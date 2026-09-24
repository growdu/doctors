// Package realname 抽象实名认证。
//
// 设计要点：
//   - 真实接入对接公安二要素（姓名 + 身份证号）；本期不接真渠道。
//   - MockVerifier 仅做格式校验 + 哈希，不调外网。
//   - 永远不返回明文身份证号；只返回 sha256(idCard + salt) 与末四位。
//   - salt 必须在 main 启动时从 config.Auth 注入；不可硬编码。
package realname

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// ErrInvalidInput 是字段缺失 / 长度不够时的错误。
var ErrInvalidInput = errors.New("realname: invalid input")

// Verifier 抽象实名核验。
type Verifier interface {
	// Verify 校验 name + idCard；通过返回 true，同时返回 hash + tail。
	// hash = sha256(idCard + salt) 的 hex 编码；tail = idCard 后 4 位。
	Verify(ctx context.Context, name, idCard string) (verified bool, hash string, tail string, err error)
}

// MockVerifier 是默认实现；接受任意非空 name + 长度 ≥ 4 的 idCard。
type MockVerifier struct {
	salt string
}

// NewMockVerifier 构造一个 MockVerifier；salt 用于 hash 拼接。
func NewMockVerifier(salt string) *MockVerifier {
	return &MockVerifier{salt: salt}
}

// Verify 模拟实名通过；真实渠道在 v2 接入。
func (v *MockVerifier) Verify(ctx context.Context, name, idCard string) (bool, string, string, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(idCard) == "" {
		return false, "", "", ErrInvalidInput
	}
	if len(idCard) < 4 {
		return false, "", "", ErrInvalidInput
	}
	hash := sha256Hash(idCard + v.salt)
	tail := idCard[len(idCard)-4:]
	return true, hash, tail, nil
}

// sha256Hash 计算 sha256 并 hex 编码。
func sha256Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}