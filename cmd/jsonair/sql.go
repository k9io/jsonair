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
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/go-sql-driver/mysql"
)

func sqlConnect() {

	var err error

	cfg := mysql.Config{
		User:                 Env.MySQLUser,
		Passwd:               Env.MySQLPass,
		Net:                  "tcp",
		Addr:                 Env.MySQLHost,
		DBName:               Env.MySQLDB,
		AllowNativePasswords: true,
	}

	/* Enable TLS */

	if Env.MySQLTLS {

		cfg.TLS = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: Env.MySQLTLSSkipVerify,
		}

	} else {

		/* Disable TLS */

		cfg.TLSConfig = ""
		cfg.TLS = nil

	}

	Env.DB, err = sql.Open("mysql", cfg.FormatDSN())

	if err != nil {
		l.Logger(l.ERROR, "Cannot connect to database. %s\n", err.Error())
		os.Exit(1)
	}

	if err = Env.DB.Ping(); err != nil {
		l.Logger(l.ERROR, "Database unreachable: %s\n", err.Error())
		os.Exit(1)
	}

	Env.DB.SetMaxOpenConns(25)
	Env.DB.SetMaxIdleConns(5)
	Env.DB.SetConnMaxLifetime(5 * time.Minute)

}

func sqlAuth(ctx context.Context, pat string) (bool, string, string) {

	var authCheck string
	var name string
	var uuid string

	/* HMAC-SHA256 the users PAT using the server-side secret */

	mac := hmac.New(sha256.New, Env.TokenHMACSecret)
	mac.Write([]byte(pat))
	hashPat := hex.EncodeToString(mac.Sum(nil))

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := Env.DB.QueryRowContext(queryCtx, "SELECT `id`,`name`,`uuid` FROM `keys` WHERE `token`=? LIMIT 1", hashPat).Scan(&authCheck, &name, &uuid)

	if err != nil && err != sql.ErrNoRows {
		l.Logger(l.ERROR, "Cannot query SQL: %v", err.Error())
		return false, "", ""
	}

	if err == sql.ErrNoRows || authCheck == "" {
		return false, "", ""
	}

	if err := sqlUpdateLastLogin(ctx, uuid, hashPat); err != nil {
		l.Logger(l.WARN, "Failed to update last_login for uuid %s: %v", uuid, err)
	}

	return true, name, uuid
}

func sqlGetConfig(ctx context.Context, uuid string, name string, jtype string) (string, error) {

	var configData string

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := Env.DB.QueryRowContext(queryCtx, "SELECT `config_data` FROM `configurations` WHERE `uuid`=? AND `name`=? AND `type`=? LIMIT 1", uuid, name, jtype).Scan(&configData)

	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("Database error: %v", err)
	}

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("Configuration '%s' not found for uuid '%s'", name, uuid)
	}

	return configData, nil

}

func sqlGetSimple(ctx context.Context, uuid string, name string, jtype string, jaType string) (string, error) {

	var query string

	switch jaType {
	case "reload":
		query = "SELECT `reload` FROM `configurations` WHERE `uuid`=? AND `name`=? AND `type`=? LIMIT 1"
	case "debug":
		query = "SELECT `debug` FROM `configurations` WHERE `uuid`=? AND `name`=? AND `type`=? LIMIT 1"
	default:
		return "", fmt.Errorf("Invalid field '%s'", jaType)
	}

	var result string

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := Env.DB.QueryRowContext(queryCtx, query, uuid, name, jtype).Scan(&result)

	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("Database error: %v", err)
	}

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("'%s' for '%s' not found for uuid '%s'", jaType, name, uuid)
	}

	return result, nil

}

func sqlUpdateLastLogin(ctx context.Context, uuid string, hashpat string) error {

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := Env.DB.ExecContext(queryCtx, "UPDATE `keys` SET `last_login`=now() WHERE `uuid`=? AND `token`=?", uuid, hashpat)

	return err

}
