package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/metrics"
	"github.com/growdu/doctors/shared/middleware"
)

// TestMetrics_GinMiddleware 验证 shared/middleware.Metrics() 是 shared/metrics.GinMiddleware()
// 的语义别名：装到 gin engine 后能自动记录 method/path/status/duration。
func TestMetrics_GinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Metrics())
	r.GET("/api/v1/items/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": "42"})
	})
	r.POST("/api/v1/items", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	// GET /api/v1/items/42 → 2xx
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/items/42", nil))
	require.Equal(t, http.StatusOK, w.Code)

	// POST /api/v1/items → 2xx
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/items", nil))
	require.Equal(t, http.StatusCreated, w.Code)

	// path 必须用路由模板（避免 label 基数爆炸）
	assert.Equal(t, 1.0,
		testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/items/:id", "2xx")),
		"GET 应被 metrics 计数（路由模板）")
	assert.Equal(t, 1.0,
		testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("POST", "/api/v1/items", "2xx")),
		"POST 应被 metrics 计数")
}

// TestMetrics_HandlerExposesEndpoint 验证 handler 暴露的 /metrics 端点能输出标准 prom 文本格式。
func TestMetrics_HandlerExposesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Metrics())
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	// 触发一次业务请求，让中间件写入数据
	r.GET("/api/v1/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/probe", nil))
	require.Equal(t, http.StatusOK, w.Code)

	// 然后抓取 /metrics
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, "http_requests_total", "应暴露 http_requests_total")
	assert.Contains(t, body, "http_request_duration_seconds", "应暴露 histogram")
	assert.Contains(t, body, "# HELP", "标准 prom 文本格式")
}
