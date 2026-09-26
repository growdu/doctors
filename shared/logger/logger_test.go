package logger_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/growdu/doctors/shared/logger"
)

// withObserved 替换全局 logger 为可观测实例。
func withObserved(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, obs := observer.New(zapcore.DebugLevel)
	logger.SetForTest(zap.New(core))
	t.Cleanup(logger.ResetForTest)
	return obs
}

func TestL_ReturnsLogger(t *testing.T) {
	l := logger.L()
	require.NotNil(t, l)
	assert.NotPanics(t, func() { l.Info("ok") })
}

func TestL_RecordsMessage(t *testing.T) {
	obs := withObserved(t)

	logger.L().Info("hello", zap.String("trace_id", "abc"), zap.Int("latency_ms", 42))

	entries := obs.AllUntimed()
	require.Len(t, entries, 1)
	assert.Equal(t, "hello", entries[0].Message)
	assert.NotEmpty(t, entries[0].ContextMap(), "应当带结构化字段")
}

func TestL_EmitsJSONToStdout(t *testing.T) {
	var buf strings.Builder
	prev := logger.SetWriterForTest(&buf)
	t.Cleanup(func() { logger.SetWriterForTest(prev) })

	logger.L().Info("to-stdout", zap.String("k", "v"))

	out := buf.String()
	assert.Contains(t, out, `"msg":"to-stdout"`)
	assert.Contains(t, out, `"k":"v"`)
}

func TestWithTrace_AddsTraceIDToContext(t *testing.T) {
	ctx := logger.WithTrace(context.Background(), "trace-xyz")
	assert.Equal(t, "trace-xyz", logger.TraceIDFrom(ctx))
}

func TestTraceIDFrom_MissingReturnsEmpty(t *testing.T) {
	assert.Equal(t, "", logger.TraceIDFrom(context.Background()))
}

func TestFromContext_LoggerHasTraceIDField(t *testing.T) {
	obs := withObserved(t)
	ctx := logger.WithTrace(context.Background(), "ctx-trace")

	logger.FromContext(ctx).Info("hi")

	entries := obs.AllUntimed()
	require.Len(t, entries, 1)
	m := entries[0].ContextMap()
	assert.Equal(t, "ctx-trace", m["trace_id"])
}

// TestFromContext_OTelSpanContext 验证 ctx 携带 OTel SpanContext 时，
// FromContext 返回的 logger 会自动注入 otel_trace_id / otel_span_id。
//
// 用 trace.NewSpanContext 直接构造一个手工 SpanContext（不依赖全局 SDK），
// 避免在单测里启动 TracerProvider；HasTraceID=true 即可触发注入分支。
func TestFromContext_OTelSpanContext(t *testing.T) {
	obs := withObserved(t)

	// 32-hex TraceID + 16-hex SpanID（OTel 标准格式）
	var tid trace.TraceID
	var sid trace.SpanID
	copy(tid[:], []byte("0123456789abcdef0123456789abcdef"))
	copy(sid[:], []byte("0123456789abcdef"))
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
		Remote:     false,
	})
	require.True(t, sc.IsValid(), "手工 SpanContext 应当合法")
	require.True(t, sc.HasTraceID())

	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	logger.FromContext(ctx).Info("otel-bound")

	entries := obs.AllUntimed()
	require.Len(t, entries, 1)
	m := entries[0].ContextMap()
	assert.Equal(t, tid.String(), m["otel_trace_id"], "otel_trace_id 应来自 SpanContext")
	assert.Equal(t, sid.String(), m["otel_span_id"], "otel_span_id 应来自 SpanContext")
}

// TestFromContext_NoSpanContext 验证 ctx 不带 SpanContext 时，
// 不会输出空 otel_trace_id 字段（避免噪声）。
func TestFromContext_NoSpanContext(t *testing.T) {
	obs := withObserved(t)

	logger.FromContext(context.Background()).Info("plain")

	entries := obs.AllUntimed()
	require.Len(t, entries, 1)
	m := entries[0].ContextMap()
	_, hasTrace := m["otel_trace_id"]
	_, hasSpan := m["otel_span_id"]
	assert.False(t, hasTrace, "无 SpanContext 时不应输出 otel_trace_id")
	assert.False(t, hasSpan, "无 SpanContext 时不应输出 otel_span_id")
}

// TestFromContext_BothTraceIDAndOTel 验证当 ctx 同时携带 WithTrace 与 OTel
// SpanContext 时，两者字段共存。
func TestFromContext_BothTraceIDAndOTel(t *testing.T) {
	obs := withObserved(t)

	var tid trace.TraceID
	var sid trace.SpanID
	copy(tid[:], []byte("aabbccddeeff00112233445566778899"))
	copy(sid[:], []byte("aabbccddeeff0011"))
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := logger.WithTrace(context.Background(), "manual-trace-id")
	ctx = trace.ContextWithSpanContext(ctx, sc)

	logger.FromContext(ctx).Info("both")

	entries := obs.AllUntimed()
	require.Len(t, entries, 1)
	m := entries[0].ContextMap()
	assert.Equal(t, "manual-trace-id", m["trace_id"])
	assert.Equal(t, tid.String(), m["otel_trace_id"])
	assert.Equal(t, sid.String(), m["otel_span_id"])
}