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
	"crypto/tls"
	"database/sql"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

type configRow struct {
	ID         int
	UUID       string
	Type       string
	Name       string
	Reload     string
	Debug      string
	ConfigData string
	Created    string
	Updated    string

	// KeyName is the name of the matching row in `keys` ("" when none matches).
	KeyName string
	// UpdatedAgo is the age of `updated` in seconds, computed by the database
	// so the result does not depend on time zone settings.
	UpdatedAgo int64
}

type keyRow struct {
	UUID string
	Name string
}

func sqlConnect() {
	cfg := mysql.Config{
		User:                 Cfg.MySQLUser,
		Passwd:               Cfg.MySQLPass,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(Cfg.MySQLHost, strconv.Itoa(Cfg.MySQLPort)),
		DBName:               Cfg.MySQLDB,
		AllowNativePasswords: true,
	}

	if Cfg.MySQLTLS {
		cfg.TLS = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: Cfg.MySQLTLSSkipVerify,
		}
	}

	// NewConnector uses the Config struct directly, so cfg.TLS is honoured.
	// sql.Open("mysql", cfg.FormatDSN()) does not work for custom TLS configs
	// because FormatDSN only serialises cfg.TLSConfig (the string name), not
	// cfg.TLS (the *tls.Config), so TLS would be silently skipped.
	connector, err := mysql.NewConnector(&cfg)
	if err != nil {
		fatalf("cannot create database connector: %v", err)
	}
	Cfg.DB = sql.OpenDB(connector)
	if err = Cfg.DB.Ping(); err != nil {
		fatalf("database unreachable: %v", err)
	}
	Cfg.DB.SetMaxOpenConns(10)
	Cfg.DB.SetMaxIdleConns(3)
	Cfg.DB.SetConnMaxLifetime(5 * time.Minute)
}

func sqlListConfigs(typeFilter, nameFilter string) ([]configRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := "SELECT c.`id`, c.`uuid`, c.`type`, c.`name`, c.`reload`, c.`debug`, c.`updated`," +
		" COALESCE((SELECT k.`name` FROM `keys` k WHERE k.`uuid` = c.`uuid` ORDER BY k.`id` LIMIT 1), '')," +
		" TIMESTAMPDIFF(SECOND, c.`updated`, NOW())" +
		" FROM `configurations` c WHERE 1=1"
	args := []any{}
	if typeFilter != "" {
		query += " AND c.`type` LIKE ?"
		args = append(args, "%"+typeFilter+"%")
	}
	if nameFilter != "" {
		query += " AND c.`name` LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}
	query += " ORDER BY c.`type`, c.`name`"

	rows, err := Cfg.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []configRow
	for rows.Next() {
		var r configRow
		if err := rows.Scan(&r.ID, &r.UUID, &r.Type, &r.Name, &r.Reload, &r.Debug, &r.Updated, &r.KeyName, &r.UpdatedAgo); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func sqlGetConfigByID(id int) (*configRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var r configRow
	err := Cfg.DB.QueryRowContext(ctx,
		"SELECT `id`,`uuid`,`type`,`name`,`reload`,`debug`,`config_data`,`created`,`updated`,"+
			" TIMESTAMPDIFF(SECOND, `updated`, NOW()) FROM `configurations` WHERE `id`=?",
		id,
	).Scan(&r.ID, &r.UUID, &r.Type, &r.Name, &r.Reload, &r.Debug, &r.ConfigData, &r.Created, &r.Updated, &r.UpdatedAgo)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// sqlFindConfigID returns the ID of the configuration with the given unique
// (uuid, type, name) combination.
func sqlFindConfigID(uuid, typ, name string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var id int
	err := Cfg.DB.QueryRowContext(ctx,
		"SELECT `id` FROM `configurations` WHERE `uuid`=? AND `type`=? AND `name`=? LIMIT 1",
		uuid, typ, name,
	).Scan(&id)
	return id, err
}

func sqlUpdateConfig(id int, uuid, typ, name, reload, debug, encryptedData string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Cfg.DB.ExecContext(ctx,
		"UPDATE `configurations` SET `uuid`=?, `type`=?, `name`=?, `reload`=?, `debug`=?, `config_data`=?, `updated`=NOW() WHERE `id`=?",
		uuid, typ, name, reload, debug, encryptedData, id,
	)
	return err
}

func sqlCreateConfig(uuid, typ, name, reload, debug, encryptedData string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Cfg.DB.ExecContext(ctx,
		"INSERT INTO `configurations` (`uuid`,`type`,`name`,`reload`,`debug`,`config_data`,`created`,`updated`) VALUES (?,?,?,?,?,?,NOW(),NOW())",
		uuid, typ, name, reload, debug, encryptedData,
	)
	return err
}

func sqlDeleteConfig(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Cfg.DB.ExecContext(ctx, "DELETE FROM `configurations` WHERE `id`=?", id)
	return err
}

func sqlListKeys() ([]keyRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := Cfg.DB.QueryContext(ctx, "SELECT `uuid`,`name` FROM `keys` ORDER BY `name`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []keyRow
	for rows.Next() {
		var k keyRow
		if err := rows.Scan(&k.UUID, &k.Name); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
