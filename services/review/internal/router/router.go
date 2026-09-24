// Package router - review-service 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/review/internal/handler"
	"github.com/growdu/doctors/services/review/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
)

func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) { httpx.OK[any](c, gin.H{"status": "ok"}) })
	v1 := r.Group("/api/v1", middleware.Auth(jwtSecret))
	h.RegisterRoutes(v1)
	return r
}
