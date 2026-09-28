// Package main 内的 helper 函数（nilEscortLoader / consumer 装配）单测。
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/shared/config"
)

// TestNilEscortLoader_ReturnsNilNil 验证 nilEscortLoader 返回空 slice + nil error，
//
//	便于 match 在没接 escort 真实数据时也能跑路由（候选打分结果为空）。
func TestNilEscortLoader_ReturnsNilNil(t *testing.T) {
	var l nilEscortLoader
	got, err := l.ListAvailable(context.Background(), "beijing")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

// TestBuildPool_EmptyAddrReturnsNopPool 验证 cfg.Redis.Addr 空时降级为 *NopPool。
func TestBuildPool_EmptyAddrReturnsNopPool(t *testing.T) {
	cfg := &config.Config{}
	cfg.Redis.Addr = ""
	p := buildPool(cfg)
	require.NotNil(t, p)
	_, ok := p.(*pool.NopPool)
	assert.True(t, ok, "空 cfg.Redis.Addr 应返回 *NopPool")
}

// TestBuildPool_NonEmptyAddrReturnsRedisPool 验证 cfg.Redis.Addr 非空时返回 *RedisPool。
func TestBuildPool_NonEmptyAddrReturnsRedisPool(t *testing.T) {
	cfg := &config.Config{}
	cfg.Redis.Addr = "127.0.0.1:6379"
	p := buildPool(cfg)
	require.NotNil(t, p)
	rp, ok := p.(*pool.RedisPool)
	assert.True(t, ok, "非空 cfg.Redis.Addr 应返回 *RedisPool")
	// 测一下底层 redis client 真的能 ping（防 buildPool 把 client close 掉的回归）
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := rp.Ping(ctx)
	assert.NoError(t, err, "RedisPool.Ping 应能联通本机 redis")
	_ = rp.Close() // 测完释放
}

// TestBuildPool_InvalidAddrReturnsRedisPool_NoPingError 验证地址无效时仍返回 RedisPool，
//
//	但 Ping 会失败；与 order cmd 的 buildPool 处理方式对齐。
func TestBuildPool_InvalidAddrReturnsRedisPool_NoPingError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Redis.Addr = "127.0.0.1:1" // 显然无服务
	p := buildPool(cfg)
	rp, ok := p.(*pool.RedisPool)
	assert.True(t, ok)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	assert.Error(t, rp.Ping(ctx), "不可达地址 ping 应失败")
	_ = rp.Close()
}

// TestDevFakeEscortLoader_ReturnsThreeVerifiedEscorts 验证 dev fake 返回 3 个 escort。
func TestDevFakeEscortLoader_ReturnsThreeVerifiedEscorts(t *testing.T) {
	var l devFakeEscortLoader
	got, err := l.ListAvailable(context.Background(), "beijing")
	require.NoError(t, err)
	require.Len(t, got, 3, "dev fake 应返回固定 3 个 escort")
	for i, e := range got {
		assert.Equal(t, "available", e.Status, "dev fake 应一律 status=available")
		assert.Greater(t, e.ID, int64(0))
		assert.Equal(t, "beijing", e.City, "dev fake 默认 city 应匹配")
		// rating 4.0+、distance 短 → 都应满足 scorer minScore 阈值
		assert.GreaterOrEqual(t, e.Rating, 4.0)
		if i > 0 {
			assert.Greater(t, got[i].ID, got[i-1].ID, "ID 应递增")
		}
	}
}

// TestDevFakeEscortLoader_DifferentCityStillReturnsSame 验证 city 不影响 dev fake 输出
// （dev mode 简化：固定返回同 3 个 escort，无论查询哪个城市）。
func TestDevFakeEscortLoader_DifferentCityStillReturnsSame(t *testing.T) {
	var l devFakeEscortLoader
	beijing, _ := l.ListAvailable(context.Background(), "beijing")
	shanghai, _ := l.ListAvailable(context.Background(), "shanghai")
	assert.Equal(t, len(beijing), len(shanghai), "dev fake 简化：city 无影响")
}

// TestMatch_KafkaConsumerConfigGating 验证 cfg.Kafka.Brokers 控制 consumer 装配行为。
func TestMatch_KafkaConsumerConfigGating(t *testing.T) {
	// 这个 test 维持原语义：cfg.Kafka.Brokers 控制 consumer 是否启动
	cfg := &config.Config{}
	cfg.Kafka.Brokers = nil
	assert.False(t, shouldStartConsumer(cfg), "空 brokers 不应启 consumer")

	cfg.Kafka.Brokers = []string{"127.0.0.1:9092"}
	assert.True(t, shouldStartConsumer(cfg), "非空 brokers 应启 consumer")
}

// TestMatch_DevFakeEscortLoader_EnvToggle 验证 env DOCTORS_DEV_FAKE_ESCORT=1 切换 fake loader。
func TestMatch_DevFakeEscortLoader_EnvToggle(t *testing.T) {
	t.Setenv("DOCTORS_DEV_FAKE_ESCORT", "1")
	loader := chooseEscortLoader()
	_, ok := loader.(devFakeEscortLoader)
	assert.True(t, ok, "env=1 应选 devFakeEscortLoader")

	t.Setenv("DOCTORS_DEV_FAKE_ESCORT", "0")
	loader = chooseEscortLoader()
	_, ok = loader.(nilEscortLoader)
	assert.True(t, ok, "env=0 应选 nilEscortLoader")

	os.Unsetenv("DOCTORS_DEV_FAKE_ESCORT")
	loader = chooseEscortLoader()
	_, ok = loader.(nilEscortLoader)
	assert.True(t, ok, "env 未设应选 nilEscortLoader")
}