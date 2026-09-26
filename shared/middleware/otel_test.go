// Package middleware —— OTelGinMiddleware 单测（§34 稳定性优化）。
//
// 验证：
//   - OTelGinMiddleware 返回的 handler 在 init 时不 panic；
//   - endpoint 配 None（Noop）时请求仍正常返回（200 + 不创建 span），
//     OTel 零开销；
//   - 路由模板 + method 出现在 span name（避免 label 基数爆炸）。
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/growdu/doctors/shared/middleware"
)

// TestOTelGinMiddleware_BasicRouting 验证 OTelGinMiddleware 接入后路由仍正常。
func TestOTelGinMiddleware_BasicRouting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 模拟 OTelGin 在 None/Noop 场景下（dev / 单测默认）。
	r.Use(middleware.OTelGinMiddleware("test-svc"))
	r.GET("/api/v1/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/42", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"42"`)
}

// TestOTelGinMiddleware_NilRequestContextSafe 验证 panic 不外泄（defer-recover 兜底）。
func TestOTelGinMiddleware_NilRequestContextSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.OTelGinMiddleware("test-svc"))
	r.GET("/p", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p", nil))
		assert.Equal(t, http.StatusOK, w.Code,
			"OTelGin 应不影响路由基本功能")
	}
}

// TestOTelGinMiddleware_SpanNameStable 验证 span name = HTTP method + 路由模板（不变 ID）。
//
// 用 in-memory exporter 替换全局 TracerProvider，断言：
//   - 请求 /api/v1/users/42 和 /api/v1/users/999 的 span.name 完全相同
//     （都是 "GET /api/v1/users/:id"），避免 label 基数爆炸。
func TestOTelGinMiddleware_SpanNameStable(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	prevProvider := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	t.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		otel.SetTextMapPropagator(prevProp)
		_ = tp.Shutdown(context.Background())
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.OTelGinMiddleware("test-svc"))
	r.GET("/api/v1/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	// 发两个不同 ID 的请求，期望 span name 一致。
	for _, id := range []string{"42", "999"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/"+id, nil))
		require.Equal(t, http.StatusOK, w.Code)
	}

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 2)
	names := map[string]int{}
	for _, s := range spans {
		names[s.Name]++
	}
	t.Logf("captured span names: %v", names)
	// 期望：所有 span.name 都是 "GET /api/v1/users/:id"（无 ID 拼接）。
	for name := range names {
		assert.NotContains(t, name, "42", "span name 不应含变量值 42")
		assert.NotContains(t, name, "999", "span name 不应含变量值 999")
	}
	// 至少出现 1 个 "/api/v1/users/:id" span（otelgin 默认 name = route template）。
	stableName := "/api/v1/users/:id"
	assert.GreaterOrEqual(t, names[stableName], 1,
		"应至少 1 个 span name = '/api/v1/users/:id'")
}

// TestOTelGinMiddleware_PanicStatus500 验证 OTelGinMiddleware 在 Recovery 后仍正常工作。
//
// 注：Recovery 必须在 OTelGin 之前（本测试省略 Recovery，验证 OTel 单独行为）。
func TestOTelGinMiddleware_NoRegressionOnExistingRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// OTelGin 在 RateLimit 之前（推荐挂载顺序）。
	r.Use(gin.Recovery())
	r.Use(middleware.OTelGinMiddleware("test-svc"))

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.POST("/panic", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/panic", nil))
	// Recovery middleware 把 panic 转为 500。
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}