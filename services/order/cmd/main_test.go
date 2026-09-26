// Package main 内的 helper 函数（buildPool / buildPublisher / nilOrderRepo）单测。
package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/order/internal/events"
	"github.com/growdu/doctors/services/order/internal/repo"
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

// TestNilOrderRepo_ReturnsSentinel 验证 nilOrderRepo 所有方法都返回相同的 sentinel error。
func TestNilOrderRepo_ReturnsSentinel(t *testing.T) {
	var r nilOrderRepo
	ctx := context.Background()
	o := &repo.Order{OrderNo: "x", PatientID: 1, HospitalID: 1, PackageID: 1, ServiceStartAt: time.Now(), Amount: 1, FinalAmount: 1, Status: "pending_payment"}
	from := "x"
	actor := int64(1)

	assert.True(t, errors.Is(r.Create(ctx, o), errNilOrderRepo))
	_, err := r.FindByID(ctx, 1)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	_, err = r.ListByPatient(ctx, 1, 10, 0)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	_, err = r.ListByEscort(ctx, 1, "", 10, 0)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	assert.True(t, errors.Is(r.UpdateStatus(ctx, 1, "x", 1, nil), errNilOrderRepo))
	assert.True(t, errors.Is(r.InsertEvent(ctx, 1, &from, "x", &actor, nil), errNilOrderRepo))
	_, err = r.ListEvents(ctx, 1)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	assert.True(t, errors.Is(r.SelectForEscort(ctx, 1, 1, time.Now(), 1), errNilOrderRepo))
	assert.True(t, errors.Is(r.ConfirmByEscort(ctx, 1, 1, time.Now(), 1), errNilOrderRepo))
	assert.True(t, errors.Is(r.RejectByEscort(ctx, 1, 1, 1), errNilOrderRepo))
	_, err = r.PendingExpired(ctx, time.Now(), 10)
	assert.True(t, errors.Is(err, errNilOrderRepo))
}

// TestNilOrderRepo_SentinelMessage 检查 sentinel 文案，提醒运维配置 DSN。
func TestNilOrderRepo_SentinelMessage(t *testing.T) {
	assert.True(t, strings.Contains(errNilOrderRepo.Error(), "dsn"),
		"sentinel error 应提示 DSN 配置缺失")
}

// TestBuildPublisher_EmptyBrokersReturnsNop 验证 cfg.Kafka.Brokers 空时降级为 NopPublisher。
//
// §32 接入策略：dev 模式（无 brokers）保留 NopPublisher，避免硬依赖 Kafka。
func TestBuildPublisher_EmptyBrokersReturnsNop(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = nil
	pub, closer := buildPublisher(cfg)
	assert.NotNil(t, pub)
	assert.Nil(t, closer, "空 brokers 不应返回非 nil closer（无资源需要释放）")
	_, ok := pub.(*events.NopPublisher)
	assert.True(t, ok, "空 brokers 应返回 *events.NopPublisher")
}

// TestBuildPublisher_NonEmptyBrokersReturnsKafka 验证 brokers 非空时构造 KafkaPublisher。
//
//	返回的 publisher 同时作为 closer（实现 events.Publisher 接口，含 Close() error）。
func TestBuildPublisher_NonEmptyBrokersReturnsKafka(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	pub, closer := buildPublisher(cfg)
	assert.NotNil(t, pub)
	assert.NotNil(t, closer, "Kafka 模式下 closer 必须非 nil 以便 shutdown hook 释放 writer")
	// publisher 和 closer 是同一个对象（双重返回便于 RegisterShutdownHook）
	assert.Same(t, pub, closer)
	_, ok := pub.(*events.KafkaPublisher)
	assert.True(t, ok, "非空 brokers 应返回 *events.KafkaPublisher")
	// KafkaPublisher 的 Close 应不 panic（writer 底层会触发连接，但单元测试不实际连 broker）
	assert.NotPanics(t, func() { _ = pub.Close() }, "KafkaPublisher.Close 不应 panic")
}