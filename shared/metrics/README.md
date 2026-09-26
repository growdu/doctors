# shared/metrics · Prometheus 业务指标接入

`shared/metrics` 封装 11 个 Go 微服务的 Prometheus 指标：HTTP 流量、DB 连接池、Kafka 消费滞后、服务自身元信息。

## 设计要点

- **全局默认 Registry**：业务指标用 `promauto` 自动注册到 `prometheus.DefaultRegisterer`，与 `promhttp.Handler()` 抓取端天然兼容。
- **Gin 适配中间件**：`GinMiddleware()` 自动记录 `http_requests_total` / `http_request_duration_seconds`，path 取 `c.FullPath()` 模板（如 `/users/:id`）避免 label 基数爆炸。
- **DB pool 注入式采集**：`InitMetrics(..., WithDBStatProvider(...))` 启动后台 goroutine 每 30s 调一次 provider，把 `pgxpool.Stat()` 同步成 gauge。
- **service_info Gauge**：单序列、值=1、标签携带 `service` / `version` / `go_version`；alertmanager 关联 build 元信息用。

## 6 个默认业务指标

| 指标名                              | 类型        | 标签                  | 含义                                                  |
| ----------------------------------- | ----------- | --------------------- | ----------------------------------------------------- |
| `http_requests_total`               | CounterVec  | `method`,`path`,`status` | HTTP 请求累计数；`status` 归桶到 `2xx/3xx/4xx/5xx`。 |
| `http_request_duration_seconds`     | HistogramVec| `method`,`path`       | HTTP 请求耗时分布（buckets: 5/10/25/50/100/250/500/1000/2500/5000 ms）。 |
| `db_pool_acquired_connections`      | GaugeVec    | `pool`                | DB 连接池当前占用连接数。                            |
| `db_pool_idle_connections`          | GaugeVec    | `pool`                | DB 连接池空闲连接数。                                |
| `db_pool_total_connections`         | GaugeVec    | `pool`                | DB 连接池总连接数（acquired + idle）。               |
| `kafka_consumer_lag`                | GaugeVec    | `topic`,`group`       | Kafka 消费者滞后（高水位 - 已提交 offset）。         |
| `service_info`                      | Gauge（值=1）| `service`,`version`,`go_version` | 服务构建元信息；单序列、静态。 |

## 使用

```go
// main.go
metrics.InitMetrics("auth-service", cfg.ServiceVersion,
    metrics.WithDBStatProvider(func() []metrics.DBPoolStat {
        s := pgxpool.Stat()
        return []metrics.DBPoolStat{{
            Name: "main", Acquired: s.AcquiredConns(),
            Idle: s.IdleConns(), TotalConns: s.TotalConns(),
        }}
    }),
)

engine := gin.New()
engine.Use(metrics.GinMiddleware())         // 自动埋 HTTP 指标
engine.GET("/metrics", gin.WrapH(metrics.Handler())) // 暴露抓取端
```

业务侧自定义指标同样自动接入：

```go
var orderCreated = promauto.NewCounter(prometheus.CounterOpts{
    Name: "order_created_total", Help: "...",
})
orderCreated.Inc() // 出现在 /metrics
```

## Prometheus 抓取配置示例

```yaml
scrape_configs:
  - job_name: doctors-services
    metrics_path: /metrics
    static_configs:
      - targets:
        - auth-service:8080
        - order-service:8080
        - user-service:8080
        # ... 共 11 个
```

## 单元测试

`metrics_test.go` 用 `prometheus/testutil.CollectAndCount` + `ToFloat64` 验证 6 个指标能注册并读写，9 个用例全 PASS。
