package service

import (
	"context"
	"time"
)

// OverviewStats 返回 dashboard overview（含 30s 缓存）。
//
// 第二次调用 < 30s 直接返回缓存；> 30s 重新拉。
// data_fresh 字段保留供前端区分（v1 不返回给 HTTP 响应，仅内部用）。
func (s *Service) OverviewStats(ctx context.Context) (*OverviewStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowFn()
	if s.cacheOverview != nil && now.Before(s.cacheExpiresAt) {
		return s.cacheOverview, nil
	}
	stats, err := s.reports.Overview(ctx)
	if err != nil {
		return nil, err
	}
	s.cacheOverview = stats
	s.cacheExpiresAt = now.Add(s.cacheTTL)
	return stats, nil
}

// InvalidateOverview 清缓存（写操作后可调；v1 简化：被动过期即可）。
func (s *Service) InvalidateOverview() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheOverview = nil
}

// ForceExpire 仅测试用：把 cacheExpiresAt 设为过去。
func (s *Service) ForceExpire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheExpiresAt = time.Now().Add(-time.Second)
}