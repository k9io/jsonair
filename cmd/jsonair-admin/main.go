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
	"embed"
	"html/template"
	"net/http"

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
		c.Next()
	})

	tmpl := template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))
	router.SetHTMLTemplate(tmpl)

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
	}

	if err := router.Run(Cfg.HTTPListen); err != nil {
		fatalf("server error: %v", err)
	}
}
