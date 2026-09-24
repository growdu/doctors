// scanner 是 T+7 冻结释放扫描器。
//
// 设计要点：
//   - 1 分钟间隔扫一次（v1 测试用 1 分钟模拟 7 天，生产可改 7*24h + daily cron）。
//   - RunScanOnce 用当前 cutoff = now - threshold 调 repo.ListCompletedOrdersBefore，
//     拿到 [{orderID, amount}] 列表，逐个调 UnfreezeToBalance(orderID, amount)。
//   - threshold 参数化：测试 1 分钟、生产 7 天，同一段代码。
//   - 单次失败不中断整个循环（warn log + continue）；多次失败由 Run loop 的 ctx 取消。
//   - allUsersFn 由 cmd 注入（扫所有 escrow 角色 user）。
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/growdu/doctors/services/wallet/internal/repo"
	"github.com/growdu/doctors/shared/logger"
)

// RepoScanner 是 scanner 需要的 repo 最小子集。
type RepoScanner interface {
	ListCompletedOrdersBefore(ctx context.Context, cutoff time.Time, limit int) ([]repo.CompletedOrder, error)
	UnfreezeToBalance(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error
}

// CompletedOrder 别名（与 repo.CompletedOrder 同义；让 scanner_test 不用 import repo）。
type CompletedOrder = repo.CompletedOrder

// Scanner T+7 冻结释放扫描器。
type Scanner struct {
	repo      RepoScanner
	threshold time.Duration
	tick      time.Duration
	now       func() time.Time
}

// NewScanner 构造 scanner。threshold=T+7 间隔（prod=7*24h；test=1m）；tick=扫描周期（prod=24h；test=1m）。
func NewScanner(repo RepoScanner, threshold time.Duration, nowFn func() time.Time) *Scanner {
	return &Scanner{
		repo:      repo,
		threshold: threshold,
		tick:      threshold, // 默认扫描周期 == 阈值；可改
		now:       nowFn,
	}
}

// WithTick 覆盖扫描周期。
func (s *Scanner) WithTick(t time.Duration) *Scanner { s.tick = t; return s }

// RunScanOnce 单次扫描（测试用 + 实际周期 tick）。
func (s *Scanner) RunScanOnce(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-s.threshold)
	logger.L().Info("scanner: T+7 tick", zap.Time("cutoff", cutoff), zap.Int64("threshold_sec", int64(s.threshold.Seconds())))

	// 单次最多处理 100 单（避免长事务）。
	const batchLimit = 100
	orders, err := s.repo.ListCompletedOrdersBefore(ctx, cutoff, batchLimit)
	if err != nil {
		return fmt.Errorf("list completed: %w", err)
	}
	for _, o := range orders {
		if o.Amount.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if err := s.repo.UnfreezeToBalance(ctx, o.UserID, o.OrderID, o.Amount); err != nil {
			logger.L().Warn("scanner: unfreeze failed",
				zap.Int64("user_id", o.UserID), zap.Int64("order_id", o.OrderID),
				zap.String("amount", o.Amount.String()), zap.Error(err))
			continue
		}
		logger.L().Info("scanner: unfrozen",
			zap.Int64("user_id", o.UserID), zap.Int64("order_id", o.OrderID),
			zap.String("amount", o.Amount.String()))
	}
	return nil
}

// Run 启动扫描循环（ctx 取消退出）。
// allUsersFn 由调用方注入（实际生产用 sql 查 users.role='escort'）。
func (s *Scanner) Run(ctx context.Context) {
	t := time.NewTicker(s.tick)
	defer t.Stop()
	logger.L().Info("scanner: started",
		zap.Duration("threshold", s.threshold),
		zap.Duration("tick", s.tick))
	for {
		select {
		case <-ctx.Done():
			logger.L().Info("scanner: stopped")
			return
		case now := <-t.C:
			if err := s.RunScanOnce(ctx, now); err != nil {
				logger.L().Warn("scanner: tick failed", zap.Error(err))
			}
		}
	}
}