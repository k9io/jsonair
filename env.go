package main

import (
	"database/sql"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	_ "github.com/go-sql-driver/mysql"
)

type Environment_Struct struct {
	DB *sql.DB

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

	AUTH_HEADER string
}

var Env Environment_Struct

func LoadEnv() {

	var err error

	godotenv.Load() // Loads .env into the system environment

	/* Sanity Checks */

	/* -- MySQL -- */

	Env.MYSQL_USER = os.Getenv("MYSQL_USERNAME")

	if Env.MYSQL_USER == "" {
		Logger(ERROR, "MYSQL_USERNAME environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_PASS = os.Getenv("MYSQL_PASSWORD")

	if Env.MYSQL_PASS == "" {
		Logger(ERROR, "MYSQL_PASSWORD environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_DB = os.Getenv("MYSQL_DATABASE")

	if Env.MYSQL_DB == "" {
		Logger(ERROR, "MYSQL_DATABASE environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_TABLE = os.Getenv("MYSQL_TABLE")

	if Env.MYSQL_TABLE == "" {
		Logger(ERROR, "MYSQL_TABLE environment variable is not set.")
		os.Exit(1)
	}

	Env.MYSQL_HOST = os.Getenv("MYSQL_HOST")

	if Env.MYSQL_HOST == "" {
		Logger(ERROR, "MYSQL_HOST environment variable is not set.")
		os.Exit(1)
	}

	tmp := os.Getenv("MYSQL_PORT")

	Env.MYSQL_PORT, err = strconv.Atoi(tmp)

	if err != nil {

		Logger(ERROR, "MYSQL_PORT environment variable is not an integer.")
		os.Exit(1)
	}

	/* -- HTTP -- */

	Env.HTTP_LISTEN = os.Getenv("HTTP_LISTEN")

	if Env.HTTP_LISTEN == "" {
		Logger(ERROR, "HTTP_LISTEN environment variable is not set.")
		os.Exit(1)
	}

	Env.HTTP_CERT = os.Getenv("HTTP_CERT")

	if Env.HTTP_CERT == "" {
		Logger(ERROR, "HTTP_CERT environment variable is not set.")
		os.Exit(1)
	}

	Env.HTTP_KEY = os.Getenv("HTTP_KEY")

	if Env.HTTP_KEY == "" {
		Logger(ERROR, "HTTP_KEY environment variable is not set.")
		os.Exit(1)
	}

	Env.HTTP_MODE = os.Getenv("HTTP_MODE")

	if Env.HTTP_MODE == "" {
		Logger(ERROR, "HTTP_MODE environment variable is not set.")
		os.Exit(1)
	}

	if Env.HTTP_MODE != "release" && Env.HTTP_MODE != "debug" && Env.HTTP_MODE != "test" && Env.HTTP_MODE != "production" {
		Logger(ERROR, "Invalid 'HTTP_MODE':  %s.  Valid 'http_modes' are 'release', 'debug', 'test' and 'production'.", Env.HTTP_MODE)
	}

	/* -- Core stuff -- */

	Env.AUTH_HEADER = os.Getenv("AUTH_HEADER")

	if Env.AUTH_HEADER == "" {
		Logger(ERROR, "AUTH_HEADER environment variable is not set.")
		os.Exit(1)
	}

}
