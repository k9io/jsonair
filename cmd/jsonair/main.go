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
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	l "github.com/k9io/jsonair/internal/logger"
	"github.com/k9io/jsonair/internal/droppriv"

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

	router.SetTrustedProxies(nil)

	router.Use(gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err any) {
		l.Logger(l.ERROR, "Panic recovered: %v\n%s", err, debug.Stack())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}))

	router.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		if Env.HTTPTLS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	})

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	StartRateLimiterCleanup(ctx)

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

		droppriv.DropPrivileges(Env.RunAs)

		l.Logger(l.INFO, "Listening on '%s' for TLS traffic as UID: %d.", Env.HTTPListen, os.Getuid())

		serveErr := make(chan error, 1)
		go func() {
			if err := server.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
				serveErr <- err
			}
			close(serveErr)
		}()

		select {
		case err := <-serveErr:
			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)
		case <-ctx.Done():
			l.Logger(l.INFO, "Shutdown signal received, draining connections...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				l.Logger(l.ERROR, "Server shutdown error: %v", err)
			}
			Env.DB.Close()
			l.Logger(l.INFO, "Server stopped cleanly.")
		}

	} else {

		ln, err := net.Listen("tcp", Env.HTTPListen)

		if err != nil {

			l.Logger(l.ERROR, "Failed to bind to port '%s': %v", Env.HTTPListen, err)
			os.Exit(1)

		}

		droppriv.DropPrivileges(Env.RunAs)

		l.Logger(l.INFO, "Listening on '%s' for traffic as UID: %d.", Env.HTTPListen, os.Getuid())

		serveErr := make(chan error, 1)
		go func() {
			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				serveErr <- err
			}
			close(serveErr)
		}()

		select {
		case err := <-serveErr:
			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)
		case <-ctx.Done():
			l.Logger(l.INFO, "Shutdown signal received, draining connections...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				l.Logger(l.ERROR, "Server shutdown error: %v", err)
			}
			Env.DB.Close()
			l.Logger(l.INFO, "Server stopped cleanly.")
		}

	}
}
