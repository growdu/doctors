// Package middleware - JWT Auth 中间件（wallet-service）。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

const (
	UserIDKey = "user_id"
	RoleKey   = "role"
)

// Auth 校验 Bearer JWT；userID 塞 ctx；后续 handler 用 mustUserID(c) 取。
// 失败统一返回 401 + 业务码 11001。
func Auth(secret string) gin.HandlerFunc {
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
		c.Set(UserIDKey, claims.UserID)
		c.Set(RoleKey, claims.Role)
		c.Next()
	}
}

// UserID 从 ctx 读 user_id。
func UserID(c *gin.Context) int64 {
	v, _ := c.Get(UserIDKey)
	id, _ := v.(int64)
	return id
}

// Role 从 ctx 读 role。
func Role(c *gin.Context) string {
	v, _ := c.Get(RoleKey)
	s, _ := v.(string)
	return s
}