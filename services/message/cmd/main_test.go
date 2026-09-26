// Package main 内的 helper 函数（memoryRepo / nilPublisher）单测。
package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/message/internal/service"
	"github.com/growdu/doctors/shared/contracts"
)

// TestMemoryRepo_CreateAndGet 验证 CreateMessage → GetMessageByID 端到端。
func TestMemoryRepo_CreateAndGet(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()

	m := &service.Message{OrderID: 1, FromUserID: 10, ToUserID: 20, Body: "hello"}
	assert.NoError(t, r.CreateMessage(ctx, m))
	assert.NotZero(t, m.ID, "CreateMessage 后 ID 必须被赋值")

	got, err := r.GetMessageByID(ctx, m.ID)
	assert.NoError(t, err)
	assert.Equal(t, m.Body, got.Body)
	assert.Equal(t, m.FromUserID, got.FromUserID)
	assert.Equal(t, m.ToUserID, got.ToUserID)
}

// TestMemoryRepo_GetNotFound 验证未命中走 ErrMessageNotFound。
func TestMemoryRepo_GetNotFound(t *testing.T) {
	r := newMemoryRepo()
	_, err := r.GetMessageByID(context.Background(), 999)
	assert.ErrorIs(t, err, service.ErrMessageNotFound)
}

// TestMemoryRepo_ListByOrder_OrderingAndLimit 验证 ListMessagesByOrder 按
//
//	CreatedAt 正序排序，limit/offset 切片正确。
func TestMemoryRepo_ListByOrder_OrderingAndLimit(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	now := time.Now()
	// 倒序写入，验证 List 时按时间正序排回来。
	for i := 3; i >= 1; i-- {
		m := &service.Message{OrderID: 1, FromUserID: 1, ToUserID: 2,
			Body: "m", CreatedAt: now.Add(time.Duration(i) * time.Second)}
		assert.NoError(t, r.CreateMessage(ctx, m))
	}

	list, err := r.ListMessagesByOrder(ctx, 1, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, list, 3)
	assert.True(t, list[0].CreatedAt.Before(list[1].CreatedAt))
	assert.True(t, list[1].CreatedAt.Before(list[2].CreatedAt))

	list2, err := r.ListMessagesByOrder(ctx, 1, 2, 0)
	assert.NoError(t, err)
	assert.Len(t, list2, 2)

	list3, err := r.ListMessagesByOrder(ctx, 1, 10, 2)
	assert.NoError(t, err)
	assert.Len(t, list3, 1)
}

// TestMemoryRepo_ListByOrder_OffsetBeyond 验证 offset 超过总数时返回空切片。
func TestMemoryRepo_ListByOrder_OffsetBeyond(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	m := &service.Message{OrderID: 1, FromUserID: 1, ToUserID: 2, Body: "x"}
	assert.NoError(t, r.CreateMessage(ctx, m))

	list, err := r.ListMessagesByOrder(ctx, 1, 10, 5)
	assert.NoError(t, err)
	assert.Empty(t, list)
}

// TestNilPublisher_Noop 验证 nilPublisher 不报错，便于 dev / 单测。
func TestNilPublisher_Noop(t *testing.T) {
	var np nilPublisher
	err := np.PublishMessageSent(context.Background(), messageSentEvt())
	assert.NoError(t, err)
}

// messageSentEvt 构造一个事件，nilPublisher 应无脑返回 nil。
func messageSentEvt() contracts.MessageSentEvent {
	return contracts.MessageSentEvent{}
}