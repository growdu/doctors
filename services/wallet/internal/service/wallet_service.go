// Package service 是 wallet-service 的业务编排层。
//
// 设计要点：
//   - 所有依赖走接口（Repo / OrderAmountLookup）；fake 替身在 _test.go。
//   - 提现最低 100 元（MinWithdrawalCents）。
//   - 提现 mock 打款：MarkWithdrawalPaid 写 external_tx_id='MOCK-TX-{id}'。
//   - 退款扣 frozen（v1 假设退款只发生在 T+7 窗口内）；窗口外的退款 v2 处理。
//   - service 暴露 OnOrderCompleted / OnRefund 给 cmd 里 Kafka consumer 用；其他业务给 handler 用。
package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/growdu/doctors/services/wallet/internal/repo"
	"github.com/growdu/doctors/shared/errs"
)

// MinWithdrawalCents 最低提现金额（单位：分）。100 元 = 10000 分。
const MinWithdrawalCents = 10000

// Repo 是 wallet-service 的仓储接口（与 repo.WalletRepo 一致；解耦）。
type Repo interface {
	Get(ctx context.Context, userID int64) (*repo.Wallet, error)
	GetOrCreate(ctx context.Context, userID int64) (*repo.Wallet, error)
	FreezeIncome(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error
	DeductFrozenForRefund(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error
	UnfreezeToBalance(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error
	CreateWithdrawal(ctx context.Context, w *repo.Withdrawal) error
	GetWithdrawal(ctx context.Context, id int64) (*repo.Withdrawal, error)
	MarkWithdrawalApproved(ctx context.Context, id, reviewerID int64) error
	MarkWithdrawalPaid(ctx context.Context, id int64, externalTxID string) error
	RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error
	ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*repo.Billing, error)
	ListPendingWithdrawals(ctx context.Context, limit int) ([]*repo.Withdrawal, error)
}

// Service 业务编排器。
type Service struct {
	repo Repo
}

// New 构造 service。
func New(r Repo) *Service { return &Service{repo: r} }

// ---- 用户视角 API（handler 调用） ----

// GetWallet 查询或建零值钱包。
func (s *Service) GetWallet(ctx context.Context, userID int64) (*repo.Wallet, error) {
	w, err := s.repo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "get wallet", err)
	}
	return w, nil
}

// CreateWithdrawal 陪诊师发起提现申请。
// amount 单位：元（与 DB NUMERIC(10,2) 一致；handler 把入参字符串解析成 decimal）。
// 最低金额 100 元（MinWithdrawalCents = 10000 分；amount.Cmp(10000/100) 即 100 元）。
func (s *Service) CreateWithdrawal(ctx context.Context, userID int64, amount decimal.Decimal, channel, account string) (*repo.Withdrawal, error) {
	if amount.Cmp(decimal.NewFromInt(MinWithdrawalCents/100)) < 0 {
		return nil, errs.New(errs.CodeParamInvalid, fmt.Sprintf("最低提现金额 %d 元", MinWithdrawalCents/100))
	}
	if channel != "wx" && channel != "alipay" {
		return nil, errs.New(errs.CodeParamInvalid, "channel must be wx|alipay")
	}
	w := &repo.Withdrawal{
		UserID:  userID,
		Amount:  amount,
		Channel: channel,
		Account: account,
		Status:  "pending",
	}
	if err := s.repo.CreateWithdrawal(ctx, w); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create withdrawal", err)
	}
	return w, nil
}

// ApproveWithdrawal admin 审核通过。
func (s *Service) ApproveWithdrawal(ctx context.Context, id, reviewerID int64) error {
	if err := s.repo.MarkWithdrawalApproved(ctx, id, reviewerID); err != nil {
		return errs.Wrap(errs.CodeInternal, "approve withdrawal", err)
	}
	return nil
}

// MarkWithdrawalPaid admin 标记已打款（v1 mock：直接 success）。
func (s *Service) MarkWithdrawalPaid(ctx context.Context, id int64) error {
	externalTxID := fmt.Sprintf("MOCK-TX-%d", id)
	if err := s.repo.MarkWithdrawalPaid(ctx, id, externalTxID); err != nil {
		return errs.Wrap(errs.CodeInternal, "mark paid", err)
	}
	return nil
}

// RejectWithdrawal admin 审核拒绝。
func (s *Service) RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error {
	if err := s.repo.RejectWithdrawal(ctx, id, reviewerID, reason); err != nil {
		return errs.Wrap(errs.CodeInternal, "reject withdrawal", err)
	}
	return nil
}

// ListTransactions 账单分页。
func (s *Service) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*repo.Billing, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	txs, err := s.repo.ListTransactions(ctx, userID, limit, offset)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list transactions", err)
	}
	return txs, nil
}

// ---- 事件入口（Kafka consumer / scanner 调用） ----

// OnOrderCompleted 订单完成事件 → 入 frozen（写 billings）。
// 订单没有 escort（escort_id=0）的异常单跳过；金额 ≤ 0 跳过。
func (s *Service) OnOrderCompleted(ctx context.Context, escortID, orderID int64, amount decimal.Decimal) error {
	if escortID == 0 || amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	if err := s.repo.FreezeIncome(ctx, escortID, orderID, amount); err != nil {
		return fmt.Errorf("freeze income: %w", err)
	}
	return nil
}

// OnRefund 退款事件 → 扣 frozen（仅 T+7 窗口内）。
func (s *Service) OnRefund(ctx context.Context, escortID, orderID int64, amount decimal.Decimal) error {
	if escortID == 0 || amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	if err := s.repo.DeductFrozenForRefund(ctx, escortID, orderID, amount); err != nil {
		return fmt.Errorf("deduct frozen: %w", err)
	}
	return nil
}