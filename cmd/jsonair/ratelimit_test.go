package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitMiddleware_AllowsUnderLimit(t *testing.T) {
	// Reset limiters for a clean test
	limitersMu.Lock()
	limiters = make(map[string]*ipLimiter)
	limitersMu.Unlock()

	router := gin.New()
	router.POST("/auth", rateLimitMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitMiddleware_BlocksOverLimit(t *testing.T) {
	// Reset limiters for a clean test
	limitersMu.Lock()
	limiters = make(map[string]*ipLimiter)
	limitersMu.Unlock()

	router := gin.New()
	router.POST("/auth", rateLimitMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Exhaust the 5-request burst
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// The 6th request should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/auth", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}
