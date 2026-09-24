package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/services/match/internal/scorer"
	"github.com/growdu/doctors/services/match/internal/service"
)

type nilLoader struct{}

func (nilLoader) ListAvailable(ctx context.Context, city string) ([]scorer.Escort, error) {
	return nil, nil
}

// TestHandleOrderCreated 验证 HandleOrderCreated 走通 service.Match。
func TestHandleOrderCreated(t *testing.T) {
	svc := service.New(pool.NewNopPool(), nilLoader{}, 0)
	ev := OrderCreatedEvent{OrderID: 100, City: "北京"}
	cands, err := HandleOrderCreated(context.Background(), svc, ev)
	require.NoError(t, err)
	// nilLoader 返回空 escort list → 0 候选
	assert.Empty(t, cands)
}

// TestOrderCreatedEvent_DecodeRoundtrip 验证 JSON 序列化可逆。
func TestOrderCreatedEvent_DecodeRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ev := OrderCreatedEvent{
		OrderID:        1,
		City:           "上海",
		ServiceStartAt: now,
		HospitalLat:    31.2,
		HospitalLng:    121.5,
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)
	var got OrderCreatedEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev.OrderID, got.OrderID)
	assert.Equal(t, ev.City, got.City)
	assert.Equal(t, ev.HospitalLat, got.HospitalLat)
}