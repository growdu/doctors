package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/contracts"
)

// TestNopPublisher_CountsCreated 验证 PublishOrderCreated 计数。
func TestNopPublisher_CountsCreated(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderCreatedEvent{OrderID: 1, City: "北京"}
	require.NoError(t, p.PublishOrderCreated(context.Background(), ev))
	require.NoError(t, p.PublishOrderCreated(context.Background(), contracts.OrderCreatedEvent{OrderID: 2}))
	assert.Equal(t, 2, p.CreatedCount)
}

// TestNopPublisher_CountsAccepted 验证 PublishOrderAccepted 计数。
func TestNopPublisher_CountsAccepted(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderAcceptedEvent{OrderID: 1, EscortID: 7, AcceptedAt: time.Now()}
	require.NoError(t, p.PublishOrderAccepted(context.Background(), ev))
	assert.Equal(t, 1, p.AcceptedCount)
}

// TestNopPublisher_CountsCancelled 验证 PublishOrderCancelled 计数。
func TestNopPublisher_CountsCancelled(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderCancelledEvent{OrderID: 1, CancelledBy: 5, Reason: "patient", CancelledAt: time.Now()}
	require.NoError(t, p.PublishOrderCancelled(context.Background(), ev))
	assert.Equal(t, 1, p.CancelledCount)
}

// TestNopPublisher_CountsSelectingEscort 验证 PublishOrderSelectingEscort 计数（v1.1 新增）。
func TestNopPublisher_CountsSelectingEscort(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderSelectingEscortEvent{OrderID: 1, PatientID: 5, City: "shanghai", OccurredAt: time.Now()}
	require.NoError(t, p.PublishOrderSelectingEscort(context.Background(), ev))
	assert.Equal(t, 1, p.SelectingEscortCount)
}

// TestNopPublisher_CountsEscortSelected 验证 PublishOrderEscortSelected 计数（v1.1 新增）。
func TestNopPublisher_CountsEscortSelected(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderEscortSelectedEvent{
		OrderID: 1, PatientID: 5, SelectedEscortID: 7,
		EscortPendingExpireAt: time.Now().Add(30 * time.Second),
		OccurredAt:            time.Now(),
	}
	require.NoError(t, p.PublishOrderEscortSelected(context.Background(), ev))
	assert.Equal(t, 1, p.EscortSelectedCount)
}

// TestNopPublisher_CountsEscortConfirmed 验证 PublishOrderEscortConfirmed 计数（v1.1 新增）。
func TestNopPublisher_CountsEscortConfirmed(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderEscortConfirmedEvent{OrderID: 1, PatientID: 5, EscortID: 7, ConfirmedAt: time.Now()}
	require.NoError(t, p.PublishOrderEscortConfirmed(context.Background(), ev))
	assert.Equal(t, 1, p.EscortConfirmedCount)
}

// TestNopPublisher_CountsEscortRejected 验证 PublishOrderEscortRejected 计数（v1.1 新增）。
func TestNopPublisher_CountsEscortRejected(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderEscortRejectedEvent{OrderID: 1, PatientID: 5, EscortID: 7, Reason: "lock_expired", OccurredAt: time.Now()}
	require.NoError(t, p.PublishOrderEscortRejected(context.Background(), ev))
	assert.Equal(t, 1, p.EscortRejectedCount)
}

// TestNopPublisher_Close 不报错。
func TestNopPublisher_Close(t *testing.T) {
	p := &NopPublisher{}
	assert.NoError(t, p.Close())
}

// TestKafkaPublisher_NilWriterReturnsError 直接用 nil writer 不应 panic，应返回错误。
func TestKafkaPublisher_NilWriterReturnsError(t *testing.T) {
	var p *KafkaPublisher
	err := p.PublishOrderCreated(context.Background(), contracts.OrderCreatedEvent{OrderID: 1})
	assert.Error(t, err)
}

// TestPublisher_InterfaceImplementations 验证 NopPublisher 满足 Publisher 接口。
func TestPublisher_InterfaceImplementations(t *testing.T) {
	var _ Publisher = (*NopPublisher)(nil)
	var _ Publisher = (*KafkaPublisher)(nil)
}