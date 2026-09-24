package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCanTransition_Legal 表驱动覆盖所有合法转换。
func TestCanTransition_Legal(t *testing.T) {
	cases := []struct{ from, to Status }{
		{StatusCreated, StatusPaid},
		{StatusCreated, StatusCanceled},
		{StatusPaid, StatusMatching},
		{StatusPaid, StatusCanceled},
		{StatusMatching, StatusAccepted},
		{StatusMatching, StatusCanceled},
		{StatusAccepted, StatusInService},
		{StatusAccepted, StatusCanceled},
		{StatusAccepted, StatusMatching},
		{StatusInService, StatusCompleted},
		{StatusCompleted, StatusReviewed},
		{StatusCompleted, StatusRefunding},
		{StatusReviewed, StatusClosed},
		{StatusRefunding, StatusRefunded},
	}
	for _, c := range cases {
		t.Run(string(c.from)+"->"+string(c.to), func(t *testing.T) {
			assert.True(t, CanTransition(c.from, c.to))
		})
	}
}

// TestCanTransition_Illegal 表驱动覆盖非法转换。
func TestCanTransition_Illegal(t *testing.T) {
	cases := []struct{ from, to Status }{
		{StatusCreated, StatusAccepted},     // 跳过 paid/matching
		{StatusPaid, StatusAccepted},        // 跳过 matching
		{StatusCompleted, StatusAccepted},   // 反向
		{StatusClosed, StatusCreated},       // 终态不能再走
		{StatusCanceled, StatusPaid},        // 取消后不能再支付
		{StatusRefunded, StatusRefunding},   // 终态
		{Status("" + "garbage"), StatusPaid}, // 非法 from
	}
	for _, c := range cases {
		t.Run(string(c.from)+"->"+string(c.to), func(t *testing.T) {
			assert.False(t, CanTransition(c.from, c.to))
		})
	}
}

// TestIsTerminal 终态：closed / canceled 不再有出向边。
func TestIsTerminal(t *testing.T) {
	assert.True(t, IsTerminal(StatusClosed))
	assert.True(t, IsTerminal(StatusCanceled))
	assert.False(t, IsTerminal(StatusPaid))
	assert.False(t, IsTerminal(StatusAccepted))
}

// TestIsValid 校验字符串是否在状态枚举内。
func TestIsValid(t *testing.T) {
	assert.True(t, IsValid(StatusCreated))
	assert.True(t, IsValid(StatusRefunded))
	assert.False(t, IsValid(Status("nope")))
	assert.False(t, IsValid(Status("")))
}

// ===== 状态机统一 plan: 新增 pending_acceptance / settling / disputed 三态 =====

// TestStatusPendingAcceptance_Exists 验证新状态在枚举中。
func TestStatusPendingAcceptance_Exists(t *testing.T) {
	assert.True(t, IsValid(StatusPendingAcceptance))
}

// TestCanTransition_Matching_ToPendingAcceptance 验证 matching → pending_acceptance 合法。
func TestCanTransition_Matching_ToPendingAcceptance(t *testing.T) {
	assert.True(t, CanTransition(StatusMatching, StatusPendingAcceptance))
}

// TestCanTransition_PendingAcceptance_ToAccepted 验证陪诊师 30s 内确认 → accepted。
func TestCanTransition_PendingAcceptance_ToAccepted(t *testing.T) {
	assert.True(t, CanTransition(StatusPendingAcceptance, StatusAccepted))
}

// TestCanTransition_PendingAcceptance_ToMatching 验证 30s 超时回退 matching。
func TestCanTransition_PendingAcceptance_ToMatching(t *testing.T) {
	assert.True(t, CanTransition(StatusPendingAcceptance, StatusMatching))
}

// TestCanTransition_Completed_ToSettling 验证结算态。
func TestCanTransition_Completed_ToSettling(t *testing.T) {
	assert.True(t, CanTransition(StatusCompleted, StatusSettling))
}

// TestCanTransition_Disputed_FromAnyActive 验证争议可从活动态转入。
func TestCanTransition_Disputed_FromAnyActive(t *testing.T) {
	for _, s := range []Status{
		StatusAccepted, StatusInService, StatusCompleted,
	} {
		assert.True(t, CanTransition(s, StatusDisputed),
			"%s → disputed 应合法", s)
	}
}

// TestCanTransition_Settling_ToClosed 验证结算后关闭。
func TestCanTransition_Settling_ToClosed(t *testing.T) {
	assert.True(t, CanTransition(StatusSettling, StatusClosed))
}

// TestCanTransition_Refunding_ToSettling 验证退款完成进入结算。
func TestCanTransition_Refunding_ToSettling(t *testing.T) {
	assert.True(t, CanTransition(StatusRefunding, StatusSettling))
}

// ===== 2026-09-24 order-matching-redesign plan: selecting_escort + escort_pending_acceptance 两态 =====
// 抢单→选人重构：paid → selecting_escort（生成候选）→ 患者选 1 位 → escort_pending_acceptance（30s 陪诊师确认窗口）
// → accepted（陪诊师确认）或 回退 selecting_escort（陪诊师拒接 / 超时）。
// 旧的 matching + pending_acceptance 流程保留（兼容），新流程并存。

// TestStatusSelectingEscort_Exists 验证 selecting_escort 在枚举中。
func TestStatusSelectingEscort_Exists(t *testing.T) {
	assert.True(t, IsValid(StatusSelectingEscort))
}

// TestStatusEscortPendingAcceptance_Exists 验证 escort_pending_acceptance 在枚举中。
func TestStatusEscortPendingAcceptance_Exists(t *testing.T) {
	assert.True(t, IsValid(StatusEscortPendingAcceptance))
}

// TestCanTransition_Paid_ToSelectingEscort 验证 paid → selecting_escort（生成候选陪诊师）。
func TestCanTransition_Paid_ToSelectingEscort(t *testing.T) {
	assert.True(t, CanTransition(StatusPaid, StatusSelectingEscort))
}

// TestCanTransition_SelectingEscort_ToEscortPendingAcceptance 验证患者选人后进入陪诊师确认窗口。
func TestCanTransition_SelectingEscort_ToEscortPendingAcceptance(t *testing.T) {
	assert.True(t, CanTransition(StatusSelectingEscort, StatusEscortPendingAcceptance))
}

// TestCanTransition_EscortPendingAcceptance_ToAccepted 验证陪诊师 30s 内 confirm → accepted。
func TestCanTransition_EscortPendingAcceptance_ToAccepted(t *testing.T) {
	assert.True(t, CanTransition(StatusEscortPendingAcceptance, StatusAccepted))
}

// TestCanTransition_EscortPendingAcceptance_ToSelectingEscort 验证陪诊师拒接 / 30s 超时 → 回退 selecting_escort。
func TestCanTransition_EscortPendingAcceptance_ToSelectingEscort(t *testing.T) {
	assert.True(t, CanTransition(StatusEscortPendingAcceptance, StatusSelectingEscort))
}

// TestCanTransition_SelectingEscort_ToCanceled 验证患者在选人阶段取消订单（v1 允许）。
func TestCanTransition_SelectingEscort_ToCanceled(t *testing.T) {
	assert.True(t, CanTransition(StatusSelectingEscort, StatusCanceled))
}

// TestIsValid_NewStates_NotEqualOldOnes 验证新状态与旧状态枚举值不同（防止字符串冲突）。
func TestIsValid_NewStates_NotEqualOldOnes(t *testing.T) {
	assert.NotEqual(t, string(StatusSelectingEscort), string(StatusMatching))
	assert.NotEqual(t, string(StatusEscortPendingAcceptance), string(StatusPendingAcceptance))
	assert.NotEqual(t, string(StatusSelectingEscort), string(StatusPendingAcceptance))
}