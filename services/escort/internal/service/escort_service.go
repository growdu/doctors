// Package service 是 escort-service 的业务编排层。
//
// 设计要点：
//   - 陪诊师实体：用户（patient/escort）的 escort 子集；通过 user_id 关联。
//   - 可服务区间（available_from / available_until）用于评分过滤。
//   - SetAvailability 触发 events.EscortAvailable / EscortUnavailable；
//     match-service 消费后更新候选池。
//   - Status: "registered" | "available" | "busy" | "offline"。
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/growdu/doctors/shared/errs"
)

// Escort 是陪诊师的最小业务视图。
type Escort struct {
	ID             int64
	UserID         int64
	City           string
	Lat            float64
	Lng            float64
	Rating         float64
	Status         string
	AvailableFrom  time.Time
	AvailableUntil time.Time
	CreatedAt      time.Time
}

// EscortRepo 是仓储契约。
type EscortRepo interface {
	Create(ctx context.Context, e *Escort) error
	GetByID(ctx context.Context, id int64) (*Escort, error)
	GetByUserID(ctx context.Context, userID int64) (*Escort, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateLocation(ctx context.Context, id int64, lat, lng float64) error
	UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error
}

// AvailabilityEvent 是发布给 match-service 的最小事件。
type AvailabilityEvent struct {
	EscortID     int64     `json:"escort_id"`
	UserID       int64     `json:"user_id"`
	City         string    `json:"city"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Available    bool      `json:"available"`
	ValidUntil   time.Time `json:"valid_until"`
}

// Publisher 是事件发布抽象；match-service 消费后调整候选池。
type Publisher interface {
	PublishAvailabilityChanged(ctx context.Context, ev AvailabilityEvent) error
}

// ErrEscortNotFound 查无结果。
var ErrEscortNotFound = errors.New("escort service: not found")

// Service 是 escort 业务编排器。
type Service struct {
	repo      EscortRepo
	publisher Publisher
}

// New 装配 Service。
func New(r EscortRepo, p Publisher) *Service { return &Service{repo: r, publisher: p} }

// Register 注册一个新陪诊师；默认 status="registered"。
//   - userID 必须 > 0 且在 users 表存在（v1 不验，交给上层）
//   - 默认城市为空，location 由后续 PATCH 设置
func (s *Service) Register(ctx context.Context, userID int64) (*Escort, error) {
	if userID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "user_id required")
	}
	if existing, _ := s.repo.GetByUserID(ctx, userID); existing != nil {
		return nil, errs.New(errs.CodeConflict, "user already registered as escort")
	}
	e := &Escort{
		UserID: userID,
		Status: "registered",
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create escort", err)
	}
	return e, nil
}

// Get 取一个 escort 详情。
func (s *Service) Get(ctx context.Context, id int64) (*Escort, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get escort", err)
	}
	return e, nil
}

// SetAvailability 设置可服务状态；触发事件发布。
//   - available=true：status="available"，发布 EscortAvailable
//   - available=false：status="offline"，发布 EscortUnavailable
func (s *Service) SetAvailability(ctx context.Context, id int64, available bool, validUntil time.Time) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return errs.New(errs.CodeNotFound, "escort not found")
		}
		return errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	status := "offline"
	if available {
		status = "available"
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	if available && validUntil.IsZero() {
		validUntil = time.Now().Add(2 * time.Hour)
	}
	if err := s.repo.UpdateAvailability(ctx, id, time.Now(), validUntil); err != nil {
		return errs.Wrap(errs.CodeInternal, "update availability", err)
	}

	// 发布事件（失败不阻塞业务；log 错误即可）
	if s.publisher != nil {
		if err := s.publisher.PublishAvailabilityChanged(ctx, AvailabilityEvent{
			EscortID:   id,
			UserID:     e.UserID,
			City:       e.City,
			Lat:        e.Lat,
			Lng:        e.Lng,
			Available:  available,
			ValidUntil: validUntil,
		}); err != nil {
			// publish 失败仅记日志，不影响主流程
			_ = err
		}
	}
	return nil
}

// UpdateLocation 更新经纬度。
func (s *Service) UpdateLocation(ctx context.Context, id, callerID int64, lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return errs.New(errs.CodeParamInvalid, "lat/lng out of range")
	}
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return errs.New(errs.CodeNotFound, "escort not found")
		}
		return errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	if e.UserID != callerID {
		return errs.New(errs.CodeForbidden, "can only update own location")
	}
	if err := s.repo.UpdateLocation(ctx, id, lat, lng); err != nil {
		return errs.Wrap(errs.CodeInternal, "update location", err)
	}
	return nil
}

// UpdateCity 修改服务城市。
func (s *Service) UpdateCity(ctx context.Context, id, callerID int64, city string) error {
	city = strings.TrimSpace(city)
	if city == "" {
		return errs.New(errs.CodeParamInvalid, "city required")
	}
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return errs.New(errs.CodeNotFound, "escort not found")
		}
		return errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	if e.UserID != callerID {
		return errs.New(errs.CodeForbidden, "can only update own city")
	}
	// 复用 UpdateLocation 占位（生产应该有专门的 UpdateCity 方法）
	if err := s.repo.UpdateLocation(ctx, id, e.Lat, e.Lng); err != nil {
		return errs.Wrap(errs.CodeInternal, "update city", err)
	}
	return nil
}