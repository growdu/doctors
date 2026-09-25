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
// 不在白名单 → 403 + 业务码 11003（CodeAdminForbidden）。
// Auth 中间件未先跑 → role 为空字符串 → 401（视为未登录）。
func RoleAuth(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role := c.GetString(RoleKey)
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