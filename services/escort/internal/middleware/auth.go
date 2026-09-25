// Package middleware - JWT Auth + role helpers。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// UserIDKey 是 gin.Context 中 user id 的 key。
const UserIDKey = "escort_user_id"

// RoleKey 是 gin.Context 中 role 的 key。
const RoleKey = "escort_role"

// Auth 校验 Bearer JWT；userIDKey / roleKey 是 ctx key。
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

// UserID 便捷取 ctx 中的 user id。
func UserID(c *gin.Context) int64 {
	v, _ := c.Get(UserIDKey)
	id, _ := v.(int64)
	return id
}

// Role 便捷取 ctx 中的 role。
func Role(c *gin.Context) string {
	v, _ := c.Get(RoleKey)
	s, _ := v.(string)
	return s
}
