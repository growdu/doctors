// Package main 内的 helper 函数（memoryRepo / nilPublisher）单测。
package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/review/internal/service"
	"github.com/growdu/doctors/shared/contracts"
)

// TestMemoryRepo_CreateAndGet 验证 Create → GetByID / GetByOrderID。
func TestMemoryRepo_CreateAndGet(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	rv := &service.Review{OrderID: 1, ReviewerID: 10, EscortID: 20, Rating: 5, Comment: "good"}
	assert.NoError(t, r.Create(ctx, rv))
	assert.NotZero(t, rv.ID)

	got, err := r.GetByID(ctx, rv.ID)
	assert.NoError(t, err)
	assert.Equal(t, rv.Comment, got.Comment)

	got2, err := r.GetByOrderID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, rv.ID, got2.ID)
}

// TestMemoryRepo_DuplicateReview 验证同 orderID 第二次创建返回 ErrDuplicateReview。
func TestMemoryRepo_DuplicateReview(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	rv1 := &service.Review{OrderID: 1, ReviewerID: 10, EscortID: 20, Rating: 5}
	rv2 := &service.Review{OrderID: 1, ReviewerID: 11, EscortID: 21, Rating: 3}
	assert.NoError(t, r.Create(ctx, rv1))
	assert.ErrorIs(t, r.Create(ctx, rv2), service.ErrDuplicateReview)
}

// TestMemoryRepo_ListByEscort 验证按 escortID 拉列表、时间倒序、limit/offset 切片。
func TestMemoryRepo_ListByEscort(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	now := time.Now()
	for i := 3; i >= 1; i-- {
		rv := &service.Review{OrderID: int64(i), ReviewerID: 1, EscortID: 20,
			Rating: 5, CreatedAt: now.Add(time.Duration(i) * time.Second)}
		assert.NoError(t, r.Create(ctx, rv))
	}

	list, err := r.ListByEscort(ctx, 20, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, list, 3)
	assert.True(t, list[0].CreatedAt.After(list[1].CreatedAt))

	list2, err := r.ListByEscort(ctx, 20, 2, 0)
	assert.NoError(t, err)
	assert.Len(t, list2, 2)
}

// TestMemoryRepo_List_Filter 验证 List 按 escortID / MinRating 过滤。
func TestMemoryRepo_List_Filter(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	_ = r.Create(ctx, &service.Review{OrderID: 1, EscortID: 100, Rating: 5})
	_ = r.Create(ctx, &service.Review{OrderID: 2, EscortID: 100, Rating: 2})
	_ = r.Create(ctx, &service.Review{OrderID: 3, EscortID: 200, Rating: 4})

	// 仅 escortID=100
	list, err := r.List(ctx, service.ListFilter{EscortID: 100})
	assert.NoError(t, err)
	assert.Len(t, list, 2)

	// escortID=100 + MinRating=4
	list2, err := r.List(ctx, service.ListFilter{EscortID: 100, MinRating: 4})
	assert.NoError(t, err)
	assert.Len(t, list2, 1)
	assert.Equal(t, int64(1), list2[0].OrderID)
}

// TestMemoryRepo_UpdateReply 验证 admin 回复。
func TestMemoryRepo_UpdateReply(t *testing.T) {
	r := newMemoryRepo()
	ctx := context.Background()
	rv := &service.Review{OrderID: 1, ReviewerID: 10, EscortID: 20, Rating: 5}
	assert.NoError(t, r.Create(ctx, rv))

	now := time.Now()
	assert.NoError(t, r.UpdateReply(ctx, rv.ID, "thanks", 99))
	assert.NotNil(t, rv.RepliedAt)
	assert.True(t, rv.RepliedAt.After(now.Add(-time.Second)))

	// 第二次回复应失败（已回复过）
	err := r.UpdateReply(ctx, rv.ID, "again", 99)
	assert.Error(t, err)
}

// TestNilPublisher_Noop 验证 nilPublisher 不报错。
func TestNilPublisher_Noop(t *testing.T) {
	var p nilPublisher
	err := p.PublishOrderReviewed(context.Background(), contracts.OrderReviewedEvent{})
	assert.NoError(t, err)
}