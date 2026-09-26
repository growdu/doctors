// Package tracing 提供全链路追踪（OpenTelemetry）初始化、Span 创建、W3C TraceContext 透传。
//
// 设计要点：
//   - 单一全局 TracerProvider：避免每个服务重复初始化；通过 otel.Tracer(name) 取 tracer。
//   - OTLP HTTP exporter：endpoint 为空时退化为 NoopTracerProvider（dev / 单测友好）。
//   - W3C TraceContext propagator：注入 `traceparent` / `tracestate` HTTP 头实现跨服务透传。
//   - 资源属性：service.name = <serviceName>，service.version = 配置注入（默认 "dev"）。
//   - 生产采样：默认 AlwaysSample；通过 WithSamplingRatio(0.1) + ParentBased 启用跨服务 10% 采样。
//   - 优雅停机：返回 Shutdown(ctx) 闭包由 main 退出时调用。
//   - 与 shared/db 集成（WithPgxPool option）：
//     自动注册 otelpgx tracer 到 pgxpool.Config，让所有 SQL 自动写 span。
package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
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

// config 聚合 InitTracer 可选项的内部状态。
type config struct {
	samplingRatio  float64 // 0~1，0 表示 NeverSample，1 表示 AlwaysSample
	serviceVersion string  // 资源属性 service.version
}

// Option 允许调用方在 InitTracer 时自定义配置。
type Option func(*config)

// WithSamplingRatio 设置全局采样率（0~1）。
//
//   - ratio == 0   → NeverSample（不采样，dev/低负载调试可用）。
//   - ratio == 1.0 → AlwaysSample（默认，dev / 集成测试友好）。
//   - 0 < ratio < 1 → ParentBased(TraceIDRatioBased(ratio))。
//
// 生产推荐 0.1（10% 采样），既保留完整调用链又不显著影响后端存储/带宽。
func WithSamplingRatio(ratio float64) Option {
	return func(c *config) {
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		c.samplingRatio = ratio
	}
}

// WithServiceVersion 设置资源属性 service.version。
//
// 默认 "dev"；CI / 生产通过此 Option 注入构建版本号（如 "v1.3.0"）。
func WithServiceVersion(version string) Option {
	return func(c *config) {
		v := strings.TrimSpace(version)
		if v != "" {
			c.serviceVersion = v
		}
	}
}

// WithPgxPool 把 otelpgx tracer 注册到给定的 pgxpool.Config。
//
// 11 个服务的 cmd/main.go 在 InitTracer 后调用：
//
//	tracing.InitTracer("wallet-service", cfg.Tracing.OTLPEndpoint, ...)
//	pcfg, _ := pgxpool.ParseConfig(cfg.DB.DSN)
//	tracing.WithPgxPool(pcfg)   // 注入 otelpgx tracer
//	pool := pgxpool.NewWithConfig(ctx, pcfg)
//
// 让所有 SQL 自动开 span（无需业务代码手动 StartSpan）。
// nil 入参时 noop（避免 dev 环境 cfg.DB.DSN 空时 panic）。
func WithPgxPool(pcfg *pgxpool.Config) {
	if pcfg == nil {
		return
	}
	pcfg.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerProvider(otel.GetTracerProvider()),
	)
}

// InitTracer 初始化全局 TracerProvider + W3C TraceContext propagator。
//
//   - serviceName：资源属性 service.name（必填，业务名 / 服务名）。
//   - otlpEndpoint：OTLP HTTP 接收端地址（如 "otel-collector:4318" 或 "http://127.0.0.1:4318"）。
//     为空字符串时退化为 NoopTracerProvider（不导出 span，便于本地开发）。
//   - opts：可选配置（如 WithSamplingRatio / WithServiceVersion）。
//
// 返回 Shutdown 闭包，main 退出前必须调用：
//
//	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
//	defer stop()
//	defer shutdown(context.Background())
//
// 注意：多次调用 InitTracer 会覆盖全局 Provider；典型用法为 main 启动时调用一次。
func InitTracer(serviceName, otlpEndpoint string, opts ...Option) (Shutdown, error) {
	// 0. 解析可选项（默认值在内部 config 结构体上）。
	cfg := &config{
		samplingRatio:  1.0,
		serviceVersion: "dev",
	}
	for _, o := range opts {
		o(cfg)
	}

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
	exporterOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(stripScheme(otlpEndpoint)),
		// 默认 insecure（K8s 内网 / dev）；生产建议开 TLS 并改 WithTLSClientConfig。
		otlptracehttp.WithInsecure(),
	}
	if strings.HasPrefix(strings.ToLower(otlpEndpoint), "https://") {
		exporterOpts = append(exporterOpts, otlptracehttp.WithEndpointURL(otlpEndpoint))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptrace.New(ctx, otlptracehttp.NewClient(exporterOpts...))
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	// 4. 构建 TracerProvider + resource（service.name + service.version 注入）。
	// 注意：不显式调用 WithSchemaURL，让 runtime detector 自动注入（避免 semconv 版本冲突）。
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(cfg.serviceVersion),
		),
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	// 5. 选择 sampler：
	//   - ratio == 0   → NeverSample（不采样）
	//   - ratio == 1.0 → AlwaysSample（默认，dev / 集成测试）
	//   - 0 < ratio < 1 → ParentBased(TraceIDRatioBased(ratio))：
	//       本服务根 span 按 TraceID 比例采样；上游 span 已采样则跟随（保留链路完整）。
	var sampler sdktrace.Sampler
	switch {
	case cfg.samplingRatio <= 0:
		sampler = sdktrace.NeverSample()
	case cfg.samplingRatio >= 1.0:
		sampler = sdktrace.AlwaysSample()
	default:
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.samplingRatio))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)

	// 6. 返回 shutdown：flush 缓冲 + 关闭 exporter。
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