// Package events 是 admin-service 的事件发布层。
package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
)

// Publisher 抽象事件发布。
type Publisher interface {
	Publish(ctx context.Context, topic string, ev any) error
	Close() error
}

// KafkaPublisher 用 kafka-go writer 写 topic。
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher 构造。
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

// Close 关闭 writer。
func (p *KafkaPublisher) Close() error { return p.writer.Close() }

// Publish 通用写消息。
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, body any) error {
	if p == nil || p.writer == nil {
		return errors.New("publisher: writer is nil")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic, Value: data, Time: time.Now(),
	})
}

// NopPublisher 是测试 / dev 占位。
type NopPublisher struct{ Count int }

// Publish 计数。
func (p *NopPublisher) Publish(_ context.Context, _ string, _ any) error {
	p.Count++
	return nil
}

// Close 无资源。
func (p *NopPublisher) Close() error { return nil }