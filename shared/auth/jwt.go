// Package auth 提供 JWT 签发与校验。
//
// 设计要点：
//   - HS256 签名，secret 必须非空。
//   - Claims 携带 user_id / roles / active / role / unionid / 标准注册声明。
//   - v2（unified-app）：Roles []string 多角色并发 + Active 当前激活角色。
//     v1 Role 字段保留兼容老客户端；v1.1+ 移除（plan §4 Phase 10）。
//   - Sign/Parse 是 pure function，无副作用，便于测试。
//   - 解析失败统一返回 error，不暴露内部细节。
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是陪诊师平台自定义的 JWT 载荷。
type Claims struct {
	UserID  int64    `json:"uid"`
	Roles   []string `json:"roles,omitempty"` // v2：多角色并发；兼职 patient+escort
	Active  string   `json:"active,omitempty"` // v2：当前激活角色；前端按此切域
	Role    string   `json:"role"`            // v1 兼容字段；取 Active 或首个 Role
	UnionID string   `json:"unionid,omitempty"`
	jwt.RegisteredClaims
}

// HasRole 判断 user 是否有指定角色（v2 多角色校验）。
//
//	v1 仅 Role 字段时，HasRole 等价于 Role == role（向后兼容）。
func (c *Claims) HasRole(role string) bool {
	if role == "" {
		return false
	}
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	// v1 兼容：单 Role 字段
	if c.Role == role {
		return true
	}
	return false
}

// ActiveIsValid 校验 Active 字段合法性（v2 严格模式）。
//
//	true 当：
//	  - Active 在 Roles 中（v2 正常路径）
//	  - Active 为空且 Roles 非空（默认首个）
//	  - Roles 为空但 Role 非空（v1 向后兼容）
//	false 当：v2 已设 Active 但不在 Roles 中（视为非法激活，应触发 401 重登录）
func (c *Claims) ActiveIsValid() bool {
	// v1 兼容路径：Roles 为空 → 视 v1 单 Role 有效
	if len(c.Roles) == 0 {
		return c.Role != ""
	}
	// v2：Active 为空 → 默认首个 role 有效
	if c.Active == "" {
		return c.Roles[0] != ""
	}
	// v2：Active 必须存在于 Roles
	for _, r := range c.Roles {
		if r == c.Active {
			return true
		}
	}
	return false
}

// Sign 用 HS256 算法签发 token。secret 为空返回错误。
func Sign(secret string, c Claims) (string, error) {
	if secret == "" {
		return "", errors.New("auth: empty secret")
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString([]byte(secret))
}

// Parse 校验 token 并解析为 Claims。失败原因用 errors.Is(err, jwt.ErrToken*) 区分。
func Parse(secret string, tokenStr string) (*Claims, error) {
	if secret == "" {
		return nil, errors.New("auth: empty secret")
	}
	c := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("auth: invalid token")
	}
	return c, nil
}