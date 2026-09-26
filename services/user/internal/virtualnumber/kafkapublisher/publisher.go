// Package kafkapublisher 是 user-service virtualnumber 模块的 Kafka publisher 实现。
//
// 设计要点：
//   - 双 topic（allocated / released）；通过 shared/kafka.NewWriter 构造底层 writer，
//     借用 TopicVirtualNumberAllocated 占位初始化，发布时按事件类型覆写 Topic。
//   - Publish* 失败只 log 错误，不阻塞业务（best-effort）。
//   - Close 由 main 注册 shutdown hook 调用，释放 writer。
package kafkapublisher

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/services/user/internal/virtualnumber"
	"github.com/growdu/doctors/shared/contracts"
	sharedkafka "github.com/growdu/doctors/shared/kafka"
)

// Publisher 实现 virtualnumber.Publisher 接口。
type Publisher struct {
	writer *kafka.Writer
}

// New 构造真实 Kafka publisher；brokers 由 main 传入。
func New(brokers []string) (*Publisher, error) {
	w, err := sharedkafka.NewWriter(brokers, contracts.TopicVirtualNumberAllocated)
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

// PublishVirtualNumberAllocated 发布 virtual_number.allocated 事件。
func (p *Publisher) PublishVirtualNumberAllocated(ctx context.Context, ev contracts.VirtualNumberAllocatedEvent) error {
	return p.publish(ctx, contracts.TopicVirtualNumberAllocated, strconv.FormatInt(ev.VirtualNumberID, 10), ev)
}

// PublishVirtualNumberReleased 发布 virtual_number.released 事件。
func (p *Publisher) PublishVirtualNumberReleased(ctx context.Context, ev contracts.VirtualNumberReleasedEvent) error {
	return p.publish(ctx, contracts.TopicVirtualNumberReleased, strconv.FormatInt(ev.VirtualNumberID, 10), ev)
}

// 引用 virtualnumber 包，确保依赖方向：user/cmd → virtualnumber/kafkapublisher → virtualnumber。
//
//	本包当前不直接引用 virtualnumber 类型，仅保留 _ 引用防止 import 被剪枝。
var _ = virtualnumber.ErrNotFound