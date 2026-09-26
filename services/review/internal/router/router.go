// Package router 注册 review-service 的 HTTP 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness）。
//   - /metrics 挂 Prometheus 抓取端（promhttp.Handler()）。
//   - Metrics 中间件（middleware.Metrics()）最先挂载：401/403 也计入 http_requests_total。
//   - /api/v1/reviews/* 全部挂 Auth（JWT）。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/review/internal/handler"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/metrics"
	"github.com/growdu/doctors/shared/middleware"
)

// New 构造 *gin.Engine。
//
// h 是业务 handler；jwtSecret 用于鉴权中间件。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	// Prometheus HTTP 指标中间件（最先挂）
	r.Use(middleware.Metrics())
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	auth := middleware.Auth(jwtSecret, "uid", middleware.RoleKey)
	v1 := r.Group("/api/v1", auth)
	h.RegisterRoutes(v1)
	return r
}