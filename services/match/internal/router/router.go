// Package router 注册 match-service HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/match/internal/handler"
	"github.com/growdu/doctors/services/match/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
)

// New 返回一个挂好路由的 *gin.Engine。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	v1 := r.Group("/api/v1", middleware.Auth(jwtSecret))
	h.RegisterRoutes(v1)

	internal := r.Group("/internal", middleware.Auth(jwtSecret))
	internal.POST("/match/dispatch", h.Dispatch)

	return r
}