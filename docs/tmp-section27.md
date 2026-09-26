---
## 27. OTel↔日志关联 + Jaeger collector + golangci-lint v2（v1.3 生产化优化）

**目标**：让 OTel 全链路真正"可观测"——日志带 trace_id（跳 Jaeger）+ Jaeger collector 接入 + lint 升级。

**3 个 commit**：

| commit | 内容 | 文件 |
| :-- | :-- | :--: |
| `2789242` | `feat(logger)` OTel↔日志 trace_id 关联（13 files +132/-39） | 13 |
| `0e928e1` | `ops(jaeger)` docker-compose 接入 jaeger all-in-one（2 files +99/-6） | 2 |
| `e26e7e2` | `chore(lint)` golangci-lint v2 配置升级 + 5 新 linter（2 files +291/-12） | 2 |

**Commit 1：OTel↔日志关联**

`shared/logger/logger.go` 的 `FromContext(ctx)` 增强：

```go
func FromContext(ctx context.Context) *zap.Logger {
    l := L()
    if id := TraceIDFrom(ctx); id != "" {
        l = l.With(zap.String("trace_id", id))
    }
    // 新增：OTel SpanContext → otel_trace_id / otel_span_id
    if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
        l = l.With(
            zap.String("otel_trace_id", sc.TraceID().String()),
            zap.String("otel_span_id", sc.SpanID().String()),
        )
    }
    return l
}
```

11 个 main.go 业务关键路径替换 `logger.L()` → `logger.FromContext(ctx)`（starting / exited / stopped + wallet 的 kafka consumer 4 处）。

**3 个新单测**：OTelSpanContext / NoSpanContext / BothTraceIDAndOTel。

**Commit 2：Jaeger collector**

`docker-compose.deploy.yml` 加 `jaeger` 服务（jaegertracing/all-in-one:latest）：
- ports：16686 UI / 4317 OTLP gRPC / 4318 OTLP HTTP / 14268/14250 collector
- healthcheck：wget `http://localhost:16686/api/services`

11 个 Go 服务 env 注入：
```
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_SERVICE_NAME=<svc>
```

`depends_on` 追加 `jaeger: condition: service_started`（OTel exporter 端点留空 → Noop 降级，不强依赖 Jaeger 就绪）。

**Commit 3：golangci-lint v2**

`.golangci.yml` 升级到 `version: "2"` schema：
- 启用 11 个 linter（6 基础 + 5 新增：bodyclose / gocritic / misspell / nakedret / prealloc）
- settings：govet enable-all + gocritic tags（diagnostic/style/performance）+ misspell locale=zh + nakedret max-func-lines=25 + prealloc simple+range-loops
- formatters：gofmt + goimports local-prefixes=github.com/growdu/doctors
- exclusions：middleware/ + contracts/ 放宽（自动生成 + 噪音）

`shared/middleware/linter_examples.go` 新增 170 行（`//go:build linter_examples` tag 隔离，CI 仅在 lint 任务启用）：11 个 linter 错误示例 vs 修正对照。

**累计测试用例**：
- shared/logger：9 PASS（原有 6 + 新增 3）
- 其他 12 个 shared 包：不变
- 服务包：不变
- **全量 13 shared 包 + 47 service 包 = 60 包 0 FAIL**

**Plan 偏差**：

1. **`build tag linter_examples`**：示例代码故意保留错误写法，加 `//go:build linter_examples` tag 避免污染生产 binary
2. **exclusions 放宽 middleware/ + contracts/**：v1 dev 期历史代码噪音较大，避免一次性大批失败阻塞 PR
3. **Jaeger healthcheck 用 wget**：jaegertracing/all-in-one 镜像默认不带 curl
4. **Jaeger depends_on service_started**：OTel endpoint 留空退化为 Noop，不强依赖 Jaeger
5. **OTEL env 显式声明 http/protobuf**：避免与 shared/tracing OTLP HTTP 实现 mismatch

**端到端联通 v1.3 目标**：

- `docker compose -f docker-compose.deploy.yml up -d` → 起 18 容器（14 服务 + 3 中间件 + jaeger）
- 业务调用 → zap 日志自动含 `otel_trace_id` 字段
- 浏览器开 `http://localhost:16686` 选 service 看 trace → 跳到对应业务日志
- golangci-lint v2 跑全仓库增量 PR → 历史代码不阻塞
- 60 包 0 FAIL
