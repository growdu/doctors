// Package service 是 match-service 的业务编排层。
//
// 设计要点：
//   - 依赖接口（Pool / EscortProvider），不依赖具体实现；测试用 fake / NopPool。
//   - EscortProvider 返回 []contracts.EscortSummary（跨服务契约），scorer 直接吃这个类型。
//   - 不持有任何 repo / DB 引用；match-service 是 stateless + cache-based。
package service

import (
	"context"
	"time"

	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/services/match/internal/scorer"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// EscortProvider 拉陪诊师列表的最小契约。
// v1 用本地 cache / mock，v2 接 escort-service 的 gRPC 或 HTTP client。
type EscortProvider interface {
	ListAvailable(ctx context.Context, city string) ([]contracts.EscortSummary, error)
}

// OrderInfo 是评分所需的最小 order 视图（接 contracts.OrderCreatedEvent）。
type OrderInfo struct {
	ID          int64
	City        string
	ServiceTime time.Time
	Lat         float64
	Lng         float64
}

// Service 是 match 业务编排器。
type Service struct {
	pool      pool.Pool
	escorts   EscortProvider
	expire    time.Duration
}

// New 装配一个 Service。
func New(p pool.Pool, e EscortProvider, expire time.Duration) *Service {
	if expire <= 0 {
		expire = 5 * time.Minute
	}
	return &Service{pool: p, escorts: e, expire: expire}
}

// Match 计算候选 + 写入 Redis 池。
func (s *Service) Match(ctx context.Context, o OrderInfo) ([]scorer.Candidate, error) {
	if o.ID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id required")
	}
	escorts, err := s.escorts.ListAvailable(ctx, o.City)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list escorts", err)
	}
	order := scorer.Order{City: o.City, ServiceTime: o.ServiceTime, Lat: o.Lat, Lng: o.Lng}
	cands := scorer.Compute(order, escorts)

	poolCands := make([]pool.Candidate, len(cands))
	for i, c := range cands {
		poolCands[i] = pool.Candidate{EscortID: c.EscortID, Score: c.Score}
	}
	if err := s.pool.Push(ctx, o.ID, poolCands, s.expire); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "push pool", err)
	}
	return cands, nil
}

// Feed 给陪诊师端拉抢单池 TopN。
func (s *Service) Feed(ctx context.Context, orderID int64, topN int) ([]int64, error) {
	if orderID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id required")
	}
	if topN <= 0 || topN > 50 {
		topN = 10
	}
	return s.pool.Peek(ctx, orderID, topN)
}

// Candidates 给订单方查候选。
func (s *Service) Candidates(ctx context.Context, orderID int64) ([]int64, error) {
	if orderID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id required")
	}
	return s.pool.Peek(ctx, orderID, 50)
}