// Package middleware 提供 auth-service 专用中间件。
//
// Auth 用 Authorization: Bearer <jwt> 解析 token，把 user_id 写入 gin.Context。
// 缺失 / 失效一律 401，业务码 11001。
//
// v2（unified-app）：额外把完整 *auth.Claims 写入 ctx，便于 handler 读 roles/active。
// 保留 UserIDKey / RoleKey（v1 单 role 兼容），老代码不变。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

const (
	// UserIDKey 是 gin.Context 里 user_id 的 key。
	UserIDKey = "auth_user_id"
	// RoleKey 是 gin.Context 里 role 的 key（v1 兼容）。
	RoleKey = "auth_role"
	// ClaimsKey 是 gin.Context 里 *auth.Claims 的 key（v2 多角色）。
	ClaimsKey = "auth_claims"
)

// Auth 校验 Bearer JWT；通过则把 claim 写入 ctx。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			httpx.Fail(c, int(errs.CodeUnauthorized), "missing bearer token")
			c.Abort()
			return
		}
		tok := strings.TrimPrefix(h, "Bearer ")
		claims, err := authpkg.Parse(secret, tok)
		if err != nil {
			httpx.Fail(c, int(errs.CodeUnauthorized), "invalid token")
			c.Abort()
			return
		}
		c.Set(UserIDKey, claims.UserID)
		c.Set(RoleKey, claims.Role)
		c.Set(ClaimsKey, claims) // v2：完整 claims 注入 ctx
		c.Next()
	}
}

// Claims 从 ctx 读取 *auth.Claims（v2 多角色场景）。
//
//	缺失返回 nil；调用方需 nil-check。
func Claims(c *gin.Context) *authpkg.Claims {
	v, ok := c.Get(ClaimsKey)
	if !ok {
		return nil
	}
	cl, _ := v.(*authpkg.Claims)
	return cl
}

// UserID 从 ctx 读取 user_id；缺失返回 0。
func UserID(c *gin.Context) int64 {
	v, ok := c.Get(UserIDKey)
	if !ok {
		return 0
	}
	id, _ := v.(int64)
	return id
}

// Role 从 ctx 读取 role。
func Role(c *gin.Context) string {
	v, _ := c.Get(RoleKey)
	s, _ := v.(string)
	return s
}