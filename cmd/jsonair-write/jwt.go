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
	"net/http"
	"strings"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

/* Tokens issued by jsonair-write carry this audience.  jsonair (the read API)
   issues tokens without one, so even if the two ever shared a signing secret a
   read token would be rejected here. */

const jwtAudience = "jsonair-write"

type claims struct {
	UUID         string `json:"uuid"`
	ClientName   string `json:"client_name"`
	AllowedTypes string `json:"allowed_types"`
	AllowedNames string `json:"allowed_names"`
	jwt.RegisteredClaims
}

func issueJWT(k *writeKey) (string, time.Duration, error) {

	ttl := time.Duration(Env.JWTTokenExpire) * time.Minute
	now := time.Now()

	cl := &claims{
		UUID:         k.UUID,
		ClientName:   k.Name,
		AllowedTypes: k.AllowedTypes,
		AllowedNames: k.AllowedNames,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, cl).SignedString(Env.JWTTokenSecret)

	return signed, ttl, err

}

func jwtMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if !strings.HasPrefix(authHeader, "Bearer ") {

			l.Logger(l.ERROR, "%s didn't send a Bearer token.", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
			return

		}

		cl := &claims{}

		token, err := jwt.ParseWithClaims(strings.TrimPrefix(authHeader, "Bearer "), cl,
			func(t *jwt.Token) (interface{}, error) { return Env.JWTTokenSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithAudience(jwtAudience),
			jwt.WithExpirationRequired())

		if err != nil || !token.Valid {

			l.Logger(l.NOTICE, "Invalid or expired token from %s", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
			return

		}

		if cl.UUID == "" || cl.ClientName == "" {

			l.Logger(l.NOTICE, "Token missing required claims from %s", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
			return

		}

		c.Set("claims", cl)

		c.Next()

	}
}

func authToken(c *gin.Context) {

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {

		l.Logger(l.ERROR, "%s sent a request missing data.", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing data"})
		return

	}

	k, err := sqlAuth(c.Request.Context(), req.Token)

	if err != nil {

		l.Logger(l.ERROR, "Cannot query SQL: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return

	}

	if k == nil {

		l.Logger(l.NOTICE, "%s sent an unknown write token.", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		return

	}

	tokenString, ttl, err := issueJWT(k)

	if err != nil {

		l.Logger(l.ERROR, "Could not generate a session for %s.", c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate session"})
		return

	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenString,
		"expires_in":   int(ttl.Seconds()),
	})

	l.Logger(l.INFO, "Got new write access token for %s [%s] from %s.", k.UUID, k.Name, c.ClientIP())

}
