# shared/tracing · OpenTelemetry 全链路追踪

11 个 Go 服务共用 `shared/tracing` 包：初始化全局 TracerProvider、创建 Span、注入 / 提取 W3C TraceContext（`traceparent` header）。

## 用法（10 行）

```go
// cmd/main.go
shutdown, err := tracing.InitTracer("auth-service", cfg.OTLPEndpoint)
if err != nil {
    log.Fatalf("init tracer: %v", err)
}
defer func() { _ = shutdown(context.Background()) }()

// 业务代码
ctx, span := tracing.StartSpan(ctx, "auth.login")
defer span.End()

// HTTP 客户端：注入 traceparent
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
tracing.Inject(ctx, tracing.HeaderCarrier(req.Header))

// HTTP 服务端：从 header 提取 traceparent
ctx = tracing.Extract(r.Context(), tracing.HeaderCarrier(r.Header))
```

## 配置

`config/<svc>.yaml` 加可选段（endpoint 留空 → Noop，零开销不导出）：

```yaml
tracing:
  otlp_endpoint: "otel-collector:4318"   # 留空 → 退化 Noop
```

环境变量覆盖：`DOCTORS_<SVC>_TRACING_OTLP_ENDPOINT=otel-collector:4318`（与 `shared/config` 一致）。

## 行为约定

- `endpoint` 为空 → `NoopTracerProvider`，所有 `span.IsRecording() == false`。
- `endpoint` 非空 → OTLP HTTP exporter + BatchSpanProcessor（5s 间隔、512 batch size）。
- 默认 `AlwaysSample`；生产建议改 `TraceIDRatioBased(0.1)`。
- 资源属性：`service.name=<svc>`、`service.version=v1.0.0` + process / host detector。
- Propagator：`TraceContext` + `Baggage`（W3C 标准）。
- Exporter 协议：OTLP/HTTP（4318 端口），默认 `insecure`；HTTPS 改 `https://` 前缀即生效。

## 与 `shared/logger` 协作

OTel TraceID 与 `logger.TraceIDFrom(ctx)` 是两个独立字段；如需把 TraceID 写进日志，可在 middleware 里读 `trace.SpanContextFromContext(ctx).TraceID()` 注入 `logger.WithTrace(ctx, ...)`。本期不强制，仅留作 §26 增量。