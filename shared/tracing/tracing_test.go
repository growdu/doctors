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
	"go.opentelemetry.io/otel/attribute"
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

// TestInitTracer_AppliesSamplingRatio 验证 WithSamplingRatio(0) → span 不被采样；
// ratio=0 应映射到 NeverSample，span.IsRecording() == false。
func TestInitTracer_AppliesSamplingRatio(t *testing.T) {
	// httptest 提供一个假 OTLP endpoint（让代码进入"非 Noop"分支，触发 sampler 设置）。
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/traces" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	shutdown, err := tracing.InitTracer("svc-ratio0", ts.URL,
		tracing.WithSamplingRatio(0),
	)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	defer func() { _ = shutdown(context.Background()) }()

	// ratio=0 → NeverSample → span 不录制。
	_, span := tracing.StartSpan(context.Background(), "should-not-record")
	assert.False(t, span.IsRecording(), "ratio=0 应该映射为 NeverSample，span 不录制")
	span.End()
}

// TestInitTracer_AppliesServiceVersion 验证 WithServiceVersion("v9.9.9") 后，
// 通过注册到 SDK provider 的 InMemoryExporter 抓到的 span resource 上能看到 service.version="v9.9.9"。
func TestInitTracer_AppliesServiceVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/traces" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	shutdown, err := tracing.InitTracer("svc-version", ts.URL,
		tracing.WithServiceVersion("v9.9.9"),
	)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	defer func() { _ = shutdown(context.Background()) }()

	// 取得全局 SDK provider，并外挂一个 InMemoryExporter 抓取 span（用于读 resource）。
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok, "全局 TracerProvider 应为 SDK provider")
	memExp := tracetest.NewInMemoryExporter()
	tp.RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(memExp))

	// 起一个 span 并结束，让 exporter 拿到 SpanStub。
	_, span := tracing.StartSpan(context.Background(), "probe-version")
	require.True(t, span.IsRecording(), "默认 AlwaysSample，span 必须 IsRecording")
	span.End()

	// 强制 flush（避免 BatchSpanProcessor 时延）。
	require.NoError(t, tp.ForceFlush(context.Background()))

	captured := memExp.GetSpans()
	require.NotEmpty(t, captured, "InMemoryExporter 必须抓到 span")

	// 取最后一个 span 的 resource 属性。
	res := captured[len(captured)-1].Resource
	require.NotNil(t, res, "span 必须带 resource")
	resAttrs := res.Attributes()

	// 查 service.version 属性。
	got, found := findStringAttr(resAttrs, "service.version")
	require.True(t, found, "resource 应包含 service.version 属性")
	assert.Equal(t, "v9.9.9", got, "service.version 应等于注入的版本")

	// 同时 service.name 也应正确（验证 WithServiceVersion 不破坏 service.name 注入）。
	gotName, foundName := findStringAttr(resAttrs, "service.name")
	require.True(t, foundName)
	assert.Equal(t, "svc-version", gotName)
}

// findStringAttr 从 attribute KV 列表中查找指定 key 的 string 值；找不到返回 ("", false)。
func findStringAttr(kvs []attribute.KeyValue, keyStr string) (string, bool) {
	for _, kv := range kvs {
		if string(kv.Key) == keyStr && kv.Value.Type() == attribute.STRING {
			return kv.Value.AsString(), true
		}
	}
	return "", false
}