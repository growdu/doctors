// Package router 注册 auth-service 的 HTTP 路由。
//
// 设计要点：
//   - 路由集中注册，避免散落在 main.go 与各 handler 包中。
//   - /healthz 直接返回 200 + OK，不挂中间件（便于 K8s liveness）。
//   - /api/v1/auth 与 /api/v1/users 前缀与文档一致。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/httpx"
)

// New 返回一个挂好全部路由的 *gin.Engine。
// 后续阶段会传入 service 层依赖；目前为空，仅占位。
func New() *gin.Engine {
	r := gin.New()
	// liveness: 不挂任何中间件，单纯探活
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/sms/send", notImplemented("auth.sms.send"))
		auth.POST("/login", notImplemented("auth.login"))
		auth.POST("/refresh", notImplemented("auth.refresh"))

		users := v1.Group("/users")
		users.POST("/real-name/auth", notImplemented("users.real_name.auth"))
		users.GET("/me", notImplemented("users.me"))
	}

	return r
}

// notImplemented 临时占位 handler：返回 501 业务码 0 + message。
// 后续阶段会替换为真实业务 handler。
func notImplemented(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.Fail(c, 0, "not implemented: "+name)
	}
}