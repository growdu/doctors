// Package main 内的 helper 函数（memoryRepo / nilPublisher / activeLookup）单测。
package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/sos/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
)

// TestMemoryRepo_CreateAndGet 验证 Create → GetByID 端到端。
func TestMemoryRepo_CreateAndGet(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	s := &service.SOS{OrderID: 1, UserID: 10, Lat: 30, Lng: 120, Note: "help"}
	assert.NoError(t, r.Create(ctx, s))
	assert.NotZero(t, s.ID)
	assert.False(t, s.RaisedAt.IsZero(), "Create 后 RaisedAt 必须被赋值")

	got, err := r.GetByID(ctx, s.ID)
	assert.NoError(t, err)
	assert.Equal(t, s.Note, got.Note)
}

// TestMemoryRepo_GetNotFound 验证未命中走 ErrSOSNotFound。
func TestMemoryRepo_GetNotFound(t *testing.T) {
	r := newMemoryRepo()
	_, err := r.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, service.ErrSOSNotFound)
}

// TestMemoryRepo_UpdateStatus 验证状态变更。
func TestMemoryRepo_UpdateStatus(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	s := &service.SOS{OrderID: 1, UserID: 10}
	assert.NoError(t, r.Create(ctx, s))

	assert.NoError(t, r.UpdateStatus(ctx, s.ID, "handling"))
	got, _ := r.GetByID(ctx, s.ID)
	assert.Equal(t, "handling", got.Status)

	// 更新不存在的 ID
	assert.ErrorIs(t, r.UpdateStatus(ctx, 999, "resolved"), service.ErrSOSNotFound)
}

// TestMemoryRepo_IsRecentDuplicate 验证 5min 内同 orderID 去重。
func TestMemoryRepo_IsRecentDuplicate(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()

	// 10 分钟前的 SOS，不算重复
	old := &service.SOS{OrderID: 1, UserID: 10, RaisedAt: time.Now().Add(-10 * time.Minute)}
	assert.NoError(t, r.Create(ctx, old))

	since := time.Now().Add(-5 * time.Minute)
	dup, err := r.IsRecentDuplicate(ctx, 1, since)
	assert.NoError(t, err)
	assert.False(t, dup, "10min 前的 SOS 不算 5min 内的重复")

	// 1 分钟前的 SOS → 重复
	fresh := &service.SOS{OrderID: 2, UserID: 10, RaisedAt: time.Now().Add(-1 * time.Minute)}
	assert.NoError(t, r.Create(ctx, fresh))

	dup2, err := r.IsRecentDuplicate(ctx, 2, since)
	assert.NoError(t, err)
	assert.True(t, dup2)
}

// TestMemoryRepo_List_Filter 验证 status / orderID 过滤。
func TestMemoryRepo_List_Filter(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		s := &service.SOS{OrderID: int64(i%2) + 1, UserID: 10, Status: "raised"} // OrderID=1/2/3/4/5
		assert.NoError(t, r.Create(ctx, s))
	}
	// 把其中两个置为 handling
	all, _ := r.List(ctx, service.ListFilter{PageSize: 100})
	for i := 0; i < 2 && i < len(all); i++ {
		_ = r.UpdateStatus(ctx, all[i].ID, "handling")
	}

	raised, err := r.List(ctx, service.ListFilter{Status: "raised", PageSize: 100})
	assert.NoError(t, err)
	assert.Len(t, raised, 3)

	handling, err := r.List(ctx, service.ListFilter{Status: "handling", PageSize: 100})
	assert.NoError(t, err)
	assert.Len(t, handling, 2)

	// 仅 orderID=1
	order1, err := r.List(ctx, service.ListFilter{OrderID: 1, PageSize: 100})
	assert.NoError(t, err)
	for _, s := range order1 {
		assert.Equal(t, int64(1), s.OrderID)
	}
}

// TestActiveLookup_AlwaysTrue 验证 activeLookup 占位实现返回 true。
func TestActiveLookup_AlwaysTrue(t *testing.T) {
	var l activeLookup
	ok, err := l.IsOrderActive(context.Background(), 1)
	assert.NoError(t, err)
	assert.True(t, ok)
}

// TestNilPublisher_Noop 验证 nilPublisher 不报错。
func TestNilPublisher_Noop(t *testing.T) {
	var p nilPublisher
	err := p.PublishSOSRaised(context.Background(), contracts.SOSRaisedEvent{})
	assert.NoError(t, err)
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