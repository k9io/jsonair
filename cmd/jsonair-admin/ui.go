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
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// dbTimeLayout is how MySQL TIMESTAMP columns arrive when scanned into a string.
const dbTimeLayout = "2006-01-02 15:04:05"

// assetVersion is a content hash of the static files. It is appended to asset
// URLs so browsers can cache them forever and still pick up new builds.
var assetVersion = func() string {
	h := sha256.New()
	_ = fs.WalkDir(staticFS, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := staticFS.ReadFile(p)
		if err != nil {
			return err
		}
		h.Write([]byte(p))
		h.Write(b)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:12]
}()

var templateFuncs = template.FuncMap{
	"asset":      func(name string) string { return "/static/" + name + "?v=" + assetVersion },
	"ago":        relativeTime,
	"fmtDate":    fmtDate,
	"isoTime":    isoTime,
	"shortUUID":  shortUUID,
	"levelClass": levelClass,
	"plural": func(n int, one, many string) string {
		if n == 1 {
			return one
		}
		return many
	},
}

func loadTemplates() *template.Template {
	return template.Must(template.New("").Funcs(templateFuncs).ParseFS(templateFS, "templates/*.html"))
}

// serveStatic serves the embedded files under /static/. Directory listings are
// never produced.
func serveStatic(c *gin.Context) {
	name := strings.TrimPrefix(c.Param("filepath"), "/")
	if name == "" || !fs.ValidPath(name) {
		c.Status(http.StatusNotFound)
		return
	}

	data, err := staticFS.ReadFile("static/" + name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	ctype := mime.TypeByExtension(path.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}

	if c.Query("v") == assetVersion {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	c.Data(http.StatusOK, ctype, data)
}

// render adds the values every page needs (signed-in user, session time left)
// and renders the template.
func render(c *gin.Context, status int, name string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	if u, ok := c.Get("user"); ok {
		data["User"] = u
	}
	if exp, ok := c.Get("sessionExp"); ok {
		data["SessionRemaining"] = sessionRemaining(exp.(time.Time))
	}
	if tok := c.GetString("csrf"); tok != "" {
		data["CSRFToken"] = tok
	}
	c.HTML(status, name, data)
}

func renderError(c *gin.Context, status int, heading, message string) {
	render(c, status, "error.html", gin.H{
		"Title":   heading,
		"Status":  status,
		"Heading": heading,
		"Message": message,
	})
}

func sessionRemaining(exp time.Time) int64 {
	secs := int64(time.Until(exp).Seconds())
	if secs < 0 {
		return 0
	}
	return secs
}

// --- Flash messages ---
//
// A flash is a one-time message carried across a redirect in a short-lived
// cookie, so messages are never put in the URL.

const flashCookie = "jsonair_admin_flash"

type flash struct {
	Kind     string // "success", "error" or "info"
	Text     string
	LinkURL  string
	LinkText string
}

func setFlash(c *gin.Context, kind, text string) {
	if len(text) > 500 {
		text = text[:500] + "…"
	}
	setCookie(c, flashCookie, kind+"|"+text, 60)
}

func takeFlash(c *gin.Context) *flash {
	v, err := c.Cookie(flashCookie)
	if err != nil || v == "" {
		return nil
	}
	setCookie(c, flashCookie, "", -1)

	kind, text, ok := strings.Cut(v, "|")
	if !ok || (kind != "success" && kind != "error" && kind != "info") {
		return nil
	}
	return &flash{Kind: kind, Text: text}
}

func errorFlash(text string) *flash {
	return &flash{Kind: "error", Text: text}
}

// --- Formatting helpers ---

// relativeTime turns an age in seconds (computed by the database, so time
// zones never matter) into "3 hours ago". Older dates fall back to the date.
func relativeTime(secs int64, ts string) string {
	n := func(v int64, unit string) string {
		if v == 1 {
			return "1 " + unit + " ago"
		}
		return strconv.FormatInt(v, 10) + " " + unit + "s ago"
	}

	switch {
	case secs < 0:
		return fmtDate(ts)
	case secs < 60:
		return "just now"
	case secs < 3600:
		return n(secs/60, "minute")
	case secs < 86400:
		return n(secs/3600, "hour")
	case secs < 2*86400:
		return "yesterday"
	case secs < 7*86400:
		return n(secs/86400, "day")
	default:
		return fmtDate(ts)
	}
}

func fmtDate(ts string) string {
	t, err := time.Parse(dbTimeLayout, ts)
	if err != nil {
		return ts
	}
	return t.Format("Jan 2, 2006")
}

func isoTime(ts string) string {
	t, err := time.Parse(dbTimeLayout, ts)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05")
}

func shortUUID(u string) string {
	if u == "" {
		return "(empty)"
	}
	if len(u) <= 14 {
		return u
	}
	return u[:8] + "…" + u[len(u)-4:]
}

func levelClass(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG", "TRACE":
		return "badge-debug"
	case "INFO":
		return "badge-info"
	case "WARN", "WARNING", "ERROR", "FATAL", "CRITICAL":
		return "badge-warn"
	default:
		return "badge-neutral"
	}
}

// --- Errors ---

func isDuplicateEntry(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// duplicateFlash builds a friendly message for a unique (uuid, type, name)
// clash, linking to the configuration that already exists when it can.
func duplicateFlash(uuid, typ, name string) *flash {
	f := &flash{
		Kind: "error",
		Text: fmt.Sprintf("A configuration with type “%s” and name “%s” already exists for this key.", typ, name),
	}
	if id, err := sqlFindConfigID(uuid, typ, name); err == nil {
		f.LinkURL = fmt.Sprintf("/configs/%d/edit", id)
		f.LinkText = "Open the existing configuration"
	}
	return f
}
