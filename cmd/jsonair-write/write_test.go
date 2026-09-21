package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cry "github.com/k9io/jsonair/internal/crypto"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
	Env.JWTTokenSecret = []byte("write-jwt-secret-write-jwt-secret-0123")
	Env.JWTTokenExpire = 15
	Env.TokenHMACSecret = []byte("write-hmac-secret")
	Env.ConfigEncryptKey = cry.DeriveKey([]byte("enc"))
}

/* --- scope --- */

func TestScopeAllows(t *testing.T) {

	tests := []struct {
		list  string
		value string
		want  bool
	}{
		{"", "anything", false},
		{" , ,", "anything", false},
		{"*", "anything", true},
		{"web", "web", true},
		{"web", "web2", false},
		{"web", "we", false},
		{"web*", "web", true},
		{"web*", "web-prod.json", true},
		{"web*", "xweb", false},
		{"db, web*", "web-1", true},
		{"db, web*", "db", true},
		{"db, web*", "cache", false},
		{"a*b", "a*b", true}, /* '*' is only special at the very end */
		{"a*b", "axb", false},
	}

	for _, tt := range tests {
		if got := scopeAllows(tt.list, tt.value); got != tt.want {
			t.Errorf("scopeAllows(%q, %q) = %v, want %v", tt.list, tt.value, got, tt.want)
		}
	}

}

func TestValidName(t *testing.T) {

	good := []string{"app.conf", "my-app_1", "A", strings.Repeat("a", 127)}
	bad := []string{"", "has space", "a/b", "../etc", "semi;colon", "üñí", "a\nb", strings.Repeat("a", 128)}

	for _, s := range good {
		if !validName(s, 127) {
			t.Errorf("validName(%q) = false, want true", s)
		}
	}

	for _, s := range bad {
		if validName(s, 127) {
			t.Errorf("validName(%q) = true, want false", s)
		}
	}

}

/* --- request parsing --- */

func TestParsePutRequest(t *testing.T) {

	tests := []struct {
		name    string
		body    string
		wantErr string /* substring; "" means success */
	}{
		{"minimal", `{"type":"t","name":"n","config_data":"x"}`, ""},
		{"with format ok", `{"type":"t","name":"n","config_data":"{\"a\":1}","format":"json"}`, ""},
		{"with format bad", `{"type":"t","name":"n","config_data":"{\"a\":","format":"json"}`, "not valid json"},
		{"unknown format", `{"type":"t","name":"n","config_data":"x","format":"toml"}`, "not valid toml"},
		{"missing type", `{"name":"n","config_data":"x"}`, "'type'"},
		{"missing name", `{"type":"t","config_data":"x"}`, "'name'"},
		{"bad type chars", `{"type":"t/../x","name":"n","config_data":"x"}`, "'type'"},
		{"bad name chars", `{"type":"t","name":"a b","config_data":"x"}`, "'name'"},
		{"missing data", `{"type":"t","name":"n"}`, "'config_data'"},
		{"empty data", `{"type":"t","name":"n","config_data":""}`, "'config_data'"},
		{"unknown field", `{"type":"t","name":"n","config_data":"x","confg":"y"}`, "invalid JSON"},
		{"uuid not accepted", `{"uuid":"other","type":"t","name":"n","config_data":"x"}`, "invalid JSON"},
		{"trailing data", `{"type":"t","name":"n","config_data":"x"} {}`, "unexpected data"},
		{"not json", `hello`, "invalid JSON"},
		{"reload too long", `{"type":"t","name":"n","config_data":"x","reload":"` + strings.Repeat("r", 256) + `"}`, "'reload'"},
		{"debug too long", `{"type":"t","name":"n","config_data":"x","debug":"` + strings.Repeat("d", 129) + `"}`, "'debug'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := parsePutRequest(strings.NewReader(tt.body))

			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error %q does not contain %q", err, tt.wantErr)
			}

		})
	}

}

/* --- JWT middleware --- */

func testRouter() *gin.Engine {

	r := gin.New()
	g := r.Group("/x")
	g.Use(jwtMiddleware())

	g.PUT("/config", putConfig)
	g.DELETE("/config", deleteConfig)
	g.GET("/who", func(c *gin.Context) { c.String(http.StatusOK, c.MustGet("claims").(*claims).UUID) })

	return r

}

func do(r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {

	req := httptest.NewRequest(method, path, strings.NewReader(body))

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w

}

func signed(t *testing.T, secret []byte, method jwt.SigningMethod, cl *claims) string {

	t.Helper()

	s, err := jwt.NewWithClaims(method, cl).SignedString(secret)

	if err != nil {
		t.Fatal(err)
	}

	return s

}

func TestJWTMiddleware(t *testing.T) {

	r := testRouter()

	valid, _, err := issueJWT(&writeKey{Name: "svc", UUID: "u-1", AllowedTypes: "*", AllowedNames: "*"})

	if err != nil {
		t.Fatal(err)
	}

	future := jwt.NewNumericDate(time.Now().Add(time.Hour))
	past := jwt.NewNumericDate(time.Now().Add(-time.Hour))

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"valid", valid, http.StatusOK},
		{"no token", "", http.StatusUnauthorized},
		{"garbage", "not.a.jwt", http.StatusUnauthorized},
		{"wrong secret", signed(t, []byte("some-other-secret-some-other-secret"), jwt.SigningMethodHS256,
			&claims{UUID: "u", ClientName: "c", RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{jwtAudience}, ExpiresAt: future}}), http.StatusUnauthorized},
		{"no audience (a read-API style token)", signed(t, Env.JWTTokenSecret, jwt.SigningMethodHS256,
			&claims{UUID: "u", ClientName: "c", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: future}}), http.StatusUnauthorized},
		{"wrong audience", signed(t, Env.JWTTokenSecret, jwt.SigningMethodHS256,
			&claims{UUID: "u", ClientName: "c", RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"jsonair"}, ExpiresAt: future}}), http.StatusUnauthorized},
		{"expired", signed(t, Env.JWTTokenSecret, jwt.SigningMethodHS256,
			&claims{UUID: "u", ClientName: "c", RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{jwtAudience}, ExpiresAt: past}}), http.StatusUnauthorized},
		{"no expiry", signed(t, Env.JWTTokenSecret, jwt.SigningMethodHS256,
			&claims{UUID: "u", ClientName: "c", RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{jwtAudience}}}), http.StatusUnauthorized},
		{"missing uuid", signed(t, Env.JWTTokenSecret, jwt.SigningMethodHS256,
			&claims{ClientName: "c", RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{jwtAudience}, ExpiresAt: future}}), http.StatusUnauthorized},
		{"other HMAC alg", signed(t, Env.JWTTokenSecret, jwt.SigningMethodHS512,
			&claims{UUID: "u", ClientName: "c", RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{jwtAudience}, ExpiresAt: future}}), http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := do(r, "GET", "/x/who", tt.token, ""); w.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}

	if w := do(r, "GET", "/x/who", valid, ""); w.Body.String() != "u-1" {
		t.Errorf("uuid in context = %q, want u-1", w.Body.String())
	}

	/* A non-Bearer scheme is rejected too. */

	req := httptest.NewRequest("GET", "/x/who", nil)
	req.Header.Set("Authorization", "Basic "+valid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Basic auth status = %d, want 401", w.Code)
	}

}

/* --- handlers: everything that is decided before the database is touched --- */

func TestHandlersRejectBeforeDatabase(t *testing.T) {

	r := testRouter()

	scoped, _, _ := issueJWT(&writeKey{Name: "svc", UUID: "u-1", AllowedTypes: "web", AllowedNames: "app-*"})

	tests := []struct {
		name   string
		method string
		body   string
		want   int
	}{
		{"put bad body", "PUT", `{`, http.StatusBadRequest},
		{"put type out of scope", "PUT", `{"type":"db","name":"app-1","config_data":"x"}`, http.StatusForbidden},
		{"put name out of scope", "PUT", `{"type":"web","name":"other","config_data":"x"}`, http.StatusForbidden},
		{"put invalid data in scope", "PUT", `{"type":"web","name":"app-1","config_data":"{","format":"json"}`, http.StatusBadRequest},
		{"delete bad body", "DELETE", `{"type":"web"}`, http.StatusBadRequest},
		{"delete type out of scope", "DELETE", `{"type":"db","name":"app-1"}`, http.StatusForbidden},
		{"delete name out of scope", "DELETE", `{"type":"web","name":"other"}`, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := do(r, tt.method, "/x/config", scoped, tt.body); w.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}

	/* Without a token nothing is reachable. */

	for _, m := range []string{"PUT", "DELETE"} {
		if w := do(r, m, "/x/config", "", `{"type":"web","name":"app-1","config_data":"x"}`); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without token = %d, want 401", m, w.Code)
		}
	}

	/* There is intentionally no way to read a configuration back. */

	if w := do(r, "GET", "/x/config", scoped, ""); w.Code != http.StatusNotFound {
		t.Errorf("GET /config = %d, want 404", w.Code)
	}

}
