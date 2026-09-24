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

	"github.com/jackc/pgx/v5"

	"github.com/growdu/doctors/services/order/internal/events"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/state"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/lock"
)

// RefundService 是 order 包定义的接口（避免反向依赖 payment/refund）。
// 实现方（refund.Service）负责按策略计算金额 + 落 refunds 表 + 发 PaymentRefundedEvent。
type RefundService interface {
	Refund(ctx context.Context, orderID int64, reason string) (*contracts.RefundResult, error)
}

// TxRunner 是事务抽象；业务层只关心 fn(tx) 是否能跑。
// v1 旧 Accept 流程用；v1.1 仍保留（cancel / refund 可能用）。
type TxRunner interface {
	WithTx(ctx context.Context, fn func(pgx.Tx) error) error
}

// OrderRepo 是仓储最小契约。
//
// v1.1（order-matching-redesign）：
//   - 删除 LockForAccept / ReleaseLock / LockExpired 3 方法（抢单锁单）
//   - 新增 SelectForEscort / ConfirmByEscort / RejectByEscort / PendingExpired 4 方法
//   - 新增 ListByEscort（v1.1 escort-order-ext：陪诊师查询「我的邀请」或「我的订单」）
type OrderRepo interface {
	Create(ctx context.Context, o *repo.Order) error
	FindByID(ctx context.Context, id int64) (*repo.Order, error)
	ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*repo.Order, error)
	ListByEscort(ctx context.Context, escortID int64, statusFilter string, limit, offset int) ([]*repo.Order, error)
	UpdateStatus(ctx context.Context, id int64, to string, expectVersion int, escortID *int64) error
	InsertEvent(ctx context.Context, orderID int64, from *string, to string, actorID *int64, payload []byte) error
	ListEvents(ctx context.Context, orderID int64) ([]*repo.OrderEvent, error)
	// 状态机 v1.1: 选人 + 30s 确认窗口四件套
	SelectForEscort(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error
	ConfirmByEscort(ctx context.Context, id int64, escortID int64, now time.Time, expectVersion int) error
	RejectByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error
	PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error)
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

// CandidatesLookup 是 v1.1 新增：查询某订单的候选陪诊师列表（SelectEscort 业务校验用）。
// 实现可由 match-service gRPC stub / Redis 缓存 / in-memory 提供；service 不依赖具体实现。
type CandidatesLookup interface {
	IsInCandidates(ctx context.Context, orderID, escortID int64) (bool, error)
}

// AvailabilityLookup 是 v1.1 新增：查询陪诊师在指定时间是否有可用时段（SelectEscort 业务校验用）。
// 实现由 escort-service gRPC stub 或 escort-client 包提供。
type AvailabilityLookup interface {
	HasAvailabilityFor(ctx context.Context, escortID int64, serviceStartAt any) (bool, error)
}

// Service 是 order 业务编排器。
type Service struct {
	orders      OrderRepo
	users       UserLookup
	txRunner    TxRunner       // 可选；v1 旧 Accept 用
	publisher   events.Publisher // 可选；nil 时不发布事件
	clockNow    func() time.Time // 用于测试注入时间
	locker      lock.Locker     // 可选；v1.1 保留兼容
	refund      RefundService   // 可选；nil = 不触发退款（v1 默认）
	candidates  CandidatesLookup  // v1.1 可选；nil = 跳过候选校验
	availability AvailabilityLookup // v1.1 可选；nil = 跳过时段校验
}

// New 装配一个 Service。
func New(orders OrderRepo, users UserLookup) *Service {
	return &Service{
		orders:   orders,
		users:    users,
		clockNow: time.Now,
		locker:   lock.NopLocker{}, // 默认 NopLocker，DB 兜底
	}
}

// WithTx 注入事务执行器（v1 旧 Accept 用；v1.1 仅 cancel/refund 可能用）。
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

// WithLocker 注入分布式锁（v1.1 保留兼容，不再用于锁单；v2 可用于其他场景）。
func (s *Service) WithLocker(l lock.Locker) *Service {
	if l == nil {
		l = lock.NopLocker{}
	}
	s.locker = l
	return s
}

// WithRefundService 注入退款服务（refund plan：触发 refund.Service.Refund）。
func (s *Service) WithRefundService(r RefundService) *Service {
	s.refund = r
	return s
}

// WithCandidatesLookup 注入候选陪诊师查询器（v1.1 SelectEscort 校验用）。
func (s *Service) WithCandidatesLookup(c CandidatesLookup) *Service {
	s.candidates = c
	return s
}

// WithAvailabilityLookup 注入陪诊师时段查询器（v1.1 SelectEscort 校验用）。
func (s *Service) WithAvailabilityLookup(a AvailabilityLookup) *Service {
	s.availability = a
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

// ListForEscort 返回某 escort 的订单分页（v1.1 escort-order-ext plan）。
//   - statusFilter 为 "" 时返回该 escort 名下所有订单（escort_id = $1）
//   - statusFilter == "invitations" 时返回 selected_escort_id = $1 AND status = 'escort_pending_acceptance'
//     （覆盖两种语义：已接受订单 vs 待确认邀请；前端可分 tab 展示）
func (s *Service) ListForEscort(ctx context.Context, escortID int64, statusFilter string, limit, offset int) ([]*repo.Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.orders.ListByEscort(ctx, escortID, statusFilter, limit, offset)
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

	// 触发退款（best-effort；失败仅 log，不阻塞主流程）
	if s.refund != nil {
		refundReason := "user_cancel"
		if actorID > 0 && o.PatientID != actorID {
			refundReason = "admin_cancel"
		}
		if _, err := s.refund.Refund(ctx, orderID, refundReason); err != nil {
			// log 但不返回 error；Cancel 已成功（DB 已写），refund 失败可走对账补单
			_ = err
		}
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