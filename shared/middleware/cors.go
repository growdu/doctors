// Package middleware — CORS（跨域资源共享）中间件（§48.K）。
//
// 用途：unified-app vite dev server（5174）跨域调用本服务（:8081-8091）。
// dev 模式只信任本地 127.0.0.1（5174 + 5180）；生产由 nginx 层加 CORS。
//
// 设计要点：
//   - 仅响应 Access-Control-Allow-* 头；不放宽业务逻辑
//   - OPTIONS 预检直接 204（gin 自动响应 + 自定义头）
//   - 默认 AllowOrigins = [127.0.0.1:5174, 127.0.0.1:5180, http://127.0.0.1:5174]
//   - 生产用 WithCORSAllowOrigins([]string{"https://app.example.com"}) 收紧
//
// 对应：dev.md §48.K（v2 unified-app 浏览器实测 CORS 缺失）。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 默认允许的 origin（dev 模式）
var defaultCORSOrigins = []string{
	"http://127.0.0.1:5174",
	"http://127.0.0.1:5180",
	"http://localhost:5174",
	"http://localhost:5180",
}

type corsConfig struct {
	allowOrigins []string
	allowMethods []string
	allowHeaders []string
}

// CORSOption 是 CORS 中间件的可选配置项。
type CORSOption func(*corsConfig)

// WithCORSAllowOrigins 设置允许的 origin 白名单。
func WithCORSAllowOrigins(origins []string) CORSOption {
	return func(c *corsConfig) {
		c.allowOrigins = origins
	}
}

// WithCORSAllowMethods 设置允许的 HTTP 方法。
func WithCORSAllowMethods(methods []string) CORSOption {
	return func(c *corsConfig) {
		c.allowMethods = methods
	}
}

// WithCORSAllowHeaders 设置允许的请求头。
func WithCORSAllowHeaders(headers []string) CORSOption {
	return func(c *corsConfig) {
		c.allowHeaders = headers
	}
}

// CORS 返回一个 gin.HandlerFunc，对每个请求设置 CORS 响应头。
//
// 用法：
//   - 挂载顺序：Metrics → Recovery → CORS → OTelGin → RateLimit → Auth
//   - CORS 必须在 Auth 之前（预检不需鉴权）
//   - OPTIONS 预检直接 204 返回，不进入下游 handler
func CORS(opts ...CORSOption) gin.HandlerFunc {
	cfg := &corsConfig{
		allowOrigins: defaultCORSOrigins,
		allowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		allowHeaders: []string{
			"Origin", "Content-Type", "Content-Length",
			"Accept-Encoding", "Authorization",
			"X-Trace-Id", "X-Requested-With",
		},
	}
	for _, o := range opts {
		o(cfg)
	}

	allowOriginsSet := make(map[string]struct{}, len(cfg.allowOrigins))
	for _, o := range cfg.allowOrigins {
		allowOriginsSet[o] = struct{}{}
	}
	allowMethodsStr := strings.Join(cfg.allowMethods, ", ")
	allowHeadersStr := strings.Join(cfg.allowHeaders, ", ")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			// 仅当 origin 在白名单内才回 Allow-Origin
			if _, ok := allowOriginsSet[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", allowMethodsStr)
				c.Header("Access-Control-Allow-Headers", allowHeadersStr)
				c.Header("Access-Control-Max-Age", "600")
			}
		}

		// 预检请求直接返回 204
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}