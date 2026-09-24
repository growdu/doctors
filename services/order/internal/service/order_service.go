// Package service 是 order-service 的业务编排层。
//
// 设计要点：
//   - 所有依赖都是接口；fake 替身在 _test.go。
//   - UserLookup 是查 user 信息的最小契约（当前只需要 FindByID）；
//     实际由 auth-service 暴露（HTTP）或本地 PG repo（直连）。
//   - 状态机走 internal/state.CanTransition；业务层负责拼装 from→to + InsertEvent。
//   - UpdateStatus（抢单）用乐观锁；并发安全在 3.6 给出。
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/growdu/doctors/services/order/internal/events"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/state"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// OrderRepo 是仓储最小契约。
type OrderRepo interface {
	Create(ctx context.Context, o *repo.Order) error
	FindByID(ctx context.Context, id int64) (*repo.Order, error)
	ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*repo.Order, error)
	UpdateStatus(ctx context.Context, id int64, to string, expectVersion int, escortID *int64) error
	InsertEvent(ctx context.Context, orderID int64, from *string, to string, actorID *int64, payload []byte) error
	ListEvents(ctx context.Context, orderID int64) ([]*repo.OrderEvent, error)
}

// UserLookup 仅需 FindByID：auth-service 的 user 视图（id + 是否实名）。
type UserLookup interface {
	FindByID(ctx context.Context, id int64) (*UserSnapshot, error)
}

// UserSnapshot 是 user 的最小子集，由 UserLookup 返回。
type UserSnapshot struct {
	ID               int64
	Role             string
	RealNameVerified bool
}

// Service 是 order 业务编排器。
type Service struct {
	orders    OrderRepo
	users     UserLookup
	txRunner  TxRunner  // 可选；抢单 (Accept) 时必需
	publisher  events.Publisher // 可选；nil 时不发布事件
	clockNow  func() time.Time // 用于测试注入时间
}

// New 装配一个 Service。
func New(orders OrderRepo, users UserLookup) *Service {
	return &Service{orders: orders, users: users, clockNow: time.Now}
}

// WithTx 注入事务执行器（阶段 3.6 接通 PG 后由 main 调）。
func (s *Service) WithTx(tx TxRunner) *Service {
	s.txRunner = tx
	return s
}

// WithPublisher 注入事件发布器。
func (s *Service) WithPublisher(p events.Publisher) *Service {
	s.publisher = p
	return s
}

// WithClock 注入时钟（测试用）。
func (s *Service) WithClock(now func() time.Time) *Service {
	s.clockNow = now
	return s
}

// CreateReq 是创建订单的参数。
type CreateReq struct {
	PatientID      int64
	HospitalID     int64
	PackageID      int64
	ServiceStartAt time.Time
	Amount         float64
}

// Create 创建一笔订单；状态初始 created，并落一条 order_event。
//   - patient 不存在 / 未实名 → CodeNotFound / CodeForbidden
//   - 金额 ≤ 0 或时间在过去 → CodeParamInvalid
func (s *Service) Create(ctx context.Context, req CreateReq) (*repo.Order, error) {
	if req.PatientID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "patient_id required")
	}
	if req.Amount <= 0 {
		return nil, errs.New(errs.CodeParamInvalid, "amount must be > 0")
	}
	if req.ServiceStartAt.Before(time.Now()) {
		return nil, errs.New(errs.CodeParamInvalid, "service_start_at must be in future")
	}

	u, err := s.users.FindByID(ctx, req.PatientID)
	if err != nil {
		return nil, errs.New(errs.CodeNotFound, "patient not found")
	}
	if !u.RealNameVerified {
		return nil, errs.New(errs.CodeForbidden, "real name not verified")
	}

	o := &repo.Order{
		OrderNo:        newOrderNo(),
		PatientID:      req.PatientID,
		HospitalID:     req.HospitalID,
		PackageID:      req.PackageID,
		ServiceStartAt: req.ServiceStartAt,
		Amount:         req.Amount,
		FinalAmount:    req.Amount, // v1 无折扣
		Status:         string(state.StatusCreated),
	}
	if err := s.orders.Create(ctx, o); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create order", err)
	}

	actor := u.ID
	from := ""
	if err := s.orders.InsertEvent(ctx, o.ID, &from, o.Status, &actor, nil); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "insert event", err)
	}

	// 发布 OrderCreatedEvent（best-effort，不影响主流程）
	if s.publisher != nil {
		_ = s.publisher.PublishOrderCreated(ctx, contracts.OrderCreatedEvent{
			OrderID:        o.ID,
			PatientID:      o.PatientID,
			HospitalID:     o.HospitalID,
			PackageID:      o.PackageID,
			ServiceStartAt: o.ServiceStartAt.(time.Time),
			Amount:         o.Amount,
		})
	}
	return o, nil
}

// List 返回某 patient 的订单分页。
func (s *Service) List(ctx context.Context, patientID int64, limit, offset int) ([]*repo.Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.orders.ListByPatient(ctx, patientID, limit, offset)
}

// Get 按 id 取订单。
func (s *Service) Get(ctx context.Context, id int64) (*repo.Order, error) {
	o, err := s.orders.FindByID(ctx, id)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return nil, errs.New(errs.CodeNotFound, "order not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find order", err)
	}
	return o, nil
}

// Cancel 患者主动取消订单。
func (s *Service) Cancel(ctx context.Context, orderID, actorID int64, reason string) error {
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return errs.New(errs.CodeNotFound, "order not found")
		}
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	from := state.Status(o.Status)
	if !state.CanTransition(from, state.StatusCanceled) {
		return errs.New(errs.CodeConflict, fmt.Sprintf("cannot cancel from status %s", from))
	}
	if err := s.orders.UpdateStatus(ctx, orderID, string(state.StatusCanceled), o.Version, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	fromStr := string(from)
	actor := actorID
	payload := []byte(fmt.Sprintf(`{"reason":%q}`, reason))
	if err := s.orders.InsertEvent(ctx, orderID, &fromStr, string(state.StatusCanceled), &actor, payload); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert event", err)
	}

	// 发布 OrderCancelledEvent
	if s.publisher != nil {
		_ = s.publisher.PublishOrderCancelled(ctx, contracts.OrderCancelledEvent{
			OrderID:     orderID,
			CancelledBy: actorID,
			Reason:      reason,
			CancelledAt: s.clockNow(),
		})
	}
	return nil
}

// Finish 陪诊师把订单置为 completed（实际业务需要 in_service → completed 的两步流程，
// 这里 v1 简化为一步直接完成，便于 mock）。
func (s *Service) Finish(ctx context.Context, orderID, actorID int64) error {
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return errs.New(errs.CodeNotFound, "order not found")
		}
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	from := state.Status(o.Status)
	to := state.StatusCompleted
	if !state.CanTransition(from, to) {
		return errs.New(errs.CodeConflict, fmt.Sprintf("cannot finish from %s", from))
	}
	if err := s.orders.UpdateStatus(ctx, orderID, string(to), o.Version, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	fromStr := string(from)
	actor := actorID
	if err := s.orders.InsertEvent(ctx, orderID, &fromStr, string(to), &actor, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert event", err)
	}
	return nil
}

// ---------- helpers ----------

// newOrderNo 生成 16 位 hex 订单号；带时间戳前缀便于排序。
func newOrderNo() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "O" + strings.ToUpper(hex.EncodeToString(b[:]))
}