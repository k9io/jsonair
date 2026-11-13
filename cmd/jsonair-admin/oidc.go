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

// Single sign-on with OpenID Connect (authorization code flow with PKCE).
//
//   GET /login/oidc     -> redirect to the identity provider
//   GET /auth/callback  <- provider redirects back with a code
//
// The callback verifies the ID token (signature, issuer, audience, expiry,
// nonce), checks the user against the configured allowlist and then issues
// the normal admin session cookie. Nothing about the user is stored.

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

const oidcStateCookie = "jsonair_admin_oidc"
const oidcStateTTL = 10 * time.Minute

type oidcConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	ProviderName string // shown on the button: "Sign in with <name>"

	AllowedEmails  map[string]bool
	AllowedDomains map[string]bool
	AllowedGroups  map[string]bool
	GroupsClaim    string
	AllowAnyUser   bool

	// HTTPClient is used for discovery, key fetches and the token exchange.
	HTTPClient *http.Client

	// The provider is discovered lazily and cached, so the admin interface
	// still starts (with SSO temporarily unavailable) if the provider is down.
	mu       sync.Mutex
	provider *oidc.Provider
}

func loadOIDCEnv() *oidcConfig {
	issuer := os.Getenv("OIDC_ISSUER")
	if issuer == "" {
		return nil
	}

	o := &oidcConfig{
		Issuer:         issuer,
		ClientID:       requireEnv("OIDC_CLIENT_ID"),
		ClientSecret:   os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:    requireEnv("OIDC_REDIRECT_URL"),
		Scopes:         splitList(os.Getenv("OIDC_SCOPES"), " ,"),
		ProviderName:   os.Getenv("OIDC_PROVIDER_NAME"),
		AllowedEmails:  lowerSet(os.Getenv("OIDC_ALLOWED_EMAILS")),
		AllowedDomains: lowerSet(os.Getenv("OIDC_ALLOWED_DOMAINS")),
		AllowedGroups:  set(os.Getenv("OIDC_ALLOWED_GROUPS")),
		GroupsClaim:    os.Getenv("OIDC_GROUPS_CLAIM"),
		AllowAnyUser:   os.Getenv("OIDC_ALLOW_ANY_USER") == "true",
		HTTPClient:     &http.Client{Timeout: 10 * time.Second},
	}

	if len(o.Scopes) == 0 {
		o.Scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}
	if o.ProviderName == "" {
		o.ProviderName = "SSO"
	}
	if o.GroupsClaim == "" {
		o.GroupsClaim = "groups"
	}

	// Refuse to start with an open door: everyone who can sign in to the
	// provider would otherwise get full admin access.
	if !o.AllowAnyUser && len(o.AllowedEmails) == 0 && len(o.AllowedDomains) == 0 && len(o.AllowedGroups) == 0 {
		fatalf("OIDC is enabled but no users are allowed: set OIDC_ALLOWED_EMAILS, OIDC_ALLOWED_DOMAINS or OIDC_ALLOWED_GROUPS (or OIDC_ALLOW_ANY_USER=true)")
	}
	return o
}

// clientContext carries the HTTP client (and its timeout) into go-oidc and
// oauth2 calls.
func (o *oidcConfig) clientContext(ctx context.Context) context.Context {
	ctx = oidc.ClientContext(ctx, o.HTTPClient)
	return context.WithValue(ctx, oauth2.HTTPClient, o.HTTPClient)
}

func (o *oidcConfig) getProvider() (*oidc.Provider, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider != nil {
		return o.provider, nil
	}
	// Background context on purpose: the provider keeps it for later
	// signing-key refreshes, so it must not be tied to one request.
	p, err := oidc.NewProvider(o.clientContext(context.Background()), o.Issuer)
	if err != nil {
		return nil, err
	}
	o.provider = p
	return p, nil
}

func (o *oidcConfig) oauth2Config(p *oidc.Provider) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  o.RedirectURL,
		Endpoint:     p.Endpoint(),
		Scopes:       o.Scopes,
	}
}

// --- Identity and authorization ---

type oidcIdentity struct {
	Subject       string
	Email         string
	EmailVerified *bool
	Username      string
	Groups        []string
}

// displayName is what the session (and the "Signed in as" label) records.
func (id oidcIdentity) displayName() string {
	switch {
	case id.Email != "":
		return id.Email
	case id.Username != "":
		return id.Username
	default:
		return id.Subject
	}
}

func identityFromClaims(sub string, claims map[string]any, groupsClaim string) oidcIdentity {
	id := oidcIdentity{Subject: sub}
	id.Email, _ = claims["email"].(string)
	id.Username, _ = claims["preferred_username"].(string)
	if v, ok := claims["email_verified"].(bool); ok {
		id.EmailVerified = &v
	}

	// Providers send groups as a list of strings or, occasionally, one string.
	switch g := claims[groupsClaim].(type) {
	case []any:
		for _, v := range g {
			if s, ok := v.(string); ok {
				id.Groups = append(id.Groups, s)
			}
		}
	case string:
		id.Groups = splitList(g, ",")
	}
	return id
}

// allowed reports whether the identity may use the admin interface.
func (o *oidcConfig) allowed(id oidcIdentity) bool {
	if o.AllowAnyUser {
		return true
	}

	for _, g := range id.Groups {
		if o.AllowedGroups[g] {
			return true
		}
	}

	// Email-based rules only trust an address the provider has not marked as
	// unverified.
	email := strings.ToLower(id.Email)
	if email == "" || (id.EmailVerified != nil && !*id.EmailVerified) {
		return false
	}
	if o.AllowedEmails[email] {
		return true
	}
	if at := strings.LastIndex(email, "@"); at != -1 && o.AllowedDomains[email[at+1:]] {
		return true
	}
	return false
}

// --- Login state (state, nonce and PKCE verifier) ---
//
// Kept in a short-lived, signed, HTTP-only cookie between the redirect to the
// provider and the callback. SameSite=Lax still sends it on the provider's
// top-level redirect back to us.

type oidcLoginState struct {
	State    string `json:"st"`
	Nonce    string `json:"nn"`
	Verifier string `json:"pv"`
	jwt.RegisteredClaims
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand failing is unrecoverable
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func saveLoginState(c *gin.Context, st oidcLoginState) {
	st.ExpiresAt = jwt.NewNumericDate(time.Now().Add(oidcStateTTL))
	signed, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, st).SignedString(Cfg.SessionSecret)
	setCookie(c, oidcStateCookie, signed, int(oidcStateTTL.Seconds()))
}

func takeLoginState(c *gin.Context) (*oidcLoginState, error) {
	raw, err := c.Cookie(oidcStateCookie)
	if err != nil || raw == "" {
		return nil, errors.New("missing login state")
	}
	setCookie(c, oidcStateCookie, "", -1) // single use

	var st oidcLoginState
	_, err = jwt.ParseWithClaims(raw, &st, func(t *jwt.Token) (any, error) {
		return Cfg.SessionSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, fmt.Errorf("invalid login state: %w", err)
	}
	return &st, nil
}

// --- Handlers ---

func renderLoginError(c *gin.Context, status int, msg string) {
	render(c, status, "login.html", loginData(gin.H{"Error": msg}))
}

// getOIDCLogin starts sign-in by redirecting to the identity provider.
func getOIDCLogin(c *gin.Context) {
	o := Cfg.OIDC
	if o == nil {
		renderError(c, http.StatusNotFound, "Page not found", "Single sign-on is not configured.")
		return
	}

	p, err := o.getProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: OIDC discovery for %s failed: %v\n", o.Issuer, err)
		renderLoginError(c, http.StatusBadGateway, "Single sign-on is unavailable right now: the identity provider could not be reached.")
		return
	}

	st := oidcLoginState{State: randomToken(), Nonce: randomToken(), Verifier: oauth2.GenerateVerifier()}
	saveLoginState(c, st)

	url := o.oauth2Config(p).AuthCodeURL(st.State,
		oidc.Nonce(st.Nonce),
		oauth2.S256ChallengeOption(st.Verifier),
	)
	c.Redirect(http.StatusFound, url)
}

// getOIDCCallback completes sign-in when the provider redirects back.
func getOIDCCallback(c *gin.Context) {
	o := Cfg.OIDC
	if o == nil {
		renderError(c, http.StatusNotFound, "Page not found", "Single sign-on is not configured.")
		return
	}

	st, stErr := takeLoginState(c)

	if e := c.Query("error"); e != "" {
		msg := "Sign-in was cancelled or refused by the identity provider"
		if d := c.Query("error_description"); d != "" {
			msg += ": " + d
		}
		renderLoginError(c, http.StatusUnauthorized, msg+".")
		return
	}

	if stErr != nil || subtle.ConstantTimeCompare([]byte(c.Query("state")), []byte(st.State)) != 1 {
		renderLoginError(c, http.StatusBadRequest, "Sign-in expired or was started in another browser. Please try again.")
		return
	}

	id, err := o.exchange(c.Request.Context(), c.Query("code"), st)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: OIDC sign-in failed: %v\n", err)
		renderLoginError(c, http.StatusUnauthorized, "Sign-in failed: the identity provider's response could not be verified.")
		return
	}

	if !o.allowed(id) {
		fmt.Fprintf(os.Stderr, "WARN: OIDC sign-in refused for %q (sub %s): not in allowlist\n", id.displayName(), id.Subject)
		renderLoginError(c, http.StatusForbidden, fmt.Sprintf("%s is not allowed to use JSONAir Admin.", id.displayName()))
		return
	}

	issueSession(c, id.displayName())
	c.Redirect(http.StatusFound, "/configs")
}

// exchange trades the authorization code for tokens and verifies the ID token.
func (o *oidcConfig) exchange(reqCtx context.Context, code string, st *oidcLoginState) (oidcIdentity, error) {
	if code == "" {
		return oidcIdentity{}, errors.New("no authorization code")
	}

	p, err := o.getProvider()
	if err != nil {
		return oidcIdentity{}, fmt.Errorf("discovery: %w", err)
	}

	ctx, cancel := context.WithTimeout(o.clientContext(reqCtx), 15*time.Second)
	defer cancel()

	tok, err := o.oauth2Config(p).Exchange(ctx, code, oauth2.VerifierOption(st.Verifier))
	if err != nil {
		return oidcIdentity{}, fmt.Errorf("token exchange: %w", err)
	}

	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		return oidcIdentity{}, errors.New("no id_token in token response")
	}

	// Checks signature, issuer, audience (our client ID) and expiry.
	idTok, err := p.Verifier(&oidc.Config{ClientID: o.ClientID}).Verify(ctx, rawID)
	if err != nil {
		return oidcIdentity{}, fmt.Errorf("verify id_token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idTok.Nonce), []byte(st.Nonce)) != 1 {
		return oidcIdentity{}, errors.New("id_token nonce mismatch")
	}

	var claims map[string]any
	if err := idTok.Claims(&claims); err != nil {
		return oidcIdentity{}, fmt.Errorf("decode claims: %w", err)
	}
	return identityFromClaims(idTok.Subject, claims, o.GroupsClaim), nil
}

// --- Small helpers ---

func splitList(s, seps string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(seps, r) }) {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func set(s string) map[string]bool {
	m := map[string]bool{}
	for _, v := range splitList(s, ",") {
		m[v] = true
	}
	return m
}

func lowerSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, v := range splitList(s, ",") {
		m[strings.ToLower(strings.TrimPrefix(v, "@"))] = true
	}
	return m
}
