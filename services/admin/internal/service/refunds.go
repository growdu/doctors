package service

import (
	"context"

	"github.com/growdu/doctors/shared/contracts"
)

// ListRefunds 调 refund client 拿退款列表（含 status 筛选）。
func (s *Service) ListRefunds(ctx context.Context, status string, page, pageSize int) ([]map[string]any, error) {
	if s.refundClient == nil {
		return nil, nil
	}
	return s.refundClient.List(ctx, status, page, pageSize)
}

// ApproveRefund 批准退款。
func (s *Service) ApproveRefund(ctx context.Context, refundID, orderID, adminID int64, note string) error {
	if err := s.refundClient.Approve(ctx, refundID, adminID, note); err != nil {
		return err
	}
	s.publish(ctx, contracts.TopicAdminRefundApproved, contracts.AdminRefundApprovedEvent{
		RefundID:   refundID,
		OrderID:    orderID,
		AdminID:    adminID,
		Note:       note,
		ApprovedAt: nowFn(),
	})
	return nil
}

// RejectRefund 拒绝退款。
func (s *Service) RejectRefund(ctx context.Context, refundID, orderID, adminID int64, note string) error {
	if err := s.refundClient.Reject(ctx, refundID, adminID, note); err != nil {
		return err
	}
	s.publish(ctx, contracts.TopicAdminRefundRejected, contracts.AdminRefundRejectedEvent{
		RefundID:   refundID,
		OrderID:    orderID,
		AdminID:    adminID,
		Note:       note,
		RejectedAt: nowFn(),
	})
	return nil
}