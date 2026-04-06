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

	"github.com/gin-gonic/gin"
)

func main() {

	Logger(BANNER, "-*> JSONAir! <*-")
	Logger(BANNER, "Version: %s", Version)
	Logger(BANNER, "Champ Clark III & The Key9, Inc. Team [https://k9.io]")
	Logger(BANNER, "Copyright (C) 2026 Key9, Inc. et al.")

	LoadEnv() /* Load environment variables */

	/* Enable remote logging, if needed */

	if Env.SYSLOG_HOST != "" {
		Init_Logger(Env.SYSLOG_HOST, Env.SYSLOG_PROTO)
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

	router.Use(Authenticate())

	router.POST("/config", GetConfig)
	router.POST("/debug", GetDebug)
	router.POST("/reload", GetReload)

	if Env.HTTP_TLS == true {

		Logger(INFO, "JSONAir is up and listening for TLS traffic on %s.", Env.HTTP_LISTEN)

		cert, err := tls.LoadX509KeyPair(Env.HTTP_CERT, Env.HTTP_KEY)

		if err != nil {

			Logger(ERROR, "Failed to load certificates: %v", err)
			os.Exit(1)

		}

		tlsConfig := &tls.Config{

			Certificates: []tls.Certificate{cert},
		}

		rawListener, err := net.Listen("tcp", Env.HTTP_LISTEN)

		if err != nil {

			Logger(ERROR, "Failed to bind to port '%s': %v", Env.HTTP_LISTEN, err)
			os.Exit(1)

		}

		tlsListener := tls.NewListener(rawListener, tlsConfig)

		DropPrivileges(Env.RUNAS)

		Logger(INFO, "Listening on '%s' for TLS traffic as UID: %d.", Env.HTTP_LISTEN, os.Getuid())

		server := &http.Server{Handler: router}

		err = server.Serve(tlsListener)

		if err != nil {

			Logger(ERROR, "Server failed: %v", err)
			os.Exit(1)

		}

	} else {

		ln, err := net.Listen("tcp", Env.HTTP_LISTEN)

		if err != nil {

			Logger(ERROR, "Failed to bind to port '%s': %v", Env.HTTP_LISTEN, err)
			os.Exit(1)

		}

		DropPrivileges(Env.RUNAS)

		Logger(INFO, "Listening on '%s' for traffic as UID: %d.", Env.HTTP_LISTEN, os.Getuid())

		err = http.Serve(ln, router)

		if err != nil {

			Logger(ERROR, "Server failed: %v", err)
			os.Exit(1)

		}

	}
}
