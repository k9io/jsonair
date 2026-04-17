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

type Claims struct {
	UUID        string `json:"uuid"`
	Client_Name string `json:"client_name"`
	jwt.RegisteredClaims
}

func JWTMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {

			l.Logger(l.ERROR, "%s didn't send a Bearer token.", c.ClientIP())

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})

			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return Env.JWT_TOKEN_SECRET, nil
		})

		if err != nil || !token.Valid {

			l.Logger(l.NOTICE, "Invalid or expired token from %s", c.ClientIP())

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
			return
		}

		l.Logger(l.NOTICE, "Authentication success for %s [%s] from %s.", c.ClientIP(), claims.UUID, claims.Client_Name)

		c.Set("uuid", claims.UUID)
		c.Set("client_name", claims.Client_Name)

		c.Next()
	}
}

func AuthToken(c *gin.Context) {

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	err := c.ShouldBindJSON(&req)

	if err != nil {

		l.Logger(l.ERROR, "%s sent a request missing data.", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing data"})
		return
	}

	auth_check, client_name, uuid := SQL_Auth(req.Token)

	if auth_check == false {

		l.Logger(l.NOTICE, "%s session expired.", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		return

	}

	/* Create the short-lived JWT */

	expirationTime := time.Now().Add(time.Duration(Env.JTW_TOKEN_EXPIRE) * time.Minute)

	claims := &Claims{
		UUID:        uuid,
		Client_Name: client_name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(Env.JWT_TOKEN_SECRET)

	if err != nil {

		l.Logger(l.ERROR, "Could not generate a sessions for %s.", c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenString,
		"expires_in":   Env.JTW_TOKEN_EXPIRE, // In seconds
	})

	l.Logger(l.INFO, "Got new access token for %s [%s] from %s.", uuid, client_name, c.ClientIP())

}
