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
	"io/ioutil"
	"net"
	"net/http"
	"os"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
)

func main() {

	l.Logger(l.BANNER, "-*> JSONAir! <*-")
	l.Logger(l.BANNER, "Version: %s", Version)
	l.Logger(l.BANNER, "Champ Clark III & The Key9, Inc. Team [https://k9.io]")
	l.Logger(l.BANNER, "Copyright (C) 2026 Key9, Inc. et al.")

	LoadEnv() /* Load environment variables */

	/* Enable remote logging, if needed */

	if Env.SYSLOG_HOST != "" {
		l.Init_Logger(Env.SYSLOG_HOST, Env.SYSLOG_PROTO)
	}

	SQL_Connect()

	if Env.HTTP_MODE == "production" || Env.HTTP_MODE == "release" {

		gin.SetMode("release")
		gin.DefaultWriter = ioutil.Discard

	} else {

		gin.SetMode(Env.HTTP_MODE)

	}

	router := gin.Default()

	if Env.HTTP_MODE != "production" && Env.HTTP_MODE != "release" {
		router.Use(HTTP_Logger())
	}

	router.POST("/api/v1/jsonair/auth/token", AuthToken)

	configGroup := router.Group("/api/v1/jsonair")

	configGroup.Use(JWTMiddleware())
	{

		configGroup.GET("/config", GetConfig)
		configGroup.GET("/reload", GetReload)
		configGroup.GET("/debug", GetDebug)

	}

	if Env.HTTP_TLS == true {

		l.Logger(l.INFO, "JSONAir is up and listening for TLS traffic on %s.", Env.HTTP_LISTEN)

		cert, err := tls.LoadX509KeyPair(Env.HTTP_CERT, Env.HTTP_KEY)

		if err != nil {

			l.Logger(l.ERROR, "Failed to load certificates: %v", err)
			os.Exit(1)

		}

		tlsConfig := &tls.Config{

			Certificates: []tls.Certificate{cert},
		}

		rawListener, err := net.Listen("tcp", Env.HTTP_LISTEN)

		if err != nil {

			l.Logger(l.ERROR, "Failed to bind to port '%s': %v", Env.HTTP_LISTEN, err)
			os.Exit(1)

		}

		tlsListener := tls.NewListener(rawListener, tlsConfig)

		DropPrivileges(Env.RUNAS)

		l.Logger(l.INFO, "Listening on '%s' for TLS traffic as UID: %d.", Env.HTTP_LISTEN, os.Getuid())

		server := &http.Server{Handler: router}

		err = server.Serve(tlsListener)

		if err != nil {

			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)

		}

	} else {

		ln, err := net.Listen("tcp", Env.HTTP_LISTEN)

		if err != nil {

			l.Logger(l.ERROR, "Failed to bind to port '%s': %v", Env.HTTP_LISTEN, err)
			os.Exit(1)

		}

		DropPrivileges(Env.RUNAS)

		l.Logger(l.INFO, "Listening on '%s' for traffic as UID: %d.", Env.HTTP_LISTEN, os.Getuid())

		err = http.Serve(ln, router)

		if err != nil {

			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)

		}

	}
}
