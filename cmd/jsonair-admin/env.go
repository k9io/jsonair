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
	"fmt"
	"os"
	"strconv"

	cry "github.com/k9io/jsonair/internal/crypto"
	"github.com/joho/godotenv"
)

type adminConfig struct {
	DB *sql.DB

	AdminUsername    string
	AdminPassword    string
	SessionSecret    []byte
	ConfigEncryptKey []byte

	MySQLUser          string
	MySQLPass          string
	MySQLDB            string
	MySQLHost          string
	MySQLPort          int
	MySQLTLS           bool
	MySQLTLSSkipVerify bool

	HTTPListen string
	HTTPTLS    bool
	HTTPCert   string
	HTTPKey    string
}

var Cfg adminConfig

func loadEnv() {
	_ = godotenv.Load()

	Cfg.AdminUsername = requireEnv("ADMIN_USERNAME")
	Cfg.AdminPassword = requireEnv("ADMIN_PASSWORD")
	Cfg.SessionSecret = []byte(requireEnv("ADMIN_SESSION_SECRET"))
	Cfg.ConfigEncryptKey = cry.DeriveKey([]byte(requireEnv("CONFIG_ENCRYPT_SECRET")))

	Cfg.MySQLUser = requireEnv("MYSQL_USERNAME")
	Cfg.MySQLPass = requireEnv("MYSQL_PASSWORD")
	Cfg.MySQLDB = requireEnv("MYSQL_DATABASE")
	Cfg.MySQLHost = requireEnv("MYSQL_HOST")

	port, err := strconv.Atoi(requireEnv("MYSQL_PORT"))
	if err != nil || port == 0 {
		fatalf("MYSQL_PORT must be a non-zero integer")
	}
	Cfg.MySQLPort = port

	Cfg.MySQLTLS = os.Getenv("MYSQL_TLS") == "true"
	Cfg.MySQLTLSSkipVerify = os.Getenv("MYSQL_TLS_SKIP_VERIFY") == "true"

	Cfg.HTTPListen = os.Getenv("HTTP_LISTEN")
	if Cfg.HTTPListen == "" {
		Cfg.HTTPListen = ":8080"
	}

	Cfg.HTTPTLS = os.Getenv("HTTP_TLS") == "true"
	Cfg.HTTPCert = os.Getenv("HTTP_CERT")
	Cfg.HTTPKey = os.Getenv("HTTP_KEY")

	if Cfg.HTTPTLS {
		if Cfg.HTTPCert == "" {
			fatalf("HTTP_CERT environment variable is not set")
		}
		if Cfg.HTTPKey == "" {
			fatalf("HTTP_KEY environment variable is not set")
		}
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("%s environment variable is not set", key)
	}
	return v
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
