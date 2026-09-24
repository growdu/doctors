// Package router 注册 order-service 的 HTTP 路由。
//
// 设计要点：
//   - 路由只做挂载，不写业务逻辑。
//   - /orders/{id}/accept /cancel /finish 走同一前缀中间件。
//   - /healthz 不挂任何中间件（K8s liveness）。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/order/internal/handler"
	"github.com/growdu/doctors/services/order/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
)

// New 返回一个挂好全部路由的 *gin.Engine。
// h 是业务 handler；jwtSecret 用于鉴权中间件。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1", middleware.Auth(jwtSecret))
	h.RegisterRoutes(v1)

	return r
}