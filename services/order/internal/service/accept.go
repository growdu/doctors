// Package service - 抢单并发核心 (AcceptOrder)
//
// 设计要点：
//   - 2026-09-24 状态机统一 plan：旧的 Accept（单次 SELECT+UPDATE）保留兼容 1 个版本；
//     新流程走 TryLock + ConfirmAccept 两步（中间穿插 §4.2 order-lock plan 的 Redis SETNX）。
//   - 旧的 Accept 用 PG `SELECT ... FOR UPDATE SKIP LOCKED` 把锁冲突变成"行不存在"，
//     避免热点行的串行等待；100 并发 → 只有 1 个拿到行锁，其余直接放弃。
//   - 行锁内再用 version 做乐观锁兜底（防御 SKIP LOCKED 边界）。
//   - 整段 SQL 在一个事务里：行锁 → 状态校验 → 写 status + version + escort → 写 event → commit。
//   - 失败语义：
//       ErrOrderLocked   → 行被另一个 escort 锁定（SKIP LOCKED 返回 0 行）
//       ErrInvalidState  → 状态不在 paid/matching
//       ErrInvalidStateForLock → TryLock 时订单不在 matching（已被别人锁或已 accepted）
//       ErrVersionConflict → version 不匹配（理论上不会发生，兜底）

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/state"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// Accept 使用的错误。
var (
	ErrOrderLocked         = errors.New("accept: order locked by another escort")
	ErrInvalidState        = errors.New("accept: order not in matching state")
	ErrVersionConflict     = errors.New("accept: version conflict")
	// ErrInvalidStateForLock 复用 repo 的同名哨兵，便于 service / repo 调用方做 errors.Is 匹配。
	ErrInvalidStateForLock = repo.ErrInvalidStateForLock
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

// Accept 保留兼容：单次 SELECT+UPDATE；§4.2 order-lock plan 引入 Redis SETNX 后删除。
//
// Deprecated: 新流程走 TryLock + ConfirmAccept；保留 1 个版本兼容。
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

	// 发布 OrderAcceptedEvent（best-effort）
	if s.publisher != nil {
		_ = s.publisher.PublishOrderAccepted(ctx, contracts.OrderAcceptedEvent{
			OrderID:    orderID,
			EscortID:   escortID,
			AcceptedAt: time.Now(),
		})
	}
	return result, nil
}

// 编译期确保 repo.Order 与 service.OrderRepo 兼容（避免误改）。
var _ OrderRepo = (*repo.OrderRepo)(nil)

// ===== 2026-09-24 状态机统一 plan：TryLock / ReleaseAcceptLock / ConfirmAccept =====
//
// 新流程：陪诊师先 TryLock 拿 30s 锁单（§4.2 order-lock plan 会在 TryLock 前后加 Redis SETNX）；
// 锁单窗口内 ConfirmAccept 把订单切到 accepted；超时 / 拒接走 ReleaseAcceptLock 回退 matching。

// TryLock 陪诊师尝试拿锁单（§4.2 order-lock plan 会在此前后加 Redis SETNX）。
//
// 失败语义：
//   - order 不存在 → CodeNotFound
//   - 订单已被另一 escort 锁单（status=pending_acceptance）→ ErrInvalidStateForLock
//   - 订单已 accepted / canceled 等非 matching 状态 → CodeConflict
//   - version 不匹配 → 内部错（repo.ErrVersionConflict）
func (s *Service) TryLock(ctx context.Context, orderID, escortID int64, ttl time.Duration) error {
	if orderID == 0 || escortID == 0 {
		return errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return errs.New(errs.CodeNotFound, "order not found")
		}
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	switch state.Status(o.Status) {
	case state.StatusMatching:
		// 正常路径
	case state.StatusPendingAcceptance:
		// 已被另一 escort 锁单 —— 返回更具体的 ErrInvalidStateForLock
		return ErrInvalidStateForLock
	default:
		return errs.New(errs.CodeConflict, fmt.Sprintf("cannot lock from status %s", o.Status))
	}
	expireAt := s.clockNow().Add(ttl)
	if err := s.orders.LockForAccept(ctx, orderID, escortID, expireAt, o.Version); err != nil {
		// repo 层的版本冲突 / 非法状态透传，便于 errors.Is 判断
		return err
	}
	return nil
}

// ReleaseAcceptLock 拒接 / 超时 → 回退到 matching。
func (s *Service) ReleaseAcceptLock(ctx context.Context, orderID, escortID int64) error {
	if orderID == 0 || escortID == 0 {
		return errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if o.Status != string(state.StatusPendingAcceptance) {
		return errs.New(errs.CodeConflict, "order not in pending_acceptance")
	}
	if o.LockOwner == nil || *o.LockOwner != escortID {
		return errs.New(errs.CodeForbidden, "lock owner mismatch")
	}
	if err := s.orders.ReleaseLock(ctx, orderID, o.Version); err != nil {
		return errs.Wrap(errs.CodeInternal, "release lock", err)
	}
	return nil
}

// ConfirmAccept 陪诊师在锁单窗口内确认 → 改 accepted；escort_id 写入。
func (s *Service) ConfirmAccept(ctx context.Context, orderID, escortID int64) (*repo.Order, error) {
	if orderID == 0 || escortID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if o.Status != string(state.StatusPendingAcceptance) {
		return nil, errs.New(errs.CodeConflict, "order not in pending_acceptance")
	}
	if o.LockOwner == nil || *o.LockOwner != escortID {
		return nil, errs.New(errs.CodeForbidden, "lock owner mismatch")
	}
	if err := s.orders.UpdateStatus(ctx, orderID, string(state.StatusAccepted), o.Version, &escortID); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "update status", err)
	}
	// 写 order_event
	actor := escortID
	from := string(state.StatusPendingAcceptance)
	if err := s.orders.InsertEvent(ctx, orderID, &from, string(state.StatusAccepted), &actor, nil); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "insert event", err)
	}
	return s.orders.FindByID(ctx, orderID)
}