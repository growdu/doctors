// Package lock 提供基于 Redis 的分布式锁抽象。
//
// 设计要点：
//   - TryLock 用 SET key value NX PX ttl；失败返回 false（不抛错）。
//   - Release 用 Lua 脚本保证"只删自己的锁"（避免误删别人续期的锁）。
//   - 不可用（Redis down）时返回 false（让上层 DB 唯一约束兜底），不 panic。
package lock

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Locker 抽象分布式锁。
type Locker interface {
	// TryLock 尝试拿锁；成功返回 true，失败 false。
	// 错误仅用于 Redis 客户端层异常；上层应把 false 当作"未拿到锁"。
	TryLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error)
	// Release 用 Lua 脚本释放锁（CAS：只删 value 等于 token 的）；best-effort。
	Release(ctx context.Context, key, token string) error
}

// releaseScript 保证只删自己 token 的锁（Lua CAS）。
const releaseScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`

// RedisLocker 是基于 Redis SETNX 的实现。
type RedisLocker struct {
	rdb *redis.Client
}

// NewRedisLocker 构造 RedisLocker。
func NewRedisLocker(rdb *redis.Client) *RedisLocker { return &RedisLocker{rdb: rdb} }

// TryLock 用 SET NX PX 实现。
func (l *RedisLocker) TryLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	ok, err := l.rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err // 上层按 false 处理；DB 兜底
	}
	return ok, nil
}

// Release 用 Lua 脚本释放。
func (l *RedisLocker) Release(ctx context.Context, key, token string) error {
	return redis.NewScript(releaseScript).Run(ctx, l.rdb, []string{key}, token).Err()
}

// NopLocker 是 dev / unit test 的占位实现：永远拿不到锁（模拟 Redis 不可用）。
type NopLocker struct{}

func (NopLocker) TryLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	return false, nil
}
func (NopLocker) Release(ctx context.Context, key, token string) error { return nil }