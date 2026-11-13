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
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// None of the requests below reach the database: they hit routes that don't
// use it, or are rejected by middleware first.

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	Cfg.AdminUsername = "admin"
	Cfg.AdminPassword = "secret"
	Cfg.SessionSecret = []byte("test-session-secret")
	Cfg.HTTPTLS = false
	Cfg.PasswordLogin = true
	Cfg.OIDC = nil
	return newRouter()
}

func do(r http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func formPost(path string, vals url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// login signs in and returns the session cookie value.
func login(t *testing.T, r http.Handler) string {
	t.Helper()
	w := do(r, formPost("/login", url.Values{"username": {"admin"}, "password": {"secret"}}))
	if w.Code != http.StatusFound {
		t.Fatalf("login: status %d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			return c.Value
		}
	}
	t.Fatal("login: no session cookie")
	return ""
}

func withSession(req *http.Request, session string) *http.Request {
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
	return req
}

func validateRequest(session, headerToken, formToken string) *http.Request {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("format", "json")
	_ = mw.WriteField("data", `{"a":1}`)
	if formToken != "" {
		_ = mw.WriteField(csrfField, formToken)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/validate", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if headerToken != "" {
		req.Header.Set(csrfHeader, headerToken)
	}
	return withSession(req, session)
}

func TestSecurityHeaders(t *testing.T) {
	r := testRouter(t)
	w := do(r, httptest.NewRequest(http.MethodGet, "/login", nil))
	if got := w.Header().Get("Content-Security-Policy"); got != contentSecurityPolicy {
		t.Errorf("CSP = %q", got)
	}
	if got := w.Header().Get("Referrer-Policy"); got != "same-origin" {
		t.Errorf("Referrer-Policy = %q", got)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
}

func TestSessionCookieIsSameSiteLax(t *testing.T) {
	r := testRouter(t)
	w := do(r, formPost("/login", url.Values{"username": {"admin"}, "password": {"secret"}}))
	raw := w.Header().Get("Set-Cookie")
	for _, want := range []string{sessionCookie + "=", "HttpOnly", "SameSite=Lax"} {
		if !strings.Contains(raw, want) {
			t.Errorf("Set-Cookie %q missing %q", raw, want)
		}
	}
}

func TestCSRFRequiredForAuthenticatedPosts(t *testing.T) {
	r := testRouter(t)
	session := login(t, r)
	token := csrfToken(session)
	otherToken := csrfToken(session + "x")

	cases := []struct {
		name string
		req  *http.Request
		want int
	}{
		{"no token", validateRequest(session, "", ""), http.StatusForbidden},
		{"wrong token", validateRequest(session, otherToken, ""), http.StatusForbidden},
		{"header token", validateRequest(session, token, ""), http.StatusOK},
		{"form token", validateRequest(session, "", token), http.StatusOK},
		{"delete without token", withSession(formPost("/configs/1/delete", nil), session), http.StatusForbidden},
		{"edit without token", withSession(formPost("/configs/1/edit", url.Values{"type": {"x"}}), session), http.StatusForbidden},
		{"new without token", withSession(formPost("/configs/new", url.Values{"type": {"x"}}), session), http.StatusForbidden},
		{"logout without token", withSession(formPost("/logout", nil), session), http.StatusForbidden},
		{"logout with token", withSession(formPost("/logout", url.Values{csrfField: {token}}), session), http.StatusFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := do(r, tc.req); w.Code != tc.want {
				t.Errorf("status %d, want %d; body: %.200s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestCSRFFailureResponses(t *testing.T) {
	r := testRouter(t)
	session := login(t, r)

	// Script requests get JSON.
	w := do(r, validateRequest(session, "bad", ""))
	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res["forbidden"] != true {
		t.Errorf("JSON 403 body = %s", w.Body.String())
	}

	// Form posts get the error page.
	w = do(r, withSession(formPost("/configs/1/delete", nil), session))
	if !strings.Contains(w.Body.String(), "This page is out of date") {
		t.Errorf("HTML 403 body missing message: %.300s", w.Body.String())
	}
}

func TestCrossSiteRequestsBlocked(t *testing.T) {
	r := testRouter(t)
	session := login(t, r)

	req := formPost("/login", url.Values{"username": {"admin"}, "password": {"secret"}})
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	if w := do(r, req); w.Code != http.StatusForbidden {
		t.Errorf("cross-site login: status %d", w.Code)
	}

	// Even with a valid token.
	req = validateRequest(session, csrfToken(session), "")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	if w := do(r, req); w.Code != http.StatusForbidden {
		t.Errorf("cross-site validate: status %d", w.Code)
	}

	// Old browsers without Sec-Fetch-Site: Origin is compared with Host.
	req = validateRequest(session, csrfToken(session), "")
	req.Host = "admin.example.com"
	req.Header.Set("Origin", "https://evil.example.net")
	if w := do(r, req); w.Code != http.StatusForbidden {
		t.Errorf("foreign Origin: status %d", w.Code)
	}

	// Same-origin requests pass.
	req = validateRequest(session, csrfToken(session), "")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if w := do(r, req); w.Code != http.StatusOK {
		t.Errorf("same-origin validate: status %d", w.Code)
	}
}

func TestSessionEndpoint(t *testing.T) {
	r := testRouter(t)
	session := login(t, r)

	w := do(r, withSession(httptest.NewRequest(http.MethodGet, "/session", nil), session))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var res struct {
		Remaining int64  `json:"remaining"`
		CSRF      string `json:"csrf"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.CSRF != csrfToken(session) {
		t.Error("session endpoint returned the wrong CSRF token")
	}
	if res.Remaining <= 0 || res.Remaining > int64(sessionTTL.Seconds()) {
		t.Errorf("remaining = %d", res.Remaining)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q", got)
	}

	// Signed out: redirected to login.
	w = do(r, httptest.NewRequest(http.MethodGet, "/session", nil))
	if w.Code != http.StatusFound || !strings.HasPrefix(w.Header().Get("Location"), "/login") {
		t.Errorf("signed out: status %d, Location %q", w.Code, w.Header().Get("Location"))
	}
}

// TestEveryPostFormHasCSRFToken guards against a new form that forgets the
// token field.
func TestEveryPostFormHasCSRFToken(t *testing.T) {
	tmpl := loadTemplates()
	rows := sampleRows()
	orig := rows[0]

	withToken := func(h gin.H) gin.H {
		h["CSRFToken"] = "tok123"
		h["User"] = "admin"
		h["SessionRemaining"] = int64(100)
		return h
	}

	pages := map[string]gin.H{
		"configs.html": withToken(gin.H{"Title": "Configurations", "Configs": rows, "Count": 2, "KeyCount": 1}),
		"edit.html":    withToken(formData(orig.Name, &orig, orig, "x", sampleKeys(), nil)),
		"new.html":     withToken(formData("New configuration", nil, configRow{}, "", sampleKeys(), nil)),
		"error.html":   withToken(gin.H{"Title": "Oops", "Status": 404, "Heading": "Oops", "Message": "gone"}),
	}

	for page, data := range pages {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, page, data); err != nil {
			t.Fatalf("%s: %v", page, err)
		}
		out := buf.String()
		posts := strings.Count(out, `method="POST"`)
		tokens := strings.Count(out, `name="csrf_token" value="tok123"`)
		if posts == 0 || posts != tokens {
			t.Errorf("%s: %d POST forms but %d CSRF fields", page, posts, tokens)
		}
		if !strings.Contains(out, `<meta name="csrf-token" content="tok123">`) {
			t.Errorf("%s: missing csrf-token meta tag", page)
		}
	}
}

func TestCredentialsMatch(t *testing.T) {
	testRouter(t) // sets admin / secret
	cases := []struct {
		user, pass string
		want       bool
	}{
		{"admin", "secret", true},
		{"admin", "wrong", false},
		{"other", "secret", false},
		{"admin", "", false},
		{"", "", false},
		{"admin", "secretx", false},
	}
	for _, tc := range cases {
		if got := credentialsMatch(tc.user, tc.pass); got != tc.want {
			t.Errorf("credentialsMatch(%q, %q) = %v, want %v", tc.user, tc.pass, got, tc.want)
		}
	}
}
