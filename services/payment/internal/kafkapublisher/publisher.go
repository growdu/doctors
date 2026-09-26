// Package kafkapublisher 是 payment-service 的 Kafka publisher 实现。
//
// 设计要点：
//   - 单底层 writer（无固定 topic），每次 WriteMessages 按事件类型路由到
//     contracts.TopicPaymentCompleted / TopicPaymentRefunded。
//   - 构造期 shared/tracing.WrapWriter 注入 OTel span（kind=Producer；span name
//     反映 writer 占位 topic "publish payment.completed"，每条消息实际 topic
//     在 Kafka 层由 msg.Topic 决定——span 仍能完整覆盖 publish I/O）。
//   - Publish 失败只 log 错误，不阻塞业务（best-effort）。
//   - Close 由 main 注册 shutdown hook 调用，释放 writer。
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

// Publisher 实现 service.Publisher 接口（PublishPaymentCompleted/Refunded）。
//
// 复用 order.events 同款模式：单一 writer + 每次 SetTopic；底层构造借 shared/kafka
// 但因 NewWriter 强制 topic 不空，所以这里先用 TopicPaymentCompleted 占位初始化，
// 实际写入时再覆盖 Topic。
type Publisher struct {
	writer *tracing.TracedWriter
}

// New 构造真实 Kafka publisher；brokers 由 main 传入。
//
// shared/kafka.NewWriter 需要非空 topic；本服务发布到 2 个 topic（completed/refunded），
//
//	借用 TopicPaymentCompleted 占位初始化，写入时再覆写 Topic 字段。
func New(brokers []string) (*Publisher, error) {
	w, err := sharedkafka.NewWriter(brokers, contracts.TopicPaymentCompleted)
	if err != nil {
		return nil, err
	}
	return &Publisher{writer: tracing.WrapWriter(w, "payment-service")}, nil
}

// Close 关闭底层 writer。
func (p *Publisher) Close() error {
	if p == nil || p.writer == nil || p.writer.W == nil {
		return nil
	}
	return p.writer.W.Close()
}

// publish 内部写一条 JSON 到 topic。
func (p *Publisher) publish(ctx context.Context, topic, key string, body any) error {
	if p == nil || p.writer == nil {
		return errors.New("kafkapublisher: writer is nil")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	})
}

// PublishPaymentCompleted 发布 payment.completed 事件。
func (p *Publisher) PublishPaymentCompleted(ctx context.Context, ev contracts.PaymentCompletedEvent) error {
	return p.publish(ctx, contracts.TopicPaymentCompleted, strconv.FormatInt(ev.PaymentID, 10), ev)
}

// PublishPaymentRefunded 发布 payment.refunded 事件。
func (p *Publisher) PublishPaymentRefunded(ctx context.Context, ev contracts.PaymentRefundedEvent) error {
	return p.publish(ctx, contracts.TopicPaymentRefunded, strconv.FormatInt(ev.RefundID, 10), ev)
}