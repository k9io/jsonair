package main

import (
	"fmt"
	"os"
	//	"errors"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

func SQL_Connect() {

	var err error

	connection_string := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", Env.MYSQL_USER, Env.MYSQL_PASS, Env.MYSQL_HOST, Env.MYSQL_PORT, Env.MYSQL_DB)

	Env.DB, err = sql.Open("mysql", connection_string)

	if err != nil {
		Logger(ERROR, "Cannot connect to database. %s\n", err.Error())
		os.Exit(1)
	}

}

func SQL_Auth(client_uuid string, api_key string) bool {

	var auth_check string

	err := Env.DB.QueryRow("SELECT `id` FROM `keys` WHERE `uuid`=? AND `key`=? LIMIT 1", client_uuid, api_key).Scan(&auth_check)

	if err != nil && err != sql.ErrNoRows {

		Logger(ERROR, "Cannot query SQL: %v", err.Error())
		return false
	}

	if err == sql.ErrNoRows || auth_check == "" {

		return false
	}

	return true
}

func SQL_GetConfig(uuid string, name string, jtype string) (string, error) {

	var config_json string

	err := Env.DB.QueryRow("SELECT `json` FROM `configurations` WHERE `uuid`=? AND `name`=? AND `type`=? LIMIT 1", uuid, name, jtype ).Scan(&config_json)

	if err != nil && err != sql.ErrNoRows {

		return "", fmt.Errorf("Database error: %v", err)

	}

	if err == sql.ErrNoRows {

		return "", fmt.Errorf("Configuration '%s' not found for uuid %s'", name, uuid)

	}

	return config_json, nil

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
