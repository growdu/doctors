// Package scorer 实现候选打分（pure function）。
//
// 设计要点：
//   - 城市 / 时段 / 评分 / 距离 四个维度。
//   - 分数越高越优先；0 表示不可用（城市不匹配 / 时段冲突）。
//   - 所有权重写死为常量；后续可在 config 里覆盖。
package scorer

import (
	"math"
	"time"
)

// Escort 是评分输入（最小子集；实际由 escort-service 提供）。
type Escort struct {
	ID        int64
	City      string
	StartTime time.Time // 可服务开始
	EndTime   time.Time // 可服务结束
	Rating    float64   // 0~5
	Distance  float64   // km
	Lat       float64
	Lng       float64
}

// Order 是评分目标。
type Order struct {
	City        string
	ServiceTime time.Time
	Lat         float64
	Lng         float64
}

// Candidate 是输出。
type Candidate struct {
	EscortID int64
	Score    float64
}

// 权重。
const (
	wCity    = 100.0
	wTime    = 80.0
	wRating  = 40.0
	wDist    = 30.0
	maxDist  = 50.0 // km，超过直接 0
	minScore = 1e-6 // 避免 -0
)

// Compute 计算所有 escort 的得分；过滤掉 0 分（不可用）。
func Compute(o Order, escorts []Escort) []Candidate {
	out := make([]Candidate, 0, len(escorts))
	for _, e := range escorts {
		s := scoreOne(o, e)
		if s > minScore {
			out = append(out, Candidate{EscortID: e.ID, Score: s})
		}
	}
	return out
}

func scoreOne(o Order, e Escort) float64 {
	// 1. 城市：必须匹配；否则 0
	if o.City != "" && e.City != "" && o.City != e.City {
		return 0
	}
	// 2. 时段：escort 可服务区间必须包含 order.service_time
	if !e.StartTime.IsZero() && !e.EndTime.IsZero() && !o.ServiceTime.IsZero() {
		if o.ServiceTime.Before(e.StartTime) || o.ServiceTime.After(e.EndTime) {
			return 0
		}
	}
	// 3. 评分：0~5 映射 0~wRating
	ratingScore := e.Rating / 5.0 * wRating
	// 4. 距离：越近越高（线性衰减到 0）
	dist := e.Distance
	if dist <= 0 {
		dist = estimateDistance(o, e)
	}
	if dist > maxDist {
		return 0 // 超过 50km 一律不可用
	}
	distScore := 0.0
	if dist > 0 {
		distScore = (1 - dist/maxDist) * wDist
	}
	return wCity + wTime + ratingScore + distScore
}

// estimateDistance 用经纬度算近似距离（km）；Haversine 公式。
func estimateDistance(o Order, e Escort) float64 {
	if o.Lat == 0 || o.Lng == 0 || e.Lat == 0 || e.Lng == 0 {
		return 0
	}
	const R = 6371.0 // km
	lat1 := o.Lat * math.Pi / 180
	lat2 := e.Lat * math.Pi / 180
	dlat := (e.Lat - o.Lat) * math.Pi / 180
	dlon := (e.Lng - o.Lng) * math.Pi / 180
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}