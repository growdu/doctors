// Package middleware 提供跨服务复用的 JWT 鉴权中间件。
//
// 与 services/*/internal/middleware 不同：本包是通用模板，
// 各服务用自己的 ctx key 包装，避免 ctx key 泄露到 shared。
//
// v2（unified-app）：额外把完整 *auth.Claims 写入 ctx，便于 RoleAuth 用 Claims.HasRole 校验多角色并发。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// ClaimsKey 是 gin.Context 里 *auth.Claims 的 key（v2 多角色）。
const ClaimsKey = "claims"

// Auth 校验 Bearer JWT；userIDKey / roleKey 是各服务自定义的 gin.Context key。
// 失败统一返回 401 + 业务码 11001。
//
// v2：同时把完整 *auth.Claims 写到 ClaimsKey，便于后续 RoleAuth 用 HasRole 校验。
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
		c.Set(roleKey, claims.Role) // v1 兼容字段
		c.Set(ClaimsKey, claims)    // v2 多角色：完整 claims 注入 ctx
		c.Next()
	}
}

// ClaimsFromCtx 读 ctx 里的 *auth.Claims；nil-safe。
func ClaimsFromCtx(c *gin.Context) *authpkg.Claims {
	v, ok := c.Get(ClaimsKey)
	if !ok {
		return nil
	}
	cl, _ := v.(*authpkg.Claims)
	return cl
}