/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// fakeIdP is a minimal OpenID Connect provider: discovery, JWKS, and a token
// endpoint that enforces the client secret, the code and PKCE.
type fakeIdP struct {
	srv *httptest.Server
	key *rsa.PrivateKey

	// From the latest /authorize redirect.
	nonce     string
	challenge string

	// Per-test knobs.
	claims   jwt.MapClaims // extra ID token claims
	nonceFor func(string) string
	aud      string
}

const testClientID = "jsonair-admin"
const testClientSecret = "client-secret"
const goodCode = "good-code"

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIdP{key: key}
	mux := http.NewServeMux()
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)

	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"issuer":                                f.srv.URL,
			"authorization_endpoint":                f.srv.URL + "/authorize",
			"token_endpoint":                        f.srv.URL + "/token",
			"jwks_uri":                              f.srv.URL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		b64 := base64.RawURLEncoding.EncodeToString
		writeJSON(w, 200, map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "k1", "use": "sig", "alg": "RS256",
			"n": b64(key.N.Bytes()),
			"e": b64(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		id, secret, ok := r.BasicAuth()
		if !ok {
			id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
		}
		if id != testClientID || secret != testClientSecret {
			writeJSON(w, 401, map[string]string{"error": "invalid_client"})
			return
		}
		sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
		if r.PostForm.Get("grant_type") != "authorization_code" ||
			r.PostForm.Get("code") != goodCode ||
			base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
			writeJSON(w, 400, map[string]string{"error": "invalid_grant"})
			return
		}

		nonce := f.nonce
		if f.nonceFor != nil {
			nonce = f.nonceFor(nonce)
		}
		aud := testClientID
		if f.aud != "" {
			aud = f.aud
		}
		claims := jwt.MapClaims{
			"iss": f.srv.URL, "aud": aud, "sub": "user-1", "nonce": nonce,
			"iat": time.Now().Unix(), "exp": time.Now().Add(5 * time.Minute).Unix(),
		}
		for k, v := range f.claims {
			claims[k] = v
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = "k1"
		signed, err := tok.SignedString(key)
		if err != nil {
			t.Error(err)
		}
		writeJSON(w, 200, map[string]any{
			"access_token": "at", "token_type": "Bearer", "expires_in": 300, "id_token": signed,
		})
	})

	return f
}

// oidcRouter returns a router with OIDC pointed at a fresh fake provider.
func oidcRouter(t *testing.T, configure func(*oidcConfig)) (*gin.Engine, *fakeIdP) {
	t.Helper()
	r := testRouter(t)
	idp := newFakeIdP(t)
	Cfg.OIDC = &oidcConfig{
		Issuer:         idp.srv.URL,
		ClientID:       testClientID,
		ClientSecret:   testClientSecret,
		RedirectURL:    "http://admin.test/auth/callback",
		Scopes:         []string{"openid", "email", "profile"},
		ProviderName:   "Key9",
		AllowedEmails:  lowerSet("alice@example.com"),
		AllowedDomains: map[string]bool{},
		AllowedGroups:  map[string]bool{},
		GroupsClaim:    "groups",
		HTTPClient:     idp.srv.Client(),
	}
	if configure != nil {
		configure(Cfg.OIDC)
	}
	t.Cleanup(func() { Cfg.OIDC = nil })
	return r, idp
}

// startLogin follows GET /login/oidc and returns the state and state cookie,
// recording the nonce and PKCE challenge on the fake provider.
func startLogin(t *testing.T, r http.Handler, idp *fakeIdP) (string, *http.Cookie) {
	t.Helper()
	w := do(r, httptest.NewRequest(http.MethodGet, "/login/oidc", nil))
	if w.Code != http.StatusFound {
		t.Fatalf("/login/oidc: status %d: %s", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil || !strings.HasPrefix(loc.String(), idp.srv.URL+"/authorize") {
		t.Fatalf("redirect to %q", w.Header().Get("Location"))
	}
	q := loc.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("client_id") != testClientID ||
		q.Get("redirect_uri") != "http://admin.test/auth/callback" || !strings.Contains(q.Get("scope"), "openid") ||
		q.Get("state") == "" || q.Get("nonce") == "" {
		t.Fatalf("authorize query missing parameters: %v", q)
	}
	idp.nonce, idp.challenge = q.Get("nonce"), q.Get("code_challenge")

	for _, c := range w.Result().Cookies() {
		if c.Name == oidcStateCookie {
			if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
				t.Errorf("state cookie flags: HttpOnly=%v SameSite=%v", c.HttpOnly, c.SameSite)
			}
			return q.Get("state"), c
		}
	}
	t.Fatal("no state cookie")
	return "", nil
}

func callback(r http.Handler, query string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?"+query, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return do(r, req)
}

// sessionSubject returns the "sub" of the session cookie set by w, or "".
func sessionSubject(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie && c.Value != "" {
			tok, err := jwt.Parse(c.Value, func(*jwt.Token) (any, error) { return Cfg.SessionSecret, nil })
			if err != nil {
				t.Fatalf("session cookie: %v", err)
			}
			sub, _ := tok.Claims.GetSubject()
			return sub
		}
	}
	return ""
}

func TestOIDCLoginSuccess(t *testing.T) {
	cases := []struct {
		name      string
		configure func(*oidcConfig)
		claims    jwt.MapClaims
		wantSub   string
	}{
		{"allowed email", nil,
			jwt.MapClaims{"email": "Alice@Example.com", "email_verified": true}, "Alice@Example.com"},
		{"email without email_verified claim", nil,
			jwt.MapClaims{"email": "alice@example.com"}, "alice@example.com"},
		{"allowed domain", func(o *oidcConfig) { o.AllowedEmails = map[string]bool{}; o.AllowedDomains = lowerSet("@example.com") },
			jwt.MapClaims{"email": "bob@EXAMPLE.com", "email_verified": true}, "bob@EXAMPLE.com"},
		{"allowed group, no email", func(o *oidcConfig) { o.AllowedGroups = set("jsonair-admins") },
			jwt.MapClaims{"preferred_username": "carol", "groups": []string{"staff", "jsonair-admins"}}, "carol"},
		{"custom groups claim", func(o *oidcConfig) { o.AllowedGroups = set("ops"); o.GroupsClaim = "roles" },
			jwt.MapClaims{"roles": []string{"ops"}}, "user-1"},
		{"allow any user", func(o *oidcConfig) { o.AllowAnyUser = true },
			jwt.MapClaims{"email": "dave@elsewhere.org"}, "dave@elsewhere.org"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, idp := oidcRouter(t, tc.configure)
			idp.claims = tc.claims
			state, cookie := startLogin(t, r, idp)

			w := callback(r, "code="+goodCode+"&state="+url.QueryEscape(state), cookie)
			if w.Code != http.StatusFound || w.Header().Get("Location") != "/configs" {
				t.Fatalf("status %d, Location %q: %.300s", w.Code, w.Header().Get("Location"), w.Body.String())
			}
			if got := sessionSubject(t, w); got != tc.wantSub {
				t.Errorf("session subject = %q, want %q", got, tc.wantSub)
			}
		})
	}
}

func TestOIDCLoginRejected(t *testing.T) {
	type tweak struct {
		configure func(*oidcConfig)
		idp       func(*fakeIdP)
		query     func(state string) string
		noCookie  bool
	}
	good := func(state string) string { return "code=" + goodCode + "&state=" + url.QueryEscape(state) }

	cases := []struct {
		name   string
		tw     tweak
		status int
		body   string
	}{
		{"email not allowed", tweak{idp: func(f *fakeIdP) { f.claims = jwt.MapClaims{"email": "mallory@example.com"} }},
			http.StatusForbidden, "mallory@example.com is not allowed"},
		{"unverified email", tweak{idp: func(f *fakeIdP) { f.claims = jwt.MapClaims{"email": "alice@example.com", "email_verified": false} }},
			http.StatusForbidden, "not allowed"},
		{"lookalike domain", tweak{
			configure: func(o *oidcConfig) { o.AllowedEmails = map[string]bool{}; o.AllowedDomains = lowerSet("example.com") },
			idp:       func(f *fakeIdP) { f.claims = jwt.MapClaims{"email": "eve@evilexample.com"} }},
			http.StatusForbidden, "not allowed"},
		{"wrong group", tweak{
			configure: func(o *oidcConfig) { o.AllowedEmails = map[string]bool{}; o.AllowedGroups = set("admins") },
			idp:       func(f *fakeIdP) { f.claims = jwt.MapClaims{"groups": []string{"staff"}} }},
			http.StatusForbidden, "not allowed"},
		{"wrong state", tweak{query: func(string) string { return "code=" + goodCode + "&state=forged" }},
			http.StatusBadRequest, "Sign-in expired"},
		{"missing state cookie", tweak{noCookie: true},
			http.StatusBadRequest, "Sign-in expired"},
		{"nonce mismatch", tweak{idp: func(f *fakeIdP) {
			f.claims = jwt.MapClaims{"email": "alice@example.com"}
			f.nonceFor = func(string) string { return "replayed-nonce" }
		}}, http.StatusUnauthorized, "could not be verified"},
		{"wrong audience", tweak{idp: func(f *fakeIdP) {
			f.claims = jwt.MapClaims{"email": "alice@example.com"}
			f.aud = "some-other-app"
		}}, http.StatusUnauthorized, "could not be verified"},
		{"bad code", tweak{query: func(state string) string { return "code=stolen&state=" + url.QueryEscape(state) }},
			http.StatusUnauthorized, "could not be verified"},
		{"wrong client secret", tweak{configure: func(o *oidcConfig) { o.ClientSecret = "nope" }},
			http.StatusUnauthorized, "could not be verified"},
		{"provider error", tweak{query: func(state string) string {
			return "error=access_denied&error_description=User+cancelled&state=" + url.QueryEscape(state)
		}}, http.StatusUnauthorized, "User cancelled"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, idp := oidcRouter(t, tc.tw.configure)
			idp.claims = jwt.MapClaims{"email": "alice@example.com"}
			if tc.tw.idp != nil {
				tc.tw.idp(idp)
			}
			state, cookie := startLogin(t, r, idp)
			if tc.tw.noCookie {
				cookie = nil
			}
			q := good(state)
			if tc.tw.query != nil {
				q = tc.tw.query(state)
			}

			w := callback(r, q, cookie)
			if w.Code != tc.status {
				t.Errorf("status %d, want %d", w.Code, tc.status)
			}
			if !strings.Contains(w.Body.String(), tc.body) {
				t.Errorf("body missing %q: %.400s", tc.body, w.Body.String())
			}
			if sub := sessionSubject(t, w); sub != "" {
				t.Errorf("session issued for %q", sub)
			}
		})
	}
}

func TestOIDCStateIsSingleUse(t *testing.T) {
	r, idp := oidcRouter(t, nil)
	idp.claims = jwt.MapClaims{"email": "alice@example.com"}
	state, cookie := startLogin(t, r, idp)
	q := "code=" + goodCode + "&state=" + url.QueryEscape(state)

	w := callback(r, q, cookie)
	if w.Code != http.StatusFound {
		t.Fatalf("first callback: status %d", w.Code)
	}
	// The response must clear the state cookie.
	cleared := false
	for _, c := range w.Result().Cookies() {
		if c.Name == oidcStateCookie && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("state cookie not cleared after use")
	}
}

func TestOIDCProviderUnreachable(t *testing.T) {
	r, idp := oidcRouter(t, nil)
	idp.srv.Close()
	w := do(r, httptest.NewRequest(http.MethodGet, "/login/oidc", nil))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "could not be reached") {
		t.Errorf("status %d: %.300s", w.Code, w.Body.String())
	}
}

func TestLoginPageMethods(t *testing.T) {
	r, _ := oidcRouter(t, nil)

	page := func() string {
		return do(r, httptest.NewRequest(http.MethodGet, "/login", nil)).Body.String()
	}

	// SSO and password both enabled.
	out := page()
	if !strings.Contains(out, `href="/login/oidc"`) || !strings.Contains(out, "Sign in with Key9") || !strings.Contains(out, `action="/login"`) {
		t.Errorf("expected SSO button and password form: %.500s", out)
	}

	// SSO only: no password form, and password posts are refused.
	Cfg.PasswordLogin = false
	out = page()
	if strings.Contains(out, `action="/login"`) || !strings.Contains(out, "Sign in with Key9") {
		t.Errorf("expected SSO button only: %.500s", out)
	}
	w := do(r, formPost("/login", url.Values{"username": {"admin"}, "password": {"secret"}}))
	if w.Code != http.StatusNotFound || sessionSubject(t, w) != "" {
		t.Errorf("password login while disabled: status %d", w.Code)
	}
}

func TestOIDCAllowed(t *testing.T) {
	yes, no := true, false
	o := &oidcConfig{
		AllowedEmails:  lowerSet("Alice@Example.com"),
		AllowedDomains: lowerSet("corp.example"),
		AllowedGroups:  set("admins"),
	}
	cases := []struct {
		id   oidcIdentity
		want bool
	}{
		{oidcIdentity{Email: "alice@example.com"}, true},
		{oidcIdentity{Email: "alice@example.com", EmailVerified: &yes}, true},
		{oidcIdentity{Email: "alice@example.com", EmailVerified: &no}, false},
		{oidcIdentity{Email: "x@corp.example"}, true},
		{oidcIdentity{Email: "x@sub.corp.example"}, false},
		{oidcIdentity{Email: "x@corp.example.evil"}, false},
		{oidcIdentity{Groups: []string{"admins"}}, true},
		{oidcIdentity{Groups: []string{"Admins"}}, false}, // group names are exact
		{oidcIdentity{Subject: "only-sub"}, false},
	}
	for _, tc := range cases {
		if got := o.allowed(tc.id); got != tc.want {
			t.Errorf("allowed(%+v) = %v, want %v", tc.id, got, tc.want)
		}
	}
}
