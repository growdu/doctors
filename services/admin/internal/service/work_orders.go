package service

import (
	"context"

	"github.com/growdu/doctors/services/admin/internal/repo"
	"github.com/growdu/doctors/shared/contracts"
)

// ListWorkOrders 按筛选分页查询。
func (s *Service) ListWorkOrders(ctx context.Context, f repo.WorkOrderListFilter) ([]*repo.WorkOrder, error) {
	return s.workOrders.List(ctx, f)
}

// CreateWorkOrder 创建工单 + 发 AdminWorkOrderCreatedEvent。
func (s *Service) CreateWorkOrder(ctx context.Context, w *repo.WorkOrder) error {
	if err := s.workOrders.Create(ctx, w); err != nil {
		return err
	}
	s.publish(ctx, contracts.TopicAdminWorkOrderCreated, contracts.AdminWorkOrderCreatedEvent{
		WorkOrderID: w.ID,
		UserID:      w.UserID,
		Category:    w.Category,
		Priority:    w.Priority,
		Title:       w.Title,
		CreatedAt:   w.CreatedAt,
	})
	return nil
}

// AssignWorkOrder 分配工单。
func (s *Service) AssignWorkOrder(ctx context.Context, id, adminID int64) error {
	return s.workOrders.Assign(ctx, id, adminID)
}

// ResolveWorkOrder 关单。
func (s *Service) ResolveWorkOrder(ctx context.Context, id int64, resolution string) error {
	return s.workOrders.Resolve(ctx, id, resolution)
}