package service

import "context"

// ListUsers 调 user client 拿用户列表（含 patient / escort 筛选）。
func (s *Service) ListUsers(ctx context.Context, role, keyword string, page, pageSize int) ([]map[string]any, error) {
	if s.userClient == nil {
		return nil, nil
	}
	return s.userClient.List(ctx, role, keyword, page, pageSize)
}