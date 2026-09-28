// Package main 内的 helper 函数（buildPool / buildPublisher / nilOrderRepo）单测。
package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// TestUnwiredUsers_ReturnsSentinel 验证 dev 占位 UserLookup 返回 sentinel error
//
//	而非 (nil, nil)，让 service.Create 走 "patient not found" 分支而不是 nil pointer panic。
//
// 背景：原实现返回 (nil, nil)，service.order_service.go:181 直接读 u.RealNameVerified
//
//	触发 nil pointer dereference，导致 POST /api/v1/orders panic 500。
func TestUnwiredUsers_ReturnsSentinel(t *testing.T) {
	var u unwiredUsers
	got, err := u.FindByID(context.Background(), 42)
	assert.Nil(t, got, "unwiredUsers 不应返回 *UserSnapshot")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errUnwiredUsers), "应返回 sentinel errUnwiredUsers 便于上层判断")
	assert.True(t, strings.Contains(err.Error(), "user lookup"),
		"sentinel error 应提示 user lookup 配置缺失")
}
// TestDevFakeUsers_ReturnsVerifiedPatient 验证 dev fake 在 DOCTORS_DEV_FAKE_USER=1 时返回 verified patient。
func TestDevFakeUsers_ReturnsVerifiedPatient(t *testing.T) {
	var u devFakeUsers
	got, err := u.FindByID(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(42), got.ID)
	assert.Equal(t, "patient", got.Role)
	assert.True(t, got.RealNameVerified, "devFakeUsers 应一律视为已实名")
}

// TestDevFakeUsers_RejectsNonPositiveID 验证 id <= 0 时返回 sentinel error（与 unwiredUsers 一致）。
func TestDevFakeUsers_RejectsNonPositiveID(t *testing.T) {
	var u devFakeUsers
	for _, id := range []int64{0, -1, -100} {
		got, err := u.FindByID(context.Background(), id)
		assert.Nil(t, got)
		assert.True(t, errors.Is(err, errUnwiredUsers), "id=%d 应返回 sentinel errUnwiredUsers", id)
	}
}
