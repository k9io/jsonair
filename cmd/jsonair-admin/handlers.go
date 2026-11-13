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
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/k9io/jsonair/internal/configdata"
	cry "github.com/k9io/jsonair/internal/crypto"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const sessionCookie = "jsonair_admin_session"
const sessionTTL = 4 * time.Hour

func requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok, err := c.Cookie(sessionCookie)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return Cfg.SessionSecret, nil
		}, jwt.WithValidMethods([]string{"HS256"}))

		if err != nil || !token.Valid {
			// A cookie was present but is no longer valid: almost always an
			// expired session, so say so on the login page.
			setCookie(c, sessionCookie, "", -1)
			c.Redirect(http.StatusFound, "/login?expired=1")
			c.Abort()
			return
		}

		if sub, err := token.Claims.GetSubject(); err == nil {
			c.Set("user", sub)
		}
		if exp, err := token.Claims.GetExpirationTime(); err == nil && exp != nil {
			c.Set("sessionExp", exp.Time)
		}
		c.Set("csrf", csrfToken(tok))

		c.Next()
	}
}

// issueSession signs the user in. subject is shown as "Signed in as".
func issueSession(c *gin.Context, subject string) {
	claims := jwt.MapClaims{
		"sub": subject,
		"exp": time.Now().Add(sessionTTL).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString(Cfg.SessionSecret)
	setCookie(c, sessionCookie, signed, int(sessionTTL.Seconds()))
}

// decryptConfigData decrypts an encrypted config_data value and returns the raw
// (pre-base64) configuration text ready for display in a form.
func decryptConfigData(encryptedData string) (string, error) {
	decrypted, err := cry.Decrypt(encryptedData, Cfg.ConfigEncryptKey)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	// The decrypted value is a base64-encoded config. Decode it to raw text.
	// Shell-generated base64 may contain newlines, so strip whitespace before decoding.
	b64 := strings.TrimSpace(string(decrypted))
	b64 = strings.ReplaceAll(b64, "\n", "")
	b64 = strings.ReplaceAll(b64, "\r", "")

	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Fall back: return the base64 as-is so the admin can see what's stored.
		return string(decrypted), nil
	}
	return string(raw), nil
}

// encryptConfigData base64-encodes the raw config text and encrypts it for storage.
func encryptConfigData(rawConfig string) (string, error) {
	return configdata.Encode(rawConfig, Cfg.ConfigEncryptKey)
}

// --- Login / Logout ---

// credentialsMatch compares in constant time, so response timing reveals
// nothing about the configured username or password. Hashing first makes the
// inputs equal length, so their lengths do not leak either, and both checks
// always run.
func credentialsMatch(username, password string) bool {
	eq := func(a, b string) int {
		ha := sha256.Sum256([]byte(a))
		hb := sha256.Sum256([]byte(b))
		return subtle.ConstantTimeCompare(ha[:], hb[:])
	}
	return eq(username, Cfg.AdminUsername)&eq(password, Cfg.AdminPassword) == 1
}

// loginData adds which sign-in methods are enabled to the login page data.
func loginData(h gin.H) gin.H {
	h["Title"] = "Sign in"
	h["PasswordLogin"] = Cfg.PasswordLogin
	if Cfg.OIDC != nil {
		h["SSOName"] = Cfg.OIDC.ProviderName
	}
	return h
}

func getLogin(c *gin.Context) {
	render(c, http.StatusOK, "login.html", loginData(gin.H{
		"Expired": c.Query("expired") == "1",
	}))
}

func postLogin(c *gin.Context) {
	if !Cfg.PasswordLogin {
		renderLoginError(c, http.StatusNotFound, "Password sign-in is disabled. Use single sign-on.")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	if !credentialsMatch(username, password) {
		render(c, http.StatusUnauthorized, "login.html", loginData(gin.H{
			"Error":    "Invalid username or password.",
			"Username": username,
		}))
		return
	}

	issueSession(c, Cfg.AdminUsername)
	c.Redirect(http.StatusFound, "/configs")
}

// postLogout is a POST (with a CSRF token) so other sites cannot sign you out.
func postLogout(c *gin.Context) {
	setCookie(c, sessionCookie, "", -1)
	c.Redirect(http.StatusFound, "/login")
}

// --- Config List ---

func listConfigs(c *gin.Context) {
	typeFilter := strings.TrimSpace(c.Query("type"))
	nameFilter := strings.TrimSpace(c.Query("name"))

	data := gin.H{
		"Title":      "Configurations",
		"TypeFilter": typeFilter,
		"NameFilter": nameFilter,
		"Filtered":   typeFilter != "" || nameFilter != "",
		"Flash":      takeFlash(c),
		"Count":      0,
		"KeyCount":   0,
		"LoadError":  false,
	}

	configs, err := sqlListConfigs(typeFilter, nameFilter)
	if err != nil {
		data["Flash"] = errorFlash("Could not load configurations: " + err.Error())
		data["LoadError"] = true
		render(c, http.StatusInternalServerError, "configs.html", data)
		return
	}

	keys := map[string]struct{}{}
	for _, cfg := range configs {
		if cfg.UUID != "" {
			keys[cfg.UUID] = struct{}{}
		}
	}

	data["Configs"] = configs
	data["Count"] = len(configs)
	data["KeyCount"] = len(keys)
	render(c, http.StatusOK, "configs.html", data)
}

// --- Shared form handling ---

// configFromForm reads the posted configuration fields.
func configFromForm(c *gin.Context) (configRow, string) {
	return configRow{
		UUID:   c.PostForm("uuid"),
		Type:   strings.TrimSpace(c.PostForm("type")),
		Name:   strings.TrimSpace(c.PostForm("name")),
		Reload: c.PostForm("reload"),
		Debug:  c.PostForm("debug"),
	}, c.PostForm("config_data")
}

// formData builds the template data shared by the New and Edit pages.
// original is the stored row (nil on the New page); shown holds the values to
// display, which differ from original after a failed save.
func formData(title string, original *configRow, shown configRow, raw string, keys []keyRow, fl *flash) gin.H {
	known := shown.UUID == ""
	keyName := ""
	for _, k := range keys {
		if k.UUID == shown.UUID {
			known = true
		}
		if original != nil && k.UUID == original.UUID {
			keyName = k.Name
		}
	}

	return gin.H{
		"Title":     title,
		"IsEdit":    original != nil,
		"Original":  original,
		"Config":    shown,
		"RawConfig": raw,
		"Keys":      keys,
		"KeyKnown":  known,
		"KeyName":   keyName,
		"Flash":     fl,
	}
}

func validateConfigFields(cfg configRow) string {
	switch {
	case cfg.UUID == "":
		return "Choose a key for this configuration."
	case cfg.Type == "":
		return "Type is required."
	case cfg.Name == "":
		return "Name is required."
	}
	return ""
}

// --- New Config ---

func showNewConfig(c *gin.Context) {
	keys, err := sqlListKeys()
	var fl *flash
	if err != nil {
		fl = errorFlash("Could not load keys: " + err.Error())
	}
	render(c, http.StatusOK, "new.html", formData("New configuration", nil, configRow{}, "", keys, fl))
}

func createConfig(c *gin.Context) {
	cfg, rawData := configFromForm(c)
	keys, _ := sqlListKeys()

	fail := func(status int, fl *flash) {
		data := formData("New configuration", nil, cfg, rawData, keys, fl)
		data["StartDirty"] = true
		render(c, status, "new.html", data)
	}

	if msg := validateConfigFields(cfg); msg != "" {
		fail(http.StatusBadRequest, errorFlash(msg))
		return
	}

	encrypted, err := encryptConfigData(rawData)
	if err != nil {
		fail(http.StatusInternalServerError, errorFlash("Encryption failed: "+err.Error()))
		return
	}

	if err := sqlCreateConfig(cfg.UUID, cfg.Type, cfg.Name, cfg.Reload, cfg.Debug, encrypted); err != nil {
		if isDuplicateEntry(err) {
			fail(http.StatusConflict, duplicateFlash(cfg.UUID, cfg.Type, cfg.Name))
			return
		}
		fail(http.StatusInternalServerError, errorFlash("Database error: "+err.Error()))
		return
	}

	setFlash(c, "success", fmt.Sprintf("Created %s / %s.", cfg.Type, cfg.Name))
	c.Redirect(http.StatusFound, "/configs")
}

// --- Edit Config ---

// loadConfigOr404 fetches a configuration by the :id route parameter, rendering
// an error page and returning nil when it can't.
func loadConfigOr404(c *gin.Context) *configRow {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, http.StatusNotFound, "Configuration not found", "That configuration ID isn't valid.")
		return nil
	}

	row, err := sqlGetConfigByID(id)
	if errors.Is(err, sql.ErrNoRows) {
		renderError(c, http.StatusNotFound, "Configuration not found", "It may have been deleted, or the link may be wrong.")
		return nil
	}
	if err != nil {
		renderError(c, http.StatusInternalServerError, "Could not load configuration", err.Error())
		return nil
	}
	return row
}

func showEditConfig(c *gin.Context) {
	row := loadConfigOr404(c)
	if row == nil {
		return
	}

	var fl *flash
	rawConfig, decErr := decryptConfigData(row.ConfigData)
	if decErr != nil {
		fl = errorFlash("Could not decrypt the stored data (" + decErr.Error() + "). Saving will replace it with what is in the editor.")
	}

	keys, err := sqlListKeys()
	if err != nil && fl == nil {
		fl = errorFlash("Could not load keys: " + err.Error())
	}

	render(c, http.StatusOK, "edit.html", formData(row.Name, row, *row, rawConfig, keys, fl))
}

func saveEditConfig(c *gin.Context) {
	row := loadConfigOr404(c)
	if row == nil {
		return
	}

	cfg, rawData := configFromForm(c)
	keys, _ := sqlListKeys()

	fail := func(status int, fl *flash) {
		data := formData(row.Name, row, cfg, rawData, keys, fl)
		data["StartDirty"] = true
		render(c, status, "edit.html", data)
	}

	if msg := validateConfigFields(cfg); msg != "" {
		fail(http.StatusBadRequest, errorFlash(msg))
		return
	}

	encrypted, err := encryptConfigData(rawData)
	if err != nil {
		fail(http.StatusInternalServerError, errorFlash("Encryption failed: "+err.Error()))
		return
	}

	if err := sqlUpdateConfig(row.ID, cfg.UUID, cfg.Type, cfg.Name, cfg.Reload, cfg.Debug, encrypted); err != nil {
		if isDuplicateEntry(err) {
			fail(http.StatusConflict, duplicateFlash(cfg.UUID, cfg.Type, cfg.Name))
			return
		}
		fail(http.StatusInternalServerError, errorFlash("Database error: "+err.Error()))
		return
	}

	setFlash(c, "success", fmt.Sprintf("Saved %s / %s. Agents pick up the change on their next poll.", cfg.Type, cfg.Name))
	c.Redirect(http.StatusFound, "/configs")
}

// --- Delete Config ---

func deleteConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/configs")
		return
	}

	row, err := sqlGetConfigByID(id)
	if err != nil {
		setFlash(c, "error", "That configuration no longer exists.")
		c.Redirect(http.StatusFound, "/configs")
		return
	}

	if err := sqlDeleteConfig(id); err != nil {
		setFlash(c, "error", "Delete failed: "+err.Error())
		c.Redirect(http.StatusFound, "/configs")
		return
	}

	setFlash(c, "success", fmt.Sprintf("Deleted %s / %s.", row.Type, row.Name))
	c.Redirect(http.StatusFound, "/configs")
}
