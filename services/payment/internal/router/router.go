// Package router 注册 payment-service 的 HTTP 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness + Docker HEALTHCHECK）。
//   - /api/v1/payments/* 挂 Auth（JWT），ctx key = "user_id" / "role"（与 handler 的 mustUserID 对齐）。
//   - 角色白名单：v1 简化——patient / escort / admin / super_admin 都能调（无 RoleAuth）；
//     后续按 plan 2026-09-24-payment.md 加细分权限。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/payment/internal/handler"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/middleware"
)

// New 构造 *gin.Engine。
//
// h 是业务 handler；jwtSecret 用于鉴权中间件。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})

	auth := middleware.Auth(jwtSecret, "user_id", middleware.RoleKey)
	v1 := r.Group("/api/v1", auth)
	h.RegisterRoutes(v1)
	return r
}