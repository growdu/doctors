// Package availability 是 escort-service 的空余时段子包（2026-09-24 escort-availability plan v1.1）。
//
// 设计要点：
//   - escort_availabilities 表存多个时段（一个 escort 可同时段有多个时段）；
//   - 状态 available / booked / canceled；order-service 在 confirm accept 时 BookByOrder，cancel / reject 时 ReleaseByOrder。
//   - 高内聚低耦合：service 包定义 AvailabilityLookup 接口；order-service 在 SelectEscort 校验时调用（v1 注入 in-memory stub）。
//   - v1 实施范围：4 API + 7 集成测试 + 8 单测（与 plan 一致）。
package availability

import (
	"errors"
	"time"
)

// Status 是 escort_availabilities.status 枚举，与 DB CHECK 对齐。
type Status string

const (
	StatusAvailable Status = "available"
	StatusBooked    Status = "booked"
	StatusCanceled  Status = "canceled"
)

// Availability 映射 escort_availabilities 表行。
type Availability struct {
	ID            int64
	EscortID      int64
	StartAt       time.Time
	EndAt         time.Time
	Status        Status
	BookedOrderID *int64
}

// ErrAvailabilityNotFound 查询无结果哨兵。
var ErrAvailabilityNotFound = errors.New("availability: not found")

// ErrAvailabilityConflict 时段重叠（escort 同时段已有 available 记录）。
var ErrAvailabilityConflict = errors.New("availability: time range conflicts with existing availability")

// ErrNotOwner 操作者非 escort owner。
var ErrNotOwner = errors.New("availability: caller is not the owner")

// ErrBookedAlready 时段已 booked（不可再 book 或取消）。
var ErrBookedAlready = errors.New("availability: already booked")

// ErrTimeInvalid 时段起止时间非法（end <= start 或在过去）。
var ErrTimeInvalid = errors.New("availability: time range invalid (end must be after start and in future)")

// ErrNotBookable 时段状态不允许 book（canceled / 已是 booked）。
var ErrNotBookable = errors.New("availability: not in bookable state")

// Validate 检查时段起止时间合法性（service 层复用）。
func (a *Availability) Validate(now time.Time) error {
	if !a.EndAt.After(a.StartAt) {
		return ErrTimeInvalid
	}
	if a.StartAt.Before(now) {
		return ErrTimeInvalid
	}
	return nil
}

// Overlaps 检查与另一时段是否时间重叠（开区间）。
func (a *Availability) Overlaps(other *Availability) bool {
	return a.StartAt.Before(other.EndAt) && other.StartAt.Before(a.EndAt)
}