// Package consumer 消费 Kafka 事件（order.created → 写抢单池）。
//
// 设计要点：
//   - 事件 schema 来自 shared/contracts（单一来源）；本包只负责反序列化 + 调度。
//   - 集成测试用 //go:build integration；单元测试只验证 handler 装配。
//   - reader 用 kafka-go；Subscribe 阻塞循环，ctx cancel 时退出。
//   - handle 失败时不 commit；下一轮 poll 会重投同一条消息。
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/services/match/internal/scorer"
	"github.com/growdu/doctors/services/match/internal/service"
	"github.com/growdu/doctors/shared/contracts"
)

// Consumer 订阅 order.created 并写抢单池。
type Consumer struct {
	reader *kafka.Reader
	svc    *service.Service
}

// New 构造 Consumer；brokers / topic / groupID 由调用方提供。
func New(brokers []string, topic, groupID string, svc *service.Service) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	return &Consumer{reader: r, svc: svc}
}

// Run 阻塞消费直到 ctx cancel。
func (c *Consumer) Run(ctx context.Context) error {
	defer c.reader.Close()
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}
		if err := c.handle(ctx, msg); err != nil {
			log.Printf("[match consumer] handle failed: %v", err)
			continue // 重投
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("[match consumer] commit failed: %v", err)
		}
	}
}

// handle 处理一条消息；目前只关心 order.created。
func (c *Consumer) handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case contracts.TopicOrderCreated:
		var ev contracts.OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			return fmt.Errorf("decode %s: %w", msg.Topic, err)
		}
		_, err := c.svc.Match(ctx, service.OrderInfo{
			ID:          ev.OrderID,
			City:        ev.City,
			ServiceTime: ev.ServiceStartAt,
			Lat:         ev.HospitalLat,
			Lng:         ev.HospitalLng,
		})
		return err
	default:
		return nil
	}
}

// HandleOrderCreated 暴露给单元测试 / 直接调用（不走消费者）。
func HandleOrderCreated(ctx context.Context, svc *service.Service, ev contracts.OrderCreatedEvent) ([]scorer.Candidate, error) {
	return svc.Match(ctx, service.OrderInfo{
		ID:          ev.OrderID,
		City:        ev.City,
		ServiceTime: ev.ServiceStartAt,
		Lat:         ev.HospitalLat,
		Lng:         ev.HospitalLng,
	})
}