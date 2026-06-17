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
	"fmt"
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
		Addr:                 fmt.Sprintf("%s:%d", Cfg.MySQLHost, Cfg.MySQLPort),
		DBName:               Cfg.MySQLDB,
		AllowNativePasswords: true,
	}

	if Cfg.MySQLTLS {
		cfg.TLS = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: Cfg.MySQLTLSSkipVerify,
		}
	}

	var err error
	Cfg.DB, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		fatalf("cannot open database: %v", err)
	}
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

	query := "SELECT `id`,`uuid`,`type`,`name`,`reload`,`debug`,`updated` FROM `configurations` WHERE 1=1"
	args := []any{}
	if typeFilter != "" {
		query += " AND `type` LIKE ?"
		args = append(args, "%"+typeFilter+"%")
	}
	if nameFilter != "" {
		query += " AND `name` LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}
	query += " ORDER BY `type`, `name`"

	rows, err := Cfg.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []configRow
	for rows.Next() {
		var r configRow
		if err := rows.Scan(&r.ID, &r.UUID, &r.Type, &r.Name, &r.Reload, &r.Debug, &r.Updated); err != nil {
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
		"SELECT `id`,`uuid`,`type`,`name`,`reload`,`debug`,`config_data`,`created`,`updated` FROM `configurations` WHERE `id`=?",
		id,
	).Scan(&r.ID, &r.UUID, &r.Type, &r.Name, &r.Reload, &r.Debug, &r.ConfigData, &r.Created, &r.Updated)
	if err != nil {
		return nil, err
	}
	return &r, nil
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
