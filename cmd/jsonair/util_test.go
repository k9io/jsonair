package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- removeUnwanted ---

func TestRemoveUnwanted(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"test.config", "test.config"},
		{"type-name_1", "type-name_1"},
		{"../etc/passwd", "..etcpasswd"},
		{"foo;bar", "foobar"},
		{"foo bar", "foobar"},
		{"type<script>", "typescript"},
		{"", ""},
	}

	for _, tt := range tests {
		got := removeUnwanted(tt.input)
		if got != tt.want {
			t.Errorf("removeUnwanted(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- getConfigName ---

func newTestContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	return c, w
}

func TestGetConfigName_Valid(t *testing.T) {
	c, _ := newTestContext(`{"type":"testsub","name":"test.config"}`)
	name, jtype, err := getConfigName(c, `{"type":"testsub","name":"test.config"}`)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "test.config" {
		t.Errorf("name = %q, want %q", name, "test.config")
	}
	if jtype != "testsub" {
		t.Errorf("jtype = %q, want %q", jtype, "testsub")
	}
}

func TestGetConfigName_MissingType(t *testing.T) {
	c, _ := newTestContext(`{"name":"test.config"}`)
	_, _, err := getConfigName(c, `{"name":"test.config"}`)

	if err == nil {
		t.Error("expected error for missing type, got nil")
	}
}

func TestGetConfigName_MissingName(t *testing.T) {
	c, _ := newTestContext(`{"type":"testsub"}`)
	_, _, err := getConfigName(c, `{"type":"testsub"}`)

	if err == nil {
		t.Error("expected error for missing name, got nil")
	}
}

func TestGetConfigName_SanitizesInput(t *testing.T) {
	c, _ := newTestContext(`{"type":"test;drop","name":"config<x>"}`)
	name, jtype, err := getConfigName(c, `{"type":"test;drop","name":"config<x>"}`)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.ContainsAny(jtype, ";") {
		t.Errorf("jtype contains unsafe characters: %q", jtype)
	}
	if strings.ContainsAny(name, "<>") {
		t.Errorf("name contains unsafe characters: %q", name)
	}
}

// --- mysqlAddr ---

func TestMysqlAddr(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"127.0.0.1", 3306, "127.0.0.1:3306"},
		{"db.internal", 3307, "db.internal:3307"},
		{"::1", 3306, "[::1]:3306"},
		// A port already in MYSQL_HOST (the old workaround for MYSQL_PORT being ignored) wins.
		{"db.internal:3307", 3306, "db.internal:3307"},
		{"[::1]:3307", 3306, "[::1]:3307"},
	}
	for _, tt := range tests {
		if got := mysqlAddr(tt.host, tt.port); got != tt.want {
			t.Errorf("mysqlAddr(%q, %d) = %q, want %q", tt.host, tt.port, got, tt.want)
		}
	}
}
