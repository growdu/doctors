// Package service 是 match-service 的业务编排层。
package service

import (
	"context"
	"errors"
	"time"

	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/services/match/internal/scorer"
)

// EscortLoader 是陪诊师来源抽象（v1 用本地 mock，v2 接 escort-service）。
type EscortLoader interface {
	ListAvailable(ctx context.Context, city string) ([]scorer.Escort, error)
}

// OrderInfo 是评分所需的最小 order 视图。
type OrderInfo struct {
	ID          int64
	City        string
	ServiceTime time.Time
	Lat         float64
	Lng         float64
}

// Service 是 match 业务编排器。
type Service struct {
	pool     pool.Pool
	escorts  EscortLoader
	expire   time.Duration
}

// New 装配一个 Service。
func New(p pool.Pool, e EscortLoader, expire time.Duration) *Service {
	if expire <= 0 {
		expire = 5 * time.Minute
	}
	return &Service{pool: p, escorts: e, expire: expire}
}

// Match 计算候选 + 写入 Redis 池。
func (s *Service) Match(ctx context.Context, o OrderInfo) ([]scorer.Candidate, error) {
	escorts, err := s.escorts.ListAvailable(ctx, o.City)
	if err != nil {
		return nil, err
	}
	order := scorer.Order{City: o.City, ServiceTime: o.ServiceTime, Lat: o.Lat, Lng: o.Lng}
	cands := scorer.Compute(order, escorts)

	// 写 Redis 池（业务层用 pool.Candidate 切片）
	poolCands := make([]pool.Candidate, len(cands))
	for i, c := range cands {
		poolCands[i] = pool.Candidate{EscortID: c.EscortID, Score: c.Score}
	}
	if err := s.pool.Push(ctx, o.ID, poolCands, s.expire); err != nil {
		return nil, err
	}
	return cands, nil
}

// Feed 给陪诊师端拉抢单池 TopN。
func (s *Service) Feed(ctx context.Context, orderID int64, topN int) ([]int64, error) {
	if topN <= 0 || topN > 50 {
		topN = 10
	}
	return s.pool.Peek(ctx, orderID, topN)
}

// Candidates 给订单方查候选。
func (s *Service) Candidates(ctx context.Context, orderID int64) ([]int64, error) {
	return s.pool.Peek(ctx, orderID, 50)
}

// errNotImplemented 是占位错误；v1 直接用 NopPool。
var errNotImplemented = errors.New("match: not implemented (placeholder)")