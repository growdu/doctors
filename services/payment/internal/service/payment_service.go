// Package service - payment-service 业务编排。
//
// 设计要点：
//   - v1 仅做 MockChannel：支付单创建/完成/退款全部走内存；不接真实渠道。
//   - 真实渠道（微信支付 V3 / 支付宝）在 v2 接入；本包只定义契约。
//   - 支付完成后触发 contracts.PaymentCompletedEvent → order-service 推进订单到 paid。
//   - 状态机：created → completed / failed / refunded。
package service

import (
	"context"
	"errors"
	"time"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// Payment 是支付单。
type Payment struct {
	ID           int64
	OrderID      int64
	Amount       float64
	Channel      string // "mock" | "wx" | "alipay"
	ExternalTxID string
	Status       string // "created" | "completed" | "failed" | "refunded"
	CreatedAt    time.Time
	CompletedAt  *time.Time
	RefundedAt   *time.Time
}

// Repo 是仓储契约。
type Repo interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id int64) (*Payment, error)
	GetByOrderID(ctx context.Context, orderID int64) (*Payment, error)
	UpdateStatus(ctx context.Context, id int64, status string, completedAt, refundedAt *time.Time, externalTxID string) error
}

// Publisher 是事件发布抽象。
type Publisher interface {
	PublishPaymentCompleted(ctx context.Context, ev contracts.PaymentCompletedEvent) error
	PublishPaymentRefunded(ctx context.Context, ev contracts.PaymentRefundedEvent) error
}

// Channel 是支付渠道抽象；mock 是默认实现。
type Channel interface {
	CreateOutTradeNo(ctx context.Context, p *Payment) (string, error)
}

// Service 是 payment 业务编排器。
type Service struct {
	repo      Repo
	publisher Publisher
	channel   Channel
}

// New 装配 Service。
func New(r Repo, p Publisher, c Channel) *Service {
	return &Service{repo: r, publisher: p, channel: c}
}

// Create 为某订单创建一笔支付单。
func (s *Service) Create(ctx context.Context, orderID int64, amount float64) (*Payment, error) {
	if orderID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id required")
	}
	if amount <= 0 {
		return nil, errs.New(errs.CodeParamInvalid, "amount must be > 0")
	}
	if existing, _ := s.repo.GetByOrderID(ctx, orderID); existing != nil {
		if existing.Status != "failed" {
			return nil, errs.New(errs.CodeConflict, "payment already exists for order")
		}
	}
	external, _ := s.channel.CreateOutTradeNo(ctx, &Payment{OrderID: orderID, Amount: amount})
	p := &Payment{
		OrderID:      orderID,
		Amount:       amount,
		Channel:      "mock",
		ExternalTxID: external,
		Status:       "created",
		CreatedAt:    time.Now(),
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create payment", err)
	}
	return p, nil
}

// Complete 标记支付完成（mock 渠道直接调用）。
func (s *Service) Complete(ctx context.Context, paymentID int64, externalTxID string) (*Payment, error) {
	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			return nil, errs.New(errs.CodeNotFound, "payment not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find payment", err)
	}
	if p.Status == "completed" {
		return p, nil // idempotent
	}
	if p.Status != "created" {
		return nil, errs.New(errs.CodeConflict, "payment not in created state")
	}
	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, paymentID, "completed", &now, nil, externalTxID); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "update status", err)
	}
	p.Status = "completed"
	p.CompletedAt = &now
	if externalTxID != "" {
		p.ExternalTxID = externalTxID
	}

	if s.publisher != nil {
		_ = s.publisher.PublishPaymentCompleted(ctx, contracts.PaymentCompletedEvent{
			PaymentID:    paymentID,
			OrderID:      p.OrderID,
			Amount:       p.Amount,
			CompletedAt:  now,
			ExternalTxID: p.ExternalTxID,
		})
	}
	return p, nil
}

// Refund 发起退款。
func (s *Service) Refund(ctx context.Context, paymentID int64) error {
	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			return errs.New(errs.CodeNotFound, "payment not found")
		}
		return errs.Wrap(errs.CodeInternal, "find payment", err)
	}
	if p.Status != "completed" {
		return errs.New(errs.CodeConflict, "only completed payments can be refunded")
	}
	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, paymentID, "refunded", nil, &now, ""); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	if s.publisher != nil {
		_ = s.publisher.PublishPaymentRefunded(ctx, contracts.PaymentRefundedEvent{
			RefundID:    paymentID,
			PaymentID:   paymentID,
			OrderID:     p.OrderID,
			Amount:      p.Amount,
			RefundedAt:  now,
		})
	}
	return nil
}

// GetByOrder 取订单的支付单。
func (s *Service) GetByOrder(ctx context.Context, orderID int64) (*Payment, error) {
	p, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			return nil, errs.New(errs.CodeNotFound, "payment not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find payment", err)
	}
	return p, nil
}

// ErrPaymentNotFound 查无结果。
var ErrPaymentNotFound = errors.New("payment service: not found")