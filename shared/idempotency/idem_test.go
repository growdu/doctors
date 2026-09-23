package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/idempotency"
)

// memStore 是测试用的内存版 Store。
type memStore struct {
	reserved map[string]struct{}
	done     map[string]time.Time
}

func newMemStore() *memStore {
	return &memStore{reserved: map[string]struct{}{}, done: map[string]time.Time{}}
}

func (m *memStore) Reserve(_ context.Context, key string, ttl time.Duration) (bool, error) {
	if _, ok := m.reserved[key]; ok {
		return false, nil
	}
	m.reserved[key] = struct{}{}
	return true, nil
}

func (m *memStore) Done(_ context.Context, key string) error {
	m.done[key] = time.Now()
	delete(m.reserved, key) // Done 后允许再次 Reserve
	return nil
}

func TestManager_FirstCallReserves(t *testing.T) {
	ctx := context.Background()
	mgr := idempotency.NewManager(newMemStore(), time.Minute)

	ok, err := mgr.Reserve(ctx, "k1")
	require.NoError(t, err)
	assert.True(t, ok, "首次调用应当成功占位")
}

func TestManager_DuplicateReserveFails(t *testing.T) {
	ctx := context.Background()
	mgr := idempotency.NewManager(newMemStore(), time.Minute)

	_, _ = mgr.Reserve(ctx, "k1")
	ok, err := mgr.Reserve(ctx, "k1")
	require.NoError(t, err)
	assert.False(t, ok, "同 key 未 Done 前第二次 Reserve 应失败")
}

func TestManager_AfterDoneCanReserveAgain(t *testing.T) {
	ctx := context.Background()
	mgr := idempotency.NewManager(newMemStore(), time.Minute)

	_, _ = mgr.Reserve(ctx, "k1")
	require.NoError(t, mgr.Done(ctx, "k1"))

	ok, err := mgr.Reserve(ctx, "k1")
	require.NoError(t, err)
	assert.True(t, ok, "Done 之后可以再次 Reserve")
}

func TestNormalizeKey_TrimsAndLowercases(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ABC", "abc"},
		{"  spaced  ", "spaced"},
		{"  Mixed-Case-Key ", "mixed-case-key"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, idempotency.NormalizeKey(tc.in))
		})
	}
}

func TestNormalizeKey_EmptyReturnsUnchanged(t *testing.T) {
	assert.Equal(t, "", idempotency.NormalizeKey(""))
}

func TestManager_NormalizesKeyBeforeReserve(t *testing.T) {
	ctx := context.Background()
	mgr := idempotency.NewManager(newMemStore(), time.Minute)

	_, _ = mgr.Reserve(ctx, idempotency.NormalizeKey("ABC"))
	ok, err := mgr.Reserve(ctx, idempotency.NormalizeKey("abc"))
	require.NoError(t, err)
	assert.False(t, ok, "规范化后同 key 应识别为重复")
}