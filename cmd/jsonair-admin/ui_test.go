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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func sampleRows() []configRow {
	return []configRow{
		{ID: 1, UUID: "3f2a8b1e-6c4d-4e8a-9f12-7b5e0d2c9c1d", Type: "suricata", Name: "prod.yaml", Reload: "suricata-reload", Debug: "INFO", Created: "2026-09-03 10:00:00", Updated: "2026-11-12 09:14:02", KeyName: "sensor-fleet", UpdatedAgo: 3 * 3600},
		{ID: 2, UUID: "", Type: "testsub", Name: "test.config", Created: "2026-09-03 10:00:00", Updated: "2026-10-01 08:00:00", UpdatedAgo: 40 * 86400},
	}
}

func sampleKeys() []keyRow {
	return []keyRow{{UUID: "3f2a8b1e-6c4d-4e8a-9f12-7b5e0d2c9c1d", Name: "sensor-fleet"}}
}

// TestTemplatesRender executes every page with representative data so that
// template errors (bad field names, nil values) fail the build, not a user.
func TestTemplatesRender(t *testing.T) {
	tmpl := loadTemplates()
	rows := sampleRows()
	orig := rows[0]

	edit := formData(orig.Name, &orig, orig, "\nleading newline\n  key: value", sampleKeys(), nil)
	editFailed := formData(orig.Name, &orig, configRow{UUID: "unknown-uuid", Type: "x", Name: "y"}, "", sampleKeys(), duplicateLikeFlash())
	editFailed["StartDirty"] = true

	cases := []struct {
		name     string
		page     string
		data     gin.H
		contains []string
	}{
		{"list", "configs.html", gin.H{"Title": "Configurations", "Configs": rows, "Count": 2, "KeyCount": 2, "User": "admin", "SessionRemaining": int64(100),
			"Flash": &flash{Kind: "success", Text: "Saved suricata / prod.yaml."}},
			[]string{"sensor-fleet", "3f2a8b1e…9c1d", "No matching key", "3 hours ago", "Oct 1, 2026", "badge-info", `data-session-remaining="100"`, "Saved suricata / prod.yaml.", "delete-dialog"}},
		{"list empty", "configs.html", gin.H{"Title": "Configurations", "Count": 0, "KeyCount": 0, "User": "admin"}, []string{"No configurations yet"}},
		{"list no matches", "configs.html", gin.H{"Title": "Configurations", "Count": 0, "KeyCount": 0, "User": "admin", "Filtered": true, "TypeFilter": "zzz"}, []string{"No matches", "<code>zzz</code>"}},
		{"list load error", "configs.html", gin.H{"Title": "Configurations", "Count": 0, "KeyCount": 0, "User": "admin", "LoadError": true, "Flash": errorFlash("boom")}, []string{"boom"}},
		{"new", "new.html", formData("New configuration", nil, configRow{}, "", sampleKeys(), nil), []string{"Create configuration", "sensor-fleet — 3f2a"}},
		{"new no keys", "new.html", formData("New configuration", nil, configRow{}, "", nil, nil), []string{"No keys found", "disabled>Create configuration"}},
		{"edit", "edit.html", edit, []string{"Save changes", `data-original="suricata"`, "rename-warning", "Sep 3, 2026", "<textarea", "\n\nleading newline"}},
		{"edit failed save", "edit.html", editFailed, []string{"No matching key — unknown-uuid", "data-start-dirty", "Open the existing configuration"}},
		{"login", "login.html", gin.H{"Title": "Sign in"}, []string{"Sign in to JSONAir Admin", "autofocus"}},
		{"login error", "login.html", gin.H{"Title": "Sign in", "Error": "Invalid username or password.", "Username": "admin"}, []string{"Invalid username", `aria-invalid="true"`, `value="admin"`}},
		{"login expired", "login.html", gin.H{"Title": "Sign in", "Expired": true}, []string{"session has expired"}},
		{"error", "error.html", gin.H{"Title": "Not found", "Status": 404, "Heading": "Configuration not found", "Message": "gone"}, []string{"404", "Configuration not found"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, tc.page, tc.data); err != nil {
				t.Fatalf("render %s: %v", tc.page, err)
			}
			out := buf.String()
			for _, want := range tc.contains {
				if !strings.Contains(out, want) {
					t.Errorf("%s: output missing %q", tc.page, want)
				}
			}
		})
	}
}

func duplicateLikeFlash() *flash {
	return &flash{Kind: "error", Text: "already exists", LinkURL: "/configs/1/edit", LinkText: "Open the existing configuration"}
}

func TestRelativeTime(t *testing.T) {
	ts := "2026-10-28 14:55:10"
	cases := map[int64]string{
		-5:         "Oct 28, 2026",
		10:         "just now",
		60:         "1 minute ago",
		125:        "2 minutes ago",
		3600:       "1 hour ago",
		5 * 3600:   "5 hours ago",
		30 * 3600:  "yesterday",
		3 * 86400:  "3 days ago",
		30 * 86400: "Oct 28, 2026",
	}
	for secs, want := range cases {
		if got := relativeTime(secs, ts); got != want {
			t.Errorf("relativeTime(%d) = %q, want %q", secs, got, want)
		}
	}
}

func TestShortUUIDAndLevels(t *testing.T) {
	if got := shortUUID("3f2a8b1e-6c4d-4e8a-9f12-7b5e0d2c9c1d"); got != "3f2a8b1e…9c1d" {
		t.Errorf("shortUUID = %q", got)
	}
	if got := shortUUID(""); got != "(empty)" {
		t.Errorf("shortUUID empty = %q", got)
	}
	for level, want := range map[string]string{"info": "badge-info", "DEBUG": "badge-debug", "Warning": "badge-warn", "DEBUGLEVEL": "badge-neutral"} {
		if got := levelClass(level); got != want {
			t.Errorf("levelClass(%q) = %q, want %q", level, got, want)
		}
	}
}
