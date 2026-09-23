package logger_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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