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
	"net"
	"os"
	"strconv"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/go-sql-driver/mysql"
)

type writeKey struct {
	Name         string
	UUID         string
	AllowedTypes string
	AllowedNames string
}

func sqlConnect() {

	cfg := mysql.Config{
		User:                 Env.MySQLUser,
		Passwd:               Env.MySQLPass,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(Env.MySQLHost, strconv.Itoa(Env.MySQLPort)),
		DBName:               Env.MySQLDB,
		AllowNativePasswords: true,
	}

	if Env.MySQLTLS {

		cfg.TLS = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: Env.MySQLTLSSkipVerify,
		}

	}

	/* See sqlConnect() in cmd/jsonair/sql.go for why NewConnector is used. */

	connector, err := mysql.NewConnector(&cfg)

	if err != nil {
		l.Logger(l.ERROR, "Cannot create database connector: %s\n", err.Error())
		os.Exit(1)
	}

	Env.DB = sql.OpenDB(connector)

	if err = Env.DB.Ping(); err != nil {
		l.Logger(l.ERROR, "Database unreachable: %s\n", err.Error())
		os.Exit(1)
	}

	Env.DB.SetMaxOpenConns(10)
	Env.DB.SetMaxIdleConns(2)
	Env.DB.SetConnMaxLifetime(5 * time.Minute)

}

func hashPAT(pat string) string {

	mac := hmac.New(sha256.New, Env.TokenHMACSecret)
	mac.Write([]byte(pat))

	return hex.EncodeToString(mac.Sum(nil))

}

/* sqlAuth looks a write PAT up in `write_keys`.  Returns nil if it is unknown. */

func sqlAuth(ctx context.Context, pat string) (*writeKey, error) {

	hashed := hashPAT(pat)

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	k := &writeKey{}

	err := Env.DB.QueryRowContext(queryCtx,
		"SELECT `name`,`uuid`,`allowed_types`,`allowed_names` FROM `write_keys` WHERE `token`=? LIMIT 1",
		hashed).Scan(&k.Name, &k.UUID, &k.AllowedTypes, &k.AllowedNames)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if _, err := Env.DB.ExecContext(queryCtx, "UPDATE `write_keys` SET `last_login`=now() WHERE `token`=?", hashed); err != nil {
		l.Logger(l.WARN, "Failed to update last_login for uuid %s: %v", k.UUID, err)
	}

	return k, nil

}

/* sqlUpsertConfig creates or replaces a configuration.  Relies on the unique key
   `idx_uuid_type_name`, which makes this atomic.  'reload' and 'debug' are only
   changed on an existing row when the caller supplied them (non-nil).

   Returns true if a new row was created.  With go-sql-driver/mysql defaults,
   RowsAffected is 1 for an INSERT and 2 for an UPDATE of an existing row. */

func sqlUpsertConfig(ctx context.Context, uuid, jtype, name string, reload, debug *string, encrypted string) (bool, error) {

	var reloadVal, debugVal string

	if reload != nil {
		reloadVal = *reload
	}

	if debug != nil {
		debugVal = *debug
	}

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := Env.DB.ExecContext(queryCtx,
		"INSERT INTO `configurations` (`uuid`,`type`,`name`,`reload`,`debug`,`config_data`,`created`,`updated`) "+
			"VALUES (?,?,?,?,?,?,NOW(),NOW()) "+
			"ON DUPLICATE KEY UPDATE `config_data`=VALUES(`config_data`), "+
			"`reload`=IF(?,VALUES(`reload`),`reload`), "+
			"`debug`=IF(?,VALUES(`debug`),`debug`), "+
			"`updated`=NOW()",
		uuid, jtype, name, reloadVal, debugVal, encrypted,
		reload != nil, debug != nil)

	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()

	if err != nil {
		return false, err
	}

	return n == 1, nil

}

/* sqlDeleteConfig removes a configuration.  Returns false if there was nothing to delete. */

func sqlDeleteConfig(ctx context.Context, uuid, jtype, name string) (bool, error) {

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := Env.DB.ExecContext(queryCtx,
		"DELETE FROM `configurations` WHERE `uuid`=? AND `type`=? AND `name`=?", uuid, jtype, name)

	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()

	if err != nil {
		return false, err
	}

	return n > 0, nil

}
