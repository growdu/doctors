---
## 30. middleware Recovery + RateLimit + 11 服务优雅停机（生产稳定性）

**目标**：补全生产级稳定性——panic 不拖垮进程 + 限流防刷 + 优雅停机释放资源。

**3 个 commit**：

| commit | 内容 | 文件 |
| :-- | :-- | :--: |
| `b2682b9` | `feat(middleware)` shared/middleware.Recovery panic 恢复 + httpx.TraceID | 2 |
| `2583088` | `feat(middleware)` shared/middleware.RateLimit IP token-bucket 限流 | 2 |
| `bac341e` | `feat(middleware)` 11 服务接入 + Server 优雅停机 (15s + ShutdownHook) | 27 |

**Commit 1：Recovery 中间件**

`shared/middleware/recovery.go`（130 行）：

```go
func Recovery(opts ...RecoveryOption) gin.HandlerFunc
type RecoveryOption func(*recoveryConfig)
func WithRecoveryLogger(*zap.Logger) RecoveryOption
func WithRecoveryStackTrace(bool) RecoveryOption
```

- 捕获 panic → 记录 stack trace → 返回 500 + 业务码 `errs.CodeInternal (500000)`
- 不再次 panic；不让进程崩溃
- 双层 defer + recover：业务 panic 捕获后，写出 JSON 时若 c.Writer 损坏仍可能 panic，第二层 defer 兜底

**4 个测试 PASS**：panic→500 / 多次 panic 不影响后续请求 / stack trace 日志 / 关闭 stack 配置。

**Commit 2：RateLimit 限流**

`shared/middleware/ratelimit.go`（204 行）：

```go
func RateLimit(opts ...RateLimitOption) gin.HandlerFunc
type RateLimitOption func(*rateLimitConfig)
func WithRateLimitPerSecond(r float64) RateLimitOption
func WithRateLimitBurst(b int) RateLimitOption
func WithRateLimitKeyFunc(fn func(*gin.Context) string) RateLimitOption
```

- 使用 `golang.org/x/time/rate` token bucket
- 每个 IP 独立限流（默认 key = `c.ClientIP()`）
- 超限返回 HTTP 429 + 业务码 `errs.CodeRateLimit (13001)`
- 防御性钳值：perSecond / burst ≤ 0 钳到 1

**6 个测试 PASS**（任务要求 3 个，多 3 个增量覆盖）：burst+block / 不同 IP 独立 / rate=0.1 burst=2 / HTTP 429 / 并发安全 / 自定义 key。

**Commit 3：11 服务接入 + 优雅停机**

**11 router 改动**（统一挂载顺序）：
```go
r.Use(sharedmw.Metrics())         // 最外层：埋点
r.Use(sharedmw.Recovery())        // panic 恢复
r.Use(sharedmw.RateLimit(...))    // 限流
r.Use(sharedmw.Auth(...))          // 鉴权
```

**11 server 改动**（统一优雅停机 + 资源释放钩子）：
- `shutdownTimeout` 从 10s → **15s**（DB / Kafka / OTel flush 需要）
- 新增 `RegisterShutdownHook(name string, fn func() error)` 方法
- 新增 `runShutdownHooks()` 私有方法：HTTP Shutdown 完成后按 **LIFO** 顺序调用所有 hook
- 每个 hook 独立 `defer recover()`：单个 hook panic 不阻断后续释放
- 日志记录 hook 名 + 错误 / panic 值

**2 个新单测**：`auth/server_test.go` 加 `TestServer_ShutdownHooksRunInLIFO` + `TestServer_ShutdownHookNilSkipped`。

**新增 shared/middleware/README.md**（137 行）：中间件列表、推荐挂载顺序、使用样例。

**累计测试**：shared/middleware **20 PASS**（+10：4 Recovery + 6 RateLimit + 2 Server ShutdownHook）。

**Plan 偏差**：

1. **Recovery 业务码**：任务要求"500 + 业务码 `500001 internal_error`"，但 `errs.CodeInternal = 500000`、`CodeUnavailable = 500001`（语义"服务暂时不可用"）。使用 `errs.CodeInternal (500000)`（语义"服务器内部错误"与 panic 完全匹配）
2. **RateLimit HTTP 429**：任务要求"超限 429"，采用 HTTP 429 + body.code 13001（贴近生产 ingress 识别）
3. **RateLimit 测试数量**：任务要求 3 个，实际交付 6 个
4. **Server hook 抽象**：没抽离到 shared/server，保持各服务独立 Server struct

**端到端 v1.3 收官（生产级别）**：

```
docker compose -f docker-compose.deploy.yml up -d
  ↓
18 容器启动
  ↓
业务调用 → RateLimit 100/s IP + Recovery 兜底 + Metrics 埋点 + OTel trace
  ↓
SIGTERM → Server.Shutdown(15s) → runShutdownHooks() LIFO 释放
  ├─ otel-tracer flush（避免丢 span）
  ├─ kafka producer close
  ├─ db pool close
  └─ metrics collector stop
  ↓
62 包 0 FAIL + 跨平台测试脚本 + Prometheus /metrics + Jaeger trace
```

### 累计交付（v1.3 收官 + 生产化）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 完整 HTTP + OTel + 采样 + Prometheus /metrics + Recovery + RateLimit + 优雅停机 |
| 前端 3 端 | ✅ 完整骨架 + 业务页 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger |
| CI | ✅ 4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测性 | ✅ OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定性 | ✅ Recovery + RateLimit + 优雅停机 (15s) |
| 测试 | ✅ 62 包 0 FAIL + 跨平台一键脚本 + golangci-lint v2 |
| 文档 | ✅ dev.md 30 章节 + README + REVIEW |
