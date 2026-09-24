// Package events 是 order-service 的事件发布层。
//
// 设计要点：
//   - Publisher 接口只暴露业务关心的事件：OrderCreated / OrderAccepted。
//   - 默认实现是 KafkaPublisher，用 segmentio/kafka-go。
//   - 集成测试用 //go:build integration 隔离。
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/services/order/internal/repo"
)

// Topic 命名常量（与 docs/03 数据契约对齐）。
const (
	TopicOrderCreated  = "order.created"
	TopicOrderAccepted = "order.accepted"
)

// Publisher 抽象订单事件发布。
type Publisher interface {
	PublishOrderCreated(ctx context.Context, o *repo.Order) error
	PublishOrderAccepted(ctx context.Context, o *repo.Order) error
	Close() error
}

// KafkaPublisher 用 kafka-go writer 直接写 topic。
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher 构造 publisher；writer 必须指向一个 broker。
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

// publish 写一条 JSON 到 topic；key = order_id 字符串。
func (p *KafkaPublisher) publish(ctx context.Context, topic string, o *repo.Order) error {
	if p == nil || p.writer == nil {
		return errors.New("publisher: writer is nil")
	}
	body, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(strconv.FormatInt(o.ID, 10)),
		Value: body,
		Time:  time.Now(),
	}
	return p.writer.WriteMessages(ctx, msg)
}

// PublishOrderCreated 发布 order.created 事件。
func (p *KafkaPublisher) PublishOrderCreated(ctx context.Context, o *repo.Order) error {
	return p.publish(ctx, TopicOrderCreated, o)
}

// PublishOrderAccepted 发布 order.accepted 事件。
func (p *KafkaPublisher) PublishOrderAccepted(ctx context.Context, o *repo.Order) error {
	return p.publish(ctx, TopicOrderAccepted, o)
}

// NopPublisher 是测试或 dev 占位实现；不打 broker，只记日志。
type NopPublisher struct {
	CreatedCount  int
	AcceptedCount int
}

// PublishOrderCreated 计数 + 返回。
func (p *NopPublisher) PublishOrderCreated(ctx context.Context, o *repo.Order) error {
	p.CreatedCount++
	return nil
}

// PublishOrderAccepted 计数 + 返回。
func (p *NopPublisher) PublishOrderAccepted(ctx context.Context, o *repo.Order) error {
	p.AcceptedCount++
	return nil
}

// Close NopPublisher 无资源。
func (p *NopPublisher) Close() error { return nil }