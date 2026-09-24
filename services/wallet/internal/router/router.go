// Package router 注册 wallet-service 的 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/wallet/internal/handler"
	"github.com/growdu/doctors/services/wallet/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
)

// New 返回挂好全部路由的 *gin.Engine。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	auth := middleware.Auth(jwtSecret)
	h.RegisterRoutes(r, auth)
	return r
}