// Package kafkapublisher 是 escort-service 的 Kafka publisher 实现。
//
// 设计要点：
//   - 双 topic（escort.available / escort.unavailable）；通过 shared/kafka.NewWriter
//     构造底层 writer，借用 TopicEscortAvailable 占位初始化，发布时按事件类型覆写 Topic。
//   - 构造期 shared/tracing.WrapWriter 注入 OTel span（kind=Producer；span name
//     反映占位 topic "publish escort.available"，实际 topic 由 msg.Topic 决定）。
//   - PublishAvailabilityChanged 失败只 log 错误，不阻塞业务（best-effort）。
//   - Close 由 main 注册 shutdown hook 调用，释放 writer。
//
// 注：service.Publisher 接口当前只暴露 PublishAvailabilityChanged(escort_id, online bool)；
//
//	v2 可扩展为更细粒度的 EscortAvailableEvent / EscortUnavailableEvent。
package kafkapublisher

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/contracts"
	sharedkafka "github.com/growdu/doctors/shared/kafka"
	"github.com/growdu/doctors/shared/tracing"
)

// Publisher 实现 service.Publisher 接口（PublishAvailabilityChanged）。
type Publisher struct {
	writer *tracing.TracedWriter
}

// New 构造真实 Kafka publisher；brokers 由 main 传入。
func New(brokers []string) (*Publisher, error) {
	w, err := sharedkafka.NewWriter(brokers, contracts.TopicEscortAvailable)
	if err != nil {
		return nil, err
	}
	return &Publisher{writer: tracing.WrapWriter(w, "escort-service")}, nil
}

// Close 关闭底层 writer；main 注册到 shutdown hook（LIFO）。
func (p *Publisher) Close() error {
	if p == nil || p.writer == nil || p.writer.W == nil {
		return nil
	}
	return p.writer.W.Close()
}

// PublishAvailabilityChanged 发布 escort 上 / 下线事件。
//
// service.AvailabilityEvent 是 escort-service 内部的最小事件（含 Available 标志）；
//
//	为兼容 contracts 事件 schema（按 Available 路由 available/unavailable），
//	内部构造完整 contracts 事件后写入。
func (p *Publisher) PublishAvailabilityChanged(ctx context.Context, ev service.AvailabilityEvent) error {
	if p == nil || p.writer == nil {
		return errors.New("kafkapublisher: writer is nil")
	}
	if ev.Available {
		data, err := json.Marshal(contracts.EscortAvailableEvent{
			EscortID:   ev.EscortID,
			City:       ev.City,
			Lat:        ev.Lat,
			Lng:        ev.Lng,
			ValidUntil: ev.ValidUntil,
		})
		if err != nil {
			return err
		}
		return p.writer.WriteMessages(ctx, kafka.Message{
			Topic: contracts.TopicEscortAvailable,
			Key:   []byte(strconv.FormatInt(ev.EscortID, 10)),
			Value: data,
		})
	}
	data, err := json.Marshal(contracts.EscortUnavailableEvent{
		EscortID: ev.EscortID,
	})
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: contracts.TopicEscortUnavailable,
		Key:   []byte(strconv.FormatInt(ev.EscortID, 10)),
		Value: data,
	})
}