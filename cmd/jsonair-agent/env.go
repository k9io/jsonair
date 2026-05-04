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
	"log"
	"os"
	"strconv"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/joho/godotenv"
)

type Environment_Struct struct {
	RUNAS string

	JSONAIR_PAT string

	JSONAIR_URL  string
	JSONAIR_TYPE string
	JSONAIR_NAME string

	CONFIG_FILE string
	SLEEP       time.Duration
	PRUNE       int
	MASK        os.FileMode
	RELOAD_COMMAND  string
}

var Env Environment_Struct

func LoadEnv() {

	var tmp string
	var err error

        if err := godotenv.Load(); err != nil {
                l.Logger(l.NOTICE, "No .env file found, using system environment variables.")
        }

	/* Sanity Checks */

	Env.JSONAIR_PAT = os.Getenv("JSONAIR_PAT")

	if Env.JSONAIR_PAT == "" {
		l.Logger(l.ERROR, "JSONAIR_PAT environment variable is not set.")
		os.Exit(1)
	}

	Env.RUNAS = os.Getenv("RUNAS")

	if Env.RUNAS == "" {
		l.Logger(l.ERROR, "RUNAS environment variable is not set.")
		os.Exit(1)
	}

	Env.JSONAIR_URL = os.Getenv("JSONAIR_URL")

	if Env.JSONAIR_URL == "" {
		l.Logger(l.ERROR, "JSONAIR_URL environment variable is not set.")
		os.Exit(1)
	}

	Env.JSONAIR_NAME = os.Getenv("JSONAIR_NAME")

	if Env.JSONAIR_NAME == "" {
		l.Logger(l.ERROR, "JSONAIR_NAME environment variable is not set.")
		os.Exit(1)
	}

	Env.JSONAIR_TYPE = os.Getenv("JSONAIR_TYPE")

	if Env.JSONAIR_TYPE == "" {
		l.Logger(l.ERROR, "JSONAIR_TYPE environment variable is not set.")
		os.Exit(1)
	}

	Env.CONFIG_FILE = os.Getenv("CONFIG_FILE")

	if Env.CONFIG_FILE == "" {
		l.Logger(l.ERROR, "CONFIG_FILE environment variable is not set.")
		os.Exit(1)
	}

	Env.RELOAD_COMMAND = os.Getenv("RELOAD_COMMAND")

	if Env.RELOAD_COMMAND == "" {
		l.Logger(l.ERROR, "RELOAD_COMMAND environment variable is not set.")
		os.Exit(1)
	}

	tmp = os.Getenv("SLEEP")

	val, err := strconv.Atoi(tmp)

	if err != nil {

		l.Logger(l.ERROR, "SLEEP environment variable is not set.")
		os.Exit(1)
	}

	if val == 0 {

		l.Logger(l.ERROR, "SLEEP must be a non-zero number.")
		os.Exit(1)

	}

	Env.SLEEP = time.Duration(val) * time.Second

	tmp = os.Getenv("PRUNE")

	Env.PRUNE, err = strconv.Atoi(tmp)

	if err != nil {

		l.Logger(l.ERROR, "PRUNE environment variable is not an integer.")
		os.Exit(1)
	}

	if Env.PRUNE == 0 {

		l.Logger(l.ERROR, "PRUNE must be greater than zero.")
		os.Exit(1)

	}

	modeStr := os.Getenv("MASK")

	if modeStr != "" {

		parsedMode, err := strconv.ParseUint(modeStr, 8, 32)

		if err == nil {

			Env.MASK = os.FileMode(parsedMode)

		} else {
			log.Printf("Invalid MASK %q, falling back to 0600", modeStr)
			Env.MASK = os.FileMode(0600)
		}

	} else {

		Env.MASK = os.FileMode(0600)

	}

}
