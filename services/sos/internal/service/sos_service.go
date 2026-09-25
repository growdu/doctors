// Package service - sos-service 业务编排。
//
// 设计要点：
//   - 紧急信号仅允许 status=in_service / matched 的订单发起。
//   - Raise 触发 contracts.SOSRaisedEvent → 通知 admin / escort / 患者紧急联系人。
//   - 5 分钟内同一订单去重（避免 panic 按错）。
package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// SOS 是紧急信号实体。
type SOS struct {
	ID         int64
	OrderID    int64
	UserID     int64
	Lat        float64
	Lng        float64
	Note       string
	Status     string // "raised" | "handling" | "resolved"
	RaisedAt   time.Time
	ResolvedAt *time.Time
}

// OrderStateLookup 是订单状态查询抽象（不直接依赖 order-service 包）。
type OrderStateLookup interface {
	IsOrderActive(ctx context.Context, orderID int64) (bool, error)
}

// Repo 是仓储契约。
type Repo interface {
	Create(ctx context.Context, s *SOS) error
	GetByID(ctx context.Context, id int64) (*SOS, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	IsRecentDuplicate(ctx context.Context, orderID int64, since time.Time) (bool, error)
	List(ctx context.Context, f ListFilter) ([]*SOS, error)
}

// ListFilter 是 List 的过滤参数。
type ListFilter struct {
	OrderID  int64
	Status   string
	Page     int
	PageSize int
}

// Publisher 是事件发布抽象。
type Publisher interface {
	PublishSOSRaised(ctx context.Context, ev contracts.SOSRaisedEvent) error
}

// Service 是 sos 业务编排器。
type Service struct {
	repo      Repo
	lookup    OrderStateLookup
	publisher Publisher
}

// New 装配 Service。
func New(r Repo, l OrderStateLookup, p Publisher) *Service {
	return &Service{repo: r, lookup: l, publisher: p}
}

// Raise 发起紧急信号。
func (s *Service) Raise(ctx context.Context, orderID, userID int64, lat, lng float64, note string) (*SOS, error) {
	if orderID == 0 || userID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id / user_id required")
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, errs.New(errs.CodeParamInvalid, "lat/lng out of range")
	}
	if utf8.RuneCountInString(strings.TrimSpace(note)) > 500 {
		return nil, errs.New(errs.CodeParamInvalid, "note too long (max 500 chars)")
	}

	// 订单必须处于活动状态
	if s.lookup != nil {
		active, err := s.lookup.IsOrderActive(ctx, orderID)
		if err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "lookup order", err)
		}
		if !active {
			return nil, errs.New(errs.CodeConflict, "order not active, cannot raise SOS")
		}
	}

	// 5 分钟内去重
	if s.repo != nil {
		dup, err := s.repo.IsRecentDuplicate(ctx, orderID, time.Now().Add(-5*time.Minute))
		if err == nil && dup {
			return nil, errs.New(errs.CodeConflict, "duplicate SOS within 5min")
		}
	}

	sos := &SOS{
		OrderID:  orderID,
		UserID:   userID,
		Lat:      lat,
		Lng:      lng,
		Note:     strings.TrimSpace(note),
		Status:   "raised",
		RaisedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, sos); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create sos", err)
	}

	if s.publisher != nil {
		_ = s.publisher.PublishSOSRaised(ctx, contracts.SOSRaisedEvent{
			SOSID: sos.ID, OrderID: orderID, UserID: userID,
			Lat: lat, Lng: lng, Note: sos.Note, RaisedAt: sos.RaisedAt,
		})
	}
	return sos, nil
}

// GetByID 取 SOS 详情。
func (s *Service) GetByID(ctx context.Context, id int64) (*SOS, error) {
	if id == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "id required")
	}
	sos, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrSOSNotFound) {
			return nil, errs.New(errs.CodeNotFound, "sos not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find sos", err)
	}
	return sos, nil
}

// List 拉 SOS 列表（按 order / status / page 过滤）。
func (s *Service) List(ctx context.Context, f ListFilter) ([]*SOS, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	return s.repo.List(ctx, f)
}

// Resolve 标记 SOS 已处理。
func (s *Service) Resolve(ctx context.Context, sosID int64) error {
	if sosID == 0 {
		return errs.New(errs.CodeParamInvalid, "sos_id required")
	}
	sos, err := s.repo.GetByID(ctx, sosID)
	if err != nil {
		if errors.Is(err, ErrSOSNotFound) {
			return errs.New(errs.CodeNotFound, "sos not found")
		}
		return errs.Wrap(errs.CodeInternal, "find sos", err)
	}
	if sos.Status == "resolved" {
		return errs.New(errs.CodeConflict, "already resolved")
	}
	if err := s.repo.UpdateStatus(ctx, sosID, "resolved"); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	return nil
}

// ErrSOSNotFound 是查无结果。
var ErrSOSNotFound = errors.New("sos service: not found")
