// Package router 注册 escort-service 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness）。
//   - /api/v1/escorts/* 全部挂 Auth（JWT）。
//   - 公开端点（ListByEscort availability）单独挂到 no-auth 的 group（来自 availability 子包 handler）。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
)

// New 返回挂好路由的 gin engine。
//
// h 是业务 escort handler；availH 是 availability 子包 handler；jwtSecret 用于鉴权中间件。
func New(h *handler.Handler, availH *availability.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})

	auth := middleware.Auth(jwtSecret, middleware.UserIDKey, middleware.RoleKey)
	v1 := r.Group("/api/v1", auth)
	h.RegisterRoutes(v1)
	availH.RegisterRoutes(v1)
	return r
}

// NewWithPublic 在 New 之上额外挂一个无 auth 的 group（用于 availability 公开接口）。
func NewWithPublic(h *handler.Handler, availH *availability.Handler, jwtSecret string) *gin.Engine {
	r := New(h, availH, jwtSecret)
	public := r.Group("/api/v1")
	availH.RegisterPublicRoutes(public)
	return r
}
