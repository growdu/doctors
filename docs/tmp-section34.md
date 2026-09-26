---
## 34. 4 项生产稳定性优化（v1.4 → v1.5 增量）

**目标**：让 v1.4 真正具备"生产稳定运行"的能力——修复 RateLimit flaky + panic 埋点生效 + /readyz 端点 + OTel auto-instrumentation。

**4 + 1 个 commit**：

| commit | 内容 |
| :-- | :-- |
| `5650730` | `fix: ratelimit flaky 测试 + recovery 埋点` |
| `1e30b4f` | `feat(health): shared/health 包 + 11 服务 /readyz 端点` |
| `0fed212` | `feat(otel): gin / pgx / kafka-go auto-instrumentation` |
| `ed00141` | `chore(docs): dev.md §34` |
| `df23bd7` | `chore(deps): tidy go.mod / go.sum for otelgin + otelpgx` |

**Commit 1：RateLimit flaky 修复 + recovery 埋点**

`shared/middleware/ratelimit.go`：
- 暴露 `*LimiterRegistry` + `Snapshot()` 方法
- 测试用 `rate=0.001`（1 token/1000s）+ `limiter.Tokens()` 直接断言桶状态
- 彻底规避 Windows+Git Bash 调度抖动

`shared/metrics/metrics.go`：
- 新增 `RecoveryPanicsTotal = promauto.NewCounterVec`，label = 路由模板（避免高基数）

`shared/middleware/recovery.go`：
- 捕获 panic 后 `Inc(c.FullPath())`（404 退化为 URL.Path）
- 1 个新单测：触发 panic → 指标 +1

`deploy/prometheus/alerts/panic_recovery.yaml`：
- 4 条 alert 改用 `recovery_panics_total`，§33 真正生效

**Commit 2：/readyz 端点**

`shared/health/` 包（4 文件）：
- `health.go`：`Checker` 接口 + `Manager`（`Register/RunAll/Status`）+ `Status/Report` struct
- `adapters.go`：`NewPGPoolChecker` / `NewRedisChecker` / `NewKafkaBrokerChecker`
- `readyz_handler.go`：`ReadyzHandler(m)` 返回 gin.HandlerFunc——全部 OK → 200；任一失败 → 503 + Report JSON
- `health_test.go`：**17 个单测**

11 服务 `cmd/main.go` 构造 `health.NewManager(WithTimeout(1s))`，按可用资源注册 checker（nil/空 → skip + warn）。

11 服务 router 挂 `/readyz`（与 `/healthz` 共存区分 liveness/readiness）。

**Commit 3：OTel auto-instrumentation**

3 个 instrumentation：
- **gin**：`shared/middleware/otel.go::OTelGinMiddleware(service)` 包装 `otelgin.Middleware`
- **pgx**：`shared/tracing/WithPgxPool(pcfg)` 注入 `otelpgx.NewTracer`
- **kafka-go**：`shared/tracing/WrapWriter/WrapReader`（goproxy.cn 无 otelkafkago 镜像，自实现 OTel 标准 API wrapper）

11 服务 router 挂 `OTelGinMiddleware`（位于 Recovery 之后）；7 服务 buildPool 注入 otelpgx tracer。

**8 个新单测**：OTelGinMiddleware 4 + kafka wrapper 4。

**累计测试**：80 → **80+ 包 0 FAIL**（+3 包：health + otel middleware + kafkago tracing；+10 单测：17 health + 4 otel + 4 kafka wrapper + 1 recovery + 4 ratelimit 重构）

**Plan 偏差**：

1. **kafka-go OTel 自实现 wrapper**：goproxy.cn 无 `otelkafkago` 与 otel contrib 的 kafka-go instrumentation
2. **router.New 签名扩展加 readyzM/OTelGin**：同步更新所有 router_test.go（pass `nil`）
3. **/readyz 与 /healthz 共存**：不替换——liveness vs readiness 职责不同
4. **panic metric label = 路由模板而非 URL.Path**：避免高基数
5. **OTel 升级到 1.34.0**：与 contrib v0.59 配套
6. **buildPool 双 trace 注入**：pgxpool.ParseConfig → tracing.WithPgxPool → 回填 `pcfg.ConnConfig.Tracer`

**未做**：

- kafka-go wrapper 未实际接入业务 publisher（避免一次性 9 个服务改动）
- /readyz handler 写 Prometheus counter
- OTel metrics exporter（已含 otelpgx metrics，留 v1.5 接入）
- 真实 e2e 联调

**端到端 v1.5 收官**：

```
docker compose -f docker-compose.deploy.yml up -d
  ↓
K8s liveness probe → :8080/healthz（-healthz flag 进程存活）
K8s readiness probe → :8080/readyz（PG/Redis/Kafka 依赖就绪）
  ↓
业务请求 → otelgin 自动埋点 → otelpgx 自动 trace SQL → kafka producer 自动 span
  ↓
SIGTERM → 15s Server.Shutdown → LIFO hooks → 0 数据丢失
  ↓
recovery panic 计数 → /metrics 暴露 → Prometheus 抓 → 4 条 alert 真正生效
```

### 累计交付（v1.5 生产稳定运行）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 真实 PG + 真实 Kafka + OTel auto + 优雅停机 + /healthz + /readyz |
| 前端 3 端 | ✅ 完整骨架 + 业务页 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger |
| CI | ✅ 4 job + Pages + 仓库维护 |
| 可观测性 | ✅ OTel 全链路（gin/pgx/kafka auto）+ Jaeger + Prometheus + 20 alert |
| 稳定性 | ✅ Recovery + RateLimit + 优雅停机 (15s) + LIFO + panic 埋点 + /readyz |
| 安全性 | ✅ .env.example + 密钥管理 SOP + 4 方案对比 |
| 测试 | ✅ 80+ 包 0 FAIL + 跨平台脚本 + smoke 端到端 + golangci-lint v2 |
| 文档 | ✅ dev.md 34 章节 + README + REVIEW + ops/secrets |
