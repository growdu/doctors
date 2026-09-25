package service

import (
	"context"

	"github.com/growdu/doctors/services/admin/internal/clients"
	"github.com/growdu/doctors/shared/contracts"
)

// ListOrdersParams 是 service 层参数（避免 service 包引用 clients 包的具体类型）。
type ListOrdersParams struct {
	Status     string
	HospitalID int64
	PatientID  int64
	Page       int
	PageSize   int
}

// toClientsParams 翻译成 clients.ListOrdersParams。
func (p ListOrdersParams) toClientsParams() clients.ListOrdersParams {
	return clients.ListOrdersParams{
		Status:     p.Status,
		HospitalID: p.HospitalID,
		PatientID:  p.PatientID,
		Page:       p.Page,
		PageSize:   p.PageSize,
	}
}

// ListOrders 调 order client 拿全量订单（含筛选）。
func (s *Service) ListOrders(ctx context.Context, p ListOrdersParams) ([]map[string]any, error) {
	if s.orderClient == nil {
		return nil, nil
	}
	return s.orderClient.ListAll(ctx, p.toClientsParams())
}

// ForceCancelOrder 管理员强制取消订单。
//
// 流程：调 order-client 强码 → 发 AdminOrderForceCancelledEvent（refund-service 监听自动退款）。
func (s *Service) ForceCancelOrder(ctx context.Context, orderID, adminID int64, reason string) error {
	if err := s.orderClient.ForceCancel(ctx, orderID, adminID, reason); err != nil {
		return err
	}
	s.publish(ctx, contracts.TopicAdminOrderForceCancelled, contracts.AdminOrderForceCancelledEvent{
		OrderID:     orderID,
		AdminID:     adminID,
		Reason:      reason,
		CancelledAt: nowFn(),
	})
	return nil
}