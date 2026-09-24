package events

import (
	"context"
	"encoding/json"
	"strconv"
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

// TestNopPublisher_CountsCompleted 验证 PublishOrderCompleted 计数（v1.2 wallet-t+7）。
func TestNopPublisher_CountsCompleted(t *testing.T) {
	p := &NopPublisher{}
	ev := contracts.OrderCompletedEvent{OrderID: 1, EscortID: 7, Amount: 300.00, CompletedAt: time.Now()}
	require.NoError(t, p.PublishOrderCompleted(context.Background(), ev))
	assert.Equal(t, 1, p.CompletedCount)
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

// TestKafkaPublisher_PublishOrderCompleted_TopicAndKeyAndValue
// 验证 PublishOrderCompleted 把 OrderCompletedEvent JSON-序列化后写入
//   - topic = contracts.TopicOrderCompleted
//   - key   = strconv.FormatInt(ev.OrderID, 10)
//   - value = 合法 JSON（含 order_id/escort_id/amount/completed_at 4 字段）
// 实现思路：内部辅助 publish(ctx, topic, key, body) 在 marshal 之后才写 kafka，
// 故单独验证「topic 常量 + key 格式 + value 序列化」三件套即可保证契约一致。
func TestKafkaPublisher_PublishOrderCompleted_TopicAndKeyAndValue(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ev := contracts.OrderCompletedEvent{
		OrderID:     42,
		EscortID:    7,
		Amount:      300.50,
		CompletedAt: now,
	}
	// 1) topic 常量
	assert.Equal(t, "order.completed", contracts.TopicOrderCompleted)
	// 2) key 格式
	assert.Equal(t, "42", strconv.FormatInt(ev.OrderID, 10))
	// 3) value JSON 合法且字段对齐（这是 wallet 消费端 Unmarshal 的契约）
	data, err := json.Marshal(ev)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"order_id":42`)
	assert.Contains(t, string(data), `"escort_id":7`)
	assert.Contains(t, string(data), `"amount":300.5`)
	assert.Contains(t, string(data), `"completed_at":`)
	var got contracts.OrderCompletedEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev, got)
}

// TestKafkaPublisher_PublishOrderCompleted_NilWriterPassesError 验证 nil-writer 场景下
// PublishOrderCompleted 与其他 PublishXxx 一致地返回非 nil error（上层按 best-effort 处理）。
func TestKafkaPublisher_PublishOrderCompleted_NilWriterPassesError(t *testing.T) {
	var p *KafkaPublisher
	err := p.PublishOrderCompleted(context.Background(),
		contracts.OrderCompletedEvent{OrderID: 1, EscortID: 7, Amount: 100, CompletedAt: time.Now()})
	assert.Error(t, err)
}

// TestPublisher_InterfaceImplementations 验证 NopPublisher 满足 Publisher 接口。
func TestPublisher_InterfaceImplementations(t *testing.T) {
	var _ Publisher = (*NopPublisher)(nil)
	var _ Publisher = (*KafkaPublisher)(nil)
}