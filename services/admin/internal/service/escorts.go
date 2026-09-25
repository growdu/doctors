package service

import (
	"context"

	"github.com/growdu/doctors/shared/contracts"
)

// ListPendingEscorts 拿待审核陪诊师。
func (s *Service) ListPendingEscorts(ctx context.Context) ([]map[string]any, error) {
	if s.escortClient == nil {
		return nil, nil
	}
	return s.escortClient.ListPendingAudit(ctx)
}

// ApproveEscort 通过陪诊师审核。
func (s *Service) ApproveEscort(ctx context.Context, escortID, adminID int64, note string) error {
	if err := s.escortClient.Approve(ctx, escortID, adminID, note); err != nil {
		return err
	}
	s.publish(ctx, contracts.TopicAdminEscortApproved, contracts.AdminEscortApprovedEvent{
		EscortID:   escortID,
		AdminID:    adminID,
		Note:       note,
		ApprovedAt: nowFn(),
	})
	return nil
}

// RejectEscort 拒绝陪诊师审核。
func (s *Service) RejectEscort(ctx context.Context, escortID, adminID int64, note string) error {
	if err := s.escortClient.Reject(ctx, escortID, adminID, note); err != nil {
		return err
	}
	s.publish(ctx, contracts.TopicAdminEscortRejected, contracts.AdminEscortRejectedEvent{
		EscortID:   escortID,
		AdminID:    adminID,
		Note:       note,
		RejectedAt: nowFn(),
	})
	return nil
}