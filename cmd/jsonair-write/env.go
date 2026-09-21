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

/* Secrets shorter than this are refused.  `openssl rand -hex 32` is 64 characters. */

const minSecretLen = 32

type environmentConfig struct {
	DB *sql.DB

	RunAs string

	MySQLUser          string
	MySQLPass          string
	MySQLDB            string
	MySQLHost          string
	MySQLPort          int
	MySQLTLS           bool
	MySQLTLSSkipVerify bool

	SyslogHost  string
	SyslogProto string

	HTTPTLS      bool
	HTTPListen   string
	HTTPCert     string
	HTTPKey      string
	HTTPClientCA string /* If set, clients must present a certificate signed by this CA (mTLS) */
	HTTPMode     string

	/* These are deliberately named differently than jsonair's JWT_TOKEN_SECRET and
	   TOKEN_HMAC_SECRET so that sharing one .env file between the two programs can
	   never make them use the same secret. */

	JWTTokenSecret   []byte
	JWTTokenExpire   int
	TokenHMACSecret  []byte
	ConfigEncryptKey []byte
}

var Env environmentConfig

func requireEnv(key string) string {

	v := os.Getenv(key)

	if v == "" {
		l.Logger(l.ERROR, "%s environment variable is not set.", key)
		os.Exit(1)
	}

	return v

}

func requireSecret(key string) string {

	v := requireEnv(key)

	if len(v) < minSecretLen {
		l.Logger(l.ERROR, "%s must be at least %d characters. Generate one with `openssl rand -hex 32`.", key, minSecretLen)
		os.Exit(1)
	}

	return v

}

func requirePositiveInt(key string) int {

	n, err := strconv.Atoi(requireEnv(key))

	if err != nil {
		l.Logger(l.ERROR, "%s environment variable is not an integer.", key)
		os.Exit(1)
	}

	if n <= 0 {
		l.Logger(l.ERROR, "%s must be greater than zero.", key)
		os.Exit(1)
	}

	return n

}

func loadEnv() {

	if err := godotenv.Load(); err != nil {
		l.Logger(l.NOTICE, "No .env file found, using system environment variables.")
	}

	/* -- MySQL -- */

	Env.MySQLUser = requireEnv("MYSQL_USERNAME")
	Env.MySQLPass = requireEnv("MYSQL_PASSWORD")
	Env.MySQLDB = requireEnv("MYSQL_DATABASE")
	Env.MySQLHost = requireEnv("MYSQL_HOST")
	Env.MySQLPort = requirePositiveInt("MYSQL_PORT")
	Env.MySQLTLS = os.Getenv("MYSQL_TLS") == "true"

	if os.Getenv("MYSQL_TLS_SKIP_VERIFY") == "true" {
		Env.MySQLTLSSkipVerify = true
		l.Logger(l.WARN, "MYSQL_TLS_SKIP_VERIFY is enabled — certificate validation is disabled.")
	}

	/* -- Syslog (optional) -- */

	var err error

	Env.SyslogHost, Env.SyslogProto, err = l.ParseSyslogEnv(os.Getenv("SYSLOG_HOST"), os.Getenv("SYSLOG_PROTO"))

	if err != nil {
		l.Logger(l.ERROR, "%v", err)
		os.Exit(1)
	}

	/* -- HTTP -- */

	Env.HTTPTLS = os.Getenv("HTTP_TLS") == "true"
	Env.HTTPListen = requireEnv("HTTP_LISTEN")
	Env.HTTPCert = os.Getenv("HTTP_CERT")
	Env.HTTPKey = os.Getenv("HTTP_KEY")
	Env.HTTPClientCA = os.Getenv("HTTP_CLIENT_CA")

	if Env.HTTPTLS {
		Env.HTTPCert = requireEnv("HTTP_CERT")
		Env.HTTPKey = requireEnv("HTTP_KEY")
	}

	if Env.HTTPClientCA != "" && !Env.HTTPTLS {
		l.Logger(l.ERROR, "HTTP_CLIENT_CA requires HTTP_TLS=true.")
		os.Exit(1)
	}

	if !Env.HTTPTLS {
		l.Logger(l.WARN, "HTTP_TLS is disabled. PATs and configuration data will cross the network unencrypted.")
	}

	Env.HTTPMode = requireEnv("HTTP_MODE")

	if Env.HTTPMode != "release" && Env.HTTPMode != "debug" && Env.HTTPMode != "test" && Env.HTTPMode != "production" {
		l.Logger(l.ERROR, "Invalid 'HTTP_MODE':  %s.  Valid 'http_modes' are 'release', 'debug', 'test' and 'production'.", Env.HTTPMode)
		os.Exit(1)
	}

	/* -- Core stuff -- */

	Env.RunAs = requireEnv("RUNAS")

	jwtSecret := requireSecret("WRITE_JWT_TOKEN_SECRET")
	hmacSecret := requireSecret("WRITE_TOKEN_HMAC_SECRET")
	encSecret := requireEnv("CONFIG_ENCRYPT_SECRET")

	if jwtSecret == hmacSecret || jwtSecret == encSecret || hmacSecret == encSecret {
		l.Logger(l.ERROR, "WRITE_JWT_TOKEN_SECRET, WRITE_TOKEN_HMAC_SECRET and CONFIG_ENCRYPT_SECRET must never share values.")
		os.Exit(1)
	}

	Env.JWTTokenSecret = []byte(jwtSecret)
	Env.TokenHMACSecret = []byte(hmacSecret)
	Env.JWTTokenExpire = requirePositiveInt("WRITE_JWT_TOKEN_EXPIRE")
	Env.ConfigEncryptKey = cry.DeriveKey([]byte(encSecret))

}
