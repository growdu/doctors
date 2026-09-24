package refund

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// OrderSnapshot 是评估退款所需的订单视图（来自 order-service 或本地读）。
type OrderSnapshot struct {
	OrderID        int64
	Amount         float64
	PaidAt         *time.Time
	AcceptedAt     *time.Time
	ServiceStartAt time.Time
	Now            time.Time
}

// RefundRepo 是仓储契约。
type RefundRepo interface {
	Create(ctx context.Context, r *Record) error
	GetByOrderID(ctx context.Context, orderID int64) ([]*Record, error)
	UpdateStatus(ctx context.Context, id int64, status string, externalTxID string, failureReason string) error
}

// Record 是 refunds 表的实体。
type Record struct {
	ID            int64
	OrderID       int64
	Amount        float64
	Reason        string
	Status        string
	ExternalTxID  string
	CreatedAt     time.Time
	CompletedAt   *time.Time
	FailureReason string
}

// Publisher 是事件发布抽象。
type Publisher interface {
	PublishRefundCompleted(ctx context.Context, ev contracts.RefundCompletedEvent) error
}

// OrderSnapshotProvider 由 order-service 实现：从 order_id 拿订单快照。
type OrderSnapshotProvider interface {
	GetSnapshot(ctx context.Context, orderID int64) (*OrderSnapshot, error)
}

// ErrAlreadyRefunded 同订单已有 active 退款时返回。
var ErrAlreadyRefunded = errors.New("refund: order already has a non-rejected refund")

// Service 是 refund 业务编排。
type Service struct {
	orders    OrderSnapshotProvider
	refunds   RefundRepo
	policies  PolicyRepo
	publisher Publisher
	scope     string // e.g. "default" / "city:beijing"
}

// New 装配 Service。scope 空则用 "default"。
func New(o OrderSnapshotProvider, r RefundRepo, p PolicyRepo, pub Publisher, scope string) *Service {
	if scope == "" {
		scope = "default"
	}
	return &Service{orders: o, refunds: r, policies: p, publisher: pub, scope: scope}
}

// Refund 触发一笔退款；返回 RefundResult。
//
//   - not eligible (refund_percent=0) → 创建一条 status='rejected' 的记录，返回 RefundResult；
//     调用方按正常路径继续（不视作错误）。
//   - 同订单已有非 rejected 退款 → ErrAlreadyRefunded（幂等性）。
//   - v1 不接第三方支付 API：approved 的退款记 status=completed（mock tx_id）。
func (s *Service) Refund(ctx context.Context, orderID int64, reason string) (*contracts.RefundResult, error) {
	if orderID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id required")
	}
	snap, err := s.orders.GetSnapshot(ctx, orderID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeNotFound, "order snapshot", err)
	}
	snap.Now = time.Now()

	// 幂等检查：同订单已有非 rejected 退款
	existing, _ := s.refunds.GetByOrderID(ctx, orderID)
	for _, r := range existing {
		if r.Status != "rejected" {
			return nil, ErrAlreadyRefunded
		}
	}

	octx := OrderContext{
		Amount:         snap.Amount,
		PaidAt:         snap.PaidAt,
		AcceptedAt:     snap.AcceptedAt,
		ServiceStartAt: snap.ServiceStartAt,
		Now:            snap.Now,
	}
	d := Decide(octx, s.policies, s.scope)
	rec := &Record{
		OrderID:   orderID,
		Amount:    RefundAmountValue(octx, d),
		Reason:    reason,
		Status:    statusFromDecision(d),
		CreatedAt: snap.Now,
	}
	if err := s.refunds.Create(ctx, rec); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create refund", err)
	}
	if rec.Status == "rejected" {
		return &contracts.RefundResult{
			ID: rec.ID, Amount: rec.Amount, Status: rec.Status,
		}, nil
	}

	// v1 不接第三方支付 API；记为 completed（生产用事件模拟）。
	completedAt := time.Now()
	if err := s.refunds.UpdateStatus(ctx, rec.ID, "completed", "mock-tx-"+fmt.Sprint(rec.ID), ""); err != nil {
		return &contracts.RefundResult{
			ID: rec.ID, Amount: rec.Amount, Status: "failed",
		}, errs.Wrap(errs.CodeInternal, "update status", err)
	}
	rec.Status = "completed"
	rec.ExternalTxID = fmt.Sprintf("mock-tx-%d", rec.ID)
	rec.CompletedAt = &completedAt

	if s.publisher != nil {
		_ = s.publisher.PublishRefundCompleted(ctx, contracts.RefundCompletedEvent{
			RefundID:     rec.ID,
			OrderID:      orderID,
			Amount:       rec.Amount,
			RefundedAt:   *rec.CompletedAt,
			ExternalTxID: rec.ExternalTxID,
		})
	}
	return &contracts.RefundResult{
		ID: rec.ID, Amount: rec.Amount, Status: rec.Status,
	}, nil
}

// statusFromDecision 把 Decision 翻译成 refunds.status。
//
// 决策规则：
//   - 找不到 policy（!Eligible）→ rejected
//   - 找到 policy 但 refund_percent = 0（in_service 期不允许退款）→ rejected
//   - 其它 → created（再走第三方支付完成 completed，v1 mock）
func statusFromDecision(d Decision) string {
	if !d.Eligible || d.RefundPercent == 0 {
		return "rejected"
	}
	return "created"
}

// RefundAmountValue 是 RefundAmount 的简单包装（仅退款金额）。
func RefundAmountValue(ctx OrderContext, d Decision) float64 {
	refund, _ := RefundAmount(ctx, d)
	return refund
}