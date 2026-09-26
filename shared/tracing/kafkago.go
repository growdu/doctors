// Package tracing —— kafka-go OTel 自动埋点（生产/消费 span）。
//
// 设计要点：
//   - 因 github.com/segmentio/otelkafkago / go.opentelemetry.io/contrib/...
//     暂无可用版本（goproxy.cn 未镜像），本包用 OTel 标准 trace API 写一个
//     极简 kafka-go wrapper，覆盖生产与消费的"全调用"语义。
//   - 关键不变量：原 kafka-go Writer / Reader 行为不变，仅在每次 I/O 前后
//     开/关 span；span name = "<operation> <topic>"（如 "publish order.completed"）。
//   - 单测验证：传入 fake writer / reader（实现 io.Writer/io.Reader 即可），
//     确保 span 数量、属性正确。
//
// 使用：
//
//	w := kafka.NewWriter(brokers, topic)
//	wrapped := sharedtracing.WrapWriter(w, "wallet-service")
//	wrapped.WriteMessages(ctx, kafka.Message{...})
//
//	r := kafka.NewReader(brokers, topic, groupID)
//	wrapped := sharedtracing.WrapReader(r, "wallet-service")
//	m, err := wrapped.ReadMessage(ctx)
//
// 未来若 otelkafkago 上线，可一行替换：
//
//	wrapped := otelkafkago.NewTracer(...)  // 不改业务代码
package tracing

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// kafkaTracerName 用于 otel.Tracer；和 tracing 包其它组件统一 namespace。
const kafkaTracerName = "github.com/growdu/doctors/shared/tracing/kafkago"

// TracedWriter 是 kafka.Writer 的 OTel wrapper。
//
// 字段：
//   - W：原 *kafka.Writer；业务方法直接用 TracedWriter 即可（结构体内嵌）。
//   - serviceName：注入到 span attribute（如 service.name=wallet-service）。
type TracedWriter struct {
	W           *kafka.Writer
	serviceName string
}

// WrapWriter 返回一个带 OTel span 包装的 *TracedWriter。
//
// serviceName 注入到 span attribute（messaging.system=kafka + service.name）；
// 为空时省略 service.name（让 default resource 接管）。
func WrapWriter(w *kafka.Writer, serviceName string) *TracedWriter {
	return &TracedWriter{W: w, serviceName: serviceName}
}

// WriteMessages 启动 "publish <topic>" span，调用 w.WriteMessages，错误时设置 span status。
func (t *TracedWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	topic := "<unknown>"
	if t.W != nil {
		topic = t.W.Topic
	}
	ctx, span := startSpan(ctx, t.serviceName, "publish", topic, len(msgs))
	defer span.End()

	err := t.W.WriteMessages(ctx, msgs...)
	finishSpan(span, err)
	return err
}

// WriteMessagesWithTimeout 与 WriteMessages 类似，但传入独立的 timeout ctx
// （用于需要 deadline 但又不影响主 ctx 的场景）。
func (t *TracedWriter) WriteMessagesWithTimeout(parent context.Context, timeout time.Duration, msgs ...kafka.Message) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return t.WriteMessages(ctx, msgs...)
}

// TracedReader 是 kafka.Reader 的 OTel wrapper。
type TracedReader struct {
	R           *kafka.Reader
	serviceName string
}

// WrapReader 返回一个带 OTel span 包装的 *TracedReader。
func WrapReader(r *kafka.Reader, serviceName string) *TracedReader {
	return &TracedReader{R: r, serviceName: serviceName}
}

// FetchMessage 启动 "consume <topic>" span，调用 r.FetchMessage。
func (t *TracedReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	topic := "<unknown>"
	if t.R != nil {
		topic = t.R.Config().Topic
	}
	ctx, span := startSpan(ctx, t.serviceName, "consume", topic, 0)
	defer span.End()

	msg, err := t.R.FetchMessage(ctx)
	if err == nil {
		// 标注消费进度（不依赖 group coordinator 状态；只标记 offset / partition）。
		span.SetAttributes(
			attribute.Int64("messaging.kafka.offset", int64(msg.Offset)),
			attribute.Int("messaging.kafka.partition", msg.Partition),
		)
	}
	finishSpan(span, err)
	return msg, err
}

// ReadMessage 启动 "consume" span，调用 r.ReadMessage（含自动 commit）。
func (t *TracedReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	topic := "<unknown>"
	if t.R != nil {
		topic = t.R.Config().Topic
	}
	ctx, span := startSpan(ctx, t.serviceName, "consume", topic, 0)
	defer span.End()

	msg, err := t.R.ReadMessage(ctx)
	if err == nil {
		span.SetAttributes(
			attribute.Int64("messaging.kafka.offset", int64(msg.Offset)),
			attribute.Int("messaging.kafka.partition", msg.Partition),
		)
	}
	finishSpan(span, err)
	return msg, err
}

// startSpan 启动一个 messaging.* 语义 span 并写入通用 attribute。
//
// operation：publish / consume（kafka-go 语义）
// topic：消息 topic 名
// msgCount：WriteMessages 的消息条数（消费场景传 0）
func startSpan(ctx context.Context, serviceName, operation, topic string, msgCount int) (context.Context, trace.Span) {
	tracer := otel.Tracer(kafkaTracerName)
	kind := trace.SpanKindProducer
	if operation == "consume" {
		kind = trace.SpanKindConsumer
	}
	ctx, span := tracer.Start(ctx, operation+" "+topic, trace.WithSpanKind(kind))
	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", "kafka"),
		attribute.String("messaging.destination.name", topic),
		attribute.String("messaging.operation.name", operation),
	}
	if msgCount > 0 {
		attrs = append(attrs, semconv.MessagingBatchMessageCount(msgCount))
	}
	if serviceName != "" {
		attrs = append(attrs, semconv.ServiceName(serviceName))
	}
	span.SetAttributes(attrs...)
	return ctx, span
}

// finishSpan 把 err 映射到 span status（Otel 官方推荐做法）。
func finishSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}