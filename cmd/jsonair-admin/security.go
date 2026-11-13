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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// contentSecurityPolicy allows only this origin's own files. The pages use no
// inline scripts, inline event handlers or style attributes, and load nothing
// from other sites.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"img-src 'self'; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'none'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

func securityHeaders(c *gin.Context) {
	c.Header("Content-Security-Policy", contentSecurityPolicy)
	c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-XSS-Protection", "1; mode=block")
	c.Header("Referrer-Policy", "same-origin")
	if Cfg.HTTPTLS {
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
	c.Next()
}

// setCookie sets an HTTP-only cookie with SameSite=Lax, so browsers do not send
// it on cross-site POSTs or on requests embedded in other sites' pages.
func setCookie(c *gin.Context, name, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", Cfg.HTTPTLS, true)
}

// --- Cross-origin request protection ---
//
// Rejects state-changing requests that a browser marks as cross-site, using
// Sec-Fetch-Site (sent by all current browsers) or, failing that, by
// comparing Origin with Host. GET, HEAD and OPTIONS are always allowed.

var crossOrigin = http.NewCrossOriginProtection()

func crossOriginGuard(c *gin.Context) {
	if err := crossOrigin.Check(c.Request); err != nil {
		rejectForbidden(c, "Request blocked", "This request came from another website, so it was refused.")
		return
	}
	c.Next()
}

// --- CSRF tokens ---
//
// Every signed-in, state-changing request must also carry a token tied to the
// current session cookie: an HMAC of the cookie under ADMIN_SESSION_SECRET.
// Signing in again issues a new cookie, and so a new token.

const csrfField = "csrf_token"
const csrfHeader = "X-CSRF-Token"

func csrfToken(session string) string {
	m := hmac.New(sha256.New, Cfg.SessionSecret)
	m.Write([]byte("jsonair-admin-csrf\x00"))
	m.Write([]byte(session))
	return hex.EncodeToString(m.Sum(nil))
}

// requireCSRF must run after requireAuth, which stores the expected token.
func requireCSRF(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		c.Next()
		return
	}

	expected := c.GetString("csrf")
	got := c.GetHeader(csrfHeader)
	if got == "" {
		got = c.PostForm(csrfField)
	}

	if expected == "" || !hmac.Equal([]byte(got), []byte(expected)) {
		rejectForbidden(c, "This page is out of date",
			"Your session changed after this page was loaded, so the request was refused. Go back, reload the page and try again.")
		return
	}
	c.Next()
}

// rejectForbidden answers 403 as JSON for script requests and as an error page
// otherwise.
func rejectForbidden(c *gin.Context, heading, message string) {
	if c.GetHeader(csrfHeader) != "" || c.GetHeader("Accept") == "application/json" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"valid": false, "forbidden": true, "error": message})
		return
	}
	renderError(c, http.StatusForbidden, heading, message)
	c.Abort()
}

// getSession reports the time left in the session and the current CSRF token.
// Pages call it to refresh both, for example after the user signs in again in
// another tab.
func getSession(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	remaining := int64(0)
	if exp, ok := c.Get("sessionExp"); ok {
		remaining = sessionRemaining(exp.(time.Time))
	}
	c.JSON(http.StatusOK, gin.H{
		"remaining": remaining,
		"csrf":      c.GetString("csrf"),
	})
}
