// Package router 注册 order-service 的 HTTP 路由。
//
// 设计要点：
//   - 路由只做挂载，不写业务逻辑。
//   - /orders/{id}/accept /cancel /finish 走同一前缀中间件。
//   - /healthz 不挂任何中间件（K8s liveness）。
//   - /metrics 挂 Prometheus 抓取端（promhttp.Handler()）。
//   - Metrics 中间件（middleware.Metrics()）最先挂载：401/403 也计入 http_requests_total。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/order/internal/handler"
	"github.com/growdu/doctors/services/order/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/metrics"
	sharedmw "github.com/growdu/doctors/shared/middleware"
)

// New 返回一个挂好全部路由的 *gin.Engine。
// h 是业务 handler；jwtSecret 用于鉴权中间件。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	// Prometheus HTTP 指标中间件（最先挂）
	r.Use(sharedmw.Metrics())
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	v1 := r.Group("/api/v1", middleware.Auth(jwtSecret))
	h.RegisterRoutes(v1)

	return r
}