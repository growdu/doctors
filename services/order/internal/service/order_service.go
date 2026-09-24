// Package service 是 order-service 的业务编排层。
//
// 设计要点（与 auth/service 保持一致）：
//   - 所有依赖是接口，便于 fake 替身单测。
//   - 业务规则放这里；SQL/状态机在子包。
//   - 后续阶段会接 pgxpool repo、Kafka publisher。
package service

import "context"

// Service 是空壳；阶段 3.5/3.6 会补齐 Create/Accept 等方法。
type Service struct{}

// New 构造 Service；参数后续接 repo / publisher。
func New() *Service { return &Service{} }

// Ping 占位接口，给 handler / 业务层测试用。
func (s *Service) Ping(ctx context.Context) error { return nil }