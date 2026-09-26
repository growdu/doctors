// Package tracing 提供全链路追踪（OpenTelemetry）初始化、Span 创建、W3C TraceContext 透传。
//
// 设计要点：
//   - 单一全局 TracerProvider：避免每个服务重复初始化；通过 otel.Tracer(name) 取 tracer。
//   - OTLP HTTP exporter：endpoint 为空时退化为 NoopTracerProvider（dev / 单测友好）。
//   - W3C TraceContext propagator：注入 `traceparent` / `tracestate` HTTP 头实现跨服务透传。
//   - 资源属性：service.name = <serviceName>，service.version 留空（CI 阶段注入）。
//   - 优雅停机：返回 Shutdown(ctx) 闭包由 main 退出时调用。
package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// tracerName 是本包内 tracer 的固定名；其它包用 otel.Tracer("<package>") 取各自 tracer。
const tracerName = "github.com/growdu/doctors/shared/tracing"

// Shutdown 是 TracerProvider 的优雅停机函数签名。
type Shutdown func(context.Context) error

// InitTracer 初始化全局 TracerProvider + W3C TraceContext propagator。
//
//   - serviceName：资源属性 service.name（必填，业务名 / 服务名）。
//   - otlpEndpoint：OTLP HTTP 接收端地址（如 "otel-collector:4318" 或 "http://127.0.0.1:4318"）。
//     为空字符串时退化为 NoopTracerProvider（不导出 span，便于本地开发）。
//
// 返回 Shutdown 闭包，main 退出前必须调用：
//
//	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
//	defer stop()
//	defer shutdown(context.Background())
//
// 注意：多次调用 InitTracer 会覆盖全局 Provider；典型用法为 main 启动时调用一次。
func InitTracer(serviceName, otlpEndpoint string) (Shutdown, error) {
	// 1. 注册 W3C TraceContext + Baggage propagator（HTTP 透传 traceparent）。
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 2. endpoint 为空 → Noop（dev / 单测）。
	if strings.TrimSpace(otlpEndpoint) == "" {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	// 3. 构建 OTLP HTTP exporter。
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(stripScheme(otlpEndpoint)),
		// 默认 insecure（K8s 内网 / dev）；生产建议开 TLS 并改 WithTLSClientConfig。
		otlptracehttp.WithInsecure(),
	}
	if strings.HasPrefix(strings.ToLower(otlpEndpoint), "https://") {
		opts = append(opts, otlptracehttp.WithEndpointURL(otlpEndpoint))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	// 4. 构建 TracerProvider + resource（service.name 注入）。
	// 注意：不显式调用 WithSchemaURL，让 runtime detector 自动注入（避免 semconv 版本冲突）。
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("v1.0.0"),
		),
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		// 默认 AlwaysSample；v1 dev 阶段全采样；生产可改 TraceIDRatioBased(0.1)。
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	// 5. 返回 shutdown：flush 缓冲 + 关闭 exporter。
	return func(_ context.Context) error {
		// 用独立超时，避免 ctx 已被 cancel 时 flush 静默失败。
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return tp.Shutdown(flushCtx)
	}, nil
}

// StartSpan 在给定 ctx 上启动 span；返回新 ctx 与 span。span 结束时由调用方 defer span.End()。
//
// 这是 OTel 标准 trace.Tracer.Start 的薄包装；保留独立函数便于：
//   - 业务代码统一加 semconv 标签（DB / HTTP / RPC system）。
//   - 未来加全局 SpanProcessor / SpanFilter 集中入口。
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, name, opts...)
}

// Inject 把 ctx 中的当前 span 注入到 carrier（通常是 http.Header），返回 carrier。
// 用途：客户端发请求前调用，把 traceparent / tracestate 写入 header。
func Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// Extract 从 carrier（通常是 http.Header）读 traceparent，构造新 ctx。
// 用途：服务端收到请求时调用，从 header 还原上游 span context。
func Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// HeaderCarrier 把 http.Header 适配为 propagation.TextMapCarrier。
//
// 用法：
//
//	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
//	tracing.HeaderCarrier(req.Header).Inject(ctx)
//	// server: ctx = tracing.Extract(r.Context(), tracing.HeaderCarrier(r.Header))
type HeaderCarrier http.Header

// Get / Set / Keys 实现 propagation.TextMapCarrier 接口。
func (h HeaderCarrier) Get(key string) string { return http.Header(h).Get(key) }
func (h HeaderCarrier) Set(key, value string) { http.Header(h).Set(key, value) }
func (h HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

// stripScheme 去掉 http(s):// 前缀；otlptracehttp.WithEndpoint 期望纯 host:port。
func stripScheme(endpoint string) string {
	ep := endpoint
	ep = strings.TrimPrefix(ep, "http://")
	ep = strings.TrimPrefix(ep, "https://")
	return ep
}