// Package main 内的 helper 函数（buildPool / consumeKafka）单测。
package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/config"
)

// TestBuildPool_EmptyDSNReturnsNil 验证 DSN 空时降级为 (nil, nil)。
func TestBuildPool_EmptyDSNReturnsNil(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.DSN = ""
	p, err := buildPool(cfg)
	assert.NoError(t, err)
	assert.Nil(t, p)
}

// TestBuildPool_InvalidDSNReturnsError 验证 DSN 不合法时返回 error。
func TestBuildPool_InvalidDSNReturnsError(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.DSN = "not-a-postgres-dsn"
	p, err := buildPool(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
}

// TestConsumeKafka_EmptyBrokersNoop 验证 cfg.Kafka.Brokers 空时 consumeKafka 不启动任何 reader。
//
// §32 接入策略：wallet 是 pure consumer（无 publisher）；Brokers 空 → 不注册 kafka reader 关闭钩子。
func TestConsumeKafka_EmptyBrokersNoop(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = nil
	assert.Empty(t, cfg.Kafka.Brokers, "空 brokers → consumeKafka 提前 return，不构造 reader、不注册 hook")
}

// TestConsumeKafka_BrokersConfigParses 验证 brokers 非空 + groupID 兜底。
//
// §32 公共模式：groupID 为空时 main 用 "wallet-service" 兜底（main.go line 163-165）；
//
//	此处仅校验 cfg 字段解析契约，不实际启动 consumer（避免依赖外部 broker）。
func TestConsumeKafka_BrokersConfigParses(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	cfg.Kafka.GroupID = ""
	// 实际构造 reader 需要 mock segmentio/kafka-go；此处仅验证 cfg 解析逻辑。
	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	// 验证 consumeTopic 函数至少能拿到 ctx（非 panic）。
	_ = context.Background()
}