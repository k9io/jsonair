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
	"crypto/sha256"
	"crypto/tls"

	"fmt"
	"os"

	l "github.com/k9io/jsonair/internal/logger"

	"database/sql"
	"github.com/go-sql-driver/mysql"
)

func SQL_Connect() {

	var err error

	cfg1 := mysql.Config{
		User:                 Env.MYSQL_USER,
		Passwd:               Env.MYSQL_PASS,
		Net:                  "tcp",
		Addr:                 Env.MYSQL_HOST,
		DBName:               Env.MYSQL_DB,
		AllowNativePasswords: true,
	}

	/* Enable TLS */

	if Env.MYSQL_TLS == true {

		cfg1.TLSConfig = "skip-verify" /* Make this a config? */
		cfg1.TLS = &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS12,
		}

	} else {

		/* Disable TLS */

		cfg1.TLSConfig = ""
		cfg1.TLS = nil

	}

	Env.DB, err = sql.Open("mysql", cfg1.FormatDSN())

	if err != nil {
		l.Logger(l.ERROR, "Cannot connect to database. %s\n", err.Error())
		os.Exit(1)
	}

}

func SQL_Auth(pat string) (bool, string, string) {

	var auth_check string
	var name string
	var uuid string

	/* SHA256 the users PAT */

	hash_pat := fmt.Sprintf("%x", sha256.Sum256([]byte(pat)))

	err := Env.DB.QueryRow("SELECT `id`,`name`,`uuid` FROM `keys` WHERE `token`=? LIMIT 1", hash_pat).Scan(&auth_check, &name, &uuid)

	if err != nil && err != sql.ErrNoRows {

		l.Logger(l.ERROR, "Cannot query SQL: %v", err.Error())
		return false, "", ""
	}

	if err == sql.ErrNoRows || auth_check == "" {

		return false, "", ""
	}

	return true, name, uuid
}

func SQL_GetConfig(uuid string, name string, jtype string) (string, error) {

	var config_data string

	err := Env.DB.QueryRow("SELECT `config_data` FROM `configurations` WHERE `uuid`=? AND `name`=? AND `type`=? LIMIT 1", uuid, name, jtype).Scan(&config_data)

	if err != nil && err != sql.ErrNoRows {

		return "", fmt.Errorf("Database error: %v", err)

	}

	if err == sql.ErrNoRows {

		return "", fmt.Errorf("Configuration '%s' not found for uuid %s'", name, uuid)

	}

	return config_data, nil

}

func SQL_GetSimple(uuid string, name string, jtype string, ja_type string) (string, error) {

	var reload string

	query := fmt.Sprintf("SELECT `%s` FROM `configurations` WHERE `uuid`=? AND `name`=? AND `type`=? LIMIT 1", ja_type)

	err := Env.DB.QueryRow(query, uuid, name, jtype).Scan(&reload)

	if err != nil && err != sql.ErrNoRows {

		return "", fmt.Errorf("Database error: %v", err)

	}

	if err == sql.ErrNoRows {

		return "", fmt.Errorf("Reload for '%s' not found for uuid %s'", name, uuid)

	}

	return reload, nil

}

func SQL_Update_Last_Login(uuid string, key string) error {

	_, err := Env.DB.Exec("UPDATE `keys` SET `last_login`=now() WHERE `uuid`=? AND `key`=?", uuid, key)

	return err

}
