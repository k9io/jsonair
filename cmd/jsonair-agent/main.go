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
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	l "github.com/k9io/jsonair/internal/logger"
	"github.com/k9io/jsonair/internal/droppriv"
	"github.com/k9io/jsonair/internal/define"
	"github.com/k9io/jsonair/internal/http_req"
)

func main() {

	LoadEnv()

	droppriv.DropPrivileges( Env.RUNAS )

	bearerToken := PAT_Auth()

	config_url := fmt.Sprintf("%s/api/%s/jsonair/config", Env.JSONAIR_URL, define.VERSION)

	config_json := fmt.Sprintf(`{"type":"%s","name":"%s","decode":false}`, Env.JSONAIR_TYPE, Env.JSONAIR_NAME)

	for {

		results, status_code := http_req.HTTP(config_json, config_url, "GET", bearerToken)

		/* If HTTP is a 401,  re-authenticated with PAT */

		if status_code == 401 {

			l.Logger(l.NOTICE, "Got 401 HTTP Status.  Getting new Bearer token.")
			bearerToken = PAT_Auth()

		}

		/* Some unknown status,  we exit */

		if status_code != 200 && status_code != 401 {

			l.Logger(l.ERROR, "Got bad response %v.", status_code)
			os.Exit(1)

		}

		/* We authenticated and we're good.  Process the configuration data */

		if status_code == 200 {

			processData(results)

		}


		/* Sleep until next round */

		l.Logger(l.INFO, "Sleeping %g seconds...", Env.SLEEP.Seconds())

		time.Sleep(Env.SLEEP)
	}

}

func processData(results string) {

	decodedBytes, err := base64.StdEncoding.DecodeString(results)

	if err != nil {

		l.Logger(l.ERROR, "Unable to decode Base64.  Skipping.")
		return
	}

	r := string(decodedBytes)
	r_hash := sha256.Sum256([]byte(r))

	r_data := []byte(r) /* Used later */

	/* Read in the stored configuration files Sha256 hash. */ 

	file, err := os.Open(Env.CONFIG_FILE)

	if err != nil {

		if errors.Is(err, os.ErrNotExist) {

			/* File doesn't exist,  try and create it */

			l.Logger(l.NOTICE, "%s does not exist.  Creating it.", Env.CONFIG_FILE)

			err = os.WriteFile(Env.CONFIG_FILE, r_data, 0644)

			if err != nil {

				l.Logger(l.ERROR, "Unable to create file: %v", err)
				os.Exit(1)
			}

			/* Since we created the file, there is no need in going any
			   further.  We return and wait for the next round */

			return

		}

		/* Any other error,  we report and exit */

		l.Logger(l.ERROR, "Error reading file: %v", err)
		os.Exit(1)

	}

	defer file.Close()

	h := sha256.New()

	_, err = io.Copy(h, file)	/* Avoid reading in the file */

	if err != nil {

		l.Logger(l.ERROR, "io.Copy error: %v", err)
		os.Exit(1)

	}

	/* Compute sha256 has for the file */

	file_HashBytes := h.Sum(nil)

	/* Compare the hash from the JSONAir and the file hash.  If they match, 
	   nothing has changed.   We return and wait for the next round.  */

	if bytes.Equal(r_hash[:], file_HashBytes) {

		l.Logger(l.INFO, "No changes to configuration data.")
		return
	}

	/* Hashes don't match,  we start processing the data */

	l.Logger(l.NOTICE, "Got updated configuration data")

	/* First make a backup of the previous confiruation.  Keep X (PRUNE) 
	   backups */

	err = backupAndPrune(Env.CONFIG_FILE, Env.PRUNE)

	if err != nil {

		l.Logger(l.WARN, "Unable to make backup: %v", err)
		return 
	}

	// 2. Write the data to the original path
	// 0644 provides read/write for owner, and read-only for others

	err = os.WriteFile(Env.CONFIG_FILE, r_data, Env.MASK)

	if err != nil {

		l.Logger(l.WARN, "Failed to overwrite file: %v", err)
		return
	}

/* Task is done,  naturally return and wait for the next round */

}

func backupAndPrune(fullPath string, keepCount int) error {

	dir := filepath.Dir(fullPath)
	baseName := filepath.Base(fullPath)

	/* 1. Create the new backup */

	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s.bak", fullPath, timestamp)

	if err := copyFile(fullPath, backupPath); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	/* 2. Find and prune old backups */
	return pruneOldBackups(dir, baseName, keepCount)
}

func pruneOldBackups(dir, baseName string, keep int) error {

	files, err := os.ReadDir(dir)

	if err != nil {
		return err
	}

	var backups []os.FileInfo
	prefix := baseName + "."
	suffix := ".bak"

	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), suffix) {
			info, _ := f.Info()
			backups = append(backups, info)
		}
	}

	/* Sort backups by modification time (oldest first) */

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime().Before(backups[j].ModTime())
	})

	/* Delete if we exceed the limit */

	if len(backups) > keep {
		toDelete := backups[:len(backups)-keep]
		for _, f := range toDelete {
			os.Remove(filepath.Join(dir, f.Name()))
		}
	}
	return nil
}

func copyFile(src, dst string) error {

	source, err := os.Open(src)

	if err != nil {
		return err
	}

	defer source.Close()

	destination, err := os.Create(dst)

	if err != nil {
		return err
	}

	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
