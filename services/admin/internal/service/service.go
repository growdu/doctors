// Package service 是 admin-service 的业务封装层。
//
// 设计要点：
//   - 业务通过 internal/clients 调其他服务；不直接查 DB（除了 dashboard overview 走 ReportsRepo）。
//   - dashboard overview 30s 内存缓存（v2 接 Redis）。
//   - 强制取消 / 审核通过 / 拒绝 都发 Kafka 事件给 notification / audit 订阅者。
//   - publisher=nil 时 publish 是 no-op（单测友好）。
package service

import (
	"context"
	"sync"
	"time"

	"github.com/growdu/doctors/services/admin/internal/clients"
	"github.com/growdu/doctors/services/admin/internal/repo"
)

// OverviewStats 是 dashboard overview 的服务层别名（与 repo.OverviewStats 等价）。
type OverviewStats = repo.OverviewStats

// ReportsRepo 是 dashboard 报表的接口（service 注入用于 mock）。
type ReportsRepo interface {
	Overview(ctx context.Context) (*OverviewStats, error)
}

// OrderClientAdmin 是 admin 视角的 order client 接口（mock 用）。
type OrderClientAdmin interface {
	ListAll(ctx context.Context, p clients.ListOrdersParams) ([]map[string]any, error)
	ForceCancel(ctx context.Context, orderID, adminID int64, reason string) error
}

// RefundClientAdmin 同上。
type RefundClientAdmin interface {
	List(ctx context.Context, status string, page, pageSize int) ([]map[string]any, error)
	Approve(ctx context.Context, id, adminID int64, note string) error
	Reject(ctx context.Context, id, adminID int64, note string) error
}

// EscortClientAdmin 同上。
type EscortClientAdmin interface {
	ListPendingAudit(ctx context.Context) ([]map[string]any, error)
	Approve(ctx context.Context, id, adminID int64, note string) error
	Reject(ctx context.Context, id, adminID int64, note string) error
}

// UserClientAdmin 同上。
type UserClientAdmin interface {
	List(ctx context.Context, role, keyword string, page, pageSize int) ([]map[string]any, error)
}

// Publisher 抽象事件发布。
type Publisher interface {
	Publish(ctx context.Context, topic string, ev any) error
}

// Service 是 admin 业务门面。
type Service struct {
	workOrders    *repo.WorkOrderRepo // 可为 nil（handler 测试用 fake）
	reports       ReportsRepo
	orderClient   OrderClientAdmin
	refundClient  RefundClientAdmin
	escortClient  EscortClientAdmin
	userClient    UserClientAdmin
	publisher     Publisher // 可为 nil（test 时不上发）
	cacheTTL      time.Duration
	mu            sync.Mutex
	cacheOverview *OverviewStats
	cacheExpiresAt time.Time
}

// New 构造。
//
// cacheTTL = 0 → 默认 30s。
func New(reports ReportsRepo, wo *repo.WorkOrderRepo,
	orderClient OrderClientAdmin, refundClient RefundClientAdmin,
	escortClient EscortClientAdmin, userClient UserClientAdmin,
	pub Publisher) *Service {
	return &Service{
		workOrders:   wo,
		reports:      reports,
		orderClient:  orderClient,
		refundClient: refundClient,
		escortClient: escortClient,
		userClient:   userClient,
		publisher:    pub,
		cacheTTL:     30 * time.Second,
	}
}

// publish 安全发布（publisher nil → 跳过）。
func (s *Service) publish(ctx context.Context, topic string, ev any) {
	if s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, topic, ev)
}