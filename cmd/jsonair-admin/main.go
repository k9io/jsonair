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
	"embed"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed templates
var templateFS embed.FS

func main() {
	loadEnv()
	sqlConnect()

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())

	// Limit request body to 20 MB (configs can be large)
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 20*1024*1024)
		c.Next()
	})

	router.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		if Cfg.HTTPTLS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	})

	tmpl := template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))
	router.SetHTMLTemplate(tmpl)

	router.GET("/health", healthCheck)

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/configs")
	})
	router.GET("/login", getLogin)
	router.POST("/login", postLogin)
	router.GET("/logout", getLogout)

	auth := router.Group("/")
	auth.Use(requireAuth())
	{
		auth.GET("/configs", listConfigs)
		auth.GET("/configs/new", showNewConfig)
		auth.POST("/configs/new", createConfig)
		auth.GET("/configs/:id/edit", showEditConfig)
		auth.POST("/configs/:id/edit", saveEditConfig)
		auth.POST("/configs/:id/delete", deleteConfig)
		auth.POST("/validate", validateConfig)
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

	if Cfg.HTTPTLS {
		cert, err := tls.LoadX509KeyPair(Cfg.HTTPCert, Cfg.HTTPKey)
		if err != nil {
			fatalf("failed to load TLS certificates: %v", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		rawListener, err := net.Listen("tcp", Cfg.HTTPListen)
		if err != nil {
			fatalf("failed to bind to '%s': %v", Cfg.HTTPListen, err)
		}

		tlsListener := tls.NewListener(rawListener, tlsConfig)

		serveErr := make(chan error, 1)
		go func() {
			if err := server.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
				serveErr <- err
			}
			close(serveErr)
		}()

		println("INFO: jsonair-admin listening on", Cfg.HTTPListen, "(TLS)")

		select {
		case err := <-serveErr:
			fatalf("server error: %v", err)
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			Cfg.DB.Close()
			os.Exit(0)
		}
	} else {
		ln, err := net.Listen("tcp", Cfg.HTTPListen)
		if err != nil {
			fatalf("failed to bind to '%s': %v", Cfg.HTTPListen, err)
		}

		serveErr := make(chan error, 1)
		go func() {
			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				serveErr <- err
			}
			close(serveErr)
		}()

		println("INFO: jsonair-admin listening on", Cfg.HTTPListen)

		select {
		case err := <-serveErr:
			fatalf("server error: %v", err)
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			Cfg.DB.Close()
			os.Exit(0)
		}
	}
}
