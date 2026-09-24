// Package router 注册 auth-service 的 HTTP 路由。
//
// 设计要点：
//   - router 只做路由挂载，不写业务逻辑。
//   - 业务 handler 由 handler 包注入，server 启动时再装配。
//   - /healthz 直接返回 200 + OK，不挂任何中间件（K8s liveness）。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/shared/httpx"
)

// New 返回一个挂好全部路由的 *gin.Engine。
// h 持有 service 引用；jwtSecret 用于 /users/* 鉴权。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	// liveness 探活
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})

	h.RegisterRoutes(r, jwtSecret)
	return r
}