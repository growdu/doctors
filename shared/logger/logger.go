// Package logger 提供全局结构化日志与 trace_id 上下文传递。
//
// 设计要点：
//   - 默认 JSON 输出到 stdout，便于容器日志采集。
//   - 全局 L() 返回 *zap.Logger，可在任何地方直接调用。
//   - WithTrace / FromContext 让 trace_id 沿 context 传递，业务代码无侵入。
//   - FromContext 同时读取 OTel SpanContext，自动注入 otel_trace_id /
//     otel_span_id 字段，实现「日志↔trace」关联（Jaeger UI 一键跳转）。
//   - 测试可注入 observer 或 writer。
package logger

import (
	"context"
	"io"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey int

const (
	ctxTraceID ctxKey = iota
)

var (
	mu    sync.RWMutex
	base  = zap.NewProductionConfig()
	stdout io.Writer = os.Stdout
	global *zap.Logger
)

func init() {
	base.Encoding = "json"
	base.EncoderConfig.TimeKey = "ts"
	base.EncoderConfig.LevelKey = "level"
	base.EncoderConfig.MessageKey = "msg"
	base.EncoderConfig.CallerKey = "caller"
	base.OutputPaths = []string{"stdout"}
	base.ErrorOutputPaths = []string{"stderr"}
	buildGlobal()
}

// buildGlobal 用当前 stdout 重建 logger。
func buildGlobal() {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(base.EncoderConfig),
		zapcore.AddSync(stdout),
		zapcore.InfoLevel,
	)
	global = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

// L 返回全局 logger。
func L() *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

// WithTrace 把 trace_id 放入 context。
func WithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxTraceID, traceID)
}

// TraceIDFrom 读取 ctx 中的 trace_id；缺失返回 ""。
func TraceIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ctxTraceID).(string); ok {
		return v
	}
	return ""
}

// FromContext 返回绑定了 trace_id 字段的 logger（无侵入）。
//
// 若 ctx 携带 OTel SpanContext（HasTraceID），则额外追加：
//   - otel_trace_id：OTel 32-hex TraceID（与 Jaeger traceId 一致）
//   - otel_span_id：OTel 16-hex SpanID
//
// 这两个字段是日志↔trace 关联的锚点：Jaeger UI 可按 traceId 检索，
// 反之日志平台也可按 otel_trace_id 反查到整条 trace 的所有 span。
func FromContext(ctx context.Context) *zap.Logger {
	l := L()
	if id := TraceIDFrom(ctx); id != "" {
		l = l.With(zap.String("trace_id", id))
	}
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		l = l.With(
			zap.String("otel_trace_id", sc.TraceID().String()),
			zap.String("otel_span_id", sc.SpanID().String()),
		)
	}
	return l
}

// SetForTest 替换全局 logger，仅测试用。
func SetForTest(l *zap.Logger) {
	mu.Lock()
	defer mu.Unlock()
	global = l
}

// ResetForTest 恢复默认全局 logger。
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	buildGlobal()
}

// SetWriterForTest 替换 stdout writer，返回原 writer。仅测试用。
func SetWriterForTest(w io.Writer) io.Writer {
	mu.Lock()
	defer mu.Unlock()
	prev := stdout
	stdout = w
	buildGlobal()
	return prev
}

// SetLevel 动态调整日志级别。
func SetLevel(level zapcore.Level) {
	mu.Lock()
	defer mu.Unlock()
	base.Level = zap.NewAtomicLevelAt(level)
	buildGlobal()
}