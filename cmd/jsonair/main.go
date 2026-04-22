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
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
)

func main() {

	l.Logger(l.BANNER, "-*> JSONAir! <*-")
	l.Logger(l.BANNER, "Version: %s", Version)
	l.Logger(l.BANNER, "Champ Clark III & The Key9, Inc. Team [https://k9.io]")
	l.Logger(l.BANNER, "Copyright (C) 2026 Key9, Inc. et al.")

	loadEnv()

	/* Enable remote logging, if needed */

	if Env.SyslogHost != "" {
		l.Init_Logger(Env.SyslogHost, Env.SyslogProto)
	}

	sqlConnect()

	if Env.HTTPMode == "production" || Env.HTTPMode == "release" {

		gin.SetMode("release")
		gin.DefaultWriter = io.Discard

	} else {

		gin.SetMode(Env.HTTPMode)

	}

	router := gin.New()

	router.Use(gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err any) {
		l.Logger(l.ERROR, "Panic recovered: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}))

	if Env.HTTPMode != "production" && Env.HTTPMode != "release" {
		router.Use(httpLogger())
	}

	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
		c.Next()
	})

	router.POST("/api/v1/jsonair/auth/token", rateLimitMiddleware(), authToken)

	configGroup := router.Group("/api/v1/jsonair")

	configGroup.Use(jwtMiddleware())
	{

		configGroup.GET("/config", getConfig)
		configGroup.GET("/reload", getReload)
		configGroup.GET("/debug", getDebug)

	}

	server := &http.Server{
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if Env.HTTPTLS {

		cert, err := tls.LoadX509KeyPair(Env.HTTPCert, Env.HTTPKey)

		if err != nil {

			l.Logger(l.ERROR, "Failed to load certificates: %v", err)
			os.Exit(1)

		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		rawListener, err := net.Listen("tcp", Env.HTTPListen)

		if err != nil {

			l.Logger(l.ERROR, "Failed to bind to port '%s': %v", Env.HTTPListen, err)
			os.Exit(1)

		}

		tlsListener := tls.NewListener(rawListener, tlsConfig)

		dropPrivileges(Env.RunAs)

		l.Logger(l.INFO, "Listening on '%s' for TLS traffic as UID: %d.", Env.HTTPListen, os.Getuid())

		err = server.Serve(tlsListener)

		if err != nil {

			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)

		}

	} else {

		ln, err := net.Listen("tcp", Env.HTTPListen)

		if err != nil {

			l.Logger(l.ERROR, "Failed to bind to port '%s': %v", Env.HTTPListen, err)
			os.Exit(1)

		}

		dropPrivileges(Env.RunAs)

		l.Logger(l.INFO, "Listening on '%s' for traffic as UID: %d.", Env.HTTPListen, os.Getuid())

		err = server.Serve(ln)

		if err != nil {

			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)

		}

	}
}
