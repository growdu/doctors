package tracing_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/growdu/doctors/shared/tracing"
)

// withTestProvider 替换全局 TracerProvider 为 in-memory exporter，并返回其快照读取器。
// 用法：defer restore() 在测试结束后恢复 otel 全局（避免污染其它测试）。
func withTestProvider(t *testing.T) (*tracetest.InMemoryExporter, *sdktrace.TracerProvider, func()) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	prev := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	restore := func() {
		otel.SetTracerProvider(prev)
		otel.SetTextMapPropagator(prevProp)
		_ = tp.Shutdown(context.Background())
	}
	return exp, tp, restore
}

func TestInitTracer_EmptyEndpoint_UsesNoop(t *testing.T) {
	shutdown, err := tracing.InitTracer("test-svc", "")
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	defer func() { _ = shutdown(context.Background()) }()

	// Noop provider：span 不应被记录。
	_, span := tracing.StartSpan(context.Background(), "noop-span")
	assert.False(t, span.IsRecording(), "noop provider 应该 IsRecording=false")
	span.End()
}

func TestInitTracer_ValidEndpoint_RegistersProvider(t *testing.T) {
	// 用 httptest 起一个假 OTLP 接收端（/v1/traces 返回 200）。
	// 注意：otlptracehttp 在初始化时会发 metadata，但首次 batch 由 batch timeout 触发。
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/traces" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	shutdown, err := tracing.InitTracer("auth-service", ts.URL)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	defer func() { _ = shutdown(context.Background()) }()

	// 创建 span：应该 IsRecording=true。
	_, span := tracing.StartSpan(context.Background(), "auth.login")
	assert.True(t, span.IsRecording(), "endpoint 配齐后 span 必须 IsRecording")
	span.End()
}

func TestInitTracer_StripScheme_HTTP(t *testing.T) {
	// endpoint 含 http:// / https:// 前缀也能被正确处理（不报错）。
	shutdown, err := tracing.InitTracer("svc", "http://127.0.0.1:65535")
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	defer func() { _ = shutdown(context.Background()) }()
}

func TestInitTracer_StripScheme_HTTPS(t *testing.T) {
	shutdown, err := tracing.InitTracer("svc", "https://127.0.0.1:65535")
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	defer func() { _ = shutdown(context.Background()) }()
}

func TestInitTracer_ShutdownIsIdempotent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	shutdown, err := tracing.InitTracer("svc", ts.URL)
	require.NoError(t, err)
	// 多次调用 shutdown 不应 panic（实现可能返回 error，但必须不 panic）。
	require.NotPanics(t, func() {
		_ = shutdown(context.Background())
		_ = shutdown(context.Background())
	})
}

func TestStartSpan_PropagatesContext(t *testing.T) {
	exp, _, restore := withTestProvider(t)
	defer restore()

	ctx, span := tracing.StartSpan(context.Background(), "outer")
	innerCtx, innerSpan := tracing.StartSpan(ctx, "inner")
	innerSpan.End()
	span.End()

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 2)

	// inner span 的 parent 应该指向 outer span。
	var outerSC, innerParentSC trace.SpanContext
	for _, s := range spans {
		if s.Name == "outer" {
			outerSC = s.SpanContext
		}
		if s.Name == "inner" {
			innerParentSC = s.Parent
		}
	}
	require.True(t, outerSC.IsValid())
	require.True(t, innerParentSC.IsValid())
	assert.Equal(t, outerSC.TraceID(), innerParentSC.TraceID(), "子 span 应继承父 TraceID")
	assert.Equal(t, outerSC.SpanID(), innerParentSC.SpanID(), "子 span.Parent 应该是父 SpanID")
	_ = innerCtx // 用 innerCtx 仅做触发性引用（防止 unused）
}

func TestInject_Extract_RoundTrip(t *testing.T) {
	exp, _, restore := withTestProvider(t)
	defer restore()

	// 1. 客户端：StartSpan → Inject 到 header。
	ctx, span := tracing.StartSpan(context.Background(), "client.req")
	header := http.Header{}
	tracing.Inject(ctx, tracing.HeaderCarrier(header))
	span.End()

	tpHeader := header.Get("traceparent")
	require.NotEmpty(t, tpHeader, "traceparent header 必须被写入")

	// 2. 服务端：Extract header → StartSpan（新 span 应继承 TraceID）。
	serverCtx := tracing.Extract(context.Background(), tracing.HeaderCarrier(header))
	_, serverSpan := tracing.StartSpan(serverCtx, "server.handle")
	serverSpan.End()

	spans := exp.GetSpans()
	require.GreaterOrEqual(t, len(spans), 2)

	// 验证 traceparent 格式：00-<32hex>-<16hex>-<2hex>
	parts := splitTraceparent(tpHeader)
	require.Len(t, parts, 4)
	assert.Equal(t, "00", parts[0])
	assert.Len(t, parts[1], 32)
	assert.Len(t, parts[2], 16)
	assert.Len(t, parts[3], 2)
}

func TestHeaderCarrier_ImplementsTextMapCarrier(t *testing.T) {
	h := http.Header{}
	c := tracing.HeaderCarrier(h)
	c.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	assert.Equal(t, "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01", c.Get("traceparent"))

	keys := c.Keys()
	// http.Header 规范化 key 首字母大写；只需验证我们设置的 key 在列表中（任意大小写）。
	found := false
	for _, k := range keys {
		if strings.EqualFold(k, "traceparent") {
			found = true
			break
		}
	}
	assert.True(t, found, "traceparent key 应该在 Keys() 中")
}

// splitTraceparent 简单拆分 W3C traceparent（00-<traceid>-<spanid>-<flags>）。
func splitTraceparent(tp string) []string {
	out := make([]string, 0, 4)
	cur := ""
	for _, r := range tp {
		if r == '-' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}