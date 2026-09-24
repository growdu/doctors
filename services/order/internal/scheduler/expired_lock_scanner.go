// Package scheduler 跑后台定时任务。
//
// 设计要点：
//   - 每个 scheduler 都是 Run(ctx) 阻塞循环；ctx cancel 即退出。
//   - 用接口注入 repo / publisher / service；scheduler 只关心"扫 + 通知"，
//     不直接操作 DB（与业务 service 一致）。
//   - 失败 best-effort：单条订单释放失败不影响下一条。
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/growdu/doctors/services/order/internal/events"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/shared/contracts"
)

// RepoLockExpired 是 scheduler 调用的最小仓储契约。
type RepoLockExpired interface {
	LockExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error)
}

// ReleaseService 是 service 暴露给 scheduler 的最小契约。
type ReleaseService interface {
	ReleaseAcceptLock(ctx context.Context, orderID, actorID int64) error
}

// ExpiredLockScanner 每 interval 秒扫一次过期锁单，自动回退 matching 并通知。
type ExpiredLockScanner struct {
	repo      RepoLockExpired
	svc       ReleaseService
	publisher events.Publisher
	interval  time.Duration
	limit     int // 每次扫描最多处理多少单
}

// NewExpiredLockScanner 构造 scanner；interval <= 0 默认 5s。
func NewExpiredLockScanner(r RepoLockExpired, s ReleaseService, p events.Publisher, interval time.Duration) *ExpiredLockScanner {
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
	expired, err := s.repo.LockExpired(ctx, time.Now(), s.limit)
	if err != nil {
		log.Printf("[expired-lock-scanner] scan failed: %v", err)
		return
	}
	for _, o := range expired {
		// lock_owner 为 nil 视为异常（DB 状态不一致）；跳过
		if o.LockOwner == nil {
			log.Printf("[expired-lock-scanner] order %d has nil lock_owner, skip", o.ID)
			continue
		}
		// 用 lock_owner 作为 actorID 调用 ReleaseAcceptLock（service 校验通过）
		if err := s.svc.ReleaseAcceptLock(ctx, o.ID, *o.LockOwner); err != nil {
			log.Printf("[expired-lock-scanner] release %d failed: %v", o.ID, err)
			continue
		}
		// 发 OrderMatchingEvent（best-effort，失败仅 log）
		if s.publisher != nil {
			if err := s.publisher.PublishOrderMatching(ctx, contracts.OrderMatchingEvent{
				OrderID:   o.ID,
				EscortID:  *o.LockOwner,
				Reason:    "lock_expired",
				RetriedAt: time.Now(),
			}); err != nil {
				log.Printf("[expired-lock-scanner] publish %d failed: %v", o.ID, err)
			}
		}
		log.Printf("[expired-lock-scanner] order %d lock_expired, retried", o.ID)
	}
}