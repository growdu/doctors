package lock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNopLocker_ReturnsFalse 验证 NopLocker 永远拿不到锁（模拟 Redis 不可用）。
func TestNopLocker_ReturnsFalse(t *testing.T) {
	l := NopLocker{}
	ok, err := l.TryLock(context.Background(), "key", "token", time.Second)
	assert.NoError(t, err)
	assert.False(t, ok)
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