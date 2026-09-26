# shared/middleware

11 个 Go 服务共用的 Gin 中间件。所有中间件都按"可选、可配置、可注入"原则设计，便于测试与服务级覆写。

## 中间件清单

| 中间件 | 顺序（在 Auth 之前的相对位置） | 作用 | 默认行为 |
|--------|------------------------------|------|----------|
| **Metrics** | 最外层 | HTTP 指标埋点（method/path/status/duration） | 复用 `shared/metrics.GinMiddleware()` |
| **Recovery** | Metrics 之内 | panic → 500 + 业务码 500000；记录 stack trace | 默认开启 stack；日志走 `shared/logger.L()` |
| **RateLimit** | Recovery 之内 | IP token bucket；超限 → 429 + 业务码 13001 | 100 req/s + burst 200；按 `c.ClientIP()` |
| **Auth** | RateLimit 之内（group 级） | JWT Bearer 校验；失败 401 + 业务码 11001 | `secret` / `userIDKey` / `roleKey` 必填 |
| **RoleAuth** | Auth 之内（路由级） | role 白名单；不通过 403 + 业务码 11003 | OR 语义，多角色白名单 |

## 推荐挂载顺序（11 服务统一）

```go
r := gin.New()
r.Use(sharedmw.Metrics())      // 最外层：401/403/panic/429 也被埋点
r.Use(sharedmw.Recovery())     // panic 恢复（业务码 500000）
r.Use(sharedmw.RateLimit())    // IP 限流（业务码 13001）

r.GET("/healthz", ...)
r.GET("/metrics", gin.WrapH(metrics.Handler()))

// 业务 group 内的鉴权（不消耗限流桶）
auth := sharedmw.Auth(jwtSecret, "uid", sharedmw.RoleKey)
v1 := r.Group("/api/v1", auth)
v1.POST("/orders", middleware.RoleAuth("order_admin"), h.Create)
```

理由：

- **Metrics 最外层**：401/403/panic/429 都计入 HTTP 指标，便于告警与 SLO 统计。
- **Recovery 在 Metrics 之内**：panic 也能被埋点（`status=500` 在 http_requests_total 可见）。
- **RateLimit 在 Auth 之前**：无效请求也消耗限流桶，避免被绕过；但放在 Auth 之内又会让"未登录暴力请求"绕过限流。本仓库采用"先限流后鉴权"，让非法流量也吃 token bucket。
  - 若某服务希望"未鉴权不限流"，把 `RateLimit()` 移到 `v1.Group("/api/v1", auth)` 之后即可。

## 中间件使用样例

### Metrics

```go
r.Use(sharedmw.Metrics())
```

无参数；自动写 `http_requests_total{method,path,status}` / `http_request_duration_seconds{method,path}`。

### Recovery

```go
r.Use(sharedmw.Recovery())
// 或带选项：
r.Use(sharedmw.Recovery(
    sharedmw.WithRecoveryLogger(myZapLogger),
    sharedmw.WithRecoveryStackTrace(false), // 生产可关
))
```

panic 后：

- HTTP 状态 `500`
- 响应体：`{"code":500000,"message":"internal error","trace_id":"..."}`
- 日志：`error` 级别，含 panic 值 + path/method/client_ip + 可选 stack trace

不变量：

- 进程 / engine 不会因一次 panic 被拖垮
- 同一 engine 内后续请求正常处理
- body 不暴露 panic 细节（仅进日志）

### RateLimit

```go
r.Use(sharedmw.RateLimit())
// 或自定义：
r.Use(sharedmw.RateLimit(
    sharedmw.WithRateLimitPerSecond(50),
    sharedmw.WithRateLimitBurst(100),
    sharedmw.WithRateLimitKeyFunc(func(c *gin.Context) string {
        return c.GetHeader("X-Tenant-ID") // 或 c.GetString("uid")
    }),
))
```

默认 100 req/s + burst 200（每个 key 独立）。

超限：

- HTTP 状态 `429`
- 响应体：`{"code":13001,"message":"rate limit exceeded","trace_id":"..."}`

内存占用提示：每个活跃 IP 一个 `*rate.Limiter`（≈ 80B）。如需防止恶意 IP 撑爆内存，可在 `limiterRegistry` 上加 LRU 淘汰；当前未实现，依赖反向代理（nginx limit_req）兜底。

### Auth

```go
auth := sharedmw.Auth(secret, "uid", "role")
v1 := r.Group("/api/v1", auth)
```

- `secret`：JWT 签名密钥（来自 cfg.Auth.JWTSecret）
- `userIDKey` / `roleKey`：注入到 `gin.Context` 的字段名，与 handler 读取方式一致

失败：`401` + 业务码 `11001`。

### RoleAuth

```go
v1.POST("/orders/:id/force-cancel",
    sharedmw.RoleAuth("super_admin", "order_admin"),
    h.ForceCancel)
```

OR 语义，role 在白名单内即放行；Auth 必须已先注入 `role` 到 ctx。

失败：`403` + 业务码 `11003`。

## 与 shared/httpx 的关系

- `Recovery` / `RateLimit` 都用 `httpx.TraceID(c)` 读取 trace id，缺失则生成 16-hex。
- 响应体用 `httpx.Resp[T]` 包装，保持全栈业务响应格式一致（`code/message/data/trace_id`）。
- 业务错误码统一定义在 `shared/errs.Code*`（`11001`/`11003`/`13001`/`500000` 等）。

## 测试

```bash
go test -count=1 ./shared/middleware/...
```

测试覆盖：

- `Auth`：4 个（无 token / 错误 scheme / 错误 secret / 正常 token）
- `RoleAuth`：4 个（无 token / 白名单命中 / 角色拒绝 / 多角色白名单）
- `Metrics`：2 个（中间件生效 + /metrics 端点暴露）
- `Recovery`：4 个（500 + 业务码 / 多次 panic 不互相影响 / stack 字段 / 关闭 stack）
- `RateLimit`：6 个（burst 内放行 / 不同 IP 独立 / 慢速 refill / HTTP 429 / 并发安全 / 自定义 key func）
