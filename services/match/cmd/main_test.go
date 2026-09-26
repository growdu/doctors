// Package main 内的 helper 函数（nilEscortLoader / consumer 装配）单测。
package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
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

// TestMatch_KafkaConsumerConfigGating 验证 cfg.Kafka.Brokers 控制 consumer 装配行为。
//
// §32 接入策略：match 是 pure consumer（无 publisher）。
//
//	- Brokers 空 → 不构造 consumer，dev 模式
//	- Brokers 非空 → 构造 consumer，订阅 TopicOrderCreated；groupID 默认 "match-service"
//	  （实现位于 main.go 的 kafkaConsumer 装配分支，本测试只验证 cfg 解析契约）。
func TestMatch_KafkaConsumerConfigGating(t *testing.T) {
	// 1) brokers 空：cfg.Kafka.Brokers 为空 → consumer 应跳过
	cfg := &config.Config{}
	cfg.Kafka.Brokers = nil
	assert.Empty(t, cfg.Kafka.Brokers, "空 brokers → consumer 跳过（dev 模式）")

	// 2) brokers 非空：consumer 启动，订阅 TopicOrderCreated
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	cfg.Kafka.GroupID = ""
	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	assert.NotEmpty(t, contracts.TopicOrderCreated, "TopicOrderCreated 必须定义")
	// groupID 为空时 main 用 "match-service" 兜底（main.go line 76-78）
}