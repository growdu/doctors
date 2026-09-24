// Package service - message-service 业务编排。
//
// 设计要点：
//   - 简化版：仅提供会话 + 消息两个核心模型。
//   - SendMessage 触发 contracts.MessageSentEvent；订阅方推送通知。
//   - 真实 WebSocket 推送在 v2 接入；v1 走 Kafka。
package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// Conversation 是会话。
type Conversation struct {
	ID        int64
	OrderID   int64
	CreatedAt time.Time
}

// Message 是单条消息。
type Message struct {
	ID         int64
	OrderID    int64
	FromUserID int64
	ToUserID   int64
	Body       string
	CreatedAt  time.Time
}

// Repo 是仓储契约。
type Repo interface {
	CreateConversation(ctx context.Context, c *Conversation) error
	CreateMessage(ctx context.Context, m *Message) error
	ListMessagesByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*Message, error)
}

// Publisher 是事件发布抽象。
type Publisher interface {
	PublishMessageSent(ctx context.Context, ev contracts.MessageSentEvent) error
}

// Service 是 message 业务编排器。
type Service struct {
	repo      Repo
	publisher Publisher
}

// New 装配 Service。
func New(r Repo, p Publisher) *Service { return &Service{repo: r, publisher: p} }

// SendMessage 发送一条消息。
//   - orderID 必须 > 0
//   - body 1..1000 字符
//   - from != to
func (s *Service) SendMessage(ctx context.Context, orderID, fromID, toID int64, body string) (*Message, error) {
	if orderID == 0 || fromID == 0 || toID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id / from_id / to_id required")
	}
	body = strings.TrimSpace(body)
	l := utf8.RuneCountInString(body)
	if l == 0 || l > 1000 {
		return nil, errs.New(errs.CodeParamInvalid, "body length must be 1..1000")
	}
	if fromID == toID {
		return nil, errs.New(errs.CodeParamInvalid, "from and to must differ")
	}

	m := &Message{OrderID: orderID, FromUserID: fromID, ToUserID: toID, Body: body}
	if err := s.repo.CreateMessage(ctx, m); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create message", err)
	}

	if s.publisher != nil {
		_ = s.publisher.PublishMessageSent(ctx, contracts.MessageSentEvent{
			MessageID: m.ID, OrderID: orderID, FromID: fromID, ToID: toID,
			Body: body, SentAt: m.CreatedAt,
		})
	}
	return m, nil
}

// ListByOrder 拉某订单的消息列表（按时间正序）。
func (s *Service) ListByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*Message, error) {
	if orderID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id required")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListMessagesByOrder(ctx, orderID, limit, offset)
}

// 确保 errors import 用上
var _ = errors.New