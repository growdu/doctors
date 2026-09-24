package refund

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSnap 是 OrderSnapshotProvider 的 fake 实现。
type fakeSnap struct {
	s        *OrderSnapshot
	override *OrderSnapshot
	err      error
}

func (f *fakeSnap) GetSnapshot(_ context.Context, id int64) (*OrderSnapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	s := f.s
	if s == nil {
		return nil, errors.New("not found")
	}
	cp := *s
	cp.OrderID = id
	return &cp, nil
}

// fakeRefundRepo 满足 RefundRepo。
type fakeRefundRepo struct {
	nextID int64
	rows   map[int64]*Record
}

func (r *fakeRefundRepo) Create(_ context.Context, rec *Record) error {
	r.nextID++
	rec.ID = r.nextID
	if r.rows == nil {
		r.rows = map[int64]*Record{}
	}
	r.rows[rec.ID] = rec
	return nil
}

func (r *fakeRefundRepo) GetByOrderID(_ context.Context, oid int64) ([]*Record, error) {
	out := make([]*Record, 0)
	for _, rec := range r.rows {
		if rec.OrderID == oid {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (r *fakeRefundRepo) UpdateStatus(_ context.Context, id int64, status, txID, fail string) error {
	if r.rows == nil {
		return errors.New("not found")
	}
	rec, ok := r.rows[id]
	if !ok {
		return errors.New("not found")
	}
	rec.Status = status
	rec.ExternalTxID = txID
	rec.FailureReason = fail
	return nil
}

// fakePub 记录被调用的次数。
type fakePub struct {
	called int
	last   *contracts.RefundCompletedEvent
}

func (f *fakePub) PublishRefundCompleted(_ context.Context, ev contracts.RefundCompletedEvent) error {
	f.called++
	cp := ev
	f.last = &cp
	return nil
}

// TestRefund_AfterPaid5Min_OK 验证付款 1 分钟退款 = 100%。
func TestRefund_AfterPaid5Min_OK(t *testing.T) {
	now := time.Now()
	snap := &OrderSnapshot{
		Amount:         200,
		PaidAt:         ptrTime(now.Add(-1 * time.Minute)),
		ServiceStartAt: now.Add(time.Hour),
	}
	svc := New(&fakeSnap{s: snap}, &fakeRefundRepo{}, newRepo(), &fakePub{}, "default")
	res, err := svc.Refund(context.Background(), 100, "user_cancel")
	require.NoError(t, err)
	assert.InDelta(t, 200.0, res.Amount, 0.01)
	assert.Equal(t, "completed", res.Status)
}

// TestRefund_InService_Rejected 验证服务已开始 → rejected。
func TestRefund_InService_Rejected(t *testing.T) {
	now := time.Now()
	snap := &OrderSnapshot{
		Amount:         200,
		PaidAt:         ptrTime(now.Add(-2 * time.Hour)),
		AcceptedAt:     ptrTime(now.Add(-1 * time.Hour)),
		ServiceStartAt: now.Add(-5 * time.Minute),
	}
	svc := New(&fakeSnap{s: snap}, &fakeRefundRepo{}, newRepo(), &fakePub{}, "default")
	res, err := svc.Refund(context.Background(), 100, "user_cancel")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, res.Amount, 0.01)
	assert.Equal(t, "rejected", res.Status)
}

// TestRefund_OrderNotFound 验证 order 不存在 → CodeNotFound 错。
func TestRefund_OrderNotFound(t *testing.T) {
	svc := New(&fakeSnap{s: nil}, &fakeRefundRepo{}, newRepo(), &fakePub{}, "default")
	_, err := svc.Refund(context.Background(), 999, "user_cancel")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRefund_AfterAccepted_95Pct 验证接单后~服务开始前 → 95%。
func TestRefund_AfterAccepted_95Pct(t *testing.T) {
	now := time.Now()
	snap := &OrderSnapshot{
		Amount:         1000,
		PaidAt:         ptrTime(now.Add(-1 * time.Hour)),
		AcceptedAt:     ptrTime(now.Add(-30 * time.Minute)),
		ServiceStartAt: now.Add(time.Hour),
	}
	pub := &fakePub{}
	svc := New(&fakeSnap{s: snap}, &fakeRefundRepo{}, newRepo(), pub, "default")
	res, err := svc.Refund(context.Background(), 100, "user_cancel")
	require.NoError(t, err)
	assert.InDelta(t, 950.0, res.Amount, 0.01)
	assert.Equal(t, "completed", res.Status)
	require.NotNil(t, pub.last)
	assert.Equal(t, int64(100), pub.last.OrderID)
	assert.True(t, len(pub.last.ExternalTxID) > 0, "ExternalTxID 应非空")
	assert.Contains(t, pub.last.ExternalTxID, "mock-tx-", "前缀应为 mock-tx-")
}