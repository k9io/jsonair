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

	l "github.com/k9io/jsonair/internal/logger"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Environment_Struct struct {
	DB *sql.DB

	RUNAS string

	MYSQL_USER  string
	MYSQL_PASS  string
	MYSQL_DB    string
	MYSQL_TABLE string
	MYSQL_HOST  string
	MYSQL_PORT  int
	MYSQL_TLS   bool

	SYSLOG_HOST  string
	SYSLOG_PROTO string

	HTTP_TLS    bool
	HTTP_LISTEN string
	HTTP_CERT   string
	HTTP_KEY    string
	HTTP_MODE   string

	JWT_TOKEN_SECRET []byte
	JTW_TOKEN_EXPIRE int
}

var Env Environment_Struct

func LoadEnv() {

	var err error

	godotenv.Load() // Loads .env into the system environment

	/* Sanity Checks */

	/* -- MySQL -- */

	Env.MYSQL_USER = os.Getenv("MYSQL_USERNAME")

	if Env.MYSQL_USER == "" {
		l.Logger(l.ERROR, "MYSQL_USERNAME environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_PASS = os.Getenv("MYSQL_PASSWORD")

	if Env.MYSQL_PASS == "" {
		l.Logger(l.ERROR, "MYSQL_PASSWORD environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_DB = os.Getenv("MYSQL_DATABASE")

	if Env.MYSQL_DB == "" {
		l.Logger(l.ERROR, "MYSQL_DATABASE environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_TABLE = os.Getenv("MYSQL_TABLE")

	if Env.MYSQL_TABLE == "" {
		l.Logger(l.ERROR, "MYSQL_TABLE environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_HOST = os.Getenv("MYSQL_HOST")

	if Env.MYSQL_HOST == "" {
		l.Logger(l.ERROR, "MYSQL_HOST environment variable is not set.")
		os.Exit(1)
	}

	tmp := os.Getenv("MYSQL_PORT")

	Env.MYSQL_PORT, err = strconv.Atoi(tmp)

	if err != nil {

		l.Logger(l.ERROR, "MYSQL_PORT environment variable is not an integer.")
		os.Exit(1)
	}

	if Env.MYSQL_PORT == 0 {

		l.Logger(l.ERROR, "MYSQL_PORT must be greater than zero.")
		os.Exit(1)

	}

	tmp = os.Getenv("MYSQL_TLS")

	if tmp == "true" {
		Env.MYSQL_TLS = true
	} else {
		Env.MYSQL_TLS = false
	}

	/* -- HTTP -- */

	Env.HTTP_LISTEN = os.Getenv("HTTP_LISTEN")

	if Env.HTTP_LISTEN == "" {
		l.Logger(l.ERROR, "HTTP_LISTEN environment variable is not set.")
		os.Exit(1)
	}

	Env.HTTP_CERT = os.Getenv("HTTP_CERT")

	if Env.HTTP_CERT == "" {
		l.Logger(l.ERROR, "HTTP_CERT environment variable is not set.")
		os.Exit(1)
	}

	Env.HTTP_KEY = os.Getenv("HTTP_KEY")

	if Env.HTTP_KEY == "" {
		l.Logger(l.ERROR, "HTTP_KEY environment variable is not set.")
		os.Exit(1)
	}

	Env.HTTP_MODE = os.Getenv("HTTP_MODE")

	if Env.HTTP_MODE == "" {
		l.Logger(l.ERROR, "HTTP_MODE environment variable is not set.")
		os.Exit(1)
	}

	if Env.HTTP_MODE != "release" && Env.HTTP_MODE != "debug" && Env.HTTP_MODE != "test" && Env.HTTP_MODE != "production" {
		l.Logger(l.ERROR, "Invalid 'HTTP_MODE':  %s.  Valid 'http_modes' are 'release', 'debug', 'test' and 'production'.", Env.HTTP_MODE)
	}

	tmp = os.Getenv("HTTP_TLS")

	if tmp == "true" {
		Env.HTTP_TLS = true
	} else {
		Env.HTTP_TLS = false
	}

	/* -- Core stuff -- */

	Env.RUNAS = os.Getenv("RUNAS")

	if Env.RUNAS == "" {
		l.Logger(l.ERROR, "RUNAS environment variable is not set.")
		os.Exit(1)
	}

	tmp = os.Getenv("JTW_TOKEN_EXPIRE")

	Env.JTW_TOKEN_EXPIRE, err = strconv.Atoi(tmp)

	if err != nil {

		l.Logger(l.ERROR, "JTW_TOKEN_EXPIRE environment variable is not an integer.")
		os.Exit(1)
	}

	if Env.JTW_TOKEN_EXPIRE == 0 {

		l.Logger(l.ERROR, "JTW_TOKEN_EXPIRE must be greater than zero.")
		os.Exit(1)

	}

	tmp = os.Getenv("JWT_TOKEN_SECRET")

	if tmp == "" {
		l.Logger(l.ERROR, "JWT_TOKEN_SECRET environment variable is not set.")
		os.Exit(1)
	}

	Env.JWT_TOKEN_SECRET = []byte(tmp)

}
