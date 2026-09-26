// Package router 注册 wallet-service 的 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/wallet/internal/handler"
	"github.com/growdu/doctors/services/wallet/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/metrics"
	sharedmw "github.com/growdu/doctors/shared/middleware"
)

// New 返回挂好全部路由的 *gin.Engine。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	// Prometheus HTTP 指标中间件（最先挂）
	r.Use(sharedmw.Metrics())
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	auth := middleware.Auth(jwtSecret)
	h.RegisterRoutes(r, auth)
	return r
}