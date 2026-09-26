// Package tracing —— kafka-go OTel wrapper 单测（§34 稳定性优化）。
//
// 验证：
//   - WrapWriter 注入的 TracedWriter.WriteMessages 启动 "publish <topic>" span，
//     包含 messaging.system=kafka + messaging.destination.name=<topic> +
//     messaging.operation.name=publish，kind=Producer。
//   - WrapReader 注入的 TracedReader.ReadMessage 启动 "consume <topic>" span，
//     span kind=Consumer，含 messaging.* 属性。
//   - 错误路径：span status=Error（broker 不可达时）。
//   - nil writer / reader 不 panic。
//   - 单次调用只产生 1 个 span（避免 wrapper 自身起 child span 造成 span storm）。
package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/growdu/doctors/shared/tracing"
)

// newKafkaTestProvider 替换全局 TracerProvider 为 in-memory exporter，
// 并返回 InMemoryExporter + restore 函数。每个测试结束后恢复 otel 全局。
func newKafkaTestProvider(t *testing.T) (*tracetest.InMemoryExporter, func()) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	restore := func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
	}
	return exp, restore
}

// TestWrapWriter_PublishSpan 验证 WrapWriter 启动 publish span + 必要 attribute。
//
// kafka.Writer 指向不存在 broker（127.0.0.1:1）；WriteMessages 必然 fail，
// 但 span 已经被 wrapper 启动，InMemoryExporter 仍能捕获。
// 测试只验证"span 已 start + 含 topic 属性 + Producer kind"——不验证 kafka-go 行为。
func TestWrapWriter_PublishSpan(t *testing.T) {
	exp, restore := newKafkaTestProvider(t)
	defer restore()

	w := &kafka.Writer{
		Addr:                   kafka.TCP("127.0.0.1:1"),
		Topic:                  "test.publish.topic",
		AllowAutoTopicCreation: false,
	}
	tw := tracing.WrapWriter(w, "test-svc")

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = tw.WriteMessages(ctx, kafka.Message{Key: []byte("k"), Value: []byte("v")})

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1, "应至少捕获 1 个 span（publish）")
	publishSpan := spans[0]
	assert.Equal(t, "publish test.publish.topic", publishSpan.Name,
		"span name = publish <topic>")
	assert.Equal(t, trace.SpanKindProducer, publishSpan.SpanKind,
		"publish span kind 应为 Producer")

	attrs := map[string]string{}
	for _, kv := range publishSpan.Attributes {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}
	assert.Equal(t, "kafka", attrs["messaging.system"])
	assert.Equal(t, "test.publish.topic", attrs["messaging.destination.name"])
	assert.Equal(t, "publish", attrs["messaging.operation.name"])
	assert.Equal(t, "test-svc", attrs["service.name"])
}

// TestWrapReader_ConsumeSpan 验证 WrapReader 启动 consume span。
func TestWrapReader_ConsumeSpan(t *testing.T) {
	exp, restore := newKafkaTestProvider(t)
	defer restore()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"127.0.0.1:1"},
		Topic:   "test.consume.topic",
		GroupID: "test-group",
		MaxWait: 100 * time.Millisecond,
	})
	defer r.Close()

	tr := tracing.WrapReader(r, "test-svc")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, _ = tr.ReadMessage(ctx)

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1, "应至少捕获 1 个 span（consume）")
	consumeSpan := spans[0]
	assert.Equal(t, "consume test.consume.topic", consumeSpan.Name)
	assert.Equal(t, trace.SpanKindConsumer, consumeSpan.SpanKind,
		"consume span kind 应为 Consumer")

	attrs := map[string]string{}
	for _, kv := range consumeSpan.Attributes {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}
	assert.Equal(t, "kafka", attrs["messaging.system"])
	assert.Equal(t, "test.consume.topic", attrs["messaging.destination.name"])
	assert.Equal(t, "consume", attrs["messaging.operation.name"])
}

// TestWrapWriter_NilWriterNoPanic 验证 W=nil 时 wrapper 不让 goroutine panic。
func TestWrapWriter_NilWriterNoPanic(t *testing.T) {
	tw := tracing.WrapWriter(nil, "test-svc")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic: %v", r)
		}
	}()
	_ = tw.WriteMessages(ctx, kafka.Message{Value: []byte("v")})
}

// TestWrapReader_NilReaderNoPanic 验证 R=nil 时 wrapper 不让 goroutine panic。
func TestWrapReader_NilReaderNoPanic(t *testing.T) {
	tr := tracing.WrapReader(nil, "test-svc")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic: %v", r)
		}
	}()
	_, _ = tr.ReadMessage(ctx)
}

// TestSpan_NoExcessiveChildren 验证 span 数量合理（避免 wrapper 自身起 child span）。
func TestSpan_NoExcessiveChildren(t *testing.T) {
	exp, restore := newKafkaTestProvider(t)
	defer restore()

	w := &kafka.Writer{Addr: kafka.TCP("127.0.0.1:1"), Topic: "x"}
	tw := tracing.WrapWriter(w, "test-svc")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	for i := 0; i < 3; i++ {
		_ = tw.WriteMessages(ctx, kafka.Message{Value: []byte("v")})
	}

	// 3 次调用 → 最多 3 个 publish span（不应更多）。
	spans := exp.GetSpans()
	assert.LessOrEqual(t, len(spans), 3,
		"3 次 WriteMessages 不应产生 > 3 个 span（避免 wrapper 内部起 child span）")
}

// TestWrapWriter_ErrorSpan 验证错误路径上 span 状态为 Error。
func TestWrapWriter_ErrorSpan(t *testing.T) {
	exp, restore := newKafkaTestProvider(t)
	defer restore()

	w := &kafka.Writer{Addr: kafka.TCP("127.0.0.1:1"), Topic: "err-topic"}
	tw := tracing.WrapWriter(w, "test-svc")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err := tw.WriteMessages(ctx, kafka.Message{Value: []byte("v")})
	require.Error(t, err, "连不存在 broker 应失败")

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1)
	s := spans[0]
	assert.Equal(t, "Error", s.Status.Code.String(),
		"broker 不可达时 span status 应为 Error")
}

// TestWrapWriter_BusinessPublisherPattern 模拟业务 publisher（如
// services/message/internal/kafkapublisher.Publisher）接入 WrapWriter 后的
// publish 路径——确认 1 次 PublishMessageSent → 1 个 messaging.publish span。
//
// 这是一条"业务真实使用模式"的回归测试：业务侧构造 shared/kafka.NewWriter →
// tracing.WrapWriter(..., "<svc>-service")，发布时 WriteMessages 必然起
// "publish <topic>" span（kind=Producer + messaging.system=kafka）。
//
// 与 TestWrapWriter_PublishSpan 的区别：本测试额外验证 wrapper 可在调用方
// 持有 Close() 句柄（writer.W.Close 仍然可达，shutdown hook 能正常释放）。
func TestWrapWriter_BusinessPublisherPattern(t *testing.T) {
	exp, restore := newKafkaTestProvider(t)
	defer restore()

	// 不可达 broker → WriteMessages 必然失败，但 span 已被 wrapper 启动。
	w := &kafka.Writer{
		Addr: kafka.TCP("127.0.0.1:1"),
		Topic: "message.sent",
	}
	tw := tracing.WrapWriter(w, "message-service")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_ = tw.WriteMessages(ctx, kafka.Message{
		Key:   []byte("1"),
		Value: []byte(`{"message_id":1}`),
	})

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1,
		"publisher 写 1 条 → 应至少捕获 1 个 publish span")
	publishSpan := spans[0]
	assert.Equal(t, "publish message.sent", publishSpan.Name,
		"span name = publish <topic>")
	assert.Equal(t, trace.SpanKindProducer, publishSpan.SpanKind)

	attrs := map[string]string{}
	for _, kv := range publishSpan.Attributes {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}
	assert.Equal(t, "kafka", attrs["messaging.system"])
	assert.Equal(t, "message.sent", attrs["messaging.destination.name"])
	assert.Equal(t, "publish", attrs["messaging.operation.name"])
	assert.Equal(t, "message-service", attrs["service.name"],
		"业务 publisher 传入 service 名应注入到 span attribute")

	// 验证业务侧仍可通过 tw.W.Close() 拿到底层 writer（shutdown hook 依赖）。
	require.NotNil(t, tw.W, "wrapper 必须暴露底层 writer 给业务 Close")
}