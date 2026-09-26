// Package main 内的 helper 函数（buildPool / buildPublisher）单测。
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/admin/internal/events"
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

// TestBuildPublisher_EmptyBrokersReturnsNop 验证 brokers 空时降级为 NopPublisher + nil kafkaPub。
//
// §32 公共模式：admin 多 topic 走 events.KafkaPublisher（topic 在 Publish 时覆写）；
//
//	dev 模式保留 events.NopPublisher。
func TestBuildPublisher_EmptyBrokersReturnsNop(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = nil
	pub, kafkaPub := buildPublisher(cfg)
	assert.NotNil(t, pub)
	assert.Nil(t, kafkaPub, "空 brokers 不应返回非 nil kafkaPub")
	_, ok := pub.(*events.NopPublisher)
	assert.True(t, ok, "空 brokers 应返回 *events.NopPublisher")
}

// TestBuildPublisher_NonEmptyBrokersReturnsKafka 验证 brokers 非空时构造 events.KafkaPublisher + 同对象 kafkaPub。
//
// §32 公共模式：admin.kafka-publisher hook 直接传 events.KafkaPublisher.Close（func() error）。
func TestBuildPublisher_NonEmptyBrokersReturnsKafka(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	pub, kafkaPub := buildPublisher(cfg)
	assert.NotNil(t, pub)
	assert.NotNil(t, kafkaPub, "Kafka 模式下 kafkaPub 必须非 nil 以便 shutdown hook 释放 writer")
	assert.Same(t, pub, kafkaPub, "pub 与 kafkaPub 必须是同一对象")
	_, ok := pub.(*events.KafkaPublisher)
	assert.True(t, ok, "非空 brokers 应返回 *events.KafkaPublisher")
	assert.NotPanics(t, func() { _ = kafkaPub.Close() }, "KafkaPublisher.Close 不应 panic")
}