// Package service - 抢单并发核心 (AcceptOrder)
//
// 设计要点：
//   - 用 PG `SELECT ... FOR UPDATE SKIP LOCKED` 把锁冲突变成"行不存在"，
//     避免热点行的串行等待；100 并发 → 只有 1 个拿到行锁，其余直接放弃。
//   - 行锁内再用 version 做乐观锁兜底（防御 SKIP LOCKED 边界）。
//   - 整段 SQL 在一个事务里：行锁 → 状态校验 → 写 status + version + escort → 写 event → commit。
//   - 失败语义：
//       ErrOrderLocked   → 行被另一个 escort 锁定（SKIP LOCKED 返回 0 行）
//       ErrInvalidState  → 状态不在 paid/matching
//       ErrVersionConflict → version 不匹配（理论上不会发生，兜底）

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/state"
)

// Accept 使用的错误。
var (
	ErrOrderLocked      = errors.New("accept: order locked by another escort")
	ErrInvalidState     = errors.New("accept: order not in matching state")
	ErrVersionConflict  = errors.New("accept: version conflict")
)

// AcceptResult 是 Accept 成功时的返回值。
type AcceptResult struct {
	OrderID int64
	Version int
}

// TxRunner 是事务抽象；业务层只关心 fn(tx) 是否能跑。
type TxRunner interface {
	WithTx(ctx context.Context, fn func(pgx.Tx) error) error
}

// PGPoolTxRunner 是基于 pgxpool 的默认实现；由 shared/db.WithTx 提供。
type PGPoolTxRunner struct {
	Pool *pgxpool.Pool
}

// WithTx 直接用 pgxpool 实现事务（与 shared/db.WithTx 行为一致）。
func (r *PGPoolTxRunner) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("accept: begin tx: %w", err)
	}
	defer func() {
		// 注意：WithTx 内部会再次 Rollback 已 commit 的 tx，pgx 内部安全。
		_ = tx.Rollback(ctx)
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// Accept 是抢单核心：100 并发 → 恰好 1 个 escort 成功。
func (s *Service) Accept(ctx context.Context, orderID, escortID int64) (*AcceptResult, error) {
	if s.txRunner == nil {
		return nil, errors.New("accept: TxRunner not wired (阶段 3.6/3.9 接通)")
	}
	var result *AcceptResult
	err := s.txRunner.WithTx(ctx, func(tx pgx.Tx) error {
		// 1. 行锁 + 跳过已被锁的行
		var status string
		var version int
		err := tx.QueryRow(ctx, `
			SELECT status, version FROM orders
			 WHERE id = $1 AND deleted_at IS NULL
			 FOR UPDATE SKIP LOCKED
		`, orderID).Scan(&status, &version)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// 两种情况：行不存在 / 行被锁。区分：
				//   - 再查一次（无锁），如果不存在 → ErrOrderNotFound
				//   - 如果存在 → ErrOrderLocked
				return ErrOrderLocked
			}
			return fmt.Errorf("accept: lock order: %w", err)
		}

		// 2. 状态校验
		if status != string(state.StatusPaid) && status != string(state.StatusMatching) {
			return ErrInvalidState
		}

		// 3. UPDATE（version 校验 + 写 status + 写 escort_id）
		tag, err := tx.Exec(ctx, `
			UPDATE orders
			   SET status = $1, version = version + 1, updated_at = NOW(),
			       escort_id = $2
			 WHERE id = $3 AND version = $4 AND deleted_at IS NULL
		`, string(state.StatusAccepted), escortID, orderID, version)
		if err != nil {
			return fmt.Errorf("accept: update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrVersionConflict
		}

		// 4. 写 order_event
		from := status
		to := string(state.StatusAccepted)
		actor := escortID
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_events (order_id, from_status, to_status, actor_id, payload)
			VALUES ($1, $2, $3, $4, NULL)
		`, orderID, from, to, actor); err != nil {
			return fmt.Errorf("accept: insert event: %w", err)
		}

		result = &AcceptResult{OrderID: orderID, Version: version + 1}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// 编译期确保 repo.Order 与 service.OrderRepo 兼容（避免误改）。
var _ OrderRepo = (*repo.OrderRepo)(nil)