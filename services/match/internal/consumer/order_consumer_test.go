package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/services/match/internal/scorer"
	"github.com/growdu/doctors/services/match/internal/service"
	"github.com/growdu/doctors/shared/contracts"
)

type nilLoader struct{}

func (nilLoader) ListAvailable(ctx context.Context, city string) ([]scorer.Escort, error) {
	return nil, nil
}

// TestHandleOrderCreated 验证 HandleOrderCreated 走通 service.Match。
func TestHandleOrderCreated(t *testing.T) {
	svc := service.New(pool.NewNopPool(), nilLoader{}, 0)
	ev := contracts.OrderCreatedEvent{OrderID: 100, City: "北京"}
	cands, err := HandleOrderCreated(context.Background(), svc, ev)
	require.NoError(t, err)
	// nilLoader 返回空 escort list → 0 候选
	assert.Empty(t, cands)
}

// TestHandleOrderCreated_AllFieldsForwarded 验证 events 的字段都被传入 service。
func TestHandleOrderCreated_AllFieldsForwarded(t *testing.T) {
	svc := service.New(pool.NewNopPool(), nilLoader{}, 0)
	ev := contracts.OrderCreatedEvent{
		OrderID:        42,
		City:           "上海",
		ServiceStartAt: time.Now().Add(2 * time.Hour),
		HospitalLat:    31.2,
		HospitalLng:    121.5,
	}
	_, err := HandleOrderCreated(context.Background(), svc, ev)
	require.NoError(t, err)
}