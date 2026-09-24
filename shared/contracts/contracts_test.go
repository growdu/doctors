package contracts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrderCreatedEvent_RoundTrip 验证 JSON 序列化可逆。
func TestOrderCreatedEvent_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ev := OrderCreatedEvent{
		OrderID:        100,
		PatientID:      1,
		HospitalID:     50,
		ServiceStartAt: now,
		City:           "北京",
		HospitalLat:    39.9,
		HospitalLng:    116.4,
		Amount:         200.0,
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var got OrderCreatedEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev.OrderID, got.OrderID)
	assert.Equal(t, ev.PatientID, got.PatientID)
	assert.Equal(t, ev.HospitalID, got.HospitalID)
	assert.Equal(t, ev.City, got.City)
	assert.Equal(t, ev.ServiceStartAt.Unix(), got.ServiceStartAt.Unix())
}

// TestOrderAcceptedEvent_RoundTrip 验证 OrderAcceptedEvent 序列化。
func TestOrderAcceptedEvent_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ev := OrderAcceptedEvent{
		OrderID:  100,
		EscortID: 7,
		AcceptedAt: now,
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var got OrderAcceptedEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev.OrderID, got.OrderID)
	assert.Equal(t, ev.EscortID, got.EscortID)
	assert.Equal(t, ev.AcceptedAt.Unix(), got.AcceptedAt.Unix())
}

// TestOrderCancelledEvent_RoundTrip 验证 OrderCancelledEvent。
func TestOrderCancelledEvent_RoundTrip(t *testing.T) {
	ev := OrderCancelledEvent{
		OrderID:     100,
		CancelledBy: 1,
		Reason:      "patient changed mind",
		CancelledAt: time.Now().Truncate(time.Second),
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var got OrderCancelledEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev, got)
}

// TestTopicConstants 验证 topic 名常量化（避免拼写漂移）。
func TestTopicConstants(t *testing.T) {
	assert.Equal(t, "order.created", TopicOrderCreated)
	assert.Equal(t, "order.accepted", TopicOrderAccepted)
	assert.Equal(t, "order.cancelled", TopicOrderCancelled)
	assert.Equal(t, "order.matching", TopicOrderMatching)
	assert.Equal(t, "user.registered", TopicUserRegistered)
	assert.Equal(t, "escort.available", TopicEscortAvailable)
	assert.Equal(t, "payment.completed", TopicPaymentCompleted)
}

// TestOrderMatchingEvent_RoundTrip 验证 OrderMatchingEvent 序列化可逆。
func TestOrderMatchingEvent_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ev := OrderMatchingEvent{
		OrderID:   100,
		EscortID:  7,
		Reason:    "lock_expired",
		RetriedAt: now,
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)
	var got OrderMatchingEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev, got)
}
// TestEscortSummary_IsAvailable 验证 IsAvailable 边界。
func TestEscortSummary_IsAvailable(t *testing.T) {
	now := time.Now()
	e := EscortSummary{
		Status:         "available",
		AvailableFrom:  now.Add(-time.Hour),
		AvailableUntil: now.Add(time.Hour),
	}
	assert.True(t, e.IsAvailable(now))

	// 窗口之前
	e2 := EscortSummary{
		Status:         "available",
		AvailableFrom:  now.Add(time.Hour),
		AvailableUntil: now.Add(2 * time.Hour),
	}
	assert.False(t, e2.IsAvailable(now))

	// 窗口之后
	e3 := EscortSummary{
		Status:         "available",
		AvailableFrom:  now.Add(-2 * time.Hour),
		AvailableUntil: now.Add(-time.Hour),
	}
	assert.False(t, e3.IsAvailable(now))

	// Status 不对
	e4 := EscortSummary{
		Status:         "busy",
		AvailableFrom:  now.Add(-time.Hour),
		AvailableUntil: now.Add(time.Hour),
	}
	assert.False(t, e4.IsAvailable(now))
}
