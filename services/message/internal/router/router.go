// Package router 注册 message-service 的 HTTP 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness）。
//   - /metrics 挂 Prometheus 抓取端（promhttp.Handler()）。
//   - /api/v1/messages/* 全部挂 Auth（JWT）；broadcast 由 handler 内部取 uid 即可。
//
// 全局中间件挂载顺序（外 → 内）：
//
//	Metrics()        最外层：401/403/panic/429 也被埋点
//	Recovery()       panic 恢复 → 500 + 业务码 500000
//	RateLimit(...)   按 IP token bucket：超限 429 + 业务码 13001
//	Auth(...)        /api/v1/messages group 内挂，未通过 token 不消耗限流桶
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/message/internal/handler"
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
	r.Use(sharedmw.OTelGinMiddleware("message-service"))
	r.Use(sharedmw.RateLimit())
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})
	// readiness 探活
	r.GET("/readyz", health.ReadyzHandler(readyzM))
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	auth := sharedmw.Auth(jwtSecret, "uid", sharedmw.RoleKey)
	v1 := r.Group("/api/v1", auth)
	h.RegisterRoutes(v1)
	return r
}
