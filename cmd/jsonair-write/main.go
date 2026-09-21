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

/* jsonair-write is the write-only companion to jsonair.  It can create, update
   and delete configurations, and it has no endpoint that returns configuration
   data.  It is meant to listen on a restricted/internal interface. */

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/k9io/jsonair/internal/define"
	"github.com/k9io/jsonair/internal/droppriv"
	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
)

/* A configuration is stored base64 encoded and then encrypted, which grows it by
   roughly 1.8x.  `config_data` is a mediumtext column (16 MiB). */

const maxBodyBytes = 1 << 20

func buildTLSConfig() *tls.Config {

	cert, err := tls.LoadX509KeyPair(Env.HTTPCert, Env.HTTPKey)

	if err != nil {
		l.Logger(l.ERROR, "Failed to load certificates: %v", err)
		os.Exit(1)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	/* mTLS: only callers holding a certificate signed by this CA can even reach the API. */

	if Env.HTTPClientCA != "" {

		pem, err := os.ReadFile(Env.HTTPClientCA)

		if err != nil {
			l.Logger(l.ERROR, "Failed to read HTTP_CLIENT_CA: %v", err)
			os.Exit(1)
		}

		pool := x509.NewCertPool()

		if !pool.AppendCertsFromPEM(pem) {
			l.Logger(l.ERROR, "HTTP_CLIENT_CA '%s' contains no usable certificates.", Env.HTTPClientCA)
			os.Exit(1)
		}

		tlsConfig.ClientCAs = pool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

		l.Logger(l.INFO, "Client certificates (mTLS) are required.")

	}

	return tlsConfig

}

func main() {

	l.Logger(l.BANNER, "-*> JSONAir Write! <*-")
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
		c.Header("Cache-Control", "no-store")
		if Env.HTTPTLS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	})

	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	})

	prefix := fmt.Sprintf("/api/%s/jsonair-write", define.VERSION)

	router.GET("/health", healthCheck)

	router.POST(prefix+"/auth/token", rateLimitMiddleware(), authToken)

	writeGroup := router.Group(prefix)

	writeGroup.Use(jwtMiddleware())
	{

		writeGroup.PUT("/config", putConfig)
		writeGroup.DELETE("/config", deleteConfig)

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

	ln, err := net.Listen("tcp", Env.HTTPListen)

	if err != nil {
		l.Logger(l.ERROR, "Failed to bind to port '%s': %v", Env.HTTPListen, err)
		os.Exit(1)
	}

	if Env.HTTPTLS {
		ln = tls.NewListener(ln, buildTLSConfig())
	}

	droppriv.DropPrivileges(Env.RunAs)

	l.Logger(l.INFO, "Listening on '%s' (TLS: %t) as UID: %d.", Env.HTTPListen, Env.HTTPTLS, os.Getuid())

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
