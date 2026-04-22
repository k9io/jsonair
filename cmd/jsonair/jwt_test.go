package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func setupTestEnv() {
	Env.JWTTokenSecret = []byte("test-secret-key-for-unit-tests")
	Env.JWTTokenExpire = 60
	Env.TokenHMACSecret = []byte("test-hmac-secret-for-unit-tests")
}

func generateTestToken(t *testing.T, uuid, clientName string, expired bool) string {
	t.Helper()

	exp := time.Now().Add(time.Hour)
	if expired {
		exp = time.Now().Add(-time.Hour)
	}

	cl := &claims{
		UUID:       uuid,
		ClientName: clientName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	tokenStr, err := token.SignedString(Env.JWTTokenSecret)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return tokenStr
}

// --- jwtMiddleware ---

func TestJWTMiddleware_ValidToken(t *testing.T) {
	setupTestEnv()

	router := gin.New()
	router.Use(jwtMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tokenStr := generateTestToken(t, "test-uuid", "test-client", false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	setupTestEnv()

	router := gin.New()
	router.Use(jwtMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	setupTestEnv()

	router := gin.New()
	router.Use(jwtMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tokenStr := generateTestToken(t, "test-uuid", "test-client", true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTMiddleware_InvalidSignature(t *testing.T) {
	setupTestEnv()

	router := gin.New()
	router.Use(jwtMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Sign with a different secret
	cl := &claims{UUID: "x", ClientName: "x"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	tokenStr, _ := token.SignedString([]byte("wrong-secret"))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTMiddleware_WrongAlgorithm(t *testing.T) {
	setupTestEnv()

	router := gin.New()
	router.Use(jwtMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Sign with RS256 (wrong algorithm for this service)
	cl := &claims{UUID: "x", ClientName: "x"}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, cl)
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- authToken ---

func TestAuthToken_MissingBody(t *testing.T) {
	setupTestEnv()

	router := gin.New()
	router.POST("/auth", authToken)

	req := httptest.NewRequest(http.MethodPost, "/auth", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthToken_MissingTokenField(t *testing.T) {
	setupTestEnv()

	router := gin.New()
	router.POST("/auth", authToken)

	body, _ := json.Marshal(map[string]string{"other": "value"})
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
