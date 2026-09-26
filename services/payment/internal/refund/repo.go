// Package refund - 第 4 章退款仓储（v1 in-memory 简化实现）。
//
// 设计要点：
//   - v1 不接真实 PG：内存里维护 refunds + refund_policies 两张"表"。
//   - repo 用 sync.RWMutex 保护；ID 由 atomic int64 自增。
//   - 重启清空（与 message in-memory 一致；v2 应替换为 pgx 直写）。
//   - 之所以保留 NewRepo(pool) 签名：
//   - 1）符合 §31 "11 服务真实 PG 接入" 接口约定（pool 参数透传，便于 v2 替换）；
//   - 2）调用方现在传 nil 也能跑；未来接 DB 时只换实现，不改 main。
//
// refund_policies：v1 默认 4 档（按 Phase 决定百分比），与 shared/contracts 文档对齐。
package refund

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// PolicyRepo 是 refund_policies 表的契约（policy.go 也有同名接口，这里是它的实现）。
//
//	内存版；DB 版留给 v2。
type MemoryRepo struct {
	mu           sync.RWMutex
	refunds      map[int64]*Record
	byOrder      map[int64][]int64
	policies     map[scopePhase]*Policy
	nextRefundID int64
}

// scopePhase 是 refund_policies 的复合主键（scope, trigger_phase）。
type scopePhase struct {
	scope string
	phase Phase
}

// NewRepo 构造内存版 refund 仓储。
//
//	pool 参数保留以兼容 §31 接口约定；当前实现不依赖 pgxpool。
//	v2 接入 DB 时此函数签名不变，只需把内部实现换成 pgx 直写即可。
func NewRepo(pool any) *MemoryRepo {
	r := &MemoryRepo{
		refunds:  make(map[int64]*Record),
		byOrder:  make(map[int64][]int64),
		policies: defaultPolicies(),
	}
	return r
}

// defaultPolicies 返回 v1 默认 4 档策略（与 refund/policy_test.go 对齐）。
func defaultPolicies() map[scopePhase]*Policy {
	m := make(map[scopePhase]*Policy)
	def := "default"
	m[scopePhase{def, PhaseBeforePaid}] = &Policy{TriggerPhase: PhaseBeforePaid, RefundPercent: 100, EscortCompensationPercent: 0}
	m[scopePhase{def, PhaseAfterPaid5Min}] = &Policy{TriggerPhase: PhaseAfterPaid5Min, RefundPercent: 100, EscortCompensationPercent: 0}
	m[scopePhase{def, PhaseAfterAccepted}] = &Policy{TriggerPhase: PhaseAfterAccepted, RefundPercent: 95, EscortCompensationPercent: 5}
	// in_service 不允许退款（0%）；保持 present 以便 Decide 返回 not-eligible。
	m[scopePhase{def, PhaseInService}] = &Policy{TriggerPhase: PhaseInService, RefundPercent: 0, EscortCompensationPercent: 0}
	return m
}

// ---------- RefundRepo 实现（Create / GetByOrderID / UpdateStatus） ----------

// Create 写入一笔退款记录；ID 由原子自增分配。
func (r *MemoryRepo) Create(ctx context.Context, rec *Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec.ID = atomic.AddInt64(&r.nextRefundID, 1)
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	r.refunds[rec.ID] = rec
	r.byOrder[rec.OrderID] = append(r.byOrder[rec.OrderID], rec.ID)
	return nil
}

// GetByOrderID 按订单 ID 拉全部退款记录（按时间正序）。
func (r *MemoryRepo) GetByOrderID(ctx context.Context, orderID int64) ([]*Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byOrder[orderID]
	out := make([]*Record, 0, len(ids))
	for _, id := range ids {
		if rec, ok := r.refunds[id]; ok {
			out = append(out, rec)
		}
	}
	return out, nil
}

// UpdateStatus 更新退款状态。
func (r *MemoryRepo) UpdateStatus(ctx context.Context, id int64, status string, externalTxID string, failureReason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.refunds[id]
	if !ok {
		return ErrNotFound
	}
	rec.Status = status
	rec.ExternalTxID = externalTxID
	rec.FailureReason = failureReason
	if status == "completed" {
		now := time.Now()
		rec.CompletedAt = &now
	}
	return nil
}

// ---------- PolicyRepo 实现（GetByScopeAndPhase） ----------

// GetByScopeAndPhase 按 (scope, phase) 取策略；未命中返回 ErrPolicyNotFound。
func (r *MemoryRepo) GetByScopeAndPhase(ctx context.Context, scope string, phase Phase) (*Policy, error) {
	if scope == "" {
		scope = "default"
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.policies[scopePhase{scope, phase}]
	if !ok {
		return nil, ErrPolicyNotFound
	}
	return p, nil
}

// ErrNotFound 退款记录未找到。
var ErrNotFound = errRefundSentinel("refund: record not found")

// ErrPolicyNotFound 策略未找到。
var ErrPolicyNotFound = errRefundSentinel("refund: policy not found for scope+phase")

type errRefundSentinel string

func (e errRefundSentinel) Error() string { return string(e) }