// Package router 注册 auth-service 的 HTTP 路由。
//
// 设计要点：
//   - router 只做路由挂载，不写业务逻辑。
//   - 业务 handler 由 handler 包注入，server 启动时再装配。
//   - /healthz 直接返回 200 + OK（K8s liveness，不依赖外部资源）。
//   - /readyz 返回 200/503（K8s readiness，反映 DB / Kafka 等依赖是否就绪）。
//   - /metrics 挂 Prometheus 抓取端（promhttp.Handler()）。
//
// 全局中间件挂载顺序（外 → 内）：
//
//	Metrics()        最外层：401/403/panic/429 也被埋点
//	Recovery()       panic 恢复 → 500 + 业务码 500000
//	RateLimit(...)   按 IP token bucket：超限 429 + 业务码 13001
//	Auth(...)        /users/* 子 group 内挂，未通过 token 校验不会被限流计数器浪费
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/metrics"
	sharedmw "github.com/growdu/doctors/shared/middleware"
)

// New 返回一个挂好全部路由的 *gin.Engine。
// h 持有 service 引用；jwtSecret 用于 /users/* 鉴权；
// readyzM 用于 /readyz 端点（K8s readinessProbe）；传 nil 时 /readyz 永远 503（fail-closed）。
func New(h *handler.Handler, jwtSecret string, readyzM *health.Manager) *gin.Engine {
	r := gin.New()
	// 中间件顺序：Metrics → Recovery → OTelGin → RateLimit（全局）
	r.Use(sharedmw.Metrics())
	r.Use(sharedmw.Recovery())
	r.Use(sharedmw.OTelGinMiddleware("auth-service"))
	r.Use(sharedmw.RateLimit())
	// liveness 探活
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	// readiness 探活：所有依赖（DB / Kafka）都 OK 才 200
	r.GET("/readyz", health.ReadyzHandler(readyzM))
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	h.RegisterRoutes(r, jwtSecret)
	return r
}
