package service

import (
	"context"
	"testing"
	"time"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- fake ----------

type fakeRepo struct {
	payments map[int64]*Payment
	byOrder  map[int64]int64
	nextID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{payments: map[int64]*Payment{}, byOrder: map[int64]int64{}}
}

func (r *fakeRepo) Create(ctx context.Context, p *Payment) error {
	r.nextID++
	p.ID = r.nextID
	r.payments[p.ID] = p
	r.byOrder[p.OrderID] = p.ID
	return nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Payment, error) {
	if p, ok := r.payments[id]; ok {
		return p, nil
	}
	return nil, ErrPaymentNotFound
}

func (r *fakeRepo) GetByOrderID(ctx context.Context, orderID int64) (*Payment, error) {
	id, ok := r.byOrder[orderID]
	if !ok {
		return nil, ErrPaymentNotFound
	}
	return r.payments[id], nil
}

func (r *fakeRepo) UpdateStatus(ctx context.Context, id int64, status string, completedAt, refundedAt *time.Time, externalTxID string) error {
	if p, ok := r.payments[id]; ok {
		p.Status = status
		p.CompletedAt = completedAt
		p.RefundedAt = refundedAt
		if externalTxID != "" {
			p.ExternalTxID = externalTxID
		}
	}
	return nil
}

type fakePub struct {
	completed contracts.PaymentCompletedEvent
	refunded  contracts.PaymentRefundedEvent
	cCount    int
	rCount    int
}

func (p *fakePub) PublishPaymentCompleted(ctx context.Context, ev contracts.PaymentCompletedEvent) error {
	p.completed = ev
	p.cCount++
	return nil
}

func (p *fakePub) PublishPaymentRefunded(ctx context.Context, ev contracts.PaymentRefundedEvent) error {
	p.refunded = ev
	p.rCount++
	return nil
}

type fakeChannel struct{}

func (fakeChannel) CreateOutTradeNo(ctx context.Context, p *Payment) (string, error) {
	return "mock-tx-" + time.Now().Format("20060102"), nil
}

func newService() (*Service, *fakeRepo, *fakePub) {
	r := newFakeRepo()
	p := &fakePub{}
	return New(r, p, fakeChannel{}), r, p
}

// ---------- 测试 ----------

func TestCreate_OK(t *testing.T) {
	s, _, _ := newService()
	p, err := s.Create(context.Background(), 100, 200.0)
	require.NoError(t, err)
	assert.Equal(t, "created", p.Status)
	assert.NotEmpty(t, p.ExternalTxID)
}

func TestCreate_MissingOrderID(t *testing.T) {
	s, _, _ := newService()
	_, err := s.Create(context.Background(), 0, 100)
	assert.Error(t, err)
}

func TestCreate_BadAmount(t *testing.T) {
	s, _, _ := newService()
	_, err := s.Create(context.Background(), 100, 0)
	assert.Error(t, err)
	_, err = s.Create(context.Background(), 100, -1)
	assert.Error(t, err)
}

func TestCreate_Duplicate(t *testing.T) {
	s, _, _ := newService()
	_, err := s.Create(context.Background(), 100, 200)
	require.NoError(t, err)
	_, err = s.Create(context.Background(), 100, 200)
	assert.Error(t, err)
}

func TestComplete_OK(t *testing.T) {
	s, _, pub := newService()
	p, err := s.Create(context.Background(), 100, 200)
	require.NoError(t, err)

	_, err = s.Complete(context.Background(), p.ID, "mock-tx-final")
	require.NoError(t, err)
	assert.Equal(t, 1, pub.cCount)
	assert.Equal(t, int64(100), pub.completed.OrderID)
}

func TestComplete_Idempotent(t *testing.T) {
	s, _, _ := newService()
	p, _ := s.Create(context.Background(), 100, 200)
	_, err := s.Complete(context.Background(), p.ID, "tx")
	require.NoError(t, err)
	_, err = s.Complete(context.Background(), p.ID, "tx2")
	require.NoError(t, err) // idempotent
}

func TestComplete_NotFound(t *testing.T) {
	s, _, _ := newService()
	_, err := s.Complete(context.Background(), 999, "tx")
	assert.Error(t, err)
}

func TestRefund_OK(t *testing.T) {
	s, _, pub := newService()
	p, _ := s.Create(context.Background(), 100, 200)
	_, _ = s.Complete(context.Background(), p.ID, "tx")
	require.NoError(t, s.Refund(context.Background(), p.ID))
	assert.Equal(t, 1, pub.rCount)
}

func TestRefund_NotCompleted(t *testing.T) {
	s, _, _ := newService()
	p, _ := s.Create(context.Background(), 100, 200)
	err := s.Refund(context.Background(), p.ID)
	assert.Error(t, err)
}

func TestRefund_NotFound(t *testing.T) {
	s, _, _ := newService()
	err := s.Refund(context.Background(), 999)
	assert.Error(t, err)
}

func TestGetByOrder_OK(t *testing.T) {
	s, _, _ := newService()
	_, _ = s.Create(context.Background(), 100, 200)
	got, err := s.GetByOrder(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, int64(100), got.OrderID)
}

func TestGetByOrder_NotFound(t *testing.T) {
	s, _, _ := newService()
	_, err := s.GetByOrder(context.Background(), 999)
	assert.Error(t, err)
}

func TestNilPublishers(t *testing.T) {
	s := New(newFakeRepo(), nil, fakeChannel{})
	p, err := s.Create(context.Background(), 100, 200)
	require.NoError(t, err)
	_, err = s.Complete(context.Background(), p.ID, "tx")
	require.NoError(t, err)
	require.NoError(t, s.Refund(context.Background(), p.ID))
}