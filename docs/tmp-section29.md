---
## 29. shared/metrics Prometheus 接入 + 11 服务 /metrics 路由（生产监控）

**目标**：补全 Prometheus 监控基础设施——6 个默认业务指标 + 11 个服务 /metrics 路由 + 中间件自动埋点 + Prometheus scrape 配置文档。

**2 个 commit**：

| commit | 内容 | 文件 |
| :-- | :-- | :--: |
| `336af58` | `feat(metrics): shared/metrics Prometheus 接入 + 6 业务指标` | 4（metrics.go + test + README + go.mod）|
| `7e5e799` | `chore(deploy): 11 服务 /metrics 路由 + middleware 接入 + Prometheus 文档` | 11 router + 11 main + 11 yaml + middleware + README |

**Commit 1：shared/metrics 包**

| 指标 | 类型 | Labels |
| :-- | :-- | :-- |
| `http_requests_total` | CounterVec | method, path, status (2xx/3xx/4xx/5xx) |
| `http_request_duration_seconds` | HistogramVec | method, path — buckets 5/10/25/50/100/250/500/1000/2500/5000 ms |
| `db_pool_acquired_connections` | GaugeVec | pool |
| `db_pool_idle_connections` | GaugeVec | pool |
| `db_pool_total_connections` | GaugeVec | pool |
| `kafka_consumer_lag` | GaugeVec | topic, group |
| `service_info` | Gauge (=1) | service, version, go_version |

**核心 API**：
- `metrics.Handler() http.Handler` — 返回 promhttp.Handler()（K8s ServiceMonitor 抓取）
- `metrics.Middleware(next http.Handler) http.Handler` — 自动埋 http_requests_total + http_request_duration_seconds
- `metrics.GinMiddleware()` — gin 适配器
- `metrics.InitMetrics(serviceName, version)` — 初始化全局指标 + 启动 30s 收集 DB pool 协程
- `metrics.WithDBStatProvider(func() []DBPoolStat)` — 注入 DB 池统计
- `metrics.WithKafkaConsumerLag(topic, group, lag)` — Kafka 消费者调用写 lag

**9 个单元测试 PASS**：指标注册 / Handler / Middleware / DB Stat / Kafka Lag / 0 值边界。

**Commit 2：11 服务接入**

- `shared/middleware/metrics.go` 新增：`Metrics()` 返回 gin.HandlerFunc（语义别名 + 2 个测试）
- `shared/config/loader.go`：`Metrics{Enabled, ServiceName}` 段（默认 enabled=true）
- **11 router 改动**：`r.Use(sharedmw.Metrics())`（在 Auth 前保证 4xx 也计数）+ `r.GET("/metrics", gin.WrapH(metrics.Handler()))`
- **11 main.go 改动**：`metrics.InitMetrics(<svc>, cfg.ServiceVersion)` + wallet 注入 `WithDBStatProvider`
- **11 yaml 改动**：`metrics: enabled: true, service_name: <svc>-service`
- README §8.4：6 指标表 + Prometheus scrape_config YAML（11 targets）+ env 覆盖表

**累计测试**：60 → 62 包 0 FAIL（+metrics 9 + middleware 2）。

**Plan 偏差**：

1. **Metrics 中间件挂载位置**：在 `router.New()` 内 `r.Use(sharedmw.Metrics())`，避免改 11 个 server.go
2. **DB pool StatProvider 接入范围**：仅 wallet 注入（pgxpool 已接），其余 10 个 pool=nil 时 InitMetrics 不启动 30s 协程；按"已接的接入，未接的不挂"实现
3. **shared/config 加 Metrics 段**：viper 不解析新字段会丢默认值；同时加 `SetDefault("metrics.enabled", true)` 保证语义
4. **`-race` 标志在 Windows 报 `0xc0000139`**：Go 1.24 race detector CGo DLL 在 Windows + Git Bash 加载失败（环境限制），用 `go test -count=1` 替代通过

**端到端 v1.3 收官**：

```
docker compose -f docker-compose.deploy.yml up -d
  ↓
18 容器 + Prometheus + Grafana
  ↓
Prometheus 抓 :8080/metrics（11 个 target）
  ↓
http_requests_total{status="5xx"} 告警 + http_request_duration_seconds{quantile="0.99"} SLO 监控
  ↓
db_pool_* 预警连接池耗尽 + kafka_consumer_lag 监控积压
```

### 累计交付（v1.3 + 生产化）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 完整 HTTP + OTel + 采样 + 指标 + 监控 |
| 前端 3 端 | ✅ 完整骨架 + 业务页 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger |
| CI | ✅ 4 job + Pages + 仓库维护 |
| 可观测性 | ✅ OTel + Jaeger + Prometheus /metrics + 跨服务 trace |
| 测试 | ✅ 62 包 0 FAIL + 跨平台一键脚本 + golangci-lint v2 |
| 文档 | ✅ dev.md 29 章节 + README + REVIEW |
