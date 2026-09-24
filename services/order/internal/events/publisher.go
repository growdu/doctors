// Package events 是 order-service 的事件发布层。
//
// 设计要点：
//   - Publisher 接口只暴露业务事件：OrderCreated / OrderAccepted。
//   - 默认实现是 KafkaPublisher，用 segmentio/kafka-go。
//   - 入参用 shared/contracts 的事件类型（不是 repo.Order），避免暴露 DB 结构。
//   - Topic 名复用 contracts 常量。
//   - Publish 失败只 log，不阻塞业务（事件 best-effort）。
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/shared/contracts"
)

// Publisher 抽象订单事件发布。
//
// v1.1（order-matching-redesign）：删除 PublishOrderMatching；新增 4 个：
//   - PublishOrderSelectingEscort：paid → selecting_escort
//   - PublishOrderEscortSelected：selecting_escort → escort_pending_acceptance
//   - PublishOrderEscortConfirmed：escort_pending_acceptance → accepted
//   - PublishOrderEscortRejected：escort_pending_acceptance → selecting_escort（拒接或 30s 超时）
//
// v1.2（wallet-t+7）：新增 PublishOrderCompleted，accepted → completed 时触发；
//   wallet-service 消费后入 frozen + billings，T+7 由 scanner 释放冻结。
type Publisher interface {
	PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error
	PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error
	PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error
	PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error
	PublishOrderEscortSelected(ctx context.Context, ev contracts.OrderEscortSelectedEvent) error
	PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error
	PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error
	PublishOrderCompleted(ctx context.Context, ev contracts.OrderCompletedEvent) error
	Close() error
}

// KafkaPublisher 用 kafka-go writer 直接写 topic。
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher 构造 publisher。
func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 50 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
	}
}

// Close 关闭底层 writer。
func (p *KafkaPublisher) Close() error { return p.writer.Close() }

// publish 写一条 JSON 到 topic。
func (p *KafkaPublisher) publish(ctx context.Context, topic string, key string, body any) error {
	if p == nil || p.writer == nil {
		return errors.New("publisher: writer is nil")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", topic, err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	})
}

// PublishOrderCreated 发布 order.created 事件。
func (p *KafkaPublisher) PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error {
	return p.publish(ctx, contracts.TopicOrderCreated, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderAccepted 发布 order.accepted 事件。
func (p *KafkaPublisher) PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error {
	return p.publish(ctx, contracts.TopicOrderAccepted, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderCancelled 发布 order.cancelled 事件。
func (p *KafkaPublisher) PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error {
	return p.publish(ctx, contracts.TopicOrderCancelled, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderSelectingEscort 发布 order.selecting_escort 事件。
func (p *KafkaPublisher) PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error {
	return p.publish(ctx, contracts.TopicOrderSelectingEscort, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderEscortSelected 发布 order.escort_selected 事件。
func (p *KafkaPublisher) PublishOrderEscortSelected(ctx context.Context, ev contracts.OrderEscortSelectedEvent) error {
	return p.publish(ctx, contracts.TopicOrderEscortSelected, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderEscortConfirmed 发布 order.escort_confirmed 事件。
func (p *KafkaPublisher) PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error {
	return p.publish(ctx, contracts.TopicOrderEscortConfirmed, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderEscortRejected 发布 order.escort_rejected 事件。
func (p *KafkaPublisher) PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error {
	return p.publish(ctx, contracts.TopicOrderEscortRejected, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderCompleted 发布 order.completed 事件（v1.2 wallet-t+7）。
// 触发时机：accepted → completed；wallet 消费后入 frozen + billings。
func (p *KafkaPublisher) PublishOrderCompleted(ctx context.Context, ev contracts.OrderCompletedEvent) error {
	return p.publish(ctx, contracts.TopicOrderCompleted, strconv.FormatInt(ev.OrderID, 10), ev)
}

// NopPublisher 是测试或 dev 占位实现。
type NopPublisher struct {
	CreatedCount            int
	AcceptedCount           int
	CancelledCount          int
	SelectingEscortCount    int
	EscortSelectedCount     int
	EscortConfirmedCount    int
	EscortRejectedCount     int
	CompletedCount          int // v1.2 wallet-t+7
}

// PublishOrderCreated 计数 + 返回。
func (p *NopPublisher) PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error {
	p.CreatedCount++
	return nil
}

// PublishOrderAccepted 计数 + 返回。
func (p *NopPublisher) PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error {
	p.AcceptedCount++
	return nil
}

// PublishOrderCancelled 计数 + 返回。
func (p *NopPublisher) PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error {
	p.CancelledCount++
	return nil
}

// PublishOrderSelectingEscort 计数 + 返回。
func (p *NopPublisher) PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error {
	p.SelectingEscortCount++
	return nil
}

// PublishOrderEscortSelected 计数 + 返回。
func (p *NopPublisher) PublishOrderEscortSelected(ctx context.Context, ev contracts.OrderEscortSelectedEvent) error {
	p.EscortSelectedCount++
	return nil
}

// PublishOrderEscortConfirmed 计数 + 返回。
func (p *NopPublisher) PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error {
	p.EscortConfirmedCount++
	return nil
}

// PublishOrderEscortRejected 计数 + 返回。
func (p *NopPublisher) PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error {
	p.EscortRejectedCount++
	return nil
}

// PublishOrderCompleted 计数 + 返回（v1.2 wallet-t+7）。
func (p *NopPublisher) PublishOrderCompleted(ctx context.Context, ev contracts.OrderCompletedEvent) error {
	p.CompletedCount++
	return nil
}

// Close NopPublisher 无资源。
func (p *NopPublisher) Close() error { return nil }