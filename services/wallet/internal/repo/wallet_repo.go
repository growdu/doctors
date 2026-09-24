// Package repo 是 wallet-service 的数据访问层。
//
// 设计要点：
//   - pgx 直写 SQL（v1 不引 ORM）；用 shopspring/decimal 防精度漂移。
//   - 所有 wallet 余额变动走原子 UPDATE（同一 SQL 里 balance += / frozen +=），避免并发读改写。
//   - billings 在 UPDATE 成功后 INSERT；二者走同一事务，保证账本与余额一致。
//   - wallet 不反向 FK refund.pairs：退款扣回由 service 监听 PaymentRefundedEvent 触发。
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Wallet 映射 wallets 表行。
type Wallet struct {
	UserID         int64
	Balance        decimal.Decimal
	Frozen         decimal.Decimal
	TotalEarned    decimal.Decimal
	TotalWithdrawn decimal.Decimal
	Currency       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Withdrawal 映射 withdrawals 表行。
type Withdrawal struct {
	ID            int64
	UserID        int64
	Amount        decimal.Decimal
	Channel       string // "wx" | "alipay" | "mock"
	Account       string
	Status        string // "pending" | "approved" | "paid" | "rejected"
	ExternalTxID  string
	FailureReason string
	CreatedAt     time.Time
	ReviewedAt    *time.Time
	ReviewedBy    *int64
	PaidAt        *time.Time
}

// Billing 映射 billings 表行。
type Billing struct {
	ID           int64
	UserID       int64
	OrderID      *int64
	Type         string
	Amount       decimal.Decimal
	BalanceAfter decimal.Decimal
	FrozenAfter  decimal.Decimal
	Note         string
	CreatedAt    time.Time
}

// ErrWalletNotFound 查询无结果时的哨兵。
var ErrWalletNotFound = errors.New("repo: wallet not found")

// ErrWithdrawalNotFound 提现工单未找到。
var ErrWithdrawalNotFound = errors.New("repo: withdrawal not found")

// ErrInsufficientFrozen 冻结余额不足（退款 / 释放时）。
var ErrInsufficientFrozen = errors.New("repo: frozen insufficient")

// ErrInsufficientBalance 可用余额不足（提现时）。
var ErrInsufficientBalance = errors.New("repo: balance insufficient")

// WalletRepo 聚合 wallets + withdrawals + billings 三表仓储。
type WalletRepo struct {
	pool *pgxpool.Pool
}

// NewWalletRepo 构造仓储。
func NewWalletRepo(pool *pgxpool.Pool) *WalletRepo { return &WalletRepo{pool: pool} }

// Pool 暴露底层 pool 给 scanner / consumer 使用。
func (r *WalletRepo) Pool() *pgxpool.Pool { return r.pool }

// ---- wallets ----

// Get 查询单行；无记录返回 (nil, nil)。
func (r *WalletRepo) Get(ctx context.Context, userID int64) (*Wallet, error) {
	const q = `
		SELECT user_id, balance, frozen, total_earned, total_withdrawn,
		       currency, created_at, updated_at
		FROM wallets WHERE user_id = $1`
	row := r.pool.QueryRow(ctx, q, userID)
	w, err := scanWallet(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return w, nil
}

// GetOrCreate 查不到就 INSERT 一行零值。
func (r *WalletRepo) GetOrCreate(ctx context.Context, userID int64) (*Wallet, error) {
	w, err := r.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if w != nil {
		return w, nil
	}
	const ins = `
		INSERT INTO wallets (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING user_id, balance, frozen, total_earned, total_withdrawn,
		          currency, created_at, updated_at`
	row := r.pool.QueryRow(ctx, ins, userID)
	w, err = scanWallet(row)
	if err != nil {
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	return w, nil
}

// FreezeIncome 订单完成入账：frozen += amount；total_earned += amount；写 billings。
func (r *WalletRepo) FreezeIncome(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := r.getOrCreateTx(ctx, tx, userID); err != nil {
		return err
	}

	var balance, frozen decimal.Decimal
	const upd = `
		UPDATE wallets
		   SET frozen = frozen + $2,
		       total_earned = total_earned + $2,
		       updated_at = NOW()
		 WHERE user_id = $1
		 RETURNING balance, frozen`
	if err := tx.QueryRow(ctx, upd, userID, amount).Scan(&balance, &frozen); err != nil {
		return fmt.Errorf("freeze income: %w", err)
	}

	if err := insertBillingTx(ctx, tx, &Billing{
		UserID: userID, OrderID: orderIDPtr(orderID), Type: "order_income",
		Amount: amount, BalanceAfter: balance, FrozenAfter: frozen,
		Note: fmt.Sprintf("order %d completed", orderID),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeductFrozenForRefund 退款扣回 frozen（仅 T+7 窗口内）。
func (r *WalletRepo) DeductFrozenForRefund(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var balance, frozen decimal.Decimal
	const upd = `
		UPDATE wallets
		   SET frozen = frozen - $2,
		       total_earned = total_earned - $2,
		       updated_at = NOW()
		 WHERE user_id = $1 AND frozen >= $2
		 RETURNING balance, frozen`
	if err := tx.QueryRow(ctx, upd, userID, amount).Scan(&balance, &frozen); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: user=%d amount=%s", ErrInsufficientFrozen, userID, amount.String())
		}
		return fmt.Errorf("deduct frozen: %w", err)
	}

	if err := insertBillingTx(ctx, tx, &Billing{
		UserID: userID, OrderID: orderIDPtr(orderID), Type: "refund_deduct",
		Amount: amount.Neg(), BalanceAfter: balance, FrozenAfter: frozen,
		Note: fmt.Sprintf("order %d refund", orderID),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UnfreezeToBalance T+7 释放：frozen -= amount；balance += amount。
func (r *WalletRepo) UnfreezeToBalance(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var balance, frozen decimal.Decimal
	const upd = `
		UPDATE wallets
		   SET frozen = frozen - $2,
		       balance = balance + $2,
		       updated_at = NOW()
		 WHERE user_id = $1 AND frozen >= $2
		 RETURNING balance, frozen`
	if err := tx.QueryRow(ctx, upd, userID, amount).Scan(&balance, &frozen); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: user=%d amount=%s", ErrInsufficientFrozen, userID, amount.String())
		}
		return fmt.Errorf("unfreeze: %w", err)
	}

	if err := insertBillingTx(ctx, tx, &Billing{
		UserID: userID, OrderID: orderIDPtr(orderID), Type: "frozen_release",
		Amount: amount, BalanceAfter: balance, FrozenAfter: frozen,
		Note: fmt.Sprintf("order %d T+7 released", orderID),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---- withdrawals ----

// CreateWithdrawal 创建提现工单 + 扣 balance（不允许 frozen 提现；CHECK 约束确保 balance >= 0）。
func (r *WalletRepo) CreateWithdrawal(ctx context.Context, w *Withdrawal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var balance, frozen decimal.Decimal
	const upd = `
		UPDATE wallets
		   SET balance = balance - $2,
		       total_withdrawn = total_withdrawn + $2,
		       updated_at = NOW()
		 WHERE user_id = $1 AND balance >= $2
		 RETURNING balance, frozen`
	if err := tx.QueryRow(ctx, upd, w.UserID, w.Amount).Scan(&balance, &frozen); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: user=%d amount=%s", ErrInsufficientBalance, w.UserID, w.Amount.String())
		}
		return fmt.Errorf("deduct balance: %w", err)
	}

	const ins = `
		INSERT INTO withdrawals (user_id, amount, channel, account, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	if err := tx.QueryRow(ctx, ins, w.UserID, w.Amount, w.Channel, w.Account, w.Status).
		Scan(&w.ID, &w.CreatedAt); err != nil {
		return fmt.Errorf("insert withdrawal: %w", err)
	}

	if err := insertBillingTx(ctx, tx, &Billing{
		UserID: w.UserID, Type: "withdraw",
		Amount: w.Amount.Neg(), BalanceAfter: balance, FrozenAfter: frozen,
		Note: fmt.Sprintf("withdrawal %d", w.ID),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetWithdrawal 查询提现工单。
func (r *WalletRepo) GetWithdrawal(ctx context.Context, id int64) (*Withdrawal, error) {
	const q = `
		SELECT id, user_id, amount, channel, account, status, external_tx_id,
		       failure_reason, created_at, reviewed_at, reviewed_by, paid_at
		FROM withdrawals WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	w, err := scanWithdrawal(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return w, nil
}

// MarkWithdrawalApproved 把 pending → approved。
func (r *WalletRepo) MarkWithdrawalApproved(ctx context.Context, id, reviewerID int64) error {
	const q = `
		UPDATE withdrawals
		   SET status = 'approved', reviewed_at = NOW(), reviewed_by = $2
		 WHERE id = $1 AND status = 'pending'`
	tag, err := r.pool.Exec(ctx, q, id, reviewerID)
	if err != nil {
		return fmt.Errorf("mark approved: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWithdrawalNotFound
	}
	return nil
}

// MarkWithdrawalPaid 把 approved → paid + 写 external_tx_id + paid_at。
// v1 mock：调用方传 "MOCK-TX-{id}"。
func (r *WalletRepo) MarkWithdrawalPaid(ctx context.Context, id int64, externalTxID string) error {
	const q = `
		UPDATE withdrawals
		   SET status = 'paid', external_tx_id = $2, paid_at = NOW()
		 WHERE id = $1 AND status = 'approved'`
	tag, err := r.pool.Exec(ctx, q, id, externalTxID)
	if err != nil {
		return fmt.Errorf("mark paid: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWithdrawalNotFound
	}
	return nil
}

// RejectWithdrawal 把 pending → rejected；balance 已扣回滚。
func (r *WalletRepo) RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 锁住工单行
	const sel = `
		SELECT id, user_id, amount, channel, account, status, external_tx_id,
		       failure_reason, created_at, reviewed_at, reviewed_by, paid_at
		FROM withdrawals WHERE id = $1 FOR UPDATE`
	row := tx.QueryRow(ctx, sel, id)
	w, err := scanWithdrawal(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWithdrawalNotFound
		}
		return fmt.Errorf("lock withdrawal: %w", err)
	}
	if w.Status != "pending" {
		return fmt.Errorf("withdrawal not pending: status=%s", w.Status)
	}

	const upd = `
		UPDATE withdrawals
		   SET status = 'rejected', reviewed_at = NOW(), reviewed_by = $2,
		       failure_reason = $3
		 WHERE id = $1`
	if _, err := tx.Exec(ctx, upd, id, reviewerID, reason); err != nil {
		return fmt.Errorf("reject withdrawal: %w", err)
	}

	var balance, frozen decimal.Decimal
	const give = `
		UPDATE wallets
		   SET balance = balance + $2,
		       total_withdrawn = total_withdrawn - $2,
		       updated_at = NOW()
		 WHERE user_id = $1
		 RETURNING balance, frozen`
	if err := tx.QueryRow(ctx, give, w.UserID, w.Amount).Scan(&balance, &frozen); err != nil {
		return fmt.Errorf("rollback balance: %w", err)
	}

	if err := insertBillingTx(ctx, tx, &Billing{
		UserID: w.UserID, Type: "withdraw_refund",
		Amount: w.Amount, BalanceAfter: balance, FrozenAfter: frozen,
		Note: fmt.Sprintf("withdrawal %d rejected: %s", id, reason),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListPendingWithdrawals admin 审核队列。
func (r *WalletRepo) ListPendingWithdrawals(ctx context.Context, limit int) ([]*Withdrawal, error) {
	const q = `
		SELECT id, user_id, amount, channel, account, status, external_tx_id,
		       failure_reason, created_at, reviewed_at, reviewed_by, paid_at
		FROM withdrawals WHERE status = 'pending'
		ORDER BY created_at ASC LIMIT $1`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	defer rows.Close()
	out := make([]*Withdrawal, 0, limit)
	for rows.Next() {
		w, err := scanWithdrawal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ---- billings ----

// ListTransactions 账单分页（最新在前）。
func (r *WalletRepo) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*Billing, error) {
	const q = `
		SELECT id, user_id, order_id, type, amount, balance_after, frozen_after,
		       note, created_at
		FROM billings WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tx: %w", err)
	}
	defer rows.Close()
	out := make([]*Billing, 0, limit)
	for rows.Next() {
		b, err := scanBilling(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListCompletedOrdersBefore scanner 查"escort 在 cutoff 之前完成的订单"（已入 frozen 但未释放的）。
// 返回 [order_id] + [amount]（同时返回 amount 避免 scanner 再查一次 orders 表）。
func (r *WalletRepo) ListCompletedOrdersBefore(ctx context.Context, cutoff time.Time, limit int) ([]CompletedOrder, error) {
	const q = `
		SELECT b.order_id, b.amount
		FROM billings b
		JOIN orders o ON o.id = b.order_id
		WHERE b.type = 'order_income'
		  AND b.amount > 0
		  AND o.status = 'completed'
		  AND o.completed_at IS NOT NULL
		  AND o.completed_at <= $1
		  AND NOT EXISTS (
		    SELECT 1 FROM billings b2
		    WHERE b2.order_id = b.order_id AND b2.user_id = b.user_id AND b2.type = 'frozen_release'
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM billings b3
		    WHERE b3.order_id = b.order_id AND b3.user_id = b.user_id AND b3.type = 'refund_deduct'
		  )
		ORDER BY o.completed_at ASC
		LIMIT $2`
	rows, err := r.pool.Query(ctx, q, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list completed: %w", err)
	}
	defer rows.Close()
	out := make([]CompletedOrder, 0, limit)
	for rows.Next() {
		var co CompletedOrder
		var oid *int64
		if err := rows.Scan(&oid, &co.Amount); err != nil {
			return nil, err
		}
		if oid != nil {
			co.OrderID = *oid
		}
		out = append(out, co)
	}
	return out, rows.Err()
}

// CompletedOrder 是 scanner 用的一条已完成订单信息。
type CompletedOrder struct {
	OrderID int64
	UserID  int64
	Amount  decimal.Decimal
}

// ---- internal helpers ----

// orderIDPtr 把 int64 转成 *int64；orderID=0 时返回 nil（让 billings.order_id 为 NULL）。
// billings.order_id 是 nullable FK；scanner / 测试 setup 时 orderID=0 表示"无具体订单"。
func orderIDPtr(orderID int64) *int64 {
	if orderID == 0 {
		return nil
	}
	return &orderID
}

// getOrCreateTx 在已有事务里 GetOrCreate。
func (r *WalletRepo) getOrCreateTx(ctx context.Context, tx pgx.Tx, userID int64) (*Wallet, error) {
	const sel = `
		SELECT user_id, balance, frozen, total_earned, total_withdrawn,
		       currency, created_at, updated_at
		FROM wallets WHERE user_id = $1`
	row := tx.QueryRow(ctx, sel, userID)
	w, err := scanWallet(row)
	if err == nil {
		return w, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	const ins = `
		INSERT INTO wallets (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING user_id, balance, frozen, total_earned, total_withdrawn,
		          currency, created_at, updated_at`
	w, err = scanWallet(tx.QueryRow(ctx, ins, userID))
	if err != nil {
		return nil, fmt.Errorf("create wallet in tx: %w", err)
	}
	return w, nil
}

func insertBillingTx(ctx context.Context, tx pgx.Tx, b *Billing) error {
	const q = `
		INSERT INTO billings (user_id, order_id, type, amount, balance_after, frozen_after, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := tx.Exec(ctx, q,
		b.UserID, b.OrderID, b.Type, b.Amount, b.BalanceAfter, b.FrozenAfter, b.Note)
	if err != nil {
		return fmt.Errorf("insert billing: %w", err)
	}
	return nil
}

func scanWallet(row pgx.Row) (*Wallet, error) {
	w := &Wallet{}
	if err := row.Scan(
		&w.UserID, &w.Balance, &w.Frozen, &w.TotalEarned, &w.TotalWithdrawn,
		&w.Currency, &w.CreatedAt, &w.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return w, nil
}

func scanWithdrawal(row pgx.Row) (*Withdrawal, error) {
	w := &Withdrawal{}
	if err := row.Scan(
		&w.ID, &w.UserID, &w.Amount, &w.Channel, &w.Account, &w.Status,
		&w.ExternalTxID, &w.FailureReason, &w.CreatedAt,
		&w.ReviewedAt, &w.ReviewedBy, &w.PaidAt,
	); err != nil {
		return nil, err
	}
	return w, nil
}

func scanBilling(rows pgx.Rows) (*Billing, error) {
	b := &Billing{}
	if err := rows.Scan(
		&b.ID, &b.UserID, &b.OrderID, &b.Type, &b.Amount,
		&b.BalanceAfter, &b.FrozenAfter, &b.Note, &b.CreatedAt,
	); err != nil {
		return nil, err
	}
	return b, nil
}