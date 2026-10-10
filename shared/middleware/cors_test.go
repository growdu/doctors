package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_DefaultAllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5174")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5174" {
		t.Errorf("expected Allow-Origin=127.0.0.1:5174, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Allow-Credentials=true, got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("disallowed origin should NOT get Allow-Origin, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.POST("/ping", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("OPTIONS", "/ping", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5174")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Errorf("preflight should set Allow-Methods")
	}
}

func TestCORS_CustomOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(WithCORSAllowOrigins([]string{"https://app.example.com"})))
	r.GET("/ping", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("custom origin should be allowed, got %q", got)
	}
}