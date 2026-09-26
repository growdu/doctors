// Package router 注册 payment-service 的 HTTP 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness + Docker HEALTHCHECK）。
//   - /metrics 挂 Prometheus 抓取端（promhttp.Handler()）。
//   - /api/v1/payments/* 挂 Auth（JWT），ctx key = "user_id" / "role"（与 handler 的 mustUserID 对齐）。
//   - 角色白名单：v1 简化——patient / escort / admin / super_admin 都能调（无 RoleAuth）；
//     后续按 plan 2026-09-24-payment.md 加细分权限。
//
// 全局中间件挂载顺序（外 → 内）：
//
//	Metrics()        最外层：401/403/panic/429 也被埋点
//	Recovery()       panic 恢复 → 500 + 业务码 500000
//	RateLimit(...)   按 IP token bucket：超限 429 + 业务码 13001
//	Auth(...)        /api/v1/payments group 内挂，未通过 token 不消耗限流桶
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/payment/internal/handler"
	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/metrics"
	sharedmw "github.com/growdu/doctors/shared/middleware"
)

// New 构造 *gin.Engine。
//
// h 是业务 handler；jwtSecret 用于鉴权中间件；
// readyzM 用于 /readyz 端点（K8s readinessProbe）；传 nil 时 /readyz 永远 503（fail-closed）。
func New(h *handler.Handler, jwtSecret string, readyzM *health.Manager) *gin.Engine {
	r := gin.New()
	// 中间件顺序：Metrics → Recovery → OTelGin → RateLimit（全局）
	r.Use(sharedmw.Metrics())
	r.Use(sharedmw.Recovery())
	r.Use(sharedmw.OTelGinMiddleware("payment-service"))
	r.Use(sharedmw.RateLimit())
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	// readiness 探活
	r.GET("/readyz", health.ReadyzHandler(readyzM))
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	auth := sharedmw.Auth(jwtSecret, "user_id", sharedmw.RoleKey)
	v1 := r.Group("/api/v1", auth)
	h.RegisterRoutes(v1)
	return r
}
