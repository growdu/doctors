// Package service - 患者选陪选单 + 陪诊师 30s 确认核心
//
// 设计要点（v1.1 order-matching-redesign）：
//   - 删旧 Accept / TryLock / ReleaseAcceptLock / 旧 ConfirmAccept（抢单锁单模式）
//   - 加 3 新方法：SelectEscort / ConfirmAccept / RejectAccept（选人模式）
//   - 流程：患者 SelectEscort → 状态 escort_pending_acceptance + 30s expire
//   - 陪诊师 ConfirmAccept → 状态 accepted + 写 escort_id
//   - 陪诊师 RejectAccept 或 30s 超时 → 状态回退 selecting_escort + 发 OrderEscortRejectedEvent
//   - 业务层 interface 隔离（OrderRepo / Publisher / Locker），handler 只做绑定。
//
// 兼容与迁移：
//   - v1 抢单流程对应的 orders.lock_owner / lock_expire_at 字段已由 0009 migration 删除；
//     DB 数据若无 lock_owner 残留则旧流程会无声失败（v1 数据迁移另行排期）。
//   - state.StatusPendingAcceptance 仍保留在枚举内，旧 v1 订单可继续走完流程；不删旧常量。

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/state"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// ErrLockTaken 复用旧 accept.go 的哨兵；service / handler 层做 errors.Is 匹配。
var ErrLockTaken = errors.New("service: order lock taken by another escort")

// SelectEscortResult 是 SelectEscort 成功时的返回值。
type SelectEscortResult struct {
	OrderID               int64
	Version               int
	SelectedEscortID      int64
	EscortPendingExpireAt time.Time
}

// SelectEscort 患者选 1 位陪诊师：
//   - 校验 order 存在 + patientID 是 owner
//   - 校验 status = selecting_escort
//   - 校验 escortID 在候选列表（由 caller 通过 CandidatesLookup 注入；service 不依赖 match-service）
//   - 校验 escort 在 service_start_at 时段内有 availability（由 AvailabilityLookup 注入）
//   - 写 selected_escort_id + escort_pending_expire_at = now+30s + 切 status = escort_pending_acceptance
//   - 发 OrderSelectingEscortEvent（best-effort）
//
// CandidatesLookup / AvailabilityLookup 是 v1.1 新增接口；v1 实现可能为 in-memory stub。
func (s *Service) SelectEscort(ctx context.Context, orderID, patientID, escortID int64) (*SelectEscortResult, error) {
	if orderID == 0 || patientID == 0 || escortID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id / patient_id / escort_id required")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repo.ErrOrderNotFound) {
			return nil, errs.New(errs.CodeNotFound, "order not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if o.PatientID != patientID {
		return nil, errs.New(errs.CodeForbidden, "not order owner")
	}
	if state.Status(o.Status) != state.StatusSelectingEscort {
		return nil, errs.New(errs.CodeConflict, fmt.Sprintf("order not in selecting_escort (current %s)", o.Status))
	}

	// 业务校验：escort 在候选列表
	if s.candidates != nil {
		ok, err := s.candidates.IsInCandidates(ctx, orderID, escortID)
		if err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "check candidates", err)
		}
		if !ok {
			return nil, errs.New(errs.CodeUnprocessable, "escort not in candidates")
		}
	}
	// 业务校验：escort 在 service_start_at 时段内有 availability
	if s.availability != nil {
		ok, err := s.availability.HasAvailabilityFor(ctx, escortID, o.ServiceStartAt)
		if err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "check availability", err)
		}
		if !ok {
			return nil, errs.New(errs.CodeUnprocessable, "escort has no availability for service time")
		}
	}

	expireAt := s.clockNow().Add(30 * time.Second)
	if err := s.orders.SelectForEscort(ctx, orderID, escortID, expireAt, o.Version); err != nil {
		if errors.Is(err, repo.ErrVersionConflict) {
			return nil, errs.New(errs.CodeConflict, "version conflict; please retry")
		}
		if errors.Is(err, repo.ErrInvalidStateForSelect) {
			return nil, errs.New(errs.CodeConflict, "order not in selecting_escort")
		}
		return nil, errs.Wrap(errs.CodeInternal, "select for escort", err)
	}

	// 写 order_event
	from := string(state.StatusSelectingEscort)
	to := string(state.StatusEscortPendingAcceptance)
	actor := patientID
	if err := s.orders.InsertEvent(ctx, orderID, &from, to, &actor, nil); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "insert event", err)
	}

	// 发 OrderEscortSelectedEvent（best-effort）
	if s.publisher != nil {
		_ = s.publisher.PublishOrderEscortSelected(ctx, contracts.OrderEscortSelectedEvent{
			OrderID:               orderID,
			PatientID:             patientID,
			SelectedEscortID:      escortID,
			EscortPendingExpireAt: expireAt,
			OccurredAt:            s.clockNow(),
		})
	}
	return &SelectEscortResult{
		OrderID:               orderID,
		Version:               o.Version + 1,
		SelectedEscortID:      escortID,
		EscortPendingExpireAt: expireAt,
	}, nil
}

// ConfirmAccept 陪诊师在 30s 确认窗口内 confirm → 切到 accepted + 写 escort_id。
// 校验：
//   - status = escort_pending_acceptance
//   - selected_escort_id = escortID（防止误确认）
//   - escort_pending_expire_at > now()（未超时）
//   - 发 OrderEscortConfirmedEvent（best-effort）
func (s *Service) ConfirmAccept(ctx context.Context, orderID, escortID int64) (*repo.Order, error) {
	if orderID == 0 || escortID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if state.Status(o.Status) != state.StatusEscortPendingAcceptance {
		return nil, errs.New(errs.CodeConflict, "order not in escort_pending_acceptance")
	}
	if o.SelectedEscortID == nil || *o.SelectedEscortID != escortID {
		return nil, errs.New(errs.CodeForbidden, "not the selected escort")
	}
	if o.EscortPendingExpireAt != nil && !o.EscortPendingExpireAt.After(s.clockNow()) {
		return nil, errs.New(errs.CodeGone, "escort invitation expired")
	}
	if err := s.orders.ConfirmByEscort(ctx, orderID, escortID, s.clockNow(), o.Version); err != nil {
		switch {
		case errors.Is(err, repo.ErrVersionConflict):
			return nil, errs.New(errs.CodeConflict, "version conflict; please retry")
		case errors.Is(err, repo.ErrInvalidStateForConfirm):
			return nil, errs.New(errs.CodeConflict, "order not in escort_pending_acceptance")
		case errors.Is(err, repo.ErrSelectedEscortMismatch):
			return nil, errs.New(errs.CodeForbidden, "not the selected escort")
		case errors.Is(err, repo.ErrInvitationExpired):
			return nil, errs.New(errs.CodeGone, "escort invitation expired")
		default:
			return nil, errs.Wrap(errs.CodeInternal, "confirm by escort", err)
		}
	}

	// 写 order_event
	from := string(state.StatusEscortPendingAcceptance)
	to := string(state.StatusAccepted)
	actor := escortID
	if err := s.orders.InsertEvent(ctx, orderID, &from, to, &actor, nil); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "insert event", err)
	}

	// 发 OrderEscortConfirmedEvent（best-effort）
	if s.publisher != nil {
		_ = s.publisher.PublishOrderEscortConfirmed(ctx, contracts.OrderEscortConfirmedEvent{
			OrderID:     orderID,
			PatientID:   o.PatientID,
			EscortID:    escortID,
			ConfirmedAt: s.clockNow(),
		})
	}
	return s.orders.FindByID(ctx, orderID)
}

// RejectAccept 陪诊师拒接或 scheduler 调用 30s 超时回退 → 状态 selecting_escort。
// reason: "escort_declined" | "lock_expired"
// 发 OrderEscortRejectedEvent（best-effort）。
func (s *Service) RejectAccept(ctx context.Context, orderID, escortID int64, reason string) error {
	if orderID == 0 || escortID == 0 {
		return errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
	}
	if reason != "escort_declined" && reason != "lock_expired" {
		return errs.New(errs.CodeParamInvalid, "reason must be escort_declined or lock_expired")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if state.Status(o.Status) != state.StatusEscortPendingAcceptance {
		return errs.New(errs.CodeConflict, "order not in escort_pending_acceptance")
	}
	if err := s.orders.RejectByEscort(ctx, orderID, escortID, o.Version); err != nil {
		if errors.Is(err, repo.ErrVersionConflict) {
			return errs.New(errs.CodeConflict, "version conflict; please retry")
		}
		if errors.Is(err, repo.ErrInvalidStateForReject) {
			return errs.New(errs.CodeConflict, "order not in escort_pending_acceptance")
		}
		return errs.Wrap(errs.CodeInternal, "reject by escort", err)
	}

	// 写 order_event
	from := string(state.StatusEscortPendingAcceptance)
	to := string(state.StatusSelectingEscort)
	actor := escortID
	if err := s.orders.InsertEvent(ctx, orderID, &from, to, &actor, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert event", err)
	}

	// 发 OrderEscortRejectedEvent（best-effort）
	if s.publisher != nil {
		_ = s.publisher.PublishOrderEscortRejected(ctx, contracts.OrderEscortRejectedEvent{
			OrderID:    orderID,
			PatientID:  o.PatientID,
			EscortID:   escortID,
			Reason:     reason,
			OccurredAt: s.clockNow(),
		})
	}
	return nil
}