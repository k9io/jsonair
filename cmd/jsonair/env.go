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
	"database/sql"
	"os"
	"strconv"

	cry "github.com/k9io/jsonair/internal/crypto"
	l "github.com/k9io/jsonair/internal/logger"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type environmentConfig struct {
	DB *sql.DB

	RunAs string

	MySQLUser          string
	MySQLPass          string
	MySQLDB            string
	MySQLTable         string
	MySQLHost          string
	MySQLPort          int
	MySQLTLS           bool
	MySQLTLSSkipVerify bool

	SyslogHost  string
	SyslogProto string

	HTTPTLS   bool
	HTTPListen string
	HTTPCert   string
	HTTPKey    string
	HTTPMode   string

	JWTTokenSecret  []byte
	JWTTokenExpire  int
	TokenHMACSecret []byte
	ConfigEncryptKey []byte
}

var Env environmentConfig

func loadEnv() {

	var err error

	if err := godotenv.Load(); err != nil {
		l.Logger(l.NOTICE, "No .env file found, using system environment variables.")
	}

	/* Sanity Checks */

	/* -- MySQL -- */

	Env.MySQLUser = os.Getenv("MYSQL_USERNAME")

	if Env.MySQLUser == "" {
		l.Logger(l.ERROR, "MYSQL_USERNAME environment variable is not set.")
		os.Exit(1)
	}

	Env.MySQLPass = os.Getenv("MYSQL_PASSWORD")

	if Env.MySQLPass == "" {
		l.Logger(l.ERROR, "MYSQL_PASSWORD environment variable is not set.")
		os.Exit(1)
	}

	Env.MySQLDB = os.Getenv("MYSQL_DATABASE")

	if Env.MySQLDB == "" {
		l.Logger(l.ERROR, "MYSQL_DATABASE environment variable is not set.")
		os.Exit(1)
	}

	Env.MySQLTable = os.Getenv("MYSQL_TABLE")

	if Env.MySQLTable == "" {
		l.Logger(l.ERROR, "MYSQL_TABLE environment variable is not set.")
		os.Exit(1)
	}

	Env.MySQLHost = os.Getenv("MYSQL_HOST")

	if Env.MySQLHost == "" {
		l.Logger(l.ERROR, "MYSQL_HOST environment variable is not set.")
		os.Exit(1)
	}

	tmp := os.Getenv("MYSQL_PORT")

	Env.MySQLPort, err = strconv.Atoi(tmp)

	if err != nil {
		l.Logger(l.ERROR, "MYSQL_PORT environment variable is not an integer.")
		os.Exit(1)
	}

	if Env.MySQLPort == 0 {
		l.Logger(l.ERROR, "MYSQL_PORT must be greater than zero.")
		os.Exit(1)
	}

	tmp = os.Getenv("MYSQL_TLS")

	if tmp == "true" {
		Env.MySQLTLS = true
	} else {
		Env.MySQLTLS = false
	}

	if os.Getenv("MYSQL_TLS_SKIP_VERIFY") == "true" {
		Env.MySQLTLSSkipVerify = true
		l.Logger(l.WARN, "MYSQL_TLS_SKIP_VERIFY is enabled — certificate validation is disabled.")
	}

	/* -- HTTP -- */

	tmp = os.Getenv("HTTP_TLS")

	if tmp == "true" {
		Env.HTTPTLS = true
	} else {
		Env.HTTPTLS = false
	}

	Env.HTTPListen = os.Getenv("HTTP_LISTEN")

	if Env.HTTPListen == "" {
		l.Logger(l.ERROR, "HTTP_LISTEN environment variable is not set.")
		os.Exit(1)
	}

	Env.HTTPCert = os.Getenv("HTTP_CERT")
	Env.HTTPKey = os.Getenv("HTTP_KEY")

	if Env.HTTPTLS {
		if Env.HTTPCert == "" {
			l.Logger(l.ERROR, "HTTP_CERT environment variable is not set.")
			os.Exit(1)
		}
		if Env.HTTPKey == "" {
			l.Logger(l.ERROR, "HTTP_KEY environment variable is not set.")
			os.Exit(1)
		}
	}

	Env.HTTPMode = os.Getenv("HTTP_MODE")

	if Env.HTTPMode == "" {
		l.Logger(l.ERROR, "HTTP_MODE environment variable is not set.")
		os.Exit(1)
	}

	if Env.HTTPMode != "release" && Env.HTTPMode != "debug" && Env.HTTPMode != "test" && Env.HTTPMode != "production" {
		l.Logger(l.ERROR, "Invalid 'HTTP_MODE':  %s.  Valid 'http_modes' are 'release', 'debug', 'test' and 'production'.", Env.HTTPMode)
		os.Exit(1)
	}

	/* -- Core stuff -- */

	Env.RunAs = os.Getenv("RUNAS")

	if Env.RunAs == "" {
		l.Logger(l.ERROR, "RUNAS environment variable is not set.")
		os.Exit(1)
	}

	tmp = os.Getenv("JWT_TOKEN_EXPIRE")

	Env.JWTTokenExpire, err = strconv.Atoi(tmp)

	if err != nil {
		l.Logger(l.ERROR, "JWT_TOKEN_EXPIRE environment variable is not an integer.")
		os.Exit(1)
	}

	if Env.JWTTokenExpire == 0 {
		l.Logger(l.ERROR, "JWT_TOKEN_EXPIRE must be greater than zero.")
		os.Exit(1)
	}

	tmp = os.Getenv("JWT_TOKEN_SECRET")

	if tmp == "" {
		l.Logger(l.ERROR, "JWT_TOKEN_SECRET environment variable is not set.")
		os.Exit(1)
	}

	Env.JWTTokenSecret = []byte(tmp)

	tmp = os.Getenv("TOKEN_HMAC_SECRET")

	if tmp == "" {
		l.Logger(l.ERROR, "TOKEN_HMAC_SECRET environment variable is not set.")
		os.Exit(1)
	}

	Env.TokenHMACSecret = []byte(tmp)

	tmp = os.Getenv("CONFIG_ENCRYPT_SECRET")

	if tmp == "" {
		l.Logger(l.ERROR, "CONFIG_ENCRYPT_SECRET environment variable is not set.")
		os.Exit(1)
	}

	Env.ConfigEncryptKey = cry.DeriveKey([]byte(tmp))

}
