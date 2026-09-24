// Package wxlogin 抽象微信小程序 / APP 登录。
//
// 设计要点：
//   - 业务层只依赖 Client 接口，不接触 wx.* API 细节。
//   - 默认 MockClient 把 code 直接当成 unionid / openid 拼接，
//     便于本地端到端走通；真渠道在 v2 接入。
//   - 真实渠道需在 wx.login → wx.rest.code2session 流程
//     拿到 unionid / openid；本包只定义契约。
package wxlogin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// ErrEmptyCode 是空 code 的错误。
var ErrEmptyCode = errors.New("wxlogin: empty code")

// Client 抽象微信 code2session；返回 unionid / openid。
type Client interface {
	Code2Session(ctx context.Context, code string) (unionid, openid string, err error)
}

// MockClient 是开发期的默认实现。
type MockClient struct{}

// NewMockClient 构造 MockClient。
func NewMockClient() *MockClient { return &MockClient{} }

// Code2Session 把 code 转成 unionid / openid：
//   - 以 "wx-mock-" 开头时，code 去掉前缀直接用作 unionid / openid。
//   - 其它情况用 sha256(code)[:16] 作为 unionid / openid，保证稳定。
func (c *MockClient) Code2Session(ctx context.Context, code string) (string, string, error) {
	if strings.TrimSpace(code) == "" {
		return "", "", ErrEmptyCode
	}
	if strings.HasPrefix(code, "wx-mock-") {
		suffix := strings.TrimPrefix(code, "wx-mock-")
		return "unionid-" + suffix, "openid-" + suffix, nil
	}
	sum := sha256.Sum256([]byte(code))
	hexStr := hex.EncodeToString(sum[:])[:16]
	return "unionid-" + hexStr, "openid-" + hexStr, nil
}