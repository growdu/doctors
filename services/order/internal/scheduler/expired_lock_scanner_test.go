package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/state"
	"github.com/growdu/doctors/shared/contracts"
)

// fakeRepo 提供一个最小的 order_repo（只实现 LockExpired）。
type fakeRepo struct {
	mu     sync.Mutex
	orders map[int64]*repo.Order
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{orders: map[int64]*repo.Order{}}
}

func int64Ptr(v int64) *int64 { return &v }

func (f *fakeRepo) LockExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*repo.Order, 0)
	for _, o := range f.orders {
		if o.Status != string(state.StatusPendingAcceptance) || o.LockExpireAt == nil {
			continue
		}
		if o.LockExpireAt.Before(now) {
			out = append(out, o)
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// fakePublisher 记录最近一次发布的 OrderMatchingEvent。
type fakePublisher struct {
	mu      sync.Mutex
	last    *contracts.OrderMatchingEvent
	allEvs  []contracts.OrderMatchingEvent
}

func (f *fakePublisher) PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error {
	return nil
}
func (f *fakePublisher) PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error {
	return nil
}
func (f *fakePublisher) PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error {
	return nil
}
func (f *fakePublisher) PublishOrderMatching(ctx context.Context, ev contracts.OrderMatchingEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := ev
	f.last = &cp
	f.allEvs = append(f.allEvs, cp)
	return nil
}
func (f *fakePublisher) Close() error { return nil }

// fakeReleaseSvc 记录被调用的 orderID + actorID。
type fakeReleaseSvc struct {
	mu     sync.Mutex
	called []int64
	actor  []int64
}

func (f *fakeReleaseSvc) ReleaseAcceptLock(ctx context.Context, orderID, actorID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = append(f.called, orderID)
	f.actor = append(f.actor, actorID)
	// 模拟 service：把订单 status 回退 matching
	return nil
}

// TestScanner_ScanOnce_PublishesOnExpire 验证扫描到过期锁单后发布事件并调 ReleaseAcceptLock。
func TestScanner_ScanOnce_PublishesOnExpire(t *testing.T) {
	expire := time.Now().Add(-1 * time.Hour)
	fr := newFakeRepo()
	fr.orders[100] = &repo.Order{
		ID:           100,
		Status:       string(state.StatusPendingAcceptance),
		LockExpireAt: &expire,
		LockOwner:    int64Ptr(7),
	}
	pub := &fakePublisher{}
	svc := &fakeReleaseSvc{}
	s := NewExpiredLockScanner(fr, svc, pub, time.Second)
	s.ScanOnce(context.Background())

	require.NotNil(t, pub.last, "应发布 OrderMatchingEvent")
	assert.Equal(t, int64(100), pub.last.OrderID)
	assert.Equal(t, int64(7), pub.last.EscortID)
	assert.Equal(t, "lock_expired", pub.last.Reason)

	require.Len(t, svc.called, 1, "ReleaseAcceptLock 应被调用一次")
	assert.Equal(t, int64(100), svc.called[0])
	assert.Equal(t, int64(7), svc.actor[0], "actorID 应等于 lock_owner")
}

// TestScanner_ScanOnce_SkipsNonExpired 验证未过期的锁单不被处理。
func TestScanner_ScanOnce_SkipsNonExpired(t *testing.T) {
	future := time.Now().Add(time.Hour)
	fr := newFakeRepo()
	fr.orders[200] = &repo.Order{
		ID:           200,
		Status:       string(state.StatusPendingAcceptance),
		LockExpireAt: &future,
		LockOwner:    int64Ptr(7),
	}
	pub := &fakePublisher{}
	svc := &fakeReleaseSvc{}
	s := NewExpiredLockScanner(fr, svc, pub, time.Second)
	s.ScanOnce(context.Background())

	assert.Nil(t, pub.last, "未过期不应发布")
	assert.Empty(t, svc.called)
}

// TestScanner_ScanOnce_SkipsNilLockOwner 验证 lock_owner=nil 时跳过（异常保护）。
func TestScanner_ScanOnce_SkipsNilLockOwner(t *testing.T) {
	expire := time.Now().Add(-1 * time.Hour)
	fr := newFakeRepo()
	fr.orders[300] = &repo.Order{
		ID:           300,
		Status:       string(state.StatusPendingAcceptance),
		LockExpireAt: &expire,
		LockOwner:    nil, // 异常：DB 状态不一致
	}
	pub := &fakePublisher{}
	svc := &fakeReleaseSvc{}
	s := NewExpiredLockScanner(fr, svc, pub, time.Second)
	s.ScanOnce(context.Background())

	assert.Nil(t, pub.last)
	assert.Empty(t, svc.called)
}

// TestScanner_Run_RespectsCtxCancel 验证 Run 在 ctx 取消后退出。
func TestScanner_Run_RespectsCtxCancel(t *testing.T) {
	fr := newFakeRepo()
	pub := &fakePublisher{}
	svc := &fakeReleaseSvc{}
	s := NewExpiredLockScanner(fr, svc, pub, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return within 2s after ctx cancel")
	}
}