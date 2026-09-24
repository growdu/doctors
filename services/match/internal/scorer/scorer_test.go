package scorer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompute_CityMismatchZero(t *testing.T) {
	o := Order{City: "北京", ServiceTime: time.Now()}
	escorts := []Escort{{ID: 1, City: "上海", Rating: 5}}
	out := Compute(o, escorts)
	assert.Len(t, out, 0, "城市不匹配应被过滤")
}

func TestCompute_TimeConflictZero(t *testing.T) {
	now := time.Now()
	o := Order{City: "北京", ServiceTime: now.Add(2 * time.Hour)}
	escorts := []Escort{{
		ID: 1, City: "北京", Rating: 5,
		StartTime: now.Add(3 * time.Hour), EndTime: now.Add(4 * time.Hour),
	}}
	out := Compute(o, escorts)
	assert.Len(t, out, 0, "时段冲突应被过滤")
}

func TestCompute_SortByScore(t *testing.T) {
	now := time.Now()
	o := Order{
		City: "北京", ServiceTime: now.Add(1 * time.Hour),
		Lat: 39.9, Lng: 116.4,
	}
	escorts := []Escort{
		{ID: 1, City: "北京", Rating: 5, Lat: 39.91, Lng: 116.41, StartTime: now, EndTime: now.Add(2 * time.Hour)},
		{ID: 2, City: "北京", Rating: 3, Lat: 40.0, Lng: 116.5, StartTime: now, EndTime: now.Add(2 * time.Hour)},
		{ID: 3, City: "上海", Rating: 5, StartTime: now, EndTime: now.Add(2 * time.Hour)}, // 城市错
	}
	out := Compute(o, escorts)
	require.Len(t, out, 2)
	// 第一个应该是评分更高且距离更近的 escort 1
	assert.Equal(t, int64(1), out[0].EscortID)
	assert.Greater(t, out[0].Score, out[1].Score)
}

func TestCompute_DistanceDecay(t *testing.T) {
	now := time.Now()
	o := Order{
		City: "北京", ServiceTime: now,
		Lat: 39.9, Lng: 116.4,
	}
	near := Escort{ID: 1, City: "北京", Rating: 5, Lat: 39.91, Lng: 116.41, StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)}
	mid := Escort{ID: 2, City: "北京", Rating: 5, Lat: 39.95, Lng: 116.45, StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)}

	out := Compute(o, []Escort{near, mid})
	require.Len(t, out, 2)
	// near should be higher than mid (closer = higher dist score)
	nearIdx, midIdx := 0, 1
	if out[0].EscortID == 2 {
		nearIdx, midIdx = 1, 0
	}
	assert.Greater(t, out[nearIdx].Score, out[midIdx].Score, "近的 escort 应得分更高")
}

func TestCompute_Over50kmZero(t *testing.T) {
	now := time.Now()
	o := Order{
		City: "北京", ServiceTime: now,
		Lat: 39.9, Lng: 116.4,
	}
	far := Escort{ID: 1, City: "北京", Rating: 5, Distance: 100, StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)}
	out := Compute(o, []Escort{far})
	assert.Len(t, out, 0, "距离 >50km 应被过滤")
}

func TestEstimateDistance_Symmetry(t *testing.T) {
	o := Order{Lat: 39.9, Lng: 116.4}
	e := Escort{Lat: 40.0, Lng: 116.5}
	d1 := estimateDistance(o, e)
	// 同样调用经纬度
	o2 := Order{Lat: 40.0, Lng: 116.5}
	e2 := Escort{Lat: 39.9, Lng: 116.4}
	d2 := estimateDistance(o2, e2)
	assert.InDelta(t, d1, d2, 0.1, "距离应对称")
	assert.Greater(t, d1, 0.0)
}