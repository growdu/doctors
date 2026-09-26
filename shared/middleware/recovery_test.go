package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/logger"
	"github.com/growdu/doctors/shared/middleware"
)

// withObservedLogger 替换 shared/logger 全局为带 observer 的 zap logger，
// 让测试能拿到 Recovery 写出的日志条目。返回 cleanup func。
//
// shared/logger 的 SetForTest 是官方测试钩子（无需新增 export），
// ResetForTest 把全局恢复默认。
func withObservedLogger(t *testing.T) (*observer.ObservedLogs, func()) {
	t.Helper()
	core, recorded := observer.New(zapcore.ErrorLevel)
	prev := zap.New(core)
	logger.SetForTest(prev)
	return recorded, func() { logger.ResetForTest() }
}

// 触发 panic 的 handler，用于验证 Recovery 能吞掉 panic。
func panickingHandler(c *gin.Context) {
	panic("boom")
}

// TestRecovery_PanicReturns500AndInternalCode 验证 panic → 500 + body.code=500000。
func TestRecovery_PanicReturns500AndInternalCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/explode", panickingHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/explode", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code,
		"panic 后必须返回 HTTP 500")
	var resp httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeInternal), resp.Code,
		"业务码必须为 errs.CodeInternal(500000)")
	assert.Equal(t, "internal error", resp.Message,
		"对外 message 不暴露 panic 细节")
	assert.NotEmpty(t, resp.TraceID, "响应须带 trace_id")
}

// TestRecovery_MultiplePanicsKeepServing 验证多次 panic 不影响后续请求：
//
//	1. 第一次请求 → panic → 500
//	2. 第二次请求 → 同样 panic → 仍 500
//	3. 第三次请求 → 正常 → 200
//
// 不变量：进程 / engine 未被一次 panic 拖垮。
func TestRecovery_MultiplePanicsKeepServing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/explode", panickingHandler)
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hello": "world"})
	})

	// 第一次 panic
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/explode", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// 第二次 panic
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/explode", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// 后续正常请求依然 200
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "world")
}

// TestRecovery_LogsStackTrace 验证默认开启 stack 抓取且日志字段齐全。
func TestRecovery_LogsStackTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logs, restore := withObservedLogger(t)
	defer restore()

	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/explode", panickingHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/explode", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// 必须记录一条 Error + 包含 stack / path / method / client_ip
	matched := logs.FilterMessage("panic recovered")
	require.Equal(t, 1, matched.Len(), "应记录 1 条 panic 日志")
	entry := matched.All()[0]
	fields := fieldsOf(entry.Context)
	assert.Contains(t, fields, "panic", "应包含 panic 字段")
	assert.Contains(t, fields, "stack", "默认开启 stack 抓取")
	assert.Contains(t, fields, "path")
	assert.Equal(t, "/explode", fields["path"])
	assert.Contains(t, fields, "method")
	assert.Equal(t, "GET", fields["method"])
}

// TestRecovery_StackTraceDisabled 验证 WithRecoveryStackTrace(false) 关闭 stack。
//
// 通过 WithRecoveryLogger 注入 observer-logger 仍能验证 stack 字段缺席。
func TestRecovery_StackTraceDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, recorded := observer.New(zapcore.ErrorLevel)
	rec := zap.New(core)

	r := gin.New()
	r.Use(middleware.Recovery(
		middleware.WithRecoveryLogger(rec),
		middleware.WithRecoveryStackTrace(false),
	))
	r.GET("/explode", panickingHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/explode", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)

	matched := recorded.FilterMessage("panic recovered")
	require.Equal(t, 1, matched.Len())
	fields := fieldsOf(matched.All()[0].Context)
	assert.NotContains(t, fields, "stack", "关闭 stack 后不应包含 stack 字段")
	assert.Contains(t, fields, "path", "其他字段仍记录")
}

// fieldsOf 把 zapcore.Field 列表拍平成 map[string]any，便于断言。
func fieldsOf(fs []zapcore.Field) map[string]any {
	out := make(map[string]any, len(fs))
	for _, f := range fs {
		out[f.Key] = fieldAny(f)
	}
	return out
}

// fieldAny 提取 zapcore.Field 的值（仅覆盖测试会用到的类型）。
func fieldAny(f zapcore.Field) any {
	switch f.Type {
	case zapcore.StringType:
		return f.String
	case zapcore.ByteStringType:
		if b, ok := f.Interface.([]byte); ok {
			return string(b)
		}
		return ""
	default:
		if f.Interface != nil {
			return f.Interface
		}
		return f.String
	}
}
