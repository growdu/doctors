// Package httpx 提供统一 HTTP 响应格式与辅助函数。
//
// 设计要点：
//   - 业务码在 body 中（code 字段），HTTP status 恒为 200。
//     这避免前端在 4xx 时拿不到完整 body，并允许 trace_id 始终可读。
//   - trace_id 优先复用 gin.Context 中已存在的 "trace_id"（由中间件注入），
//     不存在则生成新的 16 字节随机十六进制。
//   - 泛型 OK[T] 让 data 字段保持原类型（避免 interface{} 序列化噪声）。
package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// TraceIDKey 是 gin.Context 中 trace_id 的 key。
	TraceIDKey = "trace_id"
	// HeaderTraceID 是 HTTP header 中 trace_id 的名字，便于跨服务透传。
	HeaderTraceID = "X-Trace-Id"
)

// Resp 是统一的 JSON 响应结构。
type Resp[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
	TraceID string `json:"trace_id"`
}

// OK 写出成功响应。HTTP 状态固定为 200。
func OK[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, Resp[T]{
		Code:    0,
		Message: "ok",
		Data:    data,
		TraceID: traceIDOf(c),
	})
}

// Fail 写出业务错误响应。HTTP 状态固定为 200；业务码在 body.code 中。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Resp[any]{
		Code:    code,
		Message: msg,
		Data:    nil,
		TraceID: traceIDOf(c),
	})
}

// traceIDOf 复用上下文里已有的 trace_id；不存在则生成新的。
func traceIDOf(c *gin.Context) string {
	return TraceID(c)
}

// TraceID 返回当前请求的 trace_id：复用 gin.Context 中已存在的值或
// HeaderTraceID 请求头，缺失则生成新的 16-hex。供 Recover / RateLimit
// 等需要直接拼装 Resp 的中间件复用。
func TraceID(c *gin.Context) string {
	if v, ok := c.Get(TraceIDKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if h := c.GetHeader(HeaderTraceID); h != "" {
		return h
	}
	return newTraceID()
}

// newTraceID 生成 16 字节随机十六进制 trace id。
func newTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 极少见；降级为时间戳
		return "trace-fallback"
	}
	return hex.EncodeToString(b[:])
}