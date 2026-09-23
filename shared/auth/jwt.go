// Package auth 提供 JWT 签发与校验。
//
// 设计要点：
//   - HS256 签名，secret 必须非空。
//   - Claims 携带 user_id / role / unionid / 标准注册声明。
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
	UserID  int64  `json:"uid"`
	Role    string `json:"role"`
	UnionID string `json:"unionid,omitempty"`
	jwt.RegisteredClaims
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