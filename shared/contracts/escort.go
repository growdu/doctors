package contracts

import "time"

// EscortSummary 是 escort 的对外摘要（多服务共享）。
//
// 设计要点：
//   - escort-service 是生产者；match-service / notification 等是消费者。
//   - 字段尽量原子，避免暴露 escort-service 内部状态机。
//   - Rating 是 0~5 浮点（与 escort-service 一致）；score 计算在 match 侧。
//   - AvailableFrom / AvailableUntil 是陪诊师承诺可服务的窗口；为空表示不可服务。
type EscortSummary struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	City           string    `json:"city"`
	Lat            float64   `json:"lat"`
	Lng            float64   `json:"lng"`
	Rating         float64   `json:"rating"`
	AvailableFrom  time.Time `json:"available_from"`
	AvailableUntil time.Time `json:"available_until"`
	Status         string    `json:"status"` // "registered" | "available" | "busy" | "offline"
}

// IsAvailable 业务侧便捷判断：Status == "available" 且当前时间在窗口内。
func (e EscortSummary) IsAvailable(now time.Time) bool {
	if e.Status != "available" {
		return false
	}
	if !e.AvailableFrom.IsZero() && now.Before(e.AvailableFrom) {
		return false
	}
	if !e.AvailableUntil.IsZero() && now.After(e.AvailableUntil) {
		return false
	}
	return true
}