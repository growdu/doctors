// Package scheduler 跑后台定时任务。
//
// 设计要点：
//   - 每个 scheduler 都是 Run(ctx) 阻塞循环；ctx cancel 即退出。
//   - 用接口注入 repo / publisher / service；scheduler 只关心"扫 + 通知"，
//     不直接操作 DB（与业务 service 一致）。
//   - 失败 best-effort：单条订单释放失败不影响下一条。
//
// v1.1（order-matching-redesign）：
//   - 文件名沿用 expired_lock_scanner.go（避免重命名引发不必要的代码 review 噪音）。
//   - 实际语义改名为 EscortInviteExpiry（30s 选人确认窗口超时 scanner）。
//   - 扫 escort_pending_expire_at < now 的 escort_pending_acceptance 订单，
//     调 service.RejectAccept(reason="lock_expired") 回退 selecting_escort，
//     发 OrderEscortRejectedEvent（替换旧 OrderMatchingEvent）。
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/growdu/doctors/services/order/internal/events"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/shared/contracts"
)

// RepoPendingExpired 是 scheduler 调用的最小仓储契约（v1.1）。
type RepoPendingExpired interface {
	PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error)
}

// RejectService 是 service 暴露给 scheduler 的最小契约（v1.1）。
type RejectService interface {
	RejectAccept(ctx context.Context, orderID, escortID int64, reason string) error
}

// ExpiredLockScanner 每 interval 秒扫一次过期 escort_pending_expire_at，
// 自动调 RejectAccept(reason="lock_expired") 回退 selecting_escort 并发 OrderEscortRejectedEvent。
//
// 命名沿用旧文件名以便 git history；实际语义是 EscortInviteExpiry scanner。
type ExpiredLockScanner struct {
	repo      RepoPendingExpired
	svc       RejectService
	publisher events.Publisher
	interval  time.Duration
	limit     int // 每次扫描最多处理多少单
}

// NewExpiredLockScanner 构造 scanner；interval <= 0 默认 5s。
func NewExpiredLockScanner(r RepoPendingExpired, s RejectService, p events.Publisher, interval time.Duration) *ExpiredLockScanner {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &ExpiredLockScanner{
		repo:      r,
		svc:       s,
		publisher: p,
		interval:  interval,
		limit:     50,
	}
}

// Run 阻塞扫描直到 ctx 取消。
func (s *ExpiredLockScanner) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.ScanOnce(ctx)
		}
	}
}

// ScanOnce 扫描一次；可单测。
func (s *ExpiredLockScanner) ScanOnce(ctx context.Context) {
	expired, err := s.repo.PendingExpired(ctx, time.Now(), s.limit)
	if err != nil {
		log.Printf("[expired-lock-scanner] scan failed: %v", err)
		return
	}
	for _, o := range expired {
		// selected_escort_id 为 nil 视为异常（DB 状态不一致）；跳过
		if o.SelectedEscortID == nil {
			log.Printf("[expired-lock-scanner] order %d has nil selected_escort_id, skip", o.ID)
			continue
		}
		// 用 selected_escort_id 作为 escortID 调 RejectAccept（service 校验通过）
		if err := s.svc.RejectAccept(ctx, o.ID, *o.SelectedEscortID, "lock_expired"); err != nil {
			log.Printf("[expired-lock-scanner] reject %d failed: %v", o.ID, err)
			continue
		}
		// 发 OrderEscortRejectedEvent（best-effort，失败仅 log）
		if s.publisher != nil {
			if err := s.publisher.PublishOrderEscortRejected(ctx, contracts.OrderEscortRejectedEvent{
				OrderID:    o.ID,
				PatientID:  o.PatientID,
				EscortID:   *o.SelectedEscortID,
				Reason:     "lock_expired",
				OccurredAt: time.Now(),
			}); err != nil {
				log.Printf("[expired-lock-scanner] publish %d failed: %v", o.ID, err)
			}
		}
		log.Printf("[expired-lock-scanner] order %d invitation expired, retried", o.ID)
	}
}