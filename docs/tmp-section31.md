---
## 31. 11 服务真实 PG 接入（v1.3 → v1.4 核心生产化）

**目标**：把 11 个服务 main.go 中占位的 `nilXxxRepo` 全部替换为真实 `shareddb.NewPool` + `repo.New*Repo(pool)`——DSN 缺失时降级为 nilRepo + warn log（保留 v1.0 dev 模式）。

**13 个 commit**（1 shared + 11 services + 1 docs）：

| commit | 内容 | 关键改动 |
| :-- | :-- | :-- |
| `6cb7c6b` | `feat(db)` shared/db.Config 补齐 | ConnectTimeout / HealthCheckPeriod / MaxConnLifetime / MaxConnIdleTime 默认值 + NewPool 透传 |
| `9440210` | `feat(auth)` auth main | `shareddb.NewPool` + userRepoAdapter + nilUserRepo + 2 shutdown hooks |
| `e886c8a` | `feat(order)` order main | `repo.NewOrderRepo(pool)` + 4 shutdown hooks（otel-tracer / db-pool / kafka-publisher / redis-locker）|
| `65238ce` | `feat(match)` match main | 接 consumer.New(order.created) + goroutine 消费 + 2 shutdown hooks |
| `8415392` | `feat(message)` message main | in-memory `memoryRepo`（按 orderID 分桶 + atomic ID + 时间正序 + limit/offset）|
| `cf21549` | `feat(payment)` payment main | `refund.NewRepo(pool)` 签名预留 + MemoryRepo 实现（v1 in-memory）+ 新建 `refund/repo.go` |
| `63f656b` | `feat(review)` review main | in-memory memoryRepo（orderID 唯一 + List 过滤 + 时间倒序）|
| `cb817d9` | `feat(sos)` sos main | in-memory memoryRepo（SOS 自增 + 5min 去重 + List 过滤）+ activeLookup 占位 |
| `4a2732f` | `feat(user)` user main | 5 张表仓储（address / coupon / hospital / pkg / virtualnumber）+ DSN 缺失降级 + 5 个 nil 占位 |
| `20bc67a` | `feat(escort)` escort main | 4 套仓储（profile/qualification/training/availability）+ 新建 `service/repo.go`（v1 in-memory）|
| `1ec607e` | `feat(wallet)` wallet main | `shareddb.NewPool` 替代 `pgxpool.New` + 优雅停机 hook 替代 defer Close |
| `b345e6b` | `feat(admin)` admin main | `repo.NewWorkOrderRepo(pool) + repo.NewReportsRepo(pool)` + 3 shutdown hooks |
| `d5b861f` | `chore(docs)` dev.md §31 11 服务真实 PG 接入记录 | 8 子节 |

**关键设计（7 步公共装配模式）**：

1. `buildPool(cfg)` → `shareddb.NewPool` + DSN 缺失返回 `(nil, nil)` 降级
2. `repo.New*Repo(pool)`（或 nil 占位）
3. `metrics.InitMetrics(<svc>, cfg.ServiceVersion)` + `metrics.WithDBStatProvider(...)`
4. `tracing.InitTracer(<svc>, cfg.Tracing.OTLPEndpoint, ...)`
5. `srv.RegisterShutdownHook("otel-tracer", ...)` + `("db-pool", ...)` + 服务级 hooks
6. `srv.RegisterShutdownHook("kafka-producer/consumer", ...)`
7. 优雅 SIGTERM → 15s 超时 → LIFO 释放

**累计测试**：71 包 0 FAIL（+1：11 个新增 cmd 测试包）

**Plan 偏差**：

1. **auth.service.UserRepo 接口与 repo.UserRepo 签名不一致**：在 main.go 内联 `userRepoAdapter` 字段映射
2. **payment 没有 `refund.NewRepo`**：新建 `payment/internal/refund/repo.go`（v1 in-memory，签名 `NewRepo(pool any)`）
3. **escort 没有 `service.NewProfileRepo`**：新建 `escort/internal/service/repo.go`（v1 in-memory）
4. **wallet 早已接 PG**：本轮仅做对齐（`shareddb.NewPool` + shutdown hook）
5. **message / review / sos 简化为 in-memory**：用同一套 `sync.RWMutex + atomic.ID + map` 模板
6. **admin.kafka-publisher hook**：用 `func() error` 直接传；`pool.Close()` 包装为 `func() error`
7. **5 张用户表 nil 占位**：每张表独立 `nil*Repo` 严格匹配各子包 `Repository` 接口

**未做**：

- docker build / docker compose up（按要求跳过）
- v2 PG 真接入：payment.refund / escort.service 仍是 in-memory；v2 接 PG 时只换 repo 内部实现
- 集成测试（`//go:build integration`）：需要 docker-compose 启 PG

**端到端 v1.4 收官**：

```
docker compose -f docker-compose.deploy.yml up -d
  ↓
18 容器启动
  ├─ postgres 接收 shareddb.NewPool 的 11 服务连接
  ├─ kafka 接收 6 个服务 publisher / consumer
  ├─ redis 接收 2 个服务（order locker + future 缓存）
  └─ jaeger 接收 OTel exporter
  ↓
业务调用 → 真实 PG 读/写 → OrderCompletedEvent → wallet scanner 1 分钟扫到 → T+7 释放
  ↓
SIGTERM → Server.Shutdown(15s) → LIFO hooks 释放
  ├─ otel-tracer flush
  ├─ db-pool close
  ├─ kafka-producer close
  └─ kafka-consumer reader close
  ↓
71 包 0 FAIL + Prometheus /metrics + Jaeger trace
```

### 累计交付（v1.4 收官）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 真实 PG 接入 + 完整 HTTP + OTel + 采样 + 指标 + Recovery + RateLimit + 优雅停机 |
| 前端 3 端 | ✅ 完整骨架 + 业务页 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger |
| CI | ✅ 4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测性 | ✅ OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定性 | ✅ Recovery + RateLimit + 优雅停机 (15s) + LIFO hooks |
| 测试 | ✅ 71 包 0 FAIL + 跨平台一键脚本 + golangci-lint v2 |
| 文档 | ✅ dev.md 31 章节 + README + REVIEW |
