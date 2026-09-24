// Package state 实现订单状态机（pure function）。
//
// 设计要点：
//   - Status 用 string 常量，与 DB CHECK 约束对齐。
//   - transitions 是静态白名单；CanTransition 是纯函数，无副作用、易测试。
//   - 状态机变更触发的副作用（写 order_events、发 Kafka）由 service 层负责，
//     本包只回答"这个转换能不能做"。
package state

// Status 是订单状态枚举，与 orders.status CHECK 对齐。
type Status string

const (
	StatusCreated                 Status = "created"
	StatusPaid                    Status = "paid"
	StatusMatching                Status = "matching"
	StatusPendingAcceptance       Status = "pending_acceptance"
	StatusSelectingEscort         Status = "selecting_escort"
	StatusEscortPendingAcceptance Status = "escort_pending_acceptance"
	StatusAccepted                Status = "accepted"
	StatusInService               Status = "in_service"
	StatusCompleted               Status = "completed"
	StatusReviewed                Status = "reviewed"
	StatusRefunding               Status = "refunding"
	StatusRefunded                Status = "refunded"
	StatusSettling                Status = "settling"
	StatusDisputed                Status = "disputed"
	StatusClosed                  Status = "closed"
	StatusCanceled                Status = "canceled"
)

// transitions 是合法转换的白名单。
// key = 当前状态；value = 可去的下一状态。
//
// 2026-09-24 状态机统一 plan：
//   - pending_acceptance：抢单锁单期（30s），超时回退 matching 或陪诊师确认 → accepted（保留旧抢单模式，兼容）。
//   - settling：账单核对中（结算），所有"关闭"订单必经 settling → closed。
//   - disputed：争议中，可从活动态任意时段进入，处理后回流到 completed/refunding/closed。
//
// 2026-09-24 order-matching-redesign plan：
//   - selecting_escort：订单已支付，系统生成候选陪诊师列表，等待患者选择。
//   - escort_pending_acceptance：患者已选 1 位陪诊师，30s 确认窗口；陪诊师 confirm → accepted；拒接 / 超时 → 回退 selecting_escort。
//   - 两新状态与旧 matching + pending_acceptance 并存；order-lock plan 后续将旧流程下线。
var transitions = map[Status][]Status{
	StatusCreated:                 {StatusPaid, StatusCanceled},
	StatusPaid:                    {StatusMatching, StatusSelectingEscort, StatusCanceled},
	StatusMatching:                {StatusPendingAcceptance, StatusAccepted, StatusCanceled},
	StatusPendingAcceptance:       {StatusAccepted, StatusMatching, StatusCanceled}, // 30s 锁单 / 超时回退 / 拒接
	StatusSelectingEscort:           {StatusEscortPendingAcceptance, StatusCanceled},  // 患者选人或取消
	StatusEscortPendingAcceptance: {StatusAccepted, StatusSelectingEscort},          // confirm → accepted；reject/timeout → 回退
	StatusAccepted:                {StatusInService, StatusMatching, StatusCanceled, StatusDisputed},
	StatusInService:               {StatusCompleted, StatusDisputed},
	StatusCompleted:               {StatusReviewed, StatusRefunding, StatusSettling, StatusDisputed},
	StatusReviewed:                {StatusClosed},
	StatusRefunding:               {StatusRefunded, StatusSettling},
	StatusRefunded:                {StatusSettling},
	StatusSettling:                {StatusClosed},
	StatusDisputed:                {StatusCompleted, StatusRefunding, StatusClosed},
	StatusClosed:                  {},
	StatusCanceled:                {},
}

// CanTransition 判定 from → to 是否合法。
func CanTransition(from, to Status) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// IsTerminal 判定该状态是否为终态（不再有出向边）。
func IsTerminal(s Status) bool {
	return len(transitions[s]) == 0
}

// IsValid 检查字符串是否为合法的 Status 值。
func IsValid(s Status) bool {
	_, ok := transitions[s]
	return ok
}