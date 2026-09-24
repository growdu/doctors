package lock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNopLocker_PassesThrough 验证 NopLocker.TryLock 永远 ok=true（passthrough）。
// 设计意图：让上层 Service.TryLock 跳过 SETNX 检查，走 DB 唯一约束兜底。
func TestNopLocker_PassesThrough(t *testing.T) {
	l := NopLocker{}
	ok, err := l.TryLock(context.Background(), "key", "token", time.Second)
	assert.NoError(t, err)
	assert.True(t, ok, "NopLocker.TryLock 应返回 true 让上层继续")
}

// TestNopLocker_ReleaseNoError 验证 NopLocker.Release 不报错。
func TestNopLocker_ReleaseNoError(t *testing.T) {
	assert.NoError(t, NopLocker{}.Release(context.Background(), "key", "token"))
}

// TestLockerInterface_NopSatisfies 编译期检查 NopLocker 满足 Locker 接口。
func TestLockerInterface_NopSatisfies(t *testing.T) {
	var _ Locker = NopLocker{}
	var _ Locker = (*RedisLocker)(nil)
}