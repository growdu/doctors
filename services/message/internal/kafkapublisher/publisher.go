// Package kafkapublisher 是 message-service 的 Kafka publisher 实现。
//
// 设计要点：
//   - 单 topic：contracts.TopicMessageSent；通过 shared/kafka.NewWriter 构造底层 writer
//   - PublishMessageSent 失败只 log 错误，不阻塞业务（best-effort）
//   - Close 由 main 注册 shutdown hook 调用，释放 writer
package kafkapublisher

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/shared/contracts"
	sharedkafka "github.com/growdu/doctors/shared/kafka"
)

// Publisher 实现 service.Publisher 接口（PublishMessageSent）。
type Publisher struct {
	writer *kafka.Writer
}

// New 构造真实 Kafka publisher；brokers / topic 由 main 传入。
func New(brokers []string, topic string) (*Publisher, error) {
	w, err := sharedkafka.NewWriter(brokers, topic)
	if err != nil {
		return nil, err
	}
	return &Publisher{writer: w}, nil
}

// Close 关闭底层 writer；main 注册到 shutdown hook（LIFO）。
func (p *Publisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// PublishMessageSent 发布 message.sent 事件。
func (p *Publisher) PublishMessageSent(ctx context.Context, ev contracts.MessageSentEvent) error {
	if p == nil || p.writer == nil {
		return errors.New("kafkapublisher: writer is nil")
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: contracts.TopicMessageSent,
		Key:   []byte(strconv.FormatInt(ev.MessageID, 10)),
		Value: data,
	})
}