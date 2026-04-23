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
	"os"
	"testing"
	"time"
)

func setRequiredEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("JSONAIR_PAT", "test-pat-value")
	t.Setenv("RUNAS", "nobody")
	t.Setenv("JSONAIR_URL", "http://localhost:9191")
	t.Setenv("JSONAIR_NAME", "test.config")
	t.Setenv("JSONAIR_TYPE", "testsub")
	t.Setenv("CONFIG_FILE", "/tmp/test-agent.json")
	t.Setenv("RELOAD_COMMAND", "/usr/bin/true")
	t.Setenv("SLEEP", "10")
	t.Setenv("PRUNE", "5")
}

func TestLoadEnv_Happy(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("MASK", "600")

	LoadEnv()

	if Env.JSONAIR_PAT != "test-pat-value" {
		t.Errorf("JSONAIR_PAT = %q, want %q", Env.JSONAIR_PAT, "test-pat-value")
	}
	if Env.RUNAS != "nobody" {
		t.Errorf("RUNAS = %q, want %q", Env.RUNAS, "nobody")
	}
	if Env.JSONAIR_URL != "http://localhost:9191" {
		t.Errorf("JSONAIR_URL = %q, want %q", Env.JSONAIR_URL, "http://localhost:9191")
	}
	if Env.SLEEP != 10*time.Second {
		t.Errorf("SLEEP = %v, want %v", Env.SLEEP, 10*time.Second)
	}
	if Env.PRUNE != 5 {
		t.Errorf("PRUNE = %d, want 5", Env.PRUNE)
	}
	if Env.MASK != os.FileMode(0600) {
		t.Errorf("MASK = %04o, want 0600", Env.MASK)
	}
}

func TestLoadEnv_MaskDefault(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("MASK", "")

	LoadEnv()

	if Env.MASK != os.FileMode(0600) {
		t.Errorf("MASK = %04o, want 0600 (default)", Env.MASK)
	}
}

func TestLoadEnv_MaskInvalidFallsBack(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("MASK", "notamode")

	LoadEnv()

	if Env.MASK != os.FileMode(0600) {
		t.Errorf("MASK = %04o, want 0600 (fallback)", Env.MASK)
	}
}

func TestLoadEnv_MaskCustom(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("MASK", "644")

	LoadEnv()

	if Env.MASK != os.FileMode(0644) {
		t.Errorf("MASK = %04o, want 0644", Env.MASK)
	}
}

func TestLoadEnv_SleepParsed(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("SLEEP", "30")

	LoadEnv()

	if Env.SLEEP != 30*time.Second {
		t.Errorf("SLEEP = %v, want 30s", Env.SLEEP)
	}
}
