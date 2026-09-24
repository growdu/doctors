package scorer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/contracts"
)

// helper 构造一个 EscortSummary。
func mkEscort(id int64, city string, rating float64, availableFrom, availableUntil time.Time, lat, lng float64) contracts.EscortSummary {
	return contracts.EscortSummary{
		ID:             id,
		City:           city,
		Rating:         rating,
		AvailableFrom:  availableFrom,
		AvailableUntil: availableUntil,
		Lat:            lat,
		Lng:            lng,
		Status:         "available",
	}
}

func TestCompute_CityMismatchZero(t *testing.T) {
	o := Order{City: "北京", ServiceTime: time.Now()}
	escorts := []contracts.EscortSummary{mkEscort(1, "上海", 5, time.Time{}, time.Time{}, 0, 0)}
	out := Compute(o, escorts)
	assert.Len(t, out, 0, "城市不匹配应被过滤")
}

func TestCompute_TimeConflictZero(t *testing.T) {
	now := time.Now()
	o := Order{City: "北京", ServiceTime: now.Add(2 * time.Hour)}
	escorts := []contracts.EscortSummary{
		mkEscort(1, "北京", 5, now.Add(3*time.Hour), now.Add(4*time.Hour), 0, 0),
	}
	out := Compute(o, escorts)
	assert.Len(t, out, 0, "时段冲突应被过滤")
}

func TestCompute_SortByScore(t *testing.T) {
	now := time.Now()
	o := Order{City: "北京", ServiceTime: now.Add(1 * time.Hour), Lat: 39.9, Lng: 116.4}
	escorts := []contracts.EscortSummary{
		mkEscort(1, "北京", 5, now, now.Add(2*time.Hour), 39.91, 116.41),
		mkEscort(2, "北京", 3, now, now.Add(2*time.Hour), 40.0, 116.5),
		mkEscort(3, "上海", 5, now, now.Add(2*time.Hour), 0, 0),
	}
	out := Compute(o, escorts)
	require.Len(t, out, 2)
	assert.Equal(t, int64(1), out[0].EscortID)
	assert.Greater(t, out[0].Score, out[1].Score)
}

func TestCompute_DistanceDecay(t *testing.T) {
	now := time.Now()
	o := Order{City: "北京", ServiceTime: now, Lat: 39.9, Lng: 116.4}
	near := mkEscort(1, "北京", 5, now.Add(-time.Hour), now.Add(time.Hour), 39.91, 116.41)
	mid := mkEscort(2, "北京", 5, now.Add(-time.Hour), now.Add(time.Hour), 39.95, 116.45)

	out := Compute(o, []contracts.EscortSummary{near, mid})
	require.Len(t, out, 2)
	nearIdx, midIdx := 0, 1
	if out[0].EscortID == 2 {
		nearIdx, midIdx = 1, 0
	}
	assert.Greater(t, out[nearIdx].Score, out[midIdx].Score)
}

func TestCompute_Over50kmZero(t *testing.T) {
	now := time.Now()
	o := Order{City: "北京", ServiceTime: now, Lat: 39.9, Lng: 116.4}
	// 经纬度设置得足够远（北京 → 西安 ≈ 1100 km）
	far := contracts.EscortSummary{
		ID: 1, City: "北京", Rating: 5,
		AvailableFrom: now.Add(-time.Hour), AvailableUntil: now.Add(time.Hour),
		Lat: 34.3, Lng: 108.9,
		Status: "available",
	}
	out := Compute(o, []contracts.EscortSummary{far})
	assert.Len(t, out, 0, "距离 >50km 应被过滤")
}

func TestEstimateDistance_Symmetry(t *testing.T) {
	o := Order{Lat: 39.9, Lng: 116.4}
	e := contracts.EscortSummary{Lat: 40.0, Lng: 116.5}
	d1 := estimateDistance(o, e)
	o2 := Order{Lat: 40.0, Lng: 116.5}
	e2 := contracts.EscortSummary{Lat: 39.9, Lng: 116.4}
	d2 := estimateDistance(o2, e2)
	assert.InDelta(t, d1, d2, 0.1)
	assert.Greater(t, d1, 0.0)
}