// Package middleware 提供跨服务复用的 panic recovery 中间件。
//
// Recovery 捕获下游 handler / 子中间件抛出的 panic，记录 stack trace，
// 返回统一业务错误响应（500 + 业务码 CodeInternal），避免进程崩溃。
//
// 设计要点：
//   - 业务码沿用 errs.CodeInternal（500000）：与系统内部错误语义一致；
//     HTTP 状态码返回 500，让 ingress / 监控能直接识别"服务器异常"。
//   - 不再次 panic：defer + recover 已吞掉，即使后续 Abort 失败也不会让
//     goroutine 退出。
//   - 进程可继续处理下一个请求：每次 panic 只影响当前请求，连接被关闭，
//     其他 in-flight / 新建请求不受影响。
//   - 默认日志输出走 shared/logger.L()，便于通过 SetLevel / SetForTest
//     注入测试 observer。
//   - 支持 WithRecoveryLogger 注入外部 *zap.Logger；支持
//     WithRecoveryStackTrace(false) 关闭 stack 抓取（生产默认开启）。
//
// 使用：
//
//	engine.Use(middleware.Recovery())              // 默认：zap 默认 logger + stack
//	engine.Use(middleware.Recovery(                // 自定义 logger + 关闭 stack
//	    middleware.WithRecoveryLogger(myLogger),
//	    middleware.WithRecoveryStackTrace(false),
//	))
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/logger"
)

// RecoveryOption 配置 Recovery。
type RecoveryOption func(*recoveryConfig)

type recoveryConfig struct {
	logger    *zap.Logger
	withStack bool
}

// WithRecoveryLogger 注入自定义 *zap.Logger。
//
// 未注入时默认走 shared/logger.L()，便于测试时通过 logger.SetForTest 替换。
func WithRecoveryLogger(l *zap.Logger) RecoveryOption {
	return func(c *recoveryConfig) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithRecoveryStackTrace 控制是否在日志中记录 stack trace。
//
// 默认 true（开启）；生产可关闭以减少日志量，但建议保持开启便于事后排查。
func WithRecoveryStackTrace(on bool) RecoveryOption {
	return func(c *recoveryConfig) {
		c.withStack = on
	}
}

// Recovery 返回 gin.HandlerFunc：捕获 panic → 记录日志 → 返回 500 + 业务码。
//
// 关键不变量：
//   - 永远不再 panic：defer 内嵌套两层 recover —— 即使 AbortWithStatusJSON 自身
//     panic（极端边界，例如 c.Writer 已损坏）也不会让 goroutine 退出。
//   - 进程不会崩溃：连接关闭后继续接受新请求。
//   - HTTP 状态 = 500（real server error，便于 ingress / 监控识别）。
//   - body.code = errs.CodeInternal（业务码 500000）。
//   - body.message 不暴露 panic 细节（仅记录到日志），防信息泄露。
//
// 推荐挂载顺序：Metrics() → Recovery() → RateLimit() → Auth() → business。
// Recovery 必须早于业务 handler，但晚于 Metrics（让 panic 请求也能被埋点）。
func Recovery(opts ...RecoveryOption) gin.HandlerFunc {
	cfg := recoveryConfig{
		logger:    logger.L(),
		withStack: true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return func(c *gin.Context) {
		// 第一层 defer：捕获业务 handler panic
		defer func() {
			r := recover()
			if r == nil {
				return
			}

			// 1. 记录 stack trace 与 panic 值（仅日志，不进 body）
			fields := []zap.Field{
				zap.Any("panic", r),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
			}
			if cfg.withStack {
				fields = append(fields, zap.ByteString("stack", debug.Stack()))
			}
			cfg.logger.Error("panic recovered", fields...)

			// 2. 双层防御：tryResponse 里再 recover 一次，避免 c.Writer 已损坏
			//    导致 AbortWithStatusJSON 二次 panic 把 goroutine 带走。
			tryResponse(c)
		}()
		c.Next()
	}
}

// tryResponse 安全写出 500 + body。任何 panic 都吞掉。
//
// 之所以独立为函数，是为了 defer 链路更清晰，并明确"二次 panic 防御"语义。
func tryResponse(c *gin.Context) {
	defer func() {
		_ = recover()
	}()
	c.AbortWithStatusJSON(http.StatusInternalServerError, httpx.Resp[any]{
		Code:    int(errs.CodeInternal),
		Message: "internal error",
		Data:    nil,
		// httpx.TraceID 优先复用 gin.Context 已注入或 Header 的 trace_id，
		// 缺失则生成新的 16-hex —— 与 OK/Fail 行为一致。
		TraceID: httpx.TraceID(c),
	})
}
