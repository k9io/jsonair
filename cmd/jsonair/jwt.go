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
	"fmt"
	"net/http"
	"strings"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type claims struct {
	UUID       string `json:"uuid"`
	ClientName string `json:"client_name"`
	jwt.RegisteredClaims
}

func jwtMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {

			l.Logger(l.ERROR, "%s didn't send a Bearer token.", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		cl := &claims{}

		token, err := jwt.ParseWithClaims(tokenStr, cl, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return Env.JWTTokenSecret, nil
		})

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

		l.Logger(l.NOTICE, "Authentication success for %s [%s] from %s.", c.ClientIP(), cl.UUID, cl.ClientName)

		c.Set("uuid", cl.UUID)
		c.Set("client_name", cl.ClientName)

		c.Next()
	}
}

func authToken(c *gin.Context) {

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	err := c.ShouldBindJSON(&req)

	if err != nil {

		l.Logger(l.ERROR, "%s sent a request missing data.", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing data"})
		return
	}

	ok, clientName, uuid := sqlAuth(c.Request.Context(), req.Token)

	if !ok {

		l.Logger(l.NOTICE, "%s session expired.", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		return

	}

	/* Create the short-lived JWT */

	expirationTime := time.Now().Add(time.Duration(Env.JWTTokenExpire) * time.Minute)

	cl := &claims{
		UUID:       uuid,
		ClientName: clientName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	tokenString, err := token.SignedString(Env.JWTTokenSecret)

	if err != nil {

		l.Logger(l.ERROR, "Could not generate a session for %s.", c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenString,
		"expires_in":   Env.JWTTokenExpire * 60, // Convert minutes to seconds (RFC 6749)
	})

	l.Logger(l.INFO, "Got new access token for %s [%s] from %s.", uuid, clientName, c.ClientIP())

}
