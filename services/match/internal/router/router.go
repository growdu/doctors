// Package router 注册 match-service HTTP 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness）。
//   - /metrics 挂 Prometheus 抓取端（promhttp.Handler()）。
//   - /api/v1/* 与 /internal/* 全部挂 Auth（JWT）。
//
// 全局中间件挂载顺序（外 → 内）：
//
//	Metrics()        最外层：401/403/panic/429 也被埋点
//	Recovery()       panic 恢复 → 500 + 业务码 500000
//	RateLimit(...)   按 IP token bucket：超限 429 + 业务码 13001
//	Auth(...)        group 级挂载，未通过 token 不消耗限流桶
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/match/internal/handler"
	mw "github.com/growdu/doctors/services/match/internal/middleware"
	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/metrics"
	sharedmw "github.com/growdu/doctors/shared/middleware"
)

// New 返回一个挂好路由的 *gin.Engine。
// readyzM 用于 /readyz 端点（K8s readinessProbe）；传 nil 时 /readyz 永远 503（fail-closed）。
func New(h *handler.Handler, jwtSecret string, readyzM *health.Manager) *gin.Engine {
	r := gin.New()
	// 中间件顺序：Metrics → Recovery → RateLimit（全局）
	r.Use(sharedmw.Metrics())
	r.Use(sharedmw.Recovery())
	r.Use(sharedmw.RateLimit())
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	// readiness 探活：所有依赖（Kafka）都 OK 才 200
	r.GET("/readyz", health.ReadyzHandler(readyzM))
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	v1 := r.Group("/api/v1", mw.Auth(jwtSecret))
	h.RegisterRoutes(v1)

	internal := r.Group("/internal", mw.Auth(jwtSecret))
	internal.POST("/match/dispatch", h.Dispatch)

	return r
}
