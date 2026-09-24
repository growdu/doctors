package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/httpx"
)

// setupRouter 把 gin 切到 test 模式（默认 debug 模式会刷很多日志）。
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return New()
}

// TestHealthz_ReturnsOK 验证 /healthz 直接返回 200 + code=0。
func TestHealthz_ReturnsOK(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "HTTP 状态码应为 200")

	var resp httpx.Resp[map[string]string]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, "业务码应为 0")
	assert.Equal(t, "ok", resp.Message)
	assert.Equal(t, "ok", resp.Data["status"])
	assert.NotEmpty(t, resp.TraceID, "trace_id 必须存在")
}

// TestRouter_RegistersAllRoutes 验证六个 API 路由都被注册（不是 404）。
func TestRouter_RegistersAllRoutes(t *testing.T) {
	r := setupRouter()
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/sms/send"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/users/real-name/auth"},
		{http.MethodGet, "/api/v1/users/me"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			r.ServeHTTP(w, req)
			// 路由已注册 → 命中 handler（即使是占位也不会 404）
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"路由 %s %s 应被注册", tc.method, tc.path)
		})
	}
}

// TestRouter_RejectsUnknownPath 验证未注册路径返回 404。
func TestRouter_RejectsUnknownPath(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}