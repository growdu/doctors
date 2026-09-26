// Package middleware —— OTel auto-instrumentation 入口（§34）。
//
// 设计要点：
//   - gin / pgx / kafka-go 三个最常用依赖的 OTel auto-tracer
//     集中在 shared/middleware + shared/tracing 包；
//   - 业务 router / cmd/main.go 一行接入，无需知道三方包路径。
//   - OTel endpoint 为空时（dev / 单测），InitTracer 已退化为 Noop；
//     otelgin.Middleware 在 Noop 场景是 zero-cost（不创建 span）。
//
// 使用：
//
//	// router.go
//	r.Use(sharedmw.Metrics(), sharedmw.Recovery(), sharedmw.RateLimit())
//	r.Use(sharedmw.OTelGinMiddleware("wallet-service")) // 加在 Recovery 之后、RateLimit 之前
//
//	// cmd/main.go
//	tracing.InitTracer("wallet-service", cfg.Tracing.OTLPEndpoint, ...)
//	pcfg, _ := pgxpool.ParseConfig(cfg.DB.DSN)
//	tracing.WithPgxPool(pcfg)  // SQL 自动写 span
//	pool := pgxpool.NewWithConfig(ctx, pcfg)
//
//	// kafka publisher
//	w := kafka.NewWriter(brokers, topic)
//	wrapped := sharedtracing.WrapWriter(w, "wallet-service")
//
//	// kafka consumer
//	r := kafka.NewReader(brokers, topic, groupID)
//	wrapped := sharedtracing.WrapReader(r, "wallet-service")
package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// OTelGinMiddleware 返回 gin.HandlerFunc：把每个 HTTP 请求包装成 OTel span。
//
// 设计要点：
//   - otelgin 自动从 W3C traceparent 头读出上游 span context；
//     与 shared/tracing.Extract / Inject 配合实现跨服务链路透传。
//   - otelgin.Middleware(service) → span name = HTTP method + route template
//     （如 "GET /api/v1/users/:id"），比 URL.Path 更稳定（避免高基数）。
//   - 注入到 c.Request.Context()，业务 handler 通过 ctx 写 SQL / Kafka span
//     自动挂到 HTTP span 下（tracing tree）。
//
// 推荐挂载顺序：Metrics → Recovery → OTelGinMiddleware → RateLimit → Auth → business。
// Recovery 在 OTelGinMiddleware 之前——panic 时 OTel span 也被记录（状态码=500）。
func OTelGinMiddleware(service string) gin.HandlerFunc {
	return otelgin.Middleware(service)
}