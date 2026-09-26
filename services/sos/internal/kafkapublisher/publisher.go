// Package kafkapublisher 是 sos-service 的 Kafka publisher 实现。
//
// 设计要点：
//   - 单 topic：contracts.TopicSOSRaised；通过 shared/kafka.NewWriter 构造底层 writer
//   - 构造期 shared/tracing.WrapWriter 注入 OTel span（"publish <topic>"，kind=Producer）
//   - PublishSOSRaised 失败只 log 错误，不阻塞业务（best-effort）
//   - Close 由 main 注册 shutdown hook 调用，释放 writer
//
// 注：sos-service 当前 service.Publisher 接口只暴露 PublishSOSRaised；resolved 状态
//
//	由 service 层内部 UpdateStatus 标记，不广播外部事件（v2 可视情况扩展）。
package kafkapublisher

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/shared/contracts"
	sharedkafka "github.com/growdu/doctors/shared/kafka"
	"github.com/growdu/doctors/shared/tracing"
)

// Publisher 实现 service.Publisher 接口（PublishSOSRaised）。
type Publisher struct {
	writer *tracing.TracedWriter
}

// New 构造真实 Kafka publisher；brokers / topic 由 main 传入。
func New(brokers []string, topic string) (*Publisher, error) {
	w, err := sharedkafka.NewWriter(brokers, topic)
	if err != nil {
		return nil, err
	}
	return &Publisher{writer: tracing.WrapWriter(w, "sos-service")}, nil
}

// Close 关闭底层 writer；main 注册到 shutdown hook（LIFO）。
func (p *Publisher) Close() error {
	if p == nil || p.writer == nil || p.writer.W == nil {
		return nil
	}
	return p.writer.W.Close()
}

// PublishSOSRaised 发布 sos.raised 事件。
func (p *Publisher) PublishSOSRaised(ctx context.Context, ev contracts.SOSRaisedEvent) error {
	if p == nil || p.writer == nil {
		return errors.New("kafkapublisher: writer is nil")
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: contracts.TopicSOSRaised,
		Key:   []byte(strconv.FormatInt(ev.SOSID, 10)),
		Value: data,
	})
}