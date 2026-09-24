// Package consumer 消费 Kafka 事件（order.created → 写抢单池）。
//
// 设计要点：
//   - 集成测试用 //go:build integration；单元测试只验证 handler 装配。
//   - reader 用 kafka-go；Subscribe 阻塞循环，ctx cancel 时退出。
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
)

// OrderCreatedEvent 是 order.created 消息体。
type OrderCreatedEvent struct {
	OrderID        int64     `json:"order_id"`
	City           string    `json:"city"`
	ServiceStartAt time.Time `json:"service_start_at"`
	HospitalLat    float64   `json:"hospital_lat"`
	HospitalLng    float64   `json:"hospital_lng"`
}

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
			// 不 commit；下次会重投
			continue
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("[match consumer] commit failed: %v", err)
		}
	}
}

// handle 处理一条消息；目前只关心 order.created。
func (c *Consumer) handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case "order.created":
		var ev OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			return fmt.Errorf("decode order.created: %w", err)
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
		// 其它 topic 暂忽略
		return nil
	}
}

// HandleOrderCreated 暴露给单元测试 / 直接调用（不走消费者）。
func HandleOrderCreated(ctx context.Context, svc *service.Service, ev OrderCreatedEvent) ([]scorer.Candidate, error) {
	return svc.Match(ctx, service.OrderInfo{
		ID:          ev.OrderID,
		City:        ev.City,
		ServiceTime: ev.ServiceStartAt,
		Lat:         ev.HospitalLat,
		Lng:         ev.HospitalLng,
	})
}