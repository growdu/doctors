// Package middleware 提供跨服务复用的鉴权中间件。
//
// RoleAuth 假定 Auth 中间件已先注入 role 到 gin.Context。
//
// 使用：
//
//	v1.Use(middleware.Auth(secret, "uid", "role"))
//	v1.POST("/orders/:id/force-cancel",
//	    middleware.RoleAuth("super_admin", "order_admin"),
//	    h.ForceCancel)
//
// v2（unified-app）：RoleAuth 同时校验 token.active 和 roles 列表：
//   - v1 token：ctx["role"] 走 fallback
//   - v2 token：active ∈ roles 时 ctx["role"] = active；Roles 数组也校验命中
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// RoleKey 是 gin.Context 中 role 字段的 key（与 Auth 的 roleKey 一致）。
const RoleKey = "role"

// RoleAuth 校验当前请求 role 是否在白名单（OR）。
//
// v2 多角色：先看 ClaimsFromCtx → Claims.HasRole 任一命中 allowedRoles 即通过；
// 否则看 ctx[role]（v1 兼容字段，等于 active 或首个 role）。
// 不在白名单 → 403 + 业务码 11003（CodeAdminForbidden）。
// Auth 中间件未先跑 → role 为空字符串 → 401（视为未登录）。
func RoleAuth(allowedRoles ...string) gin.HandlerFunc {
	return RoleAuthWithKey(RoleKey, allowedRoles...)
}

// RoleAuthWithKey 是 RoleAuth 的 ctx-key 可配置版本；用于各 service 自定义 ctx key
// （如 order 用 "order_role"，match 用 "match_role"）。
//
// 优先级：v2 ClaimsFromCtx（与 RoleKey 无关）→ ctx[roleKey]。
func RoleAuthWithKey(roleKey string, allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		// v2 优先：Claims.HasRole
		if cl := ClaimsFromCtx(c); cl != nil {
			for _, r := range allowedRoles {
				if cl.HasRole(r) {
					c.Next()
					return
				}
			}
			if _, ok := allowed[cl.Role]; ok {
				c.Next()
				return
			}
			httpx.Fail(c, int(errs.CodeAdminForbidden),
				"active role not in allowlist")
			c.Abort()
			return
		}
		// v1 fallback：ctx[roleKey]
		role := c.GetString(roleKey)
		if role == "" {
			httpx.Fail(c, int(errs.CodeUnauthorized), "missing role")
			c.Abort()
			return
		}
		if _, ok := allowed[role]; !ok {
			httpx.Fail(c, int(errs.CodeAdminForbidden),
				"role "+role+" not in allowlist")
			c.Abort()
			return
		}
		c.Next()
	}
}