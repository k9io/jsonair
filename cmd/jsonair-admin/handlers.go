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
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Next()
	}
}

func issueSession(c *gin.Context) {
	claims := jwt.MapClaims{
		"sub": Cfg.AdminUsername,
		"exp": time.Now().Add(sessionTTL).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString(Cfg.SessionSecret)
	c.SetCookie(sessionCookie, signed, int(sessionTTL.Seconds()), "/", "", false, true)
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
	b64 := base64.StdEncoding.EncodeToString([]byte(rawConfig))
	return cry.Encrypt([]byte(b64), Cfg.ConfigEncryptKey)
}

// --- Login / Logout ---

func getLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{"Error": ""})
}

func postLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username != Cfg.AdminUsername || password != Cfg.AdminPassword {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Error": "Invalid username or password."})
		return
	}

	issueSession(c)
	c.Redirect(http.StatusFound, "/configs")
}

func getLogout(c *gin.Context) {
	c.SetCookie(sessionCookie, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// --- Config List ---

func listConfigs(c *gin.Context) {
	typeFilter := c.Query("type")
	nameFilter := c.Query("name")
	msg := c.Query("msg")

	configs, err := sqlListConfigs(typeFilter, nameFilter)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "configs.html", gin.H{
			"Configs":    nil,
			"TypeFilter": typeFilter,
			"NameFilter": nameFilter,
			"Message":    "",
			"Error":      err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "configs.html", gin.H{
		"Configs":    configs,
		"TypeFilter": typeFilter,
		"NameFilter": nameFilter,
		"Message":    msg,
		"Error":      "",
	})
}

// --- New Config ---

func showNewConfig(c *gin.Context) {
	keys, err := sqlListKeys()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new.html", gin.H{
			"Keys":  nil,
			"Error": err.Error(),
		})
		return
	}
	c.HTML(http.StatusOK, "new.html", gin.H{
		"Keys":  keys,
		"Error": "",
	})
}

func createConfig(c *gin.Context) {
	uuid := c.PostForm("uuid")
	typ := c.PostForm("type")
	name := c.PostForm("name")
	reload := c.PostForm("reload")
	dbg := c.PostForm("debug")
	rawData := c.PostForm("config_data")

	encrypted, err := encryptConfigData(rawData)
	if err != nil {
		keys, _ := sqlListKeys()
		c.HTML(http.StatusInternalServerError, "new.html", gin.H{
			"Keys":  keys,
			"Error": "Encryption failed: " + err.Error(),
		})
		return
	}

	if err := sqlCreateConfig(uuid, typ, name, reload, dbg, encrypted); err != nil {
		keys, _ := sqlListKeys()
		c.HTML(http.StatusInternalServerError, "new.html", gin.H{
			"Keys":  keys,
			"Error": "Database error: " + err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/configs?msg=Configuration+created+successfully.")
}

// --- Edit Config ---

func showEditConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/configs")
		return
	}

	row, err := sqlGetConfigByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "edit.html", gin.H{
			"Config":    nil,
			"RawConfig": "",
			"Keys":      nil,
			"Error":     "Configuration not found.",
		})
		return
	}

	rawConfig, decErr := decryptConfigData(row.ConfigData)
	decryptWarning := ""
	if decErr != nil {
		decryptWarning = "Could not decrypt existing data: " + decErr.Error()
	}

	keys, _ := sqlListKeys()

	c.HTML(http.StatusOK, "edit.html", gin.H{
		"Config":         row,
		"RawConfig":      rawConfig,
		"Keys":           keys,
		"Error":          decryptWarning,
	})
}

func saveEditConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/configs")
		return
	}

	uuid := c.PostForm("uuid")
	typ := c.PostForm("type")
	name := c.PostForm("name")
	reload := c.PostForm("reload")
	dbg := c.PostForm("debug")
	rawData := c.PostForm("config_data")

	encrypted, err := encryptConfigData(rawData)
	if err != nil {
		row, _ := sqlGetConfigByID(id)
		keys, _ := sqlListKeys()
		c.HTML(http.StatusInternalServerError, "edit.html", gin.H{
			"Config":    row,
			"RawConfig": rawData,
			"Keys":      keys,
			"Error":     "Encryption failed: " + err.Error(),
		})
		return
	}

	if err := sqlUpdateConfig(id, uuid, typ, name, reload, dbg, encrypted); err != nil {
		row, _ := sqlGetConfigByID(id)
		keys, _ := sqlListKeys()
		c.HTML(http.StatusInternalServerError, "edit.html", gin.H{
			"Config":    row,
			"RawConfig": rawData,
			"Keys":      keys,
			"Error":     "Database error: " + err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/configs?msg=Configuration+saved+successfully.")
}

// --- Delete Config ---

func deleteConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/configs")
		return
	}

	if err := sqlDeleteConfig(id); err != nil {
		c.Redirect(http.StatusFound, "/configs?msg=Delete+failed:+"+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/configs?msg=Configuration+deleted.")
}
