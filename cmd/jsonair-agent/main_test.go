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
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- copyFile ---

func TestCopyFile_CopiesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("hello agent"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile() error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile(dst) error: %v", err)
	}
	if string(got) != "hello agent" {
		t.Errorf("content = %q, want %q", string(got), "hello agent")
	}
}

func TestCopyFile_MissingSource(t *testing.T) {
	err := copyFile("/nonexistent/path/file.txt", t.TempDir()+"/dst.txt")
	if err == nil {
		t.Error("expected error for missing source, got nil")
	}
}

// --- pruneOldBackups ---

func TestPruneOldBackups_RemovesOldest(t *testing.T) {
	dir := t.TempDir()
	base := "config.json"

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("%s.2026040%d-120000-000000000.bak", base, i+1)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := pruneOldBackups(dir, base, 3); err != nil {
		t.Fatalf("pruneOldBackups() error: %v", err)
	}

	files, _ := os.ReadDir(dir)
	var baks []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bak") {
			baks = append(baks, f.Name())
		}
	}
	if len(baks) != 3 {
		t.Errorf("backup count = %d, want 3", len(baks))
	}
}

func TestPruneOldBackups_UnderLimit_NoDelete(t *testing.T) {
	dir := t.TempDir()
	base := "config.json"

	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("%s.2026040%d-120000-000000000.bak", base, i+1)
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0600)
	}

	if err := pruneOldBackups(dir, base, 5); err != nil {
		t.Fatalf("pruneOldBackups() error: %v", err)
	}

	files, _ := os.ReadDir(dir)
	var baks []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bak") {
			baks = append(baks, f.Name())
		}
	}
	if len(baks) != 2 {
		t.Errorf("backup count = %d, want 2 (nothing should be pruned)", len(baks))
	}
}

// --- backupAndPrune ---

func TestBackupAndPrune_CreatesBackup(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "config.json")

	if err := os.WriteFile(src, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := backupAndPrune(src, 5); err != nil {
		t.Fatalf("backupAndPrune() error: %v", err)
	}

	files, _ := os.ReadDir(dir)
	var baks []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bak") {
			baks = append(baks, f.Name())
		}
	}
	if len(baks) == 0 {
		t.Error("no backup file created")
	}

	got, _ := os.ReadFile(filepath.Join(dir, baks[0]))
	if string(got) != "original" {
		t.Errorf("backup content = %q, want %q", string(got), "original")
	}
}

func TestBackupAndPrune_BackupTimestampUnique(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "config.json")
	os.WriteFile(src, []byte("data"), 0600)

	if err := backupAndPrune(src, 10); err != nil {
		t.Fatal(err)
	}
	if err := backupAndPrune(src, 10); err != nil {
		t.Fatal(err)
	}

	files, _ := os.ReadDir(dir)
	var baks []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bak") {
			baks = append(baks, f.Name())
		}
	}
	if len(baks) < 2 {
		t.Errorf("expected 2 unique backups, got %d — timestamps may collide", len(baks))
	}
}

// --- processData ---

func setupProcessEnv(t *testing.T, dir string) {
	t.Helper()
	Env.CONFIG_FILE = filepath.Join(dir, "config.json")
	Env.PRUNE = 3
	Env.MASK = 0600
}

func TestProcessData_InvalidBase64(t *testing.T) {
	setupProcessEnv(t, t.TempDir())
	processData("!!!not-valid-base64!!!")
	// Should return without panic or os.Exit
}

func TestProcessData_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	setupProcessEnv(t, dir)

	content := "brand new config"
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	processData(encoded)

	got, err := os.ReadFile(Env.CONFIG_FILE)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", string(got), content)
	}
}

func TestProcessData_NoChangeWhenHashMatches(t *testing.T) {
	dir := t.TempDir()
	setupProcessEnv(t, dir)

	content := "stable config content"
	os.WriteFile(Env.CONFIG_FILE, []byte(content), 0600)

	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	processData(encoded)

	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bak") {
			t.Errorf("unexpected backup created when content unchanged: %s", f.Name())
		}
	}
}

func TestProcessData_UpdatesFileWhenChanged(t *testing.T) {
	dir := t.TempDir()
	setupProcessEnv(t, dir)

	os.WriteFile(Env.CONFIG_FILE, []byte("old config"), 0600)

	newContent := "updated config from server"
	encoded := base64.StdEncoding.EncodeToString([]byte(newContent))
	processData(encoded)

	got, err := os.ReadFile(Env.CONFIG_FILE)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != newContent {
		t.Errorf("content = %q, want %q", string(got), newContent)
	}

	files, _ := os.ReadDir(dir)
	var baks []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bak") {
			baks = append(baks, f.Name())
		}
	}
	if len(baks) == 0 {
		t.Error("expected a backup to be created when content changes")
	}
}

func TestProcessData_RespectsFileMask(t *testing.T) {
	dir := t.TempDir()
	setupProcessEnv(t, dir)
	Env.MASK = 0640

	content := "masked content"
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	processData(encoded)

	info, err := os.Stat(Env.CONFIG_FILE)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("file mode = %04o, want 0640", info.Mode().Perm())
	}
}
