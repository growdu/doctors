// Package refund 实现退款策略计算与单据管理。
//
// 设计要点：
//   - Policy 是纯函数策略；输入 OrderContext → 输出 Decision（百分比 + 资格）。
//   - Phase 表示订单当前所处的退款触发阶段；DetectPhase 按时间窗口推断。
//   - v1 默认四档：未付 100% / 付款 5min 内 100% / 5min~接单前 100% /
//     接单后~服务开始前 95%（5% 陪诊师补偿）/ 服务已开始 0%。
package refund

import (
	"context"
	"errors"
	"time"
)

// Phase 表示订单当前所处的退款触发阶段。
type Phase string

const (
	PhaseBeforePaid    Phase = "before_paid"
	PhaseAfterPaid5Min Phase = "after_paid_5min"
	PhaseAfterAccepted Phase = "after_accepted"
	PhaseInService     Phase = "in_service"
)

// Policy 描述一档退款规则。
type Policy struct {
	TriggerPhase              Phase
	RefundPercent             float64 // 0~100
	EscortCompensationPercent float64 // 0~100
}

// Decision 是策略评估结果。
type Decision struct {
	Eligible           bool
	RefundPercent      float64
	EscortCompensation float64
	Reason             string // "no_policy_for_phase" / ""（eligible 时为空）
}

// OrderContext 是策略评估的输入。
type OrderContext struct {
	Amount         float64
	PaidAt         *time.Time
	AcceptedAt     *time.Time
	ServiceStartAt time.Time
	Now            time.Time
}

// DetectPhase 根据 order context 推断退款阶段。
//
// 规则：
//   - 未付款（PaidAt == nil）→ before_paid
//   - 已付款、未接单 → after_paid_5min（不论 5 分钟内外，统一一档 100%）
//   - 已接单、服务未开始 → after_accepted
//   - 服务已开始 → in_service
func DetectPhase(ctx OrderContext) Phase {
	if ctx.PaidAt == nil {
		return PhaseBeforePaid
	}
	if ctx.AcceptedAt == nil {
		return PhaseAfterPaid5Min
	}
	if ctx.ServiceStartAt.IsZero() || ctx.Now.Before(ctx.ServiceStartAt) {
		return PhaseAfterAccepted
	}
	return PhaseInService
}

// PolicyRepo 是 refund_policies 表的契约。
type PolicyRepo interface {
	GetByScopeAndPhase(ctx context.Context, scope string, phase Phase) (*Policy, error)
}

// Decide 根据 order context + policy repo 计算退款决策。
func Decide(octx OrderContext, repo PolicyRepo, scope string) Decision {
	phase := DetectPhase(octx)
	p, err := repo.GetByScopeAndPhase(context.Background(), scope, phase)
	if err != nil || p == nil {
		return Decision{Eligible: false, Reason: "no_policy_for_phase"}
	}
	return Decision{
		Eligible:           true,
		RefundPercent:      p.RefundPercent,
		EscortCompensation: p.EscortCompensationPercent,
	}
}

// RefundAmount 把 Decision 应用到 OrderContext 上，返回退款金额 + 陪诊师补偿。
func RefundAmount(ctx OrderContext, d Decision) (refund, escort float64) {
	if !d.Eligible {
		return 0, 0
	}
	refund = ctx.Amount * d.RefundPercent / 100.0
	escort = ctx.Amount * d.EscortCompensation / 100.0
	return
}

// ErrInvalidPhase 是 Phase 字符串不合法的错误。
var ErrInvalidPhase = errors.New("refund: invalid phase")