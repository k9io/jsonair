package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func newRouter(cfg Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth", New(cfg).Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

/* send fires n requests from remoteAddr and returns the status codes and the last response. */

func send(router *gin.Engine, remoteAddr string, n int) ([]int, *httptest.ResponseRecorder) {
	var codes []int
	var w *httptest.ResponseRecorder
	for i := 0; i < n; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth", nil)
		req.RemoteAddr = remoteAddr
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		codes = append(codes, w.Code)
	}
	return codes, w
}

func count(codes []int, code int) int {
	n := 0
	for _, c := range codes {
		if c == code {
			n++
		}
	}
	return n
}

func TestParseEnv_Defaults(t *testing.T) {
	cfg, err := ParseEnv(envMap(nil))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.PerMinute != DefaultPerMinute || cfg.Burst != DefaultBurst || len(cfg.Exempt) != 0 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestParseEnv_Custom(t *testing.T) {
	cfg, err := ParseEnv(envMap(map[string]string{
		"RATE_LIMIT_ENABLED":    "FALSE",
		"RATE_LIMIT_PER_MINUTE": "120",
		"RATE_LIMIT_BURST":      "50",
		"RATE_LIMIT_EXEMPT":     " 10.0.5.12 , 192.168.1.77/24,::1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || cfg.PerMinute != 120 || cfg.Burst != 50 {
		t.Errorf("unexpected config: %+v", cfg)
	}
	want := []string{"10.0.5.12/32", "192.168.1.0/24", "::1/128"}
	if len(cfg.Exempt) != len(want) {
		t.Fatalf("expected %d exempt entries, got %v", len(want), cfg.Exempt)
	}
	for i, w := range want {
		if cfg.Exempt[i].String() != w {
			t.Errorf("exempt[%d]: expected %s, got %s", i, w, cfg.Exempt[i])
		}
	}
}

func TestParseEnv_Invalid(t *testing.T) {
	cases := []map[string]string{
		{"RATE_LIMIT_ENABLED": "yes"},
		{"RATE_LIMIT_PER_MINUTE": "0"},
		{"RATE_LIMIT_PER_MINUTE": "abc"},
		{"RATE_LIMIT_BURST": "-1"},
		{"RATE_LIMIT_EXEMPT": "10.0.0.300"},
		{"RATE_LIMIT_EXEMPT": "10.0.0.0/33"},
	}
	for _, env := range cases {
		if _, err := ParseEnv(envMap(env)); err == nil {
			t.Errorf("expected error for %v", env)
		}
	}
}

func TestMiddleware_AllowsBurstThenBlocks(t *testing.T) {
	router := newRouter(Config{Enabled: true, PerMinute: 5, Burst: 5})

	codes, w := send(router, "192.0.2.1:1234", 6)

	if count(codes[:5], http.StatusOK) != 5 {
		t.Errorf("expected first 5 requests to pass, got %v", codes)
	}
	if codes[5] != http.StatusTooManyRequests {
		t.Errorf("expected 6th request to be 429, got %d", codes[5])
	}
	ra, err := strconv.Atoi(w.Header().Get("Retry-After"))
	if err != nil || ra < 1 || ra > 12 {
		t.Errorf("expected Retry-After between 1 and 12 seconds, got %q", w.Header().Get("Retry-After"))
	}
}

func TestMiddleware_CustomBurst(t *testing.T) {
	router := newRouter(Config{Enabled: true, PerMinute: 5, Burst: 20})

	codes, _ := send(router, "192.0.2.1:1234", 21)

	if count(codes, http.StatusOK) != 20 || codes[20] != http.StatusTooManyRequests {
		t.Errorf("expected 20 passes then a 429, got %v", codes)
	}
}

func TestMiddleware_PerIP(t *testing.T) {
	router := newRouter(Config{Enabled: true, PerMinute: 5, Burst: 1})

	send(router, "192.0.2.1:1234", 1)
	codes, _ := send(router, "192.0.2.2:1234", 1)

	if codes[0] != http.StatusOK {
		t.Errorf("a different IP should have its own bucket, got %d", codes[0])
	}
}

func TestMiddleware_Disabled(t *testing.T) {
	router := newRouter(Config{Enabled: false, PerMinute: 5, Burst: 5})

	codes, _ := send(router, "192.0.2.1:1234", 50)

	if count(codes, http.StatusOK) != 50 {
		t.Errorf("expected all requests to pass when disabled, got %v", codes)
	}
}

func TestMiddleware_Exempt(t *testing.T) {
	cfg, err := ParseEnv(envMap(map[string]string{
		"RATE_LIMIT_BURST":  "1",
		"RATE_LIMIT_EXEMPT": "10.0.5.12,192.168.0.0/24",
	}))
	if err != nil {
		t.Fatal(err)
	}
	router := newRouter(cfg)

	for _, addr := range []string{"10.0.5.12:1234", "192.168.0.99:1234"} {
		codes, _ := send(router, addr, 20)
		if count(codes, http.StatusOK) != 20 {
			t.Errorf("%s should be exempt, got %v", addr, codes)
		}
	}

	codes, _ := send(router, "10.0.5.13:1234", 2)
	if codes[1] != http.StatusTooManyRequests {
		t.Errorf("non-exempt IP should still be limited, got %v", codes)
	}
}
