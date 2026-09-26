// Package middleware 中间件补充：metrics 接入。
//
// 与 services/*/internal/middleware 不同：本文件是 shared/metrics 的薄包装，
// 各服务在 gin engine 上 Use 即可自动埋 HTTP 指标。
//
// 使用：
//
//	import "github.com/growdu/doctors/shared/middleware"
//
//	engine := gin.New()
//	engine.Use(middleware.Metrics())   // 全局：每个请求自动记 method/path/status/duration
//	engine.GET("/metrics", gin.WrapH(metrics.Handler()))
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/metrics"
)

// Metrics 返回 gin.HandlerFunc，自动把每个请求写入：
//   - http_requests_total{method, path, status}
//   - http_request_duration_seconds{method, path}
//
// 这是 shared/metrics.GinMiddleware() 的语义别名；放在 shared/middleware
// 便于 11 服务统一导入路径，避免每个 router 直接依赖 shared/metrics。
//
// 推荐挂载顺序：Metrics() 应早于业务中间件（Auth / RoleAuth），
// 以便 401/403 也被计入 http_requests_total。
func Metrics() gin.HandlerFunc {
	return metrics.GinMiddleware()
}
