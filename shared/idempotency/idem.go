// Package idempotency 提供幂等键抽象。
//
// 设计要点：
//   - Store 接口允许 Redis / PG / 内存多种实现。
//   - Reserve 第一次返回 true；同 key 未 Done 前重复 Reserve 返回 false。
//   - NormalizeKey 把入参统一 trim + 小写，避免空格 / 大小写造成漏判。
package idempotency

import (
	"context"
	"strings"
	"time"
)

// Store 抽象幂等键的持久化存储。
type Store interface {
	// Reserve 试图占用 key；若已被占用返回 false。
	Reserve(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Done 标记 key 已处理完成；之后可再次 Reserve。
	Done(ctx context.Context, key string) error
}

// Manager 是幂等键的对外门面。
type Manager struct {
	store Store
	ttl   time.Duration
}

// NewManager 构造一个 Manager。
func NewManager(store Store, ttl time.Duration) *Manager {
	return &Manager{store: store, ttl: ttl}
}

// Reserve 占用 key，返回是否成功。
func (m *Manager) Reserve(ctx context.Context, key string) (bool, error) {
	return m.store.Reserve(ctx, NormalizeKey(key), m.ttl)
}

// Done 标记 key 完成。
func (m *Manager) Done(ctx context.Context, key string) error {
	return m.store.Done(ctx, NormalizeKey(key))
}

// NormalizeKey 把入参规范化：trim 前后空白、转小写。
// 空串原样返回（由调用方决定是否拒绝）。
func NormalizeKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.ToLower(s)
}