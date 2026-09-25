// Package router 注册 admin-service 的 HTTP 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness）。
//   - /api/v1/admin/* 全部挂 Auth（JWT）+ RoleAuth（白名单按 API 区分）。
//   - 角色白名单矩阵与 admin-web-design.md §3.2 一致。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/admin/internal/handler"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/middleware"
)

// New 构造 *gin.Engine。
//
// h 是业务 handler；jwtSecret 用于鉴权中间件。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})

	auth := middleware.Auth(jwtSecret, "uid", middleware.RoleKey)
	v1 := r.Group("/api/v1", auth)
	h.RegisterRoutes(v1)
	return r
}