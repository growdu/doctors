// Package main 内的 helper 函数（buildPool / nil*Repo）单测。
package main

import (
	"context"
	"errors"
	"testing"
	"time"

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

// TestNilProfileRepo_ReturnsSentinel 验证 nilProfileRepo 所有方法返回 errNil。
func TestNilProfileRepo_ReturnsSentinel(t *testing.T) {
	var r nilProfileRepo
	ctx := context.Background()
	assert.True(t, errors.Is(r.Create(ctx, nil), errNil))
	_, err := r.GetByID(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	_, err = r.GetByUserID(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	assert.True(t, errors.Is(r.UpdateStatus(ctx, 1, "x"), errNil))
	assert.True(t, errors.Is(r.UpdateLocation(ctx, 1, 0, 0), errNil))
	assert.True(t, errors.Is(r.UpdateCity(ctx, 1, "x"), errNil))
	assert.True(t, errors.Is(r.UpdateAvailability(ctx, 1, time.Now(), time.Now()), errNil))
}

// TestNilQualRepo_ReturnsSentinel 验证 nilQualRepo 所有方法返回 errNil。
func TestNilQualRepo_ReturnsSentinel(t *testing.T) {
	var r nilQualRepo
	ctx := context.Background()
	assert.True(t, errors.Is(r.Create(ctx, nil), errNil))
	_, err := r.GetByID(ctx, 1, 1)
	assert.True(t, errors.Is(err, errNil))
	_, err = r.ListByEscort(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	assert.True(t, errors.Is(r.Update(ctx, nil), errNil))
	assert.True(t, errors.Is(r.Delete(ctx, 1, 1), errNil))
}

// TestNilTrainingRepo_ReturnsSentinel 验证 nilTrainingRepo 所有方法返回 errNil。
func TestNilTrainingRepo_ReturnsSentinel(t *testing.T) {
	var r nilTrainingRepo
	ctx := context.Background()
	assert.True(t, errors.Is(r.Create(ctx, nil), errNil))
	_, err := r.ListByEscort(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
}

// TestNilAvailabilityRepo_ReturnsSentinel 验证 nilAvailabilityRepo 所有方法返回 errNil。
func TestNilAvailabilityRepo_ReturnsSentinel(t *testing.T) {
	var r nilAvailabilityRepo
	ctx := context.Background()
	_, err := r.Create(ctx, 1, time.Now(), time.Now())
	assert.True(t, errors.Is(err, errNil))
	_, err = r.FindByID(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	assert.True(t, errors.Is(r.Delete(ctx, 1, 1), errNil))
	_, err = r.ListByEscort(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	_, err = r.ListAvailableByTime(ctx, time.Now(), time.Now(), 10)
	assert.True(t, errors.Is(err, errNil))
	assert.True(t, errors.Is(r.BookByOrder(ctx, 1, 1), errNil))
	assert.True(t, errors.Is(r.ReleaseByOrder(ctx, 1), errNil))
}

// TestNilRepos_SentinelMessage 验证 sentinel 文案提示 DSN。
func TestNilRepos_SentinelMessage(t *testing.T) {
	assert.True(t, contains(errNil.Error(), "dsn"))
}

// contains 是 strings.Contains 的本地别名（避免 import）。
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBuildPublisher_EmptyBrokersReturnsNil 验证 brokers 空时降级为 nilPublisher + nil closer。
func TestBuildPublisher_EmptyBrokersReturnsNil(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = nil
	pub, closer := buildPublisher(cfg)
	assert.NotNil(t, pub)
	assert.Nil(t, closer, "空 brokers 不应返回非 nil closer")
	_, ok := pub.(nilPublisher)
	assert.True(t, ok, "空 brokers 应返回 nilPublisher")
}

// TestBuildPublisher_NonEmptyBrokersReturnsKafka 验证 brokers 非空时构造 kafkapublisher + 同对象 closer。
func TestBuildPublisher_NonEmptyBrokersReturnsKafka(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = []string{"localhost:9092"}
	pub, closer := buildPublisher(cfg)
	assert.NotNil(t, pub)
	assert.NotNil(t, closer, "Kafka 模式下 closer 必须非 nil")
	assert.NotPanics(t, func() { _ = closer() }, "closer 不应 panic")
}