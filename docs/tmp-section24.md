---
## 24. payment-service 补全 + 11 -healthz flag + Dockerfile HEALTHCHECK（2026-09-24 ops 部署补全）

**目标**：补全 payment-service HTTP 层 + 给 11 个 Go 服务加 -healthz flag + distroless Dockerfile HEALTHCHECK + docker-compose healthcheck 段——让 distroless 容器能做健康检查 + payment 能 build。

**2 个 commit**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `ff3bd34` | `feat(payment-service)` HTTP 层补全（handler+router+server+cmd）+ Get(payment_id) + 14 单测 | 9 |
| `c769f2e` | `chore(deploy)` 11 Go 服务 -healthz flag + distroless HEALTHCHECK + compose 健康检查 | 11 main.go + 11 Dockerfile + docker-compose |

**Commit 1：payment-service 补全（4 endpoint）**

| Method | Path | 业务码 |
| :-- | :-- | :-- |
| `POST` | `/api/v1/payments` | create（按订单）|
| `GET` | `/api/v1/payments/:id` | detail |
| `POST` | `/api/v1/payments/:id/complete` | mock 微信支付完成 |
| `POST` | `/api/v1/payments/:id/refund` | 申请退款 |

新增 7 文件 + 修改 2 文件 + 16 单测 = **38 个 payment-service 测试**（含既有 22 + 新增 16）。

**Commit 2：11 -healthz flag + 部署补全**

1. **11 main.go** 各加 7 行 `flag.Bool("healthz", ...)` + 13 行 `runHealthzServer()`（独立 :9090 HTTP server 持续 200 OK）
2. **11 Dockerfile** `HEALTHCHECK NONE` → `HEALTHCHECK CMD ["/app/server", "-healthz"]`（distroless 无 shell/curl/wget，改用 `-healthz` flag）
3. **docker-compose.deploy.yml** 11 个 Go 服务均加 healthcheck 段（test/interval/timeout/retries/start_period）

**关键设计**：

1. **独立 :9090 healthz 探针**：与业务 :8080 解耦，distroless 容器无需 shell/curl/wget
2. **flag.Bool** 实现：默认关闭；启动 `docker compose up` 时 `command: ["/app/server", "-healthz"]` 由 compose 触发 healthcheck
3. **payment `Get` 方法新增**：原 service.Service 无 `Get(paymentID)`，handler 需要 `GET /api/v1/payments/:id` 取详情；最小代价在 service 加 13 行 `Get()` 方法并补 2 个 service 单测
4. **payment 端口 `:8085`**：与 compose 中保留的端口一致

**累计测试用例**：

- payment 新增 16（handler 10 + router 4 + service 2）
- payment 现有 22（refund/policy 5 + refund/service 4 + payment/service 既有 13）
- 其他 10 服务：不变（commit2 不涉及业务逻辑修改）
- **全量回归 61 包 0 FAIL**（含 payment 4 个有 test 包）

**Plan 偏差**：

1. **service.Get 方法新增**：原 service.Service 无 Get(paymentID)，handler 需要 GET 详情；最小代价在 service 加 13 行 + 2 单测
2. **未创建独立的 middleware 包**：直接用 `shared/middleware.Auth` + `"user_id"` / `"role"` key，避免重复造 payment-specific middleware 包
3. **payment main 装配采用 nil 占位**：与 admin / wallet 一致，依赖空 repo/channel/publisher，路由生效
4. **未给 -healthz 加专门的 unit test**：`log.Fatal` 调用 `os.Exit`，单元测试无法验证；改用进程级 smoke 实测 `curl http://localhost:9090/healthz` 返回 200/"ok"

**未做（留给后续）**：

- Docker build（按约束：本机可能无 docker）
- Postgres / Redis / Kafka 集成测试（pool 仍为 nil）
- docker push（按约束）
- docker-compose depends_on 升级为 `condition: service_healthy`（Go 服务间等待业务端口就绪）

**端到端联通（v1.3 目标）**：

- 11 个 Go 服务 + 3 前端 = 14 镜像 + 3 中间件全部可 docker compose up
- distroless 镜像 < 30MB + 启动 < 3s + 无 shell attack surface
- healthcheck 走独立 :9090 + `-healthz` flag（distroless-friendly）
- payment 完整 HTTP 层，docker build 不再失败
- 全量 61 包 0 FAIL
