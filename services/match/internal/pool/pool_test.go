package pool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCtx() context.Context { return context.Background() }

// TestNopPool_PushAndPeek 验证 Push 后 Peek 返回 TopN。
func TestNopPool_PushAndPeek(t *testing.T) {
	p := NewNopPool()
	cands := []Candidate{
		{EscortID: 1, Score: 90},
		{EscortID: 2, Score: 95},
		{EscortID: 3, Score: 80},
	}
	require.NoError(t, p.Push(newCtx(), 100, cands, time.Minute))

	got, err := p.Peek(newCtx(), 100, 3)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, int64(2), got[0], "score 95 应排第一")
	assert.Equal(t, int64(1), got[1])
	assert.Equal(t, int64(3), got[2])
}

// TestNopPool_PeekRespectsTopN 验证 topN 截断。
func TestNopPool_PeekRespectsTopN(t *testing.T) {
	p := NewNopPool()
	cands := []Candidate{
		{EscortID: 1, Score: 90},
		{EscortID: 2, Score: 95},
		{EscortID: 3, Score: 80},
	}
	require.NoError(t, p.Push(newCtx(), 100, cands, time.Minute))

	got, _ := p.Peek(newCtx(), 100, 1)
	assert.Len(t, got, 1)
	assert.Equal(t, int64(2), got[0])
}

// TestNopPool_PopRemovesMember 验证 Pop 后从池里删除。
func TestNopPool_PopRemovesMember(t *testing.T) {
	p := NewNopPool()
	cands := []Candidate{{EscortID: 1, Score: 90}, {EscortID: 2, Score: 95}}
	require.NoError(t, p.Push(newCtx(), 100, cands, time.Minute))

	ok, err := p.Pop(newCtx(), 100, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	// 再次 Pop 应失败
	ok, err = p.Pop(newCtx(), 100, 1)
	require.NoError(t, err)
	assert.False(t, ok)

	got, _ := p.Peek(newCtx(), 100, 5)
	assert.Len(t, got, 1)
	assert.Equal(t, int64(2), got[0])
}

// TestNopPool_PopMissingEscort 验证 Pop 不存在的 escort 返回 false。
func TestNopPool_PopMissingEscort(t *testing.T) {
	p := NewNopPool()
	ok, err := p.Pop(newCtx(), 999, 1)
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestNopPool_PeekMissingOrder 不存在的订单返回 nil。
func TestNopPool_PeekMissingOrder(t *testing.T) {
	p := NewNopPool()
	got, _ := p.Peek(newCtx(), 999, 5)
	assert.Nil(t, got)
}

// TestNopPool_OverwritePush 第二次 Push 覆盖第一次。
func TestNopPool_OverwritePush(t *testing.T) {
	p := NewNopPool()
	require.NoError(t, p.Push(newCtx(), 100, []Candidate{{1, 50}}, time.Minute))
	require.NoError(t, p.Push(newCtx(), 100, []Candidate{{2, 99}}, time.Minute))

	got, _ := p.Peek(newCtx(), 100, 5)
	assert.Len(t, got, 1)
	assert.Equal(t, int64(2), got[0])
}