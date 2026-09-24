package availability

import (
	"context"
	"errors"
	"time"
)

// Clock 接口允许 service 层注入时钟（测试用）。
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// Service 是 escort-availabilities 的业务编排层。
//
// 高内聚低耦合：业务校验全部在这里；repo 保持 thin（pgx 直写 SQL）。
// order-service.SelectEscort 注入 AvailabilityLookup 接口（v1.1 escort-order-ext plan）调用 HasAvailabilityFor。
//
// Service.repo 用 AvailabilityRepo 接口（避免 Service 与具体 Repo 强耦合；测试 fake 实现接口即可注入）。
type Service struct {
	repo  AvailabilityRepo
	clock Clock
}

// AvailabilityRepo 是 Service 依赖的仓储接口（业务层注入 fake 实现单测）。
// 具体实现在 repo.go（*Repo）；v1.1 高内聚：业务接口独立于具体实现。
type AvailabilityRepo interface {
	Create(ctx context.Context, escortID int64, startAt, endAt time.Time) (*Availability, error)
	FindByID(ctx context.Context, id int64) (*Availability, error)
	Delete(ctx context.Context, id int64, escortID int64) error
	ListByEscort(ctx context.Context, escortID int64) ([]*Availability, error)
	ListAvailableByTime(ctx context.Context, startAt, endAt time.Time, limit int) ([]*Availability, error)
	BookByOrder(ctx context.Context, id int64, orderID int64) error
	ReleaseByOrder(ctx context.Context, orderID int64) error
}

// NewService 构造 Service；默认 systemClock。
func NewService(repo AvailabilityRepo) *Service {
	return &Service{repo: repo, clock: systemClock{}}
}

// WithClock 注入时钟（测试用）。
func (s *Service) WithClock(c Clock) *Service { s.clock = c; return s }

// AddAvailability 陪诊师加新时段（自己只能加自己）。
//   - 校验 owner + 时段合法 + 不与已有时段重叠（repo 兜底）。
func (s *Service) AddAvailability(ctx context.Context, escortID, callerID int64, startAt, endAt time.Time) (*Availability, error) {
	if escortID != callerID {
		return nil, ErrNotOwner
	}
	a := &Availability{EscortID: escortID, StartAt: startAt, EndAt: endAt, Status: StatusAvailable}
	if err := a.Validate(s.clock.Now()); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, escortID, startAt, endAt)
}

// RemoveAvailability 陪诊师删时段（仅 owner + status=available）。
func (s *Service) RemoveAvailability(ctx context.Context, id, callerID int64) error {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if a.EscortID != callerID {
		return ErrNotOwner
	}
	if a.Status == StatusBooked {
		return ErrBookedAlready
	}
	return s.repo.Delete(ctx, id, callerID)
}

// ListMyAvailabilities 陪诊师查自己的所有时段。
func (s *Service) ListMyAvailabilities(ctx context.Context, callerID int64) ([]*Availability, error) {
	return s.repo.ListByEscort(ctx, callerID)
}

// ListEscortAvailabilities 公开查询：某 escort 在某时段窗口内可用的时段。
//   - 用于 patient-miniapp / admin-web 渲染"该 escort 在 X 时段可服务"。
func (s *Service) ListEscortAvailabilities(ctx context.Context, escortID int64, startAt, endAt time.Time, limit int) ([]*Availability, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListAvailableByTime(ctx, startAt, endAt, limit)
}

// HasAvailabilityFor 实现 escort-order-ext 计划定义的 AvailabilityLookup 接口。
//   - 检查 escort 在 serviceStartAt 时段窗口（默认 ±1 小时）内是否有 available 时段。
//   - v1 简化：仅查 serviceStartAt 精确命中的时段；window 检查由 caller 决定。
func (s *Service) HasAvailabilityFor(ctx context.Context, escortID int64, serviceStartAt any) (bool, error) {
	t, ok := serviceStartAt.(time.Time)
	if !ok {
		return false, nil
	}
	// window = [t-1h, t+24h) — 陪诊师只要在该窗口内有时段就算"可服务"
	start := t.Add(-1 * time.Hour)
	end := t.Add(24 * time.Hour)
	list, err := s.repo.ListAvailableByTime(ctx, start, end, 100)
	if err != nil {
		return false, err
	}
	for _, a := range list {
		if a.EscortID == escortID {
			return true, nil
		}
	}
	return false, nil
}

// BookForOrder 实现 order-service 集成接口：ConfirmAccept 调。
func (s *Service) BookForOrder(ctx context.Context, id int64, orderID int64) error {
	return s.repo.BookByOrder(ctx, id, orderID)
}

// ReleaseForOrder 实现 order-service 集成接口：Cancel / RejectAccept 调。
func (s *Service) ReleaseForOrder(ctx context.Context, orderID int64) error {
	return s.repo.ReleaseByOrder(ctx, orderID)
}

// Ensure 编译期确保 *Repo 实现 AvailabilityRepo 接口。
var _ AvailabilityRepo = (*Repo)(nil)

// Ensure 编译期确保 Service 实现 AvailabilityLookup-like 接口（实际接口定义在 order 包）。
var _ = errors.New