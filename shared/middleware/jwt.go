// Package middleware 提供跨服务复用的 JWT 鉴权中间件。
//
// 与 services/*/internal/middleware 不同：本包是通用模板，
// 各服务用自己的 ctx key 包装，避免 ctx key 泄露到 shared。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Auth 校验 Bearer JWT；userIDKey / roleKey 是各服务自定义的 gin.Context key。
// 失败统一返回 401 + 业务码 11001。
func Auth(secret, userIDKey, roleKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			httpx.Fail(c, int(errs.CodeUnauthorized), "missing bearer token")
			c.Abort()
			return
		}
		claims, err := authpkg.Parse(secret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			httpx.Fail(c, int(errs.CodeUnauthorized), "invalid token")
			c.Abort()
			return
		}
		c.Set(userIDKey, claims.UserID)
		c.Set(roleKey, claims.Role)
		c.Next()
	}
}