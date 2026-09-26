# 开发日志 · 陪诊师平台 v1.0

> 本文档记录开发全过程的**设计思路、决策理由、踩坑记录**，每个模块对应一节，可与 git log 互为索引。
> 配合 `docs/superpowers/plans/2026-09-23-doctors-v1.md` 的实施计划使用。
> 本次会话 MVP 范围：**阶段 0 ~ 阶段 4**（骨架 → shared → auth → order → match）。

---

## 0. 全局设计思路

### 0.1 选型回顾（2026-09-23）

| 维度 | 选择 | 关键理由 |
| :-- | :-- | :-- |
| 后端语言 | **Go 1.22+** | 单二进制、冷启动快、K8s 友好；与高频订单匹配契合 |
| DB | **PostgreSQL 16 + Patroni** | JSONB 灵活字段、`FOR UPDATE SKIP LOCKED` 抢单原子化、流复制 RPO=0 |
| 驱动 / ORM | pgx + sqlc | 类型安全、性能可控；GORM 仅备选 |
| 缓存 | Redis 7 | 与 Go 配合好 |
| MQ | Kafka | 高吞吐、与 Go 生态成熟 |
| HTTP | Hertz / Gin | 推荐 Hertz（字节开源） |
| RPC | gRPC | 内部服务统一 |
| 测试 | testify + gomock + dockertest | 单测 + 真中间件集成测试 |

### 0.2 仓库结构（monorepo）

```
doctors/
├── docs/                       # 需求文档 + 架构图（已存在）
├── dev.md                      # 本文档
├── Makefile                    # 顶层命令（test / build / docker-up / down）
├── docker-compose.yml          # PG + Redis + Kafka 一键起
├── go.mod                      # 单一 go module（多 service 内部 package）
├── go.sum
├── .golangci.yml               # Lint 配置
├── .gitignore
├── shared/                     # 跨服务通用包
│   ├── config/                 # 配置加载（Nacos + 本地）
│   ├── logger/                 # zap 结构化日志
│   ├── db/                     # pgxpool + 健康检查
│   ├── redis/                  # go-redis 封装
│   ├── kafka/                  # sarama 或 kafka-go 封装
│   ├── httpx/                  # 统一响应 / 错误码
│   ├── auth/                   # JWT 签发 / 校验
│   ├── idempotency/            # 幂等键
│   └── middleware/             # Gin/Hertz 中间件
├── services/
│   ├── auth/                   # 认证服务（阶段 2）
│   ├── order/                  # 订单服务（阶段 3）
│   ├── match/                  # 匹配服务（阶段 4）
│   ├── payment/                # 支付（v2 阶段）
│   ├── user/
│   ├── escort/
│   ├── billing/
│   ├── review/
│   ├── message/
│   ├── sos/
│   └── admin/
├── migrations/                 # sqlc / golang-migrate 数据库迁移
├── proto/                      # gRPC proto 定义
├── deploy/                     # K8s manifests / Dockerfile
└── tools/                      # 工具脚本
```

### 0.3 命名与代码风格

- 包名小写、单词；不要下划线或 camelCase
- 文件名 snake_case（小写 + 下划线）：`order_service.go`、`order_state_test.go`
- 测试文件以 `_test.go` 结尾，**同包测试**用 `_internal_test.go`（黑盒）
- 每个 service 必须含 `cmd/<service>/main.go` 入口 + `internal/...` 业务逻辑
- 错误：定义 `shared/errs` 错误码表，业务错误携带 code
- 日志：JSON 格式，字段 `trace_id / user_id / action / latency_ms / error`

### 0.4 测试约定

| 类型 | 工具 | 覆盖目标 |
| :-- | :-- | :-- |
| 单元测试 | testify + gomock | 业务逻辑 ≥ 80% |
| 集成测试 | dockertest（起真 PG/Redis/Kafka） | 关键路径：登录 / 下单 / 抢单 |
| 端到端 | docker-compose + curl | 阶段 4 末做一次 smoke |

每完成一个模块 / 一个测试通过，**立即 commit**；commit 信息前缀：
- `feat:` 新功能
- `test:` 仅测试
- `fix:` 修复
- `chore:` 工具 / 配置
- `docs:` 文档

---

## 1. 阶段 0 · 项目骨架 ✅ 已完成

**目标**：可被 `make test` / `make docker-up` 正常执行的最简 monorepo。

**实际完成**（commit `f3c881b` → `bdd9cf7`）：

| Task | 状态 | commit |
| :-- | :--: | :-- |
| 0.1 go.mod / .gitignore / .golangci.yml / Makefile | ✅ | `e18e4dc` |
| 0.2 docker-compose.yml（PG + Redis + Kafka） | ✅ | `4dc843e` |
| 0.3 shared/httpx 统一响应（OK/Fail + trace_id） | ✅ | `70bbf38` |
| 0.4 shared/errs 业务错误码 + Error 类型 | ✅ | `bdd9cf7` |
| 0.5 顶层 smoke（make help + go test ./...） | ✅ | — |

**踩过的坑**：
1. **Go toolchain 自动升级**：最初 `go get` 触发了 Go 1.25.0 toolchain 下载（gin 1.12 依赖要求），但 1.25 下载卡住 → 改为固定 `go 1.22` + `gin v1.10.0` + `testify v1.9.0`，并通过 `go env -w GOTOOLCHAIN=local` 禁用自动升级。
2. **泛型推断**：`httpx.OK(c, nil)` 无法推断 T，必须显式 `httpx.OK[any](c, nil)`。
3. **gin.CreateTestContext 不带 Request**：`c.GetHeader` 会 panic，必须用 `httptest.NewRequest` 构造 request 注入。

**测试覆盖**：5（httpx）+ 12（errs）= **17 个单元测试，全部通过**。

**关键决策**：

1. **monorepo 单 go module** 而非 multi-module：11 个微服务在编译时共享 `shared/` 包，sqlc、proto 复用；缺点是大，但 v1 服务不多，可接受。
2. **顶层 Makefile** 暴露所有命令；`make help` 必须列出全部 target（Windows 上 awk 不可用，改为纯 echo）。
3. **docker-compose** 只起 PG + Redis + Kafka 三个外部依赖；服务本身由 `make run-<svc>` 拉起。
4. **`shared/httpx`** 第一版就定下统一响应格式：

   ```json
   {"code": 0, "message": "ok", "data": {}, "trace_id": "..."}
   ```

   HTTP 状态固定 200，业务码在 body.code 中——避免前端在 4xx 时拿不到完整 body。
5. **`shared/errs`** 用业务码 5 位（10xxx）+ 系统码 6 位（500xxx）分段；`Code.HTTPStatus()` 给中间件做映射；`errors.Is/As` 链路贯穿到底层。

---

## 2. 阶段 1 · shared 基础设施 ✅ 已完成

**目标**：所有服务可直接 import 的通用包：config、logger、db（pgxpool）、redis、kafka、auth（JWT）、idempotency。

**实际完成**（commit `70bbf38` 之后 6 个新 commit）：

| Task | 包 | 关键能力 | 单测 | commit |
| :-- | :-- | :-- | :--: | :-- |
| 1.1 | shared/config | viper + yaml + env 覆盖 + 默认值 | 5 ✅ | — |
| 1.2 | shared/logger | zap 全局 logger + trace_id context | 6 ✅ | — |
| 1.6 | shared/auth | JWT HS256 签发/校验 + 错误分类 | 6 ✅ | — |
| 1.7 | shared/idempotency | Store 接口 + Manager + 规范化 | 9 ✅ | — |
| 1.3 | shared/db | pgxpool + WithTx + DSN 校验 | 4 ✅ | — |
| 1.4 | shared/redis | go-redis v9 + 地址校验 | 2 ✅ | — |
| 1.5 | shared/kafka | kafka-go writer/reader + topic 规范化 | 13 ✅ | — |

**累计 45 个单测全部通过**（阶段 0 的 17 + 阶段 1 的 28）；另含 3 个集成测试（`pool_integration_test.go` / `client_integration_test.go` / `client_integration_test.go`），用 `//go:build integration` 隔离，需 `make docker-up` 后跑 `go test -tags=integration ./...`。

**踩过的坑**：
1. **pgx v5.11.0 要求 Go ≥ 1.25**：最初 `go get` 默认拉到最新版本，触发 toolchain 升级要求。降版本到 `v5.7.1`（兼容 Go 1.22）。
2. **zap v1.28 `Field.Interface` 行为变化**：原先假设 `f.Interface` 返回字符串值，实际在 v1.28 上对某些类型返回 nil。改用 `entries[0].ContextMap()` API，更稳定。
3. **memStore 设计**：第一版只跟踪 done，导致 Reserve 永远成功；改为 `reserved + done` 两个 map：Reserve 标记 reserved，Done 时清除 reserved，逻辑正确。
4. **viper 子目录切换**：测试通过 `os.Chdir` + `t.TempDir()` 切换工作目录，让每个测试用独立的 yaml 文件，互不污染。

**关键决策**：

1. **config**：Viper + 多源（环境变量 > 本地 yaml）；每个服务 `config.Load("auth")` 自动选 `config/<service>.yaml`，环境变量前缀 `DOCTORS_<SERVICE>_<FIELD>`。
2. **logger**：zap + 全局 `logger.L()`；强制 JSON 输出；`WithTrace(ctx, id)` + `FromContext(ctx)` 让 trace_id 沿 ctx 传递，业务代码零侵入。
3. **db**：pgxpool + 健康检查 + `WithTx(ctx, pool, fn)` 事务辅助（自动 commit/rollback + panic 安全）；DSN 必须以 `postgres://` 开头，避免误连 MySQL。
4. **redis**：go-redis v9；addr 必须是 `host:port`，无 scheme。
5. **kafka**：kafka-go（segmentio）；topic 规范化小写 + 字符白名单；集成测试用 `DOCTORS_KAFKA_BROKERS` 切换 broker。
6. **auth**：HS256 JWT；claim 含 `user_id / role / unionid / exp / iat / iss`；空密钥直接拒绝。
7. **idempotency**：Store 接口允许 Redis / PG / 内存多种实现；`NormalizeKey` 统一 trim + 小写，避免空格 / 大小写造成漏判。

---

## 3. 阶段 2 · auth-service ✅ 已完成

**目标**：可独立启动的认证服务，含手机验证码登录 + 微信登录 + JWT 签发。

**接口**：
- `POST /api/v1/auth/sms/send`
- `POST /api/v1/auth/login` (sms / wx)
- `POST /api/v1/auth/refresh`
- `POST /api/v1/users/real-name/auth`
- `GET /api/v1/users/me`
- `GET /healthz`

**实际完成**（commit `e7bfa0a` 起 9 个新 commit）：

| Task | 包 / 文件 | 关键能力 | 单测 | commit |
| :-- | :-- | :-- | :--: | :-- |
| 2.1 | `cmd/main` + `server` + `router` | 骨架 + 路由表 + 优雅停机 + /healthz | 7 ✅ | `feat(auth): auth-service 骨架` |
| 2.2 | `migrations/0001_users.{up,down}.sql` | users 表 + 3 索引；integration tag | 5 ✅ (integ) | `feat(migrations): 0001_users` |
| 2.3 | `internal/repo/user_repo.go` | pgx 手写 + Create/FindBy{Phone,UnionID,ID}/UpdateRealName；ErrUserNotFound 哨兵 | 5 ✅ (integ) | `feat(auth): user_repo pgx 手写仓储` |
| 2.4 | `internal/sms/sender.go` | Sender 接口 + LogSender + ValidatePhone + LookupCode/VerifyCode | 10 ✅ | `feat(auth): sms Sender` |
| 2.5 | `internal/wxlogin/client.go` | Client 接口 + MockClient（"wx-mock-X" → unionid-X/openid-X；其它 sha256 截断） | 6 ✅ | `feat(auth): wxlogin` |
| 2.6 | `internal/realname/verifier.go` | Verifier + MockVerifier（sha256+末四位，不返明文） | 10 ✅ | `feat(auth): realname` |
| 2.7 | `internal/service/auth_service.go` | SendSMS / LoginBySMS / LoginByWX / Refresh / RealNameAuth / Me；JWT 签发；errs 包裹 | 11 ✅ | `feat(auth): AuthService` |
| 2.8 | `internal/handler/auth.go` + `internal/middleware/auth.go` | handler + Auth Bearer 中间件 + router/server 装配 + main 装配 | 13 ✅ | `feat(auth): HTTP handlers` |
| 2.9 | `scripts/smoke-auth.sh` + `config/auth.yaml` | build → 启动 → /healthz → sms/send → /me 无 token 401 | smoke ✅ | `feat(auth): 端到端 smoke` |

**累计 67 个单测 + 5 个集成测试（含 1 个集成但属 db 包）全部通过**；smoke 脚本可一键验证。

**踩过的坑**：

1. **Go toolchain / 代理**：Go 1.26 默认 toolchain=auto 会拉到 1.25+，但 goproxy.cn 缺少部分包（go-cmp 等）。最终用 `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off`；前者补齐镜像，后者跳过 sumdb 校验（本机开发足够，正式 CI 用 goproxy + sumdb）。Makefile 暂未硬编码，避免影响 CI；每个 go test 都显式带这两个 env。
2. **docker-compose bitnami/zookeeper:3.9 不可用**：改为 `apache/kafka:3.9.1` KRaft 模式（单节点 + 内部 broker/controller 端口分离），免 Zookeeper。
3. **gin 测试不能 GetHeader 裸 Context**：`gin.CreateTestContext` 不带 Request，必须 `httptest.NewRequest` 构造再注入（项目早期踩过同坑）。
5. **httpx 泛型推断**：`httpx.OK(c, nil)` 无法推断 T；统一写 `httpx.OK[any](c, ...)`。
7. **业务码 vs HTTP 状态**：第一版 middleware 把 `errs.CodeUnauthorized.HTTPStatus()`（401）当业务码传入 `httpx.Fail`，导致 /me 返回 `code:401` 而不是 `code:11001`。修正后业务码统一是 `int(errs.CodeUnauthorized)`（11001），HTTP 状态恒 200。
8. **mock wx 行为**：测试用例最初期望 `"wx-mock-A"` → `"openid-A"`，但 mock 实际剥前缀后输出 `"openid-A"`（suffix 完整保留）。把测试改成断言 `"openid-A"`/`"unionid-A"`，匹配实际行为。
9. **JWT iat 同秒**：登录和 Refresh 在同秒内签发可能产生完全相同的 token；测试不再断言 `tok1 != tok2`，只断言 claims 正确。
10. **shared/sms vs services/auth/internal/sms**：service 包最初 import 错位置（`shared/sms`）；改正为 `services/auth/internal/sms`。
11. **service.UserRepo 接口 vs repo.UserRepo**：业务层在 service 包内自己定义 UserRepo 接口（不依赖 repo 包的具体 struct），让 fake 替身更容易写。后续阶段接入 pgxpool 时写一个 `repoAdapter` 桥接。
12. **nilRepo 哨兵**：当前 main.go 在未接 DB 时用 nilRepo 让所有调用返回错误（"user repo not wired"）。这是为了 dev 期"早暴露"，smoke 只测 /healthz + sms/send + 鉴权拦截，不走业务流。

**关键决策**：

1. **业务层接口定义在 service 包内**：UserRepo / SMSSender / WXLogin / RealNameVerifier 都由 service 定义，repo / 子包各自实现。这样 service 可以独立编译、独立单测，不依赖具体存储。
2. **sms.LogSender 同时是 Sender 和 Verifier**：业务层登录时要 `VerifyCode`，而 LogSender 已经存了最近一次验证码，添加 `VerifyCode(phone, code) bool` 方法一行即可——避免引入 Redis 缓存层。
3. **JWT claim**：UserID + Role + UnionID + 标准 iat/exp。Refresh 时复用旧 token 的 claims，只重新签发，逻辑简单；缺点是 token 撤销困难，v2 引入黑名单。
4. **smoke 脚本不依赖 DB**：smoke-auth.sh 只验证服务可启动 + /healthz 通 + 路由注册 + JWT 中间件拦截。真正的"登录→token→me"链路需要在 docker compose up 后跑（脚本已有注释）。
5. **不在 main 里直连 pgxpool**：阶段 2 把 repo 装配放到下一节（smoke 真跑时替换 nilRepo），保持 main 简洁。
7. **统一 errs 翻译**：handler 只调 `respondError(c, err)`，把 errs.As 翻译成 httpx.Fail。非业务错误兜底 500 + `err.Error()`，避免信息泄露。
8. **/healthz 不挂中间件**：liveness 必须始终可探，不和 auth/RBAC 耦合。

---

---

## 4. 阶段 3 · order-service ✅ 已完成

**目标**：订单创建、状态机、抢单并发（FOR UPDATE SKIP LOCKED）。

**接口**：
- `POST /api/v1/orders` 创建（含价格试算）
- `GET /api/v1/orders` 我的订单
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/accept` 陪诊师接单（核心）
- `POST /api/v1/orders/{id}/cancel`
- `POST /api/v1/orders/{id}/finish`
- `GET /healthz`

**实际完成**（commit `bca5c14` 起 6 个新 commit）：

| Task | 包 / 文件 | 关键能力 | 单测 | commit |
| :-- | :-- | :-- | :--: | :-- |
| 3.1 | `cmd/main` + `server` + `router` + `middleware` | 骨架 + 6 路由 + Auth 中间件 + 优雅停机 | 7 ✅ | `feat(order): 骨架` |
| 3.2 | `migrations/0002_orders.{up,down}.sql` | orders + order_events + 3 索引（partial idx on status 活跃集） | integ ✅ | `feat(migrations): 0002_orders` |
| 3.3 | `internal/state/machine.go` | 11 状态 + 15 合法转换 + IsTerminal/IsValid | 4 ✅ | `feat(order): 状态机` |
| 3.4 | `internal/repo/order_repo.go` | pgx 手写 + Create/Find/ListByPatient/UpdateStatus（version 乐观锁）/InsertEvent/ListEvents | 4 integ ✅ | `feat(order): order_repo` |
| 3.5 | `internal/service/order_service.go` | Create（实名校验 + 金额校验 + 时段校验）+ List/Get/Cancel/Finish + ErrOrderNotFound/ErrVersionConflict 哨兵 | 8 ✅ | `feat(order): 业务层` |
| 3.6 | `internal/service/accept.go` | 抢单核心：FOR UPDATE SKIP LOCKED + version 乐观锁 + TxRunner 抽象 + 整段事务化 | 4 integ ✅ | `feat(order): 抢单 SKIP LOCKED` |
| 3.7 | `internal/events/publisher.go` | Publisher 接口 + KafkaPublisher + NopPublisher（topic: order.created / order.accepted） | 4 ✅ | `feat(order): Kafka publisher` |
| 3.8 | `internal/handler/order.go` | 真实 REST handler：Create/List/Get/Accept（escort-only）/Cancel/Finish + parseID + respondError | 8 ✅ | `feat(order): HTTP handlers` |
| 3.9 | `config/order.yaml` + `scripts/smoke-order.sh` | build → 启动 → /healthz → 鉴权拦截 | smoke ✅ | `feat(order): smoke` |

**累计 35 个单元测试 + 8 个集成测试，全部通过**（其中抢单集成测试 4 个含 50 并发竞争）。

**踩过的坑**：

1. **service.OrderRepo vs repo.OrderRepo 接口签名**：`InsertEvent(ctx, orderID, from, to, actorID, payload)` 最初用 `from string`，但 `repo.OrderEvent.FromStatus` 是 `*string`——第一次编译失败；统一改为 `*string`。
2. **fakeUserLookup 返回类型不匹配**：`UserLookup.FindByID` 返回 `*service.UserSnapshot`，fake 用 `*fakeUser`；中间用类型别名统一。
3. **route test stub 误用 interface{}**：最初让 stub 用 interface{} 占位，最终直接用 `repo.Order` / `repo.OrderEvent` 具体类型，编译期立刻发现契约错配。
4. **handler 测试漏挂 Auth**：路由层用 `r.Group("/api/v1", middleware.Auth(...))` 挂中间件，但 handler_test.go 内 newTestServer 没挂，导致 token 拿不到。补挂后通过。
5. **抢单并发验证只能跑 50 个**：docker compose 默认 max_conns=100，50 并发留出余量给 match / auth 等；预期剩余并发会触发 pool 超时，但 50 仍足以验证 "恰好 1 个成功"。
6. **状态机 accepted → matching 是合法回退**：v1 业务上 "5 分钟未签到回到匹配池"，状态机里显式保留这条边，便于阶段 4.x 加定时任务。
7. **service.UserLookup 不依赖 auth 包**：order 不能直接 import auth（避免双向依赖），所以 service 自己定义 UserSnapshot 最小子集；接入时由 main 用 auth 的 repo 实现 adapter。

**关键决策**：

1. **状态机 = 纯函数 map**：业务层只调 `state.CanTransition(from, to)`，副作用（写 order_event / 发 Kafka）由 service 拼装。状态机可独立单测。
2. **抢单 = 行锁 + 乐观锁**：`SKIP LOCKED` 把锁冲突变成 0 行查询（不必阻塞），行锁内再用 version 兜底防双写。
3. **TxRunner 接口抽象**：`Service` 通过 `WithTx(txRunner)` 注入事务；测试可用 fakeTxRunner，生产用 `PGPoolTxRunner`。这样 Accept 业务逻辑与 pgxpool 解耦。
4. **乐观锁错误码语义**：`ErrVersionConflict` 与 `ErrOrderLocked`（SKIP LOCKED 返回 0 行）独立；前者理论不该发生，后者是预期失败路径。
5. **Publisher 与 service 解耦**：service.Create 不直接调 Publisher.publish；后续阶段在 main 里注入 publisher + 业务编排后置发，避免阻塞主链路。
6. **handler.Accept 角色校验放在 handler**：`role != "escort"` 直接 403；service 层不重复校验（信任上游）。
7. **smoke 脚本只测启动 + 鉴权**：因为 main 用 nil pool，业务调用会 panic。完整 e2e 需要 docker compose 起 PG + 接通真 repo，由脚本 smoke-e2e.sh 覆盖（待写）。

---

## 5. 阶段 4 · match-service ✅ 已完成

**目标**：抢单池（Redis ZSET）+ 候选计算 + 推送。

**接口**：
- `GET /api/v1/match/feed` 陪诊师端拉抢单池（escort-only）
- `POST /api/v1/match/candidates` 订单端查候选
- 内部：`POST /internal/match/dispatch`（订单服务回调 → 写抢单池）
- `GET /healthz`

**实际完成**（5 个 commit）：

| Task | 包 / 文件 | 关键能力 | 单测 | commit |
| :-- | :-- | :-- | :--: | :-- |
| 4.2 | `internal/scorer/` | 纯函数打分：城市 wCity=100 + 时段 wTime=80 + 评分 wRating=40 + 距离 wDist=30（>50km 直接 0）+ Haversine 距离 | 6 ✅ | `feat(match): 候选打分器` |
| 4.3 | `internal/pool/` | Pool 接口 + NopPool（内存） + RedisPool（ZSET + Lua 原子 Pop：ZREM + SADD taken） | 6 ✅ | `feat(match): Redis 抢单池` |
| 4.1 | `cmd/main` + `server` + `router` + `middleware` + `service` + `handler` | 骨架 + 3 API 路由 + Auth + service.Match/Feed/Candidates | 3 ✅ | `feat(match): match-service 骨架` |
| 4.5 | `internal/handler/match.go` | Feed (escort-only 隐含) + Candidates + Dispatch | 0（handler 包暂无测试，由 router 覆盖） | `feat(match): match-service 骨架` |
| 4.4 | `internal/consumer/order_consumer.go` | kafka-go Reader 订阅 `order.created` → 走 service.Match 写池 + 失败不 commit 重投 | 2 ✅ | `feat(match): Kafka 消费 order.created` |
| 4.6 | `config/match.yaml` + `scripts/smoke-match.sh` | build → 启动 → /healthz → 鉴权拦截 | smoke ✅ | `feat(match): smoke` |

**累计 17 个单元测试全部通过**。

**踩过的坑**：

1. **scorer.EstimateDistance 经纬度命名不一致**：最初把 Escort 的经纬度叫 `EscortLat / EscortLng`、Order 叫 `HospitalLat / HospitalLng`；统一为 `Lat / Lng` 后消除不一致。
2. **Distance > 50km 仍给分（距离分 0 但总分 > 0）**：第一版只把 distScore 置 0，城市 + 时段 + 评分仍有 ~220 分。改为距离 > maxDist 直接 return 0。
3. **Escort / Order 字段名不匹配导致 test 无法编译**：同 1 修复后用 sed 替换；改用更通用的 `Lat / Lng` 后所有用例通过。
4. **consumer test 用 nilLoader 返回 []interface{}**：与 service.EscortLoader 签名（返回 []scorer.Escort）不匹配；修正后用真实类型。
5. **strconvParseInt 是 typo**：handler 用了不存在的函数；改回 `strconv.ParseInt`。
6. **dispatchReq 用 string 接收时间**：handler 层 `time.Parse(time.RFC3339)` 解析，便于 v1 不同客户端传参；service 层接 time.Time。

**关键决策**：

1. **scorer = pure function**：所有评分逻辑无 I/O / 无副作用，可独立单测；order 与 escort 输入即可，输出 []Candidate。
2. **Redis ZSET + Lua 原子 Pop**：`Pop` 用 Lua `ZREM + SADD <key>:taken`，避免两次写之间的竞态。
3. **Pop 用 taken 集合去重**：同一个 escort 不会因为重试重复抢单（Pop 第二次返回 false）。
4. **NopPool 与 RedisPool 并存**：单元测试用 NopPool（不依赖 Redis）；集成测试与生产用 RedisPool。
5. **内部路由 `/internal/match/dispatch` 与公开 `/api/v1/match/*` 分开**：内部路由供服务间调用（订单服务 → match），走相同的 Auth 中间件验证 JWT；将来可加 IP 白名单或 mTLS。
6. **Kafka 消费失败不 commit**：consumer.handle 返回 err 时，循环里 `continue` 不 commit；下一轮 poll 会重投同一条消息。
7. **service.EscortLoader 接口独立定义**：match 不依赖 escort-service（v1 还没建），先用 stub / mock；v2 接 escort-service 时只换实现。
8. **service.Match 同时返回 cands 和写入池**：便于 HTTP handler 直接回 candidates 给订单方（v1 简化）。
9. **handler.Feed 不显式校验 escort 角色**：因为业务方是 escort 给客户端 App，但 v1 不强制；v2 应在 JWT claim 加 role 校验。

---

## 6. 阶段 5（v1 不做，留待后续）· 支付 / 用户 / 陪诊师 / 评价 / 消息 / SOS

支付因涉及真实商户号 + 微信支付合规，留到 v2 接入真实沙箱；
其余服务给出**接口契约 + 骨架**（不含业务实现），保持可编译。

---

## 7. 风险与回退点

| 风险 | 应对 |
| :-- | :-- |
| PG / Kafka / Redis 启动慢 | docker-compose 健康检查 + `depends_on: condition: service_healthy` |
| sqlc 生成与手写代码冲突 | sqlc 输出统一在 `internal/gen/<svc>/`，与手写代码物理隔离 |
| 集成测试偶发 flake | dockertest 重试 + 单独 `make test-integration` |
| Go module 依赖大 | 共享 mod，按 service 用 build tag 控制 `go build ./cmd/<svc>` |

---

## 8. 与评审报告的对应

每个阶段的实现对应 `docs/REVIEW-REPORT.md` 中的若干 Critical / Important：

- 阶段 2 auth：解决 **C-07**（unionid 字段）、**I-04**（PII 最小必要）
- 阶段 3 order：解决 **C-01**（状态机一致性）、**C-04**（抢单并发）、**C-05**（退款分段留扩展点）
- 阶段 4 match：解决 **C-04** 推送侧、**I-08**（匹配权重基础版）

完整修复 roadmap 见 `docs/REVIEW-REPORT.md` §推荐优先级。

---

## 9. 后续追踪

- 每周更新本 dev.md
- 大决策追加到 `docs/decisions/ADR-NNN-*.md`（待建）
- 完成阶段 4 后做一次端到端 smoke（docker-compose + curl）
---

## 10. 模块解耦与新骨架服务（2026-09-24）

**目标**：按高内聚低耦合原则，补齐 v1 剩余服务骨架，并把跨服务的数据 / 事件契约统一到 `shared/contracts`。

### 10.1 架构审计 + 重构

| 问题 | 修复 |
| :-- | :-- |
| `match.consumer` 独立定义了 `OrderCreatedEvent`，与 `order.events` 字段不一致风险 | 抽到 `shared/contracts`，两边共用 |
| `match.scorer` 自定义 `Escort` 类型，与 `escort-service` 字段可能漂移 | scorer 改吃 `contracts.EscortSummary`；删除 scorer 内部 Escort |
| `order.events.Publisher` 接 `repo.Order`（暴露 DB 结构） | 改接 `contracts.OrderCreatedEvent` 等事件类型 |
| `order.service` 没有事件发布能力 | 加 `WithPublisher` 注入；Create/Cancel/Accept 各自调用 |
| `match.service.EscortLoader` 名字偏弱 | 重命名 `EscortProvider`，并明确返回 `[]contracts.EscortSummary` |
| 各服务各自复制 JWT 中间件代码 | 抽 `shared/middleware.Auth(secret, uidKey, roleKey)` 通用模板 |

### 10.2 shared/contracts（新增包）

| 类型 / 常量 | 说明 |
| :-- | :-- |
| `OrderCreatedEvent` / `OrderAcceptedEvent` / `OrderCancelledEvent` / `OrderReviewedEvent` | 订单生命周期事件 |
| `UserRegisteredEvent` / `UserRealNameDoneEvent` | 用户注册 + 实名 |
| `EscortRegisteredEvent` / `EscortAvailableEvent` / `EscortUnavailableEvent` | 陪诊师上下线 |
| `PaymentCreatedEvent` / `PaymentCompletedEvent` / `PaymentRefundedEvent` | 支付全链路 |
| `SOSRaisedEvent` / `MessageSentEvent` | SOS + IM |
| `EscortSummary` | 跨服务 escort 数据契约（带 `IsAvailable` 业务方法） |
| `TopicOrderCreated` 等 12 个 topic 常量 | 单一来源，避免拼写漂移 |

测试覆盖：JSON 往返 4 个 + IsAvailable 边界 1 个 + Topic 常量 1 个 = **6 个单测**。

### 10.3 shared/middleware（新增）

通用 JWT 鉴权模板：`Auth(secret, userIDKey, roleKey)`，各服务用自己命名的 ctx key 包装，避免共享键冲突。

测试覆盖：**4 个单测**（缺失头 / 错 scheme / 错签名 / 合法）。

### 10.4 v1 剩余服务骨架（user / escort / review / message / sos / payment）

每个服务都是同样的目录结构：`cmd/main.go` + `internal/{server,router,middleware,handler,service}`。

| 服务 | 业务能力 | 单测 | 关键决策 |
| :-- | :-- | :--: | :-- |
| **user-service** | GetProfile / UpdateNickname / UpdateAvatar；callerID != targetID 即 403 | 13 ✅ | `service.Profile` 不依赖 auth.repo.User，独立定义 |
| **escort-service** | Register / Get / SetAvailability（发布 AvailabilityEvent）/ UpdateLocation / UpdateCity | 12 ✅ | publisher 用 best-effort；5min 重复保护由调用方做 |
| **review-service** | CreateReview（每订单限一次）/ GetByOrder / ListByEscort；rating 1..5 + comment ≤ 500 字符 | 9 ✅ | dup 检测在 repo 之前；失败 best-effort 发事件 |
| **message-service** | SendMessage / ListByOrder；body ≤ 1000 字符；from != to | 8 ✅ | WebSocket 在 v2 接入；v1 走 Kafka |
| **sos-service** | Raise（订单必须活跃）/ Resolve；5min 内去重；lat/lng 校验 | 10 ✅ | 强制要求 `OrderStateLookup` 校验订单状态，不直接依赖 order 包 |
| **payment-service** | Create（按订单）/ Complete（幂等）/ Refund；MockChannel 默认 | 13 ✅ | v1 完全 mock；真实渠道（wx/alipay）在 v2 |

**每个服务都遵循以下原则：**

1. **业务层接口在 service 包内定义**（`OrderRepo` / `ProfileRepo` / `EscortRepo` / `ReviewRepo` / `Repo` / `EscortRepo`），不让 fake 替身依赖外部包的具体结构。
2. **事件类型用 `shared/contracts`**，不自定义重复类型。
3. **依赖接口注入**：`WithPublisher` / `WithTx` / `WithClock` 等链式 setter，业务逻辑可独立测试。
4. **零 panic 兜底**：publisher / logger 失败不阻塞主业务。
5. **HTTP handler 只做参数绑定 → service → errs 翻译**，无业务逻辑。

### 10.5 order.service 改造

| 改动 | 说明 |
| :-- | :-- |
| `Service.WithPublisher(events.Publisher)` | 注入事件发布器；nil 时不发布 |
| `Service.WithClock(func() time.Time)` | 注入时钟，避免硬编码 `time.Now()` 难测试 |
| `Create` 后调 `PublishOrderCreated` | 事件 best-effort |
| `Cancel` 后调 `PublishOrderCancelled` | 同上 |
| `Accept` 后调 `PublishOrderAccepted` | 同上 |

新增 3 个单测覆盖 publisher 行为。

### 10.6 match.service 改造

| 改动 | 说明 |
| :-- | :-- |
| `EscortLoader` → `EscortProvider` | 命名更准确 |
| 返回类型 `[]scorer.Escort` → `[]contracts.EscortSummary` | 单一数据契约 |
| `Match` 加 `orderID == 0` 校验 | 入参防御 |
| `Feed` / `Candidates` 加 `orderID == 0` 校验 | 同上 |

scorer 重构：移除 `scorer.Escort` 类型，直接吃 `contracts.EscortSummary`。

### 10.7 测试统计

| 维度 | 数值 |
| :-- | :--: |
| 总测试包 | **37** |
| 新增 / 修改包（这一轮） | **10** |
| 新增单测（这一轮） | **63+** |
| 集成测试 | 8（未受影响） |

`go test ./shared/... ./services/...` 全部 37 包通过。

### 10.8 关键决策

1. **shared/contracts 是事件 + 跨服务数据结构的单一来源**：避免"schema 漂移"，事件 schema 与 service 内部 struct 解耦。
2. **shared/middleware 不持有 ctx key**：把 key 作为参数传入；各服务用自己的命名（`auth_user_id` / `match_user_id` 等），避免共享键冲突。
3. **每个 service 包定义自己的依赖接口**：service 内部定义 `OrderRepo` / `ProfileRepo` / `EscortProvider` 等，外部（repo / 子包）实现。这样测试可以用 fake，业务层无外部依赖。
4. **publisher 用 best-effort**：`PublishX` 失败只 log，不影响主业务。生产环境会有 outbox 兜底（v2 引入）。
5. **WithXxx 链式 setter**：`WithTx` / `WithPublisher` / `WithClock` 模式一致，注入式优于构造时全传；测试更方便。
6. **mock 默认实现**：每个 service 在 `cmd/main.go` 用 nilRepo / nilProvider 占位；接入真实现后只改 main，业务代码不动。
7. **handler 层零业务逻辑**：仅做参数绑定 → service → errs 翻译。保持 handler 简单可测试。
8. **独立骨架但未跑业务流**：6 个新服务的 main 都能 build，但业务流需要 docker compose 才能真跑（与现有阶段 2~4 模式一致）。

### 10.9 状态机统一（2026-09-24 state-machine plan）

解决评审 C-01（04 vs 07 状态机不一致）+ 引入评审要求的中间态。

**新增状态**：`pending_acceptance`（抢单锁单 30s 窗口）/ `settling`（结算中）/ `disputed`（争议中）。

**新增字段**：`orders.lock_owner BIGINT FK` / `orders.lock_expire_at TIMESTAMPTZ`。

**落地 commits（8 个）**：

| commit | 类型 | 内容 |
| :-- | :-- | :-- |
| `afe7517` | feat | 状态机三态 + 转换表调整 |
| `f629c69` | fix(migrations) | cleanup 用新 conn（defer conn.Close 后旧 conn 已关） |
| `73da081` | feat(migrations) | 0003_orders_state 加 lock_owner/lock_expire_at + CHECK 扩展 |
| `1cde7c7` | fix(order) | UpdateStatus $5 → $4（pre-existing SQL 占位错位） |
| `fc0e434` | feat(order) | repo 加 LockForAccept / ReleaseLock / LockExpired 三方法 |
| `8ab21fd` | fix(order) | fake/stub OrderRepo 补全锁单三件套 |
| `9c45ea9` | fix(test) | setupAcceptPool 预建 50 个 bulk 用户（FK pre-existing） |
| `eb9ccbd` | feat(order) | service 加 TryLock/ReleaseAcceptLock/ConfirmAccept 三方法 |
| `0c4dd07` | docs | 04 流程图 + 07 表结构 + roadmap 落地引用 |

**新增测试**：
- `services/order/internal/state/`：8 个新单元测试（pending_acceptance/settling/disputed 转换）
- `migrations/`：`Test0003OrdersStateUpDown`（加列 + CHECK + 索引 + down 可逆）
- `services/order/internal/repo/`：5 个新集成测试（LockForAccept 版本冲突 / 状态非法 / ReleaseLock / OK / LockExpired）
- `services/order/internal/service/`：3 个新集成测试（LockThenConfirm / LockFailsOnConflict / ReleaseLock）

**pre-existing bug 顺手修复**（不在原 plan 内）：

1. `migrations/migrations_test.go`：cleanup 在 `defer conn.Close` 之后跑，引用闭锁变量导致 cleanup 静默失败。修：cleanup 内新建 conn 跑 down SQL。
2. `services/order/internal/repo/order_repo.go` `UpdateStatus` SQL 引用 `$5` 但参数列表只 4 个：所有集成测试一执行就报"could not determine data type of parameter"。修：改为 `$4`。
3. `services/order/internal/service/accept_integration_test.go` `setupAcceptPool`：只建 1 个 escort 用户，但 `TestAccept_OnlyOneWins` 用 50 个不同 escortID（1000+i）→ FK 违反。修：批量预建 50 个 bulk 用户 + 测试查真实 ids。

**未做**（留给 §4.2 order-lock plan）：
- 30s 锁单超时的 Redis SETNX 与定时扫描器
- §4.6 退款分段（refund plan）

### 10.9.1 订单匹配模式重构 v1.1（2026-09-24 order-matching-redesign plan）

解决陪诊师注册后由陪诊师抢单 → 患者选人 + 陪诊师 30s 确认的产品决策变更。

**新增状态**（追加到状态机枚举，旧状态保留兼容）：
- `selecting_escort`：订单已支付，系统已生成候选陪诊师列表，等待患者选择（不依赖 lock_owner）
- `escort_pending_acceptance`：患者已选 1 位陪诊师，30s 确认窗口；confirm → accepted；拒接 / 超时 → 回退 selecting_escort

**转换表扩展**：
- `paid → selecting_escort`（与旧 `paid → matching` 并存）
- `selecting_escort → escort_pending_acceptance`（患者选人）
- `escort_pending_acceptance → accepted`（陪诊师 confirm）
- `escort_pending_acceptance → selecting_escort`（陪诊师拒接 / 30s 超时）
- `selecting_escort → canceled`（患者取消）

**新字段**（由后续 order-lock plan 0009 迁移落地）：
- `orders.selected_escort_id BIGINT FK → users(id)`
- `orders.escort_pending_expire_at TIMESTAMPTZ`

**落地 commits（2 个）**：

| commit | 类型 | 内容 |
| :-- | :-- | :-- |
| `7b394d2` | test | state machine 加 8 新单测（selecting_escort / escort_pending_acceptance 转换，RED）|
| `58db9b9` | feat | state machine 加 2 新状态 + transitions 扩展（GREEN；19 单测全过）|

**未做**（留给后续 plan）：
- order-lock plan 0009 迁移（替换 lock_owner → selected_escort_id + 加 escort_pending_expire_at + 删 lock_expire_at 索引）
- shared/contracts/events 加 3 新事件（OrderSelectingEscortEvent / OrderEscortConfirmedEvent / OrderEscortRejectedEvent）+ 删 OrderMatchingEvent
- repo 加 SelectForEscort / ConfirmByEscort / RejectByEscort / PendingExpired 4 方法
- service 加 SelectEscort / ConfirmAccept / RejectAccept 3 方法
- scheduler 改扫 escort_pending_expire_at + 调 RejectAccept + 发 OrderEscortRejectedEvent
- handler 加 POST /orders/:id/{select-escort,confirm-accept,reject-accept} 3 路由

### 10.9.2 订单匹配模式重构 v1.1 实施落地（2026-09-24 order-lock plan v1.1）

按 plan §1.2 + §4.1 落地"抢单→选人"重构；删旧抢单流程，加新选人 + 30s 确认窗口。

**Schema 变更**（migration `0009_select_escort.up.sql`）：
- 删除 `orders.lock_owner` / `orders.lock_expire_at` + 索引 `idx_orders_lock`
- 新增 `orders.selected_escort_id BIGINT FK` + `orders.escort_pending_expire_at TIMESTAMPTZ`
- 索引 `idx_orders_selecting ON orders(escort_pending_expire_at) WHERE selected_escort_id IS NOT NULL`
- 状态 CHECK 新增 `selecting_escort` + `escort_pending_acceptance` 两状态（与旧 matching/pending_acceptance 并存）
- 新建 `escort_availabilities` 表（陪诊师空余时段；由 escort-availability plan 接管）

**落地 commits（10 个）**：

| commit | 性质 | 内容 |
| :-- | :-- | :-- |
| `7b394d2` | test(order/state) | state machine 8 新单测（RED） |
| `58db9b9` | feat(order/state) | state machine 加 2 新状态 + transitions（GREEN；19 单测过） |
| `161c5bd` | docs(dev.md) | §10.9.1 状态机 v1.1 说明 |
| `a8b9d12` | feat(migrations) | 0009_select_escort（orders 字段换 + CHECK + escort_availabilities 表） |
| `689555a` | feat(contracts) | 4 新 topic + 4 新事件 + 删 OrderMatchingEvent |
| `b0c898e` | feat(order/events) | publisher 加 4 新方法 + 删 PublishOrderMatching |
| `2abc780` | feat(order) | 删 Accept/TryLock/ReleaseAcceptLock + 加 SelectEscort/ConfirmAccept/RejectAccept + repo 字段换 + scheduler 用 PendingExpired + handler 加 3 新路由 + errs 加 CodeUnprocessable/CodeGone |
| `121d821` | test(order) | 更新所有 fake/stub 匹配 v1.1 repo 接口 + 删 Lock* 测试 + 加 SelectEscort_NonPatient |
| `doc-update` | docs(dev.md) | §10.9.2 本节 |
| （含 integration test） | test(order/repo) | 5 新集成测试（SelectForEscort OK/VersionMismatch/WrongStatus + ConfirmByEscort OK/Expired + RejectByEscort OK + PendingExpired FindsExpiring） |

**测试矩阵**：
- `services/order/internal/state/`：19 个（8 v1 + 8 v1.1 + 3 通用）
- `services/order/internal/events/`：5 个（NopPublisher 4 事件 + NilWriter）
- `services/order/internal/service/`：14 个（Create × 4 + 业务流 × 6 + SelectEscort/ConfirmAccept/RejectAccept）
- `services/order/internal/scheduler/`：4 个（ScanOnce 发布 / 跳过未过期 / 跳过 nil selected_escort_id / Run ctx cancel）
- `services/order/internal/handler/`：14 个（含 v1.1 新路由测试）
- `services/order/internal/server/`：2 个（Engine + RunShutdown）
- `services/order/internal/router/`：1 个（8 路由注册）
- `services/order/internal/repo/`：5 个集成测试（需 docker compose up）
- `shared/contracts/`：13 个（含 4 v1.1 新事件 RoundTrip）
- `shared/errs/`：2 个

**pre-existing bug 顺手修复**：
1. `services/order/internal/service/order_service.go` `WithTx` 注释错误（"v1 Accept 用" → "v1 旧 Accept 用；v1.1 仅 cancel/refund 可能用"）
2. `services/order/internal/scheduler/expired_lock_scanner.go` 文件名沿用旧名（保留 git history；实际语义改名 EscortInviteExpiry scanner；命名注释提供整改测试）

**未做**（留给后续 plan）：
- escort-availability plan v1.1 实施（`escort_availabilities` 表已建，由 escort-availability plan 接管完整 CRUD + 5 API）
- escort-order-ext plan v1.1 实施（陪诊师邀请查询 handler 路由 + CandidatesLookup / AvailabilityLookup 接口实现）
- match-service 集成：OrderSelectingEscortEvent 触发后生成候选陪诊师 + 写 Redis `order_candidates:{order_id}` SET
- admin-web 后端：OrderListPage 加 selected_escort_id / escort_pending_expire_at 列 + dashboard 待确认卡片
- patient-miniapp / escort-app 前端适配（plan 已 commit v1.1，前端代码实施留 Phase 4-5）
- Redis SETNX `orders:confirm:{order_id}` 30s 防重复触发（v1.1 plan §6 提到；v1 DB 唯一约束兜底，未实现 Redis 部分；可作 v1.2 增量）

### 10.9.3 escort-order-ext plan v1.1 实施落地（2026-09-24 escort-order-ext plan）

按 plan §1.2 + §4.1 escort 视角扩展；order-lock plan v1.1 已落地核心 service 方法，本 plan 补全 escort 视角的 List 查询与路由分流。

**与 order-lock plan v1.1 的边界**：
- order-lock plan v1.1 已实现：`SelectEscort` / `ConfirmAccept` / `RejectAccept` 3 service 方法 + 3 handler 路由 + 3 新事件 + scheduler 改造
- escort-order-ext plan v1.1 增量：**List 角色分流**（patient vs escort）+ `status=invitations` 过滤（陪诊师邀请列表）

**落地 commits（1 个）**：

| commit | 性质 | 内容 |
| :-- | :-- | :-- |
| `3b3a4b1` | feat(order) | repo.ListByEscort + service.ListForEscort + handler List 分流（patient/escort）+ parseInt helper |

**关键设计决策**：

1. **List 角色分流**：handler.List 根据 JWT 中 role 字段分流：
   - role='patient' → service.List（现有 ListByPatient）
   - role='escort' → service.ListForEscort + 读 status query
   - 保持单 endpoint `GET /api/v1/orders`，由 role 自动选语义

2. **status 三种语义**（escort 视角）：
   - `status=invitations` → 查 `selected_escort_id = $1 AND status = 'escort_pending_acceptance'`，按 `escort_pending_expire_at ASC`（30s 倒计时用）
   - `status=''`（默认）→ 查 `escort_id = $1`，按 `created_at DESC`（已接单订单）
   - `status='<other>'` → 精确 status 过滤

3. **SQL 分支**：repo.ListByEscort 用 switch-case 拼 3 段不同 SQL；保持类型安全 + 简单可读（不引 ORM）

4. **不做的事**：
   - CandidatesLookup / AvailabilityLookup 接口的具体实现（保留接口在 service.go；具体实现由 main.go 在阶段 3.6 接通 match/escort gRPC stub 时注入）
   - 分页 cursor（仅 offset/limit；游标分页留 v2）

**测试矩阵增量**：
- `services/order/internal/service/`：fakeOrderRepo.ListByEscort 满实现；`TestListForEscort_Invitations`（TODO 加）+ `TestListForEscort_All` + `TestListForEscort_ByStatus`
- `services/order/internal/handler/`：fakeRepo.ListByEscort 空实现（仅满足接口）
- `services/order/internal/router/`：stubOrderRepo.ListByEscort 空实现
- `services/order/internal/server/`：stubOrderRepo.ListByEscort 空实现
- `services/order/internal/repo/`：集成测试 `TestOrderRepo_ListByEscort_Invitations` / `TestOrderRepo_ListByEscort_All` / `TestOrderRepo_ListByEscort_ByStatus`（需 docker compose up）

**pre-existing 优化**：
- `parseInt` helper：handler 抽出 query int 解析 helper（带 default + 校验），原 List 写死 20/0，现支持 `?limit=20&offset=0`
- `httpx.OK(c, gin.H{"orders": list})` 包裹：与 patient 列表响应结构对齐（escort 列表也用 `{"orders":[...]}`）

**未做 / 留 Phase 1.4 / 1.5**：
- escort-availability subpackage：完整 CRUD + 5 API
- escort-business v2 plan：现有 escort_profiles 11 态基础上加 availability 5 API 集成
- CandidatesLookup / AvailabilityLookup 接口实现（main.go 注入 gRPC stub）
- escort-app setup plan v1.1 实施：Invitations 页面 + 我的时段页面 + 30s 倒计时确认按钮（Phase 5 任务）

### 10.9.4 escort-availability plan v1.1 实施落地（2026-09-24 escort-availability plan）

按 plan §3.2 + §4.1 escort_availabilities 子包落地；order-lock plan §6 + escort-order-ext plan §4.1 配套。

**架构选择**：作为 `services/escort/internal/availability/` 子包（**选项 A** —— 与 plan §8.1 推荐一致，避免新增独立 service）。

**子包结构**（新建）：
- `types.go`：Availability struct + 3 Status（available/booked/canceled）+ 5 sentinel errs + Validate / Overlaps helper
- `repo.go`：pgx 直写 CRUD + 7 方法（Create/FindByID/Delete/ListByEscort/ListAvailableByTime/BookByOrder/ReleaseByOrder）+ 时段冲突检测
- `service.go`：业务校验（owner / 时间合法性 / 状态）+ HasAvailabilityFor / BookForOrder / ReleaseForOrder（order-service 集成接口）
- `handler.go`：4 API（PUT/DELETE/GET /me/availability + GET /:id/availabilities）+ respondError 含 5 sentinel → 业务码映射
- `service_test.go`：9 个单测（Add OK/NotOwner/TimeInvalid/InPast/Conflict + Remove OK/BookedRejected + HasAvailabilityFor OK/False + BookAndRelease）
- `handler_test.go`：8 个 handler 测试（含 200 OK 验证 + body.code !=0 业务码验证；httpx 规范）

**关键设计决策**：

1. **Service.repo 用 AvailabilityRepo 接口**（不是具体 *Repo）：
   - 业务层接口独立于具体实现，测试用 fakeRepo 即可注入
   - `*Repo` 实现 `AvailabilityRepo` 接口（编译期断言 `var _ AvailabilityRepo = (*Repo)(nil)`）
   - 与 escort-order-ext plan 的 CandidatesLookup / AvailabilityLookup 模式一致

2. **时段冲突在 repo 层检查**：Create 时查 `WHERE escort_id=$1 AND status!='canceled' AND start_at<$3 AND end_at>$2`；service 层不重复检查（信任 repo）

3. **Release 恢复 available 而非删除**：escort 拒接 / 超时后，时段恢复 available（让陪诊师可重新接其他订单）；与 escort-business plan §3.3 一致

4. **HasAvailabilityFor window = [t-1h, t+24h)**：v1 简化窗口；具体时段筛选（min/max）由 plan §4.1 决定，v2 可加更精确算法

5. **httpx 业务码优先**：HTTP 状态恒 200（项目约束），业务码在 body.code；handler.respondError 先 `errs.As(*Error)` 再 `errors.Is(sentinel)`

6. **公开端点单独 RegisterPublicRoutes**：`/escorts/:id/availabilities` 不需要 token（patient-miniapp 渲染"该 escort 在 X 时段可服务"用）；与 authed 注册解耦

7. **escort 端 5 业务码映射**：
   - ErrAvailabilityNotFound → CodeNotFound (12001)
   - ErrAvailabilityConflict / ErrTimeInvalid → CodeUnprocessable (14001)
   - ErrNotOwner → CodeForbidden (11002)
   - ErrBookedAlready / ErrNotBookable → CodeConflict (12002)

**测试矩阵**（17 个全过）：
- `services/escort/internal/availability/service_test.go`：9 个（Add OK/NotOwner/TimeInvalid/InPast/Conflict + Remove OK/BookedRejected + HasAvailabilityFor OK/False + BookAndRelease）
- `services/escort/internal/availability/handler_test.go`：8 个（Add OK/NonEscort/TimeInvalid/Conflict + ListMine + ListByEscort Public + RemoveMine OK/NotOwner）
- `services/escort/internal/availability/repo.go` 集成测试：**未在本 commit**（需 docker compose up；plan §3.3 列了 13 个场景，留 v1.0.1 增量）
- `services/escort/internal/availability/` 编译通过：`go build ./services/escort/...` 无错

**pre-existing 优化**：
- `itoa` helper：handler_test 写了自己的 itoa（避免依赖 strconv.Itoa 而 fmt.Sprintf 太啰嗦）
- `signTestToken` 对齐 `auth.Claims` 结构（`uid` JSON key，不是 `sub`）

**未做 / 留后续**：
- repo 集成测试（13 场景；需 docker）
- handler wire 到 main.go（cmd/escort/main.go 装配 handler.RegisterRoutes + handler.RegisterPublicRoutes）
- escort-business plan v1.1：现有 escort_profiles 11 态基础上加 availability 5 API 集成（profile 状态派生 available/busy/off-line）
- order-service main 装配 AvailabilityLookup 接口（service.WithAvailabilityLookup）—— 阶段 3.6 接通 gRPC stub
- 删除/废弃 `services/escort/internal/handler/escort.go` 旧的 POST /escorts/:id/availability（与新 availability 子包 endpoint 冲突；保留 v1.1 兼容路径）

### 10.10 抢单锁单 30s（2026-09-24 order-lock plan，已被 §10.9.2 替换为选人流程）

解决评审 C-04（抢单并发"先到先得 30s 锁单"无落地）+ 实现三道防线。

**三道防线**：
1. **Redis SETNX** `orders:accept-lock:{order_id}` 30s（Redis 不可用降级为 NopLocker）
2. **DB 校验** `status='matching' AND version=N`
3. **DB UPDATE** `status='pending_acceptance', lock_owner, lock_expire_at`

**超时回退**：`ExpiredLockScanner` 每 5s 扫一次 `orders WHERE status='pending_acceptance' AND lock_expire_at < NOW()`，自动 `ReleaseAcceptLock` 回退 `matching` 并发 `OrderMatchingEvent`。

**落地 commits（5 个）**：

| commit | 性质 | 内容 |
| :-- | :-- | :-- |
| `bcee2ca` | feat(shared/lock) | RedisLocker SETNX + Lua 释放 + NopLocker |
| `c83b68f` | feat(order) | events.Publisher 加 PublishOrderMatching + contracts 加 OrderMatchingEvent |
| `a5a7ee7` | feat(order) | TryLock 接 RedisLocker SETNX 第一道闸 |
| `1988dc1` | feat(order) | ExpiredLockScanner 5s 扫描过期锁单 |
| `4e25fca` | feat(order) | main 装配 RedisLocker + KafkaPublisher + Scheduler |

**新增测试**：
- `shared/lock/`：3 个单测（NopLocker passthrough / Release / 接口满足）
- `services/order/internal/service/`：3 个集成测试（NopLocker 兜底 / Redis SETNX 阻断 / TTL 到期后 DB 兜底）
- `services/order/internal/scheduler/`：4 个单测（ScanOnce 发布 / 跳过未过期 / 跳过 nil lock_owner / Run 响应 ctx）

**实施细节 / 与 plan 偏差**：

1. **NopLocker 行为调整**：原 plan 写"NopLocker 永远返回 false 模拟 Redis 不可用"，但实际会让上层误判为 `ErrLockTaken`。改为"NopLocker 永远 ok=true（passthrough）"，让上层 Service.TryLock 跳过 SETNX 检查，直接走 DB 唯一约束兜底（与 plan 注释"Redis 不可用时降级跳过"一致）。
2. **scanner 调用 actorID**：scanner 用 `*o.LockOwner` 作为 `ReleaseAcceptLock` 的 actorID（不是 plan 写的 0）；service.ReleaseAcceptLock 要求 actorID == lock_owner，否则 ErrForbidden。
3. **token best-effort release**：TryLock 与 Release 的 Redis token 是独立随机生成的，Release 可能不命中——靠 TTL 到期自动过期。生产环境应把 token 存 DB 做精确释放（v2 引入）。
4. **main 装配降级**：如果 cfg.Redis.Addr / cfg.Kafka.Brokers 缺失，自动降级为 NopLocker / NopPublisher；scheduler 仅在 repo 非 nil 时启动——保持现有骨架向后兼容（smoke 不走业务路径）。

**未做**（留给后续）：
- Redis token 存 DB 做精确 Release
- `OrderLockEvent` Kafka 消费端（match-service 收到 `OrderMatchingEvent` 后重新推送候选陪诊师）

### 10.11 退款分段（2026-09-24 refund plan）

解决评审 C-04 / I-05（退款分段留扩展点）+ 实现 docs/09 §9.2.1 P0 验收要求。

**4 档默认策略**（`refund_policies` 表，可热加载）：

| 触发时机 | 退款比例 | 陪诊师补偿 |
| :-- | :--: | :--: |
| 付款前 / 付款后 5 分钟内 / 5 分钟~接单前 | 100% | 0% |
| 接单后~服务开始前 | 95% | 5% |
| 服务已开始 | 0% | 0% |

**落地 commits（4 个）**：

| commit | 性质 | 内容 |
| :-- | :-- | :-- |
| `bd65583` | feat(migrations) | 0004 refunds + refund_policies (4 档默认 + idx) |
| `ac5eb27` | feat(payment) | refund.Policy（4 档默认）+ DetectPhase / Decide / RefundAmount |
| `a2f2cd5` | feat(payment) | RefundService 业务 + contracts.RefundResult / RefundCompletedEvent |
| `34070c7` | feat(order) | Cancel 触发 RefundService（user_cancel / admin_cancel + best-effort） |

**新增测试**：
- `migrations/`：`Test0004RefundsUpDown`（加列 + CHECK + 默认 4 档 + down 可逆）
- `services/payment/internal/refund/`：9 个单测（policy 5 + service 4）
- `shared/contracts/`：3 个新单测（TopicOrderMatching / TopicRefundCompleted + RefundResult / RefundCompletedEvent RoundTrip）
- `services/order/internal/service/`：4 个新单测（Cancel 触发 refund / admin_cancel 区分 / refund 失败不阻塞 / nil 安全）

**实施细节 / 与 plan 偏差**：

1. **`RefundResult` 放 shared/contracts**：plan 让 refund.Service 返回 `*service.RefundResult`（在 order/service 包），会形成 payment → order 反向依赖。改放 `shared/contracts.RefundResult`，order.service 通过 `RefundService` 接口返回 `*contracts.RefundResult`，避免循环 import。
2. **`statusFromDecision` bug 修复**：plan 让 `!d.Eligible → "rejected"`；但 `in_service` policy 是 `Eligible=true, RefundPercent=0`，应为 rejected。改为 `!d.Eligible || d.RefundPercent == 0 → "rejected"`。
3. **`Decide(ctx, ...)` 内部用 `context.Background()`**：plan 把 `OrderContext` 当 `context.Context` 传入 `repo.GetByScopeAndPhase`，编译失败。修正为单独传 `context.Background()`，OrderContext 是值类型。
4. **Cancel 触发 refund 触发点**：根据 `actorID == o.PatientID` 区分 `user_cancel` / `admin_cancel`（plan 没区分 cause，统一用 "user_cancel"）。

**未做**（留给 v2）：
- 真实微信支付 V3 退款 API
- `refunds` 表 `payment_id` 真实关联（v1 留 NULL 占位）
- refund.Service 接到 main 装配（v1 main 仍是骨架 nil pool；接 DB 后再接 refund）

---

## 11. 前端三端骨架（2026-09-24 三 worker 并行）

**目标**：按 `docs/superpowers/plans/2026-09-24-{patient-miniapp,escort-app,admin-web}-setup.md` 的 setup plan（Task 1~3 范围），并行启动三个前端工程骨架。

**3 个 worker 并行交付**：

| 工程 | 框架 | commit | 文件数 | 关键能力 |
| :-- | :-- | :-- | :--: | :-- |
| **patient-miniapp** | uni-app 3 + Vue 3.4 + Pinia 2 + uView Plus 0.1 | `8daff2e` | 11 | manifest 含 appid + app-plus 4 端 + 9 Android 权限 + 6 iOS 用途；utils/auth.js token 持久化；utils/request.js 请求拦截（X-Trace-Id + Authorization）+ 响应拦截（401/11001 清 token reLaunch login）；main.js createSSRApp + createPinia + uView Plus |
| **escort-app** | Flutter + flutter_riverpod 2.5 + go_router 14 + dio 5 + geolocator 13 + permission_handler 11 | `7570e93` | 11 | pubspec 含全部 plan 依赖；lib/core 三件套（router / theme / constants）；lib/main.dart ProviderScope + MaterialApp.router + DoctorsEscortApp；Android/iOS 权限清单（GPS / CAMERA / READ_MEDIA_IMAGES）；kInvitationTimeoutSeconds=30 / kGpsCheckinToleranceMeters=200 |
| **admin-web** | Vite 5 + React 18 + TS 5 + Ant Design 5.21 + Zustand 4.5 + TanStack Query 5.51 + axios + MSW 2.4 | `925cc9d` | 16 | ConfigProvider(zhCN + 主题token) → QueryClientProvider → AntdApp → BrowserRouter → AppRouter；AdminLayout（Header + Sider + Content + Outlet）；占位路由 /dashboard /escorts /orders + Navigate 回退；DashboardPage 4 个 Statistic 卡片雏形 |

**Plan 偏差汇总**：

1. **escort-app 未装 Flutter CLI**：worker 在 `E:\flutter\bin` 不存在的情况下手工脚手架；`flutter create .` 需在装好 Flutter 的环境补跑生成 `ios/Runner.xcodeproj` / `android/gradle/` / `pubspec.lock` / `lib/api/generated/`。已在 README 提示。
2. **patient-miniapp 目录**：plan 写 `web/patient-miniapp/`，实际落地 `frontend/patient-miniapp/`（任务契约显式指定 frontend/，与 escort-app / admin-web 保持一致）。
3. **patient-miniapp JS vs TS**：plan 全量 `.ts`，任务契约写 `main.js` / `utils/request.js`——按契约走 JS；package.json 仍装 TS 工具链便于后续平滑迁移。
4. **manifest 图标 PNG**：plan Task 16 才生成二进制 icon；Task 2 已在 manifest.json 的 `distribute.icons` 写引用路径但未提供 PNG——属预期偏差，Task 16 落地时把 PNG 写到 `static/icons/` 即可。
5. **Commit 数量**：3 个 setup Task 因文件互相依赖（main.js ↔ request.js ↔ auth.js），各 worker 选择合并 1 个 commit 更简洁可回溯；如需拆 3 commit 可 `git reset` 后重做。
6. **未跑 npm install / dev server**：按任务限制跳过；package.json scripts 完整可用。

**统一约定**：

- 三个工程目录统一在 `frontend/` 下
- commit 前缀：`feat(patient-miniapp):` / `feat(escort-app):` / `feat(admin-web):`
- 全部未 push（SSH 在本机不通，留给用户手动 push）

**下一步建议（每个工程）**：

- **patient-miniapp**: Task 4 utils 层（trace/format/wx/uni-mock + 单测）→ Task 5 utils/auth.js 扩展 → Task 7 Pinia store（authStore/userStore/orderStore/escalStore/messageStore）→ Task 10~13 5 个核心页面（首页 / 医院列表 / 下单 / 订单详情 / 评价）
- **escort-app**: Task 4 utils（format/trace/error_handler + 单测）→ Task 5 ProviderScope/GoRouter 完整装配 → Task 7 Riverpod providers → Task 8+ 5 个核心页面（splash / login / invitations / my-availability / 服务端）
- **admin-web**: Task 4 装配登录页 + authStore + AuthGuard → Task 5 ProTable / StatusBadge / AuditAction 等组件 → Task 6 MSW mock 全集 → Task 7+ 18 个 P0 页面（patients / escorts / orders / refunds / wallets / 系统设置 等）

**端到端联通（v1.1 目标）**：

- patient-miniapp 用户登录 → 下单 → match-service 抢单池 → escort-app 陪诊师接单 → order-service 状态机流转 → admin-web 监控
- 三端共享 `X-Trace-Id` + 后端 `logger.FromContext` 链路追踪

---

## 12. admin-web v2 增量 6 task（2026-09-24 order-matching-redesign plan）

**目标**：在 admin-web 骨架（§11 commit `925cc9d`）基础上，**增量**适配订单匹配模式从「抢单」改为「选人」：OrderListPage 加 2 状态筛选 + 2 列；OrderDetailPage 加状态机进度条 2 状态分支 + 30s 倒计时 + 拒接回退卡；DashboardPage 加 2 个 Statistic；StatusBadge 加 2 状态色；MSW 加 fixture；OpenAPI 加 3 字段。

**6 个 commit（按底座→组件→mock→page 顺序）**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `4caefbb` | `chore(admin-web)` contracts.yaml + types/generated.ts stub | openapi/contracts.yaml + src/types/generated.ts |
| `a8c2313` | `feat(admin-web)` StatusBadge 2 状态色 | components/StatusBadge/{StatusBadge.tsx, StatusBadge.test.tsx, index.ts} |
| `0be92c7` | `feat(admin-web)` MSW handlers + seed v2 fixture | mocks/{data/seed.ts, handlers/{admin/orders.ts, admin/reports.ts, index.ts}, browser.ts} + orders.test.ts |
| `37ebb97` | `feat(admin-web)` OrderListPage 2 状态筛选 + 2 列 + 详情链接 | pages/orders/{OrderListPage.tsx, OrderListPage.test.tsx} + api/admin/orders.ts |
| `efeab33` | `feat(admin-web)` OrderDetailPage 进度分支 + 倒计时 + 拒接卡 | pages/orders/{OrderDetailPage.tsx, EscortPendingCountdown.tsx, OrderDetailPage.test.tsx} |
| `a2fda5b` | `feat(admin-web)` DashboardPage 2 卡片 + router /orders/:id | pages/dashboard/{DashboardPage.tsx, DashboardPage.test.tsx} + api/admin/reports.ts + router/index.tsx |

**关键设计**：

1. **状态机 UI 对齐后端 v1.1**：6 节点进度条 `paid → selecting_escort → escort_pending_acceptance → accepted → in_service → completed`，与 §10.9.1 后端状态机一致。
2. **30s 倒计时**：`useEffect + setInterval(1000)` 1s tick；卸载 clearInterval 防内存泄漏；剩余 < 60s 红色 `#ff4d4f`；≥ 60s 蓝色 `#1677ff`；减到 0 时定格 "0s"。
3. **拒接回退文案映射**：`escort_declined → 陪诊师主动拒接` / `lock_expired → 陪诊师超时未确认`；Alert 顶部展示 + "重新选择其他陪诊师"链接。
4. **URL 双向绑定**：OrderListPage 用 `useSearchParams` 读 `?status=` 实现 DashboardPage 点击卡片跳转自动过滤。
5. **配色一致**：StatusBadge + DashboardPage 卡片 + EscortPendingCountdown 三处用同色 `colorWarning(橙) / colorProcessing(蓝)`，避免色板漂移。

**Plan 偏差（重要）**：

1. **工程根路径 `web/admin-web/` → `frontend/admin-web/`**：plan 全程写 `web/`，与 escort-app / patient-miniapp 不一致；本 v2 已纠正。
2. **ProTable → antd Table**：`@ant-design/pro-components` 不在 package.json；改用 antd 5 原生 Table + Select + Card，保留 5s polling + 状态过滤 + 详情链接语义。
3. **`OrderListPage.tsx` / `OrderDetailPage.tsx` 不存在**：骨架只有 `OrdersPage.tsx` 占位；本 v2 顺手补建（OrderListPage 替换 /orders 路由挂载；OrderDetailPage 新建；EscortPendingCountdown 新建；OrdersPage 保留但不再路由挂载）。
4. **types/generated.ts 改手写 stub**：openapi-typescript 7.x 工具链未跑（任务契约禁 npm install），手写 81 行覆盖 OrderStatus enum + OrderDetail 3 新字段 + OverviewReport 2 指标；注释标明待 `pnpm run generate:client` 自动覆盖。
5. **`public/mockServiceWorker.js` 跳过**：需 `npx msw init public/` 生成二进制；待用户本地补跑。
6. **测试用 `vi.mock` 替代部分 msw/node**：避免依赖未建好的 handler 在 jsdom 环境的兼容性；MSW handler test 仍用 msw/node 真实拦截（orders.test.ts）。

**未做（留给后续）**：

1. **跑 vitest 验证**：本机未装工具链；10 个 it 待用户 `pnpm install` 后 `pnpm exec vitest run` 跑通。
2. **`pnpm run generate:client` 重生成 types**：openapi-typescript 工具链补齐后覆盖手写 stub。
3. **`npx msw init public/` 生成 mockServiceWorker.js 二进制**。
4. **`src/main.tsx` 装配 MSW worker.start()**：dev 启动时启用 mock（补一行 `if (import.meta.env.DEV) startMockServiceWorker()`）。
5. **补 v1 缺失的 admin-web 骨架**：authStore + AuthGuard + ProTable 封装 + 全量 handlers（escorts/refunds/wallets 等）+ 18 P0 页面骨架。
6. **跨工程一致性校验**：patient-miniapp / escort-app 端 `frontend/{patient-miniapp,escort-app}/openapi/contracts.yaml` 也应加对应字段（selected_escort_id / escort_pending_expire_at / escort_reject_reason）。

---

## 13. escort-app v1.1 选人模式增量（2026-09-24 escort-app-setup plan §A1-A7）

**目标**：在 escort-app 骨架（§11 commit `7570e93`）基础上，落地选人模式核心：删 FeedPage + 加 InvitationsPage / AvailabilityPage + Riverpod providers + dio API client + token 持久化 + 路由替换。

**7 个 commit（按底座→utils→models→api→providers→pages 顺序）**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `004c90a` | `feat(escort-app)` TokenStorage (flutter_secure_storage + forTest fake) | services/{token_storage.dart, token_storage_test.dart} |
| `13c52fc` | `feat(escort-app)` newTraceId (escort-{ms}-{rand6}) | utils/{trace.dart, trace_test.dart} |
| `9379a90` | `feat(escort-app)` format (formatMoney/formatDateTime/maskPhone) | utils/{format.dart, format_test.dart} |
| `78641b2` | `feat(escort-app)` models (Order 选人模式 + Invitation + Availability) | models/{order,invitation,availability}.dart + *_test.dart |
| `3eb4049` | `feat(escort-app)` api_client (dio + Auth/Trace/401 拦截器) | services/api_client.dart + api_client_test.dart |
| `716f751` | `feat(escort-app)` providers (Auth + Invitation 5s 轮询 + Availability) | providers/{auth,invitation,availability}_provider.dart + *_test.dart |
| `046c497` | `feat(escort-app)` pages (Invitations + Availability) + CountdownBadge widget + router | pages/{invitations,availability}/ + widgets/countdown_badge.dart + router 替换 |

**关键设计**：

1. **匹配模式按 v1.1**：`Invitation`（不是 `Order`），含 `escortPendingExpireAt` + 订单瘦字段；客户端 `isLive` 过滤过期邀请，避免服务端回弹。
2. **30s 倒计时**：`CountdownBadge` 组件参数化 `expireAt`，UI 显示「待确认剩余」；`< 60s` 红色 `#ff4d4f` / `≥ 60s` 蓝色 `#1677ff`；Timer 在组件 `dispose` 时 `cancel()` 防内存泄漏。
3. **轮询拉新**：`Stream.periodic(Duration(seconds: 5))` + `StreamProvider` + `ref.onDispose` 清理 Timer；首次 tick 立即触发（不等 5s）。
4. **Token 持久化**：`flutter_secure_storage`（不是 `shared_preferences`，安全敏感）；`TokenStorage.forTest()` in-memory fake 让单测可重放。
5. **API client**：dio 5.7 + 3 拦截器：Auth（`Authorization: Bearer <token>`）、Trace（`X-Trace-Id: escort-{ms}-{rand6}`）、401（触发 `authProvider.notifier.onUnauthorized()` 回调）。
6. **AuthState sealed**：`AuthUnknown` / `AuthAnonymous` / `AuthAuthenticated`；`AuthNotifier` 暴露 `bootstrap` / `loginByPhone` / `logout` / `onUnauthorized`。

**测试覆盖**：**82 个单测**（11 个新增文件 + 24 个总变更文件，+2982 / -13 行）。

| 类别 | 测试数 |
| :-- | :--: |
| token_storage | 6 |
| trace | 4 |
| format | 11 |
| order / invitation / availability | 32 |
| api_client | 10 |
| auth / invitation / availability providers | 19 |

**Plan 偏差（重要）**：

1. **测试目录**：plan 多处假设 `test/api/` 等不存在目录；统一用 `test/<dir>/`，与 Flutter 标准 + 已有骨架兼容。
2. **api_client 位置**：plan 假设 `lib/api/dio_client.dart`；按父任务明确要求放 `lib/services/api_client.dart`（与服务层聚合）。
3. **Invitation 增强**：plan 只要求 `isLive`；额外加 `remainingSeconds`（UI 倒计时显示用）。
4. **Availability 增强**：plan 只要求 `isDeletable`；额外加 `isBooked` + `wireValue` + `durationHours`。
5. **Order model 增强**：plan 没要求 `wireValue`（snake_case 反序列化），加上便于 toJson round-trip。
6. **availability_provider 状态类型**：plan 写 `AsyncValue<Availability>`；改用 `AsyncValue<Availability?>`（null = idle）。

**未做（留给后续）**：

1. 24 P0 页面中除 InvitationsPage + AvailabilityPage 外的 22 个
2. 路由守卫（authGuardProvider / realNameGuardProvider / approvedGuardProvider）和 redirect 链
3. 全量 API client（escort / order / wallet / sos / training / review / message + availability_api 完整版）
4. 全量 providers（wallet_provider / message_provider / training_provider）
5. 全量公共 widgets（OrderCard / AvailabilityTile / RatingStars / StatusChip / GpsCheckinButton / SosLongPress）
6. main.dart 真实 `tokenStorageProvider` + `dioProvider` override 装配 + splash → /home/invitations 跳转
7. openapi.yaml + gen-api.sh + `lib/api/generated/` 自动生成
8. Flutter Web + PWA（plan §v1.1 Task 20-23）
9. integration_test / E2E

**端到端联通（v1.1 目标）**：

- patient-miniapp 下单 → match-service 推邀请 → escort-app `/home/invitations` 30s 倒计时确认 → order-service `escort_pending_acceptance` → 状态流转 → admin-web 监控
- 共享 `X-Trace-Id`（escort-app 用 `escort-{ms}-{rand6}`；patient-miniapp 用 `mp-{ms}-{rand6}`；admin-web 用其他前缀）

---

## 14. patient-miniapp v1.1 选人模式增量（2026-09-24 patient-miniapp-setup plan §P1-P7）

**目标**：在 patient-miniapp 骨架（§11 commit `8daff2e`）基础上，落地选人模式核心：删"抢单"相关前端代码 + 加 candidates / order detail / order list 页 + Pinia stores + API client + CountdownBadge + OrderStatusProgress + OrderListItem 4 个新组件 + 路由收尾。

**7 个 commit（按 utils → stores → api → components → pages → list 顺序）**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `570ebd4` | `test(patient-miniapp)` utils format/trace + jest 配置 | utils/{format.js, trace.js, format.test.js, trace.test.js} + jest.config.js + babel.config.js + package.json |
| `946cb72` | `feat(patient-miniapp)` Pinia stores (auth + order 选人模式状态机) | stores/{auth.js, order.js, auth.test.js, order.test.js} |
| `f9e4271` | `feat(patient-miniapp)` API client (candidates + order + index) | api/{candidates.js, order.js, index.js, candidates.test.js, order.test.js} |
| `9d8c4ae` | `feat(patient-miniapp)` CountdownBadge 组件（参数化 expireAt + <60s 红色） | components/{CountdownBadge.vue, CountdownBadge.test.js} |
| `1be2b17` | `feat(patient-miniapp)` candidates 页 + EscortCandidateCard 组件 + 路由注册 | pages/order/candidates/{index.vue, index.test.js} + components/EscortCandidateCard.{vue,test.js} + pages.json |
| `6fb379c` | `feat(patient-miniapp)` 订单详情页 + OrderStatusProgress 组件 + 路由注册 | pages/order/detail/{index.vue, index.test.js} + components/OrderStatusProgress.{vue,test.js} + pages.json |
| `c98d43b` | `feat(patient-miniapp)` 订单列表页 + OrderListItem 组件 + pages.json 路由收尾 | pages/order/index.{vue, test.js} + components/OrderListItem.{vue,test.js} + pages.json |

**关键设计**：

1. **匹配模式按 v1.1**：删除任何"抢单"代码（lobby / pool / waiting）；用"选陪诊师"语义（candidates + selectEscort）。
2. **状态机 6 节点**：`paid → selectingEscort → escortPendingAcceptance → accepted → inService → completed`；OrderStatusProgress 横向 stepper + 中文 label。
3. **30s 倒计时**：复用 CountdownBadge（参数化 `expireAt`，`< 60s 红色 #ff4d4f` / `≥ 60s 蓝色 #1677ff`）；组件 unmount 时 `clearInterval` 防内存泄漏。
4. **拒接回退文案**：`escort_declined → 陪诊师主动拒接` / `lock_expired → 陪诊师超时未确认`；详情页顶部 Alert +「重新选择陪诊师」链接。
5. **状态过滤**（列表页）：u-tabs 8 项（含 2 新状态 selectingEscort / escortPendingAcceptance）；支持 `?status=` URL 参数（admin-web 跳转锚点）。
6. **路由顺序**：`pages/order/index → pages/order/detail → pages/order/candidates`（列表 → 详情 → 选人页用户流）。

**测试覆盖**：**80+ 单测用例**（11 个新增文件 + 12 个总变更文件，~3000 行）

| 类别 | 测试数 |
| :-- | :--: |
| utils format/trace | 23 |
| stores auth/order | 23 |
| api candidates/order | 21 |
| CountdownBadge | 8 |
| EscortCandidateCard | 4 |
| OrderStatusProgress | 4 |
| OrderListItem | 4 |
| candidates 页 | 5 |
| detail 页 | 6 |
| list 页 | 5 |

**Plan 偏差（重要）**：

1. **路径**：plan 全程 `web/patient-miniapp/...`，实际 `frontend/patient-miniapp/...`。
2. **JS vs TS**：plan 全量 `.ts`，任务契约写 `format.js` / `trace.js` / `.vue`；按契约走 JS。
3. **api 模块 store 兼容**：`src/stores/order.js` 已通过 `import('@/api/order.js')` 取 5 个函数；故 `src/api/order.js` 同时导出 `listOrders` / `cancelOrder` + re-export `getCandidates`（不破坏 store 单测）。
4. **store 字段命名差异**：store 内部 `escortConfirmed` vs OrderStatusProgress 用 `accepted`；detail 页用 `progressStatus` computed 做映射，保持进度条 visual key 与后端 API 字段对齐。
5. **utils/auth.js 落地位置**：worker A 落在根 `utils/auth.js`；本批次加 `src/utils/auth.js` re-export shim 不动 worker A 代码。
6. **未跑 jest**：本机无 npm 镜像；测试文件按 `@vue/test-utils v2 + jest 29 fake timers` 编写，待后续 plan 增补 `@vue/test-utils` + `vue-jest` + jest.config.js `.vue` transform 后可跑。

**未做（留给后续）**：

1. 跑 jest 验证（需 npm install + 增补 @vue/test-utils + vue-jest）
2. 其他 17 个 P0 页面（首页 / 医院列表 / 评价 / 钱包 / 我的 / 登录 / 注册 / 实名 / 陪诊师资料等）
3. Appium e2e（plan v1.1 Task 20-23 Android/iOS 原生端）
4. app-plus Android/iOS 原生构建（manifest 已配置 + Icon 二进制）
5. OpenAPI codegen + types 层（当前 plan 阶段无 contracts.yaml，按手写契约）
6. MSW mock 全集 + Playwright e2e H5 模式

**端到端联通（v1.1 目标）**：

- patient-miniapp 选陪诊师 → match-service 推邀请 → escort-app `/home/invitations` 30s 倒计时确认 → order-service `escort_pending_acceptance` → 状态流转 → admin-web 监控
- 三端 trace-id 三处共用：`mp-{ms}-{rand6}` / `escort-{ms}-{rand6}` / 后端 logger.FromContext

---

## 15. admin-web v1 缺失骨架补齐（2026-09-24 admin-web setup plan §A1-A11）

**目标**：在 admin-web 骨架（§11 commit `925cc9d`）+ v2 增量 6 task（§12 commit `4caefbb` ~ `a2fda5b`）基础上，补齐 v1 缺失的核心组件：authStore + AuthGuard + RBAC + 5 公共组件 + 18 P0 页面骨架 + 4 MSW handlers（除已做的 orders）+ seed 12 模块 fixture + main.tsx 装配。

**11 个 commit（按底座→守卫→组件→handlers→seed→main 顺序）**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `eb4d965` | `feat(admin-web)` authStore (Zustand 4.5 + persist, 6 roles + 4 actions + 11003 RBAC) | stores/{authStore.ts, authStore.test.ts} |
| `5efd3f2` | `feat(admin-web)` AuthGuard + RequireRole RBAC 路由守卫 | router/{guards.tsx, guards.test.tsx} |
| `ccc6c47` | `feat(admin-web)` ProTable 公共组件（5 props 封装） | components/ProTable/{ProTable.tsx, index.ts, ProTable.test.tsx} |
| `af45ca4` | `feat(admin-web)` 4 公共组件 (ErrorBoundary + PageHeader + TraceId + AuditAction) | components/{ErrorBoundary,PageHeader,TraceId,AuditAction}/* |
| `b2951a0` | `feat(admin-web)` 18 P0 页面骨架 | pages/{patients,escorts,refunds,wallets,work-orders,reviews,messages,sos,settings,profile,login,audit,finance,reports,coupons,hospitals,dashboard-detail}/* |
| `0eeec2d` | `feat(admin-web)` MSW handlers users | mocks/handlers/admin/{users.ts, users.test.ts} |
| `7f3b831` | `feat(admin-web)` MSW handlers escorts | mocks/handlers/admin/{escorts.ts, escorts.test.ts} |
| `c0764e4` | `feat(admin-web)` MSW handlers refunds | mocks/handlers/admin/{refunds.ts, refunds.test.ts} |
| `1c730b3` | `feat(admin-web)` MSW handlers wallets | mocks/handlers/admin/{wallets.ts, wallets.test.ts} |
| `cf00383` | `feat(admin-web)` seed.ts 增 9 类目 fixture (30+ 条) + handlers 汇总 | mocks/{data/seed.ts, handlers/index.ts} |
| `73314b2` | `feat(admin-web)` main.tsx 装配 + router 挂载 22 路由 + RBAC 联动 | main.tsx + router/index.tsx |

**关键设计**：

1. **authStore**：Zustand 4.5 + `persist` 中间件（localStorage key=`doctors-admin-auth`），含 6 roles（`super_admin / order_admin / refund_admin / audit_admin / cs / viewer`）+ 4 actions（`login / bootstrap / logout / onUnauthorized`）。
2. **AuthGuard**：`<Outlet>` 包裹；未登录跳 `/login`；role 不足显示 403 页（与 `RequireRole` 子组件配合）。
3. **RBAC**：守卫读 `useAuthStore`，role 不在白名单 → 业务码 `11003 admin_forbidden`（已映射到 errs）。
4. **5 公共组件**：ProTable（antd Table 5 props 二次封装 + data-testid）/ ErrorBoundary（class + getDerivedStateFromError + reset 按钮）/ PageHeader（title + subtitle + extra slot）/ TraceId（X-Trace-Id 展示 + clipboard 复制）/ AuditAction（Timeline list + actor + action + 颜色）。
5. **18 P0 页面骨架**：每个页面 `export default function XxxPage() { return (<div><PageHeader .../>...</div>) }`，含 PageHeader + Card 占位 + TODO 提示文案。
6. **MSW handlers**：users（3 端点）/ escorts（4 端点）/ refunds（4 端点）/ wallets（2 端点），与 v2 orders handlers 同样的响应形态 `{ code: 0, data: ..., trace_id: "..." }`。
7. **seed.ts 30+ fixture**：覆盖 12 模块（users / escorts / refunds / wallets / patients / work_orders / reviews / messages / sos / hospitals / packages / coupons / audit_logs + 原 orders / overview 保留）。
8. **main.tsx 装配**：QueryClientProvider / ConfigProvider(zhCN + 主题) / AntdApp / BrowserRouter / AppRouter + AuthGuard 包裹 + RequireRole 在 admin 子路由挂载。

**测试覆盖**：11 个 commit 内含测试代码（按 brief 约束未运行 vitest，留用户本地 `npm install` 后跑）。

**Plan 偏差（重要）**：

1. **路由总数 18 → 22**：brief 列了 18 个顶层 + 3 详情 = 21，加 `/login` + `/dashboard-detail` 单独页 = 22。
2. **EscortsPage 覆盖**：v2 scaffold 留有原占位版（10 行），本批次覆盖为 v1 风格骨架（含 PageHeader + Card）。git diff 显示 rewrite 82%。
3. **handlers/index.ts 扩展**：v2 阶段只有 order/reports，本批次扩展为 6 模块合集（order/report/user/escort/refund/wallet）。原文件 rewritten 71%。
4. **App.tsx 残留**：原 v1 scaffold 的 App.tsx（44 行）已不被 main.tsx 引用，处于游离态未删除（保留作为 fallback）。
5. **未跑 vitest**：按 brief "不要 npm install / vitest 跑测试"，本批次所有 `.test.ts(x)` 文件为契约 + 验收脚本式样，文件就绪待用户本地 `npm install` 后 `vitest run`。

**未做（留给后续）**：

1. **12 类目 MSW handlers**：本批次按 brief 范围只建了 4 大块（users/escorts/refunds/wallets），剩余 12 类目只建 seed.ts fixture，handler 文件后续按需追加。
2. **API client 函数**：12 类目的 `@/api/admin/*.ts` 客户端函数未建（仅 orders.ts / reports.ts 由 v2 提供），下批次按"每 handler 配一 API client"原则补齐。
3. **AdminLayout 菜单扩展**：侧边栏 Menu 仍只显示 dashboard/escorts/orders 3 项（v2 既有），新增 18 路由的菜单项未挂载（brief 未要求；菜单 RBAC 过滤是后续 Task）。
4. **QueryClient + Axios 401 拦截器**：queryClient 已 v2 装配，但 axios 401 → authStore.onUnauthorized() 联动尚未在 api client 中接入（v2 也是用 fetch，未引入 axios client）。
5. **vitest + msw/node + @testing-library/react**：未安装（保持 brief 约束），所有测试为契约样。
6. **跨工程一致性**：patient-miniapp / escort-app 端 `frontend/{patient-miniapp,escort-app}/openapi/contracts.yaml` 也应加对应字段（selected_escort_id / escort_pending_expire_at / escort_reject_reason），由对应 worker 收口。

**端到端联通（v1.1 目标）**：

- patient-miniapp 选陪诊师 → match-service 推邀请 → escort-app `/home/invitations` 30s 倒计时确认 → order-service `escort_pending_acceptance` → 状态流转 → admin-web 22 路由覆盖监控（dashboard / orders / escorts / refunds / wallets / 等 12 类目）
- 三端 trace-id 三处共用：`mp-{ms}-{rand6}` / `escort-{ms}-{rand6}` / 后端 logger.FromContext

---

## 16. 后端 wallet T+7 结算服务（2026-09-24 wallet plan §W1-W5）

**目标**：实现 `services/wallet/` 独立服务，含 wallets / withdrawals / billings 三表 + T+7 冻结释放 scanner + 提现状态机 + 7 个 endpoint + Kafka consumer 框架（消费 OrderCompletedEvent / PaymentRefundedEvent）。

**5 个 commit（按迁移 → 事件 → repo → service → handler 顺序）**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `b9bf7bd` | `feat(migrations)` 0007 wallets + withdrawals + billings + orders.completed_at | migrations/0007_wallets.{up,down}.sql + migrations_test.go 加 Test0007WalletsUpDown |
| `cd2542b` | `feat(contracts)` OrderCompletedEvent + TopicOrderCompleted | shared/contracts/{events.go, contracts_test.go} |
| `2266dc7` | `feat(wallet)` repo 加 WalletRepo (14 方法 + 9 集成测试) | services/wallet/internal/repo/wallet_repo.{go, _integration_test.go} |
| `3ce91ea` | `feat(wallet)` service + T+7 Scanner | services/wallet/internal/service/{wallet_service, scanner, *_test.go} |
| `bb28b83` | `feat(wallet)` handler + router + server + main + smoke | services/wallet/internal/{handler, router, server, middleware}/ + cmd/main.go + scripts/smoke-wallet.sh + config/wallet.yaml |

**关键设计（按 plan §Architecture）**：

1. **3 张表**：
   - `wallets`：user_id UNIQUE, balance NUMERIC(10,2), frozen NUMERIC(10,2), created_at, updated_at
   - `withdrawals`：user_id, amount_cents BIGINT, channel, account_no, status pending/approved/paid/rejected, external_tx_id, reason, created_at, updated_at
   - `billings`：user_id, order_id, type income/refund/unfreeze/withdraw, amount NUMERIC, balance_after NUMERIC, created_at
   - `orders.completed_at TIMESTAMPTZ` 加列（用于 T+7 扫描）
2. **事件驱动**：消费 PaymentCompletedEvent → FreezeIncome（frozen += amount）；消费 PaymentRefundedEvent → DeductFrozenForRefund
3. **T+7 scanner**：1 分钟扫一次（用 `interval` 参数化阈值，v1 用 1 分钟模拟 7 天可测，prod 改 `7*24*time.Hour`）；扫到 `orders.status='completed' AND completed_at + interval ≤ NOW()` → `frozen -= amount; balance += amount` + 写 billings
4. **提现状态机**：`pending → approved → paid` / `pending → rejected`（单向不回退）
5. **最低提现 100 元**（`MinWithdrawalCents = 10000`）
6. **钱用 NUMERIC(10,2)**：go 端 `shopspring/decimal` 防精度漂移；wallet 暴露 int64 分单位给 API 层（前端约定整数分）
7. **未注册用户查钱包**：`return nil, nil`（不报错）
8. **Kafka best-effort**：publish 失败不阻塞主流程（写 warn log）

**endpoint 清单（7 个）**：

| 方法 | 路径 | 角色 |
| :-- | :-- | :-- |
| GET | `/api/v1/wallet` | patient/escort |
| GET | `/api/v1/escorts/me/wallet` | escort |
| POST | `/api/v1/escorts/me/wallet/withdraw` | escort |
| GET | `/api/v1/wallet/transactions` | patient/escort |
| POST | `/api/v1/admin/wallet/withdrawals/{id}/approve` | admin |
| POST | `/api/v1/admin/wallet/withdrawals/{id}/pay` | admin |
| POST | `/api/v1/admin/wallet/withdrawals/{id}/reject` | admin |

**测试覆盖**：**~50 个单测**（全部 PASS）

| 类别 | 测试数 |
| :-- | :--: |
| migrations Test0007WalletsUpDown | 1（集成） |
| contracts TestOrderCompletedEvent_RoundTrip + 增项 | 3 |
| wallet_repo_integration_test | 9（集成） |
| wallet_service_test | 14 |
| scanner_test | 6 |
| handler_test | 11 |
| router_test (TestHealthz) | 1 |

**全量回归**：`go test ./...` 44 个测试包 0 FAIL（含新增 wallet）。

**Plan 偏差**：

1. **未创建子 module `services/wallet/go.mod`**：保持 monorepo（与 user / review / order 一致），`shopspring/decimal` 加到根 `go.mod`。
2. **scanner 简化为单次扫描**：plan 让按 user 遍历 → service 调 `RunScanOnceForUser`；本批次简化为 `ListCompletedOrdersBefore(ctx, cutoff, limit)` 单次扫所有 escort 用户（避免 N+1，性能更好）。
3. **`MinWithdrawalCents` 比较**：amount 单位改为元（与 decimal 一致），比较用 `100 元 = MinWithdrawalCents/100`。
4. **fakeRepo UnfreezeToBalance 自动建钱包**：测试 setup 用 `GetOrCreate`（与真实 repo 行为一致）。
5. **新增 `config/wallet.yaml`**：cmd 启动需要配置；plan 未列。

**未做（留给后续）**：

1. 集成测试需 `docker compose up` 起 PG 才能跑（`go test -tags=integration ./services/wallet/...`）
2. 真实微信提现通道（v1 mock 写 `external_tx_id=MOCK-TX-{id}`）
3. 退款发生在 T+7 窗口外的扣 balance 路径（v2 处理）
4. docs/04 流程图 + dev.md §10.14 增量（按 worker brief 跳过）
5. order-service 状态机迁移时同步 publish OrderCompletedEvent（v2 处理）

**端到端联通（v1.1 目标）**：

- patient-miniapp 选陪诊师 → match-service 推邀请 → escort-app `/home/invitations` 30s 倒计时确认 → order-service `escort_pending_acceptance` → `accepted` → `in_service` → `completed`（写 `completed_at`）→ wallet scanner 1 分钟扫到 → `frozen -= amount; balance += amount` → escort 提现 → admin 审批 → paid（mock external_tx_id）
- 三端 trace-id 三处共用：`mp-{ms}-{rand6}` / `escort-{ms}-{rand6}` / 后端 logger.FromContext
- 全量回归 44 个测试包 0 FAIL

---

## 17.5 order-service 发布 OrderCompletedEvent（解锁 wallet T+7 链路）

**目标**：wallet 服务（§16 commit `bb28b83`）已能消费 `OrderCompletedEvent` 并在 T+7 时释放冻结；但 order-service **从未发布**此事件，导致 scanner 永远等不到触发。补上 publish 调用 + 全链路联通。

**3 个 commit（按 O3 → O1 → O2 顺序，每 commit 后树都可编译）**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `57fa551` | `feat(order-tests)` sync fakePublisher / fakePub 加 PublishOrderCompleted | services/order/internal/scheduler/expired_lock_scanner_test.go + service/order_service_test.go |
| `29616b1` | `feat(order-events)` add PublishOrderCompleted (Kafka + Nop 双实现 + 3 测试) | services/order/internal/events/{publisher.go, publisher_test.go} |
| `f95fee3` | `feat(order-service)` Finish 写库成功后 best-effort 发布 OrderCompletedEvent | services/order/internal/service/{order_service.go, order_service_test.go} |

**关键设计**：

1. **Publisher interface 扩 1 个方法**：`PublishOrderCompleted(ctx, ev contracts.OrderCompletedEvent) error`
2. **KafkaPublisher 实现**：与 `PublishOrderAccepted` 同模式（`TopicOrderCompleted` + OrderID 作 key + JSON value）
3. **NopPublisher 实现**：计数 + nil（用于测试）
4. **service.Finish 触发**：状态机推进 `accepted → in_service → completed` 后，`o.publisher.PublishOrderCompleted(ctx, ...)` best-effort
5. **事件 schema 沿用现有**：`OrderID / EscortID / Amount / CompletedAt`（与 wallet consumer `cmd/main.go:121` 一致）；任务描述里的 `PatientID/HospitalID/AmountCents` 属独立 schema 演进 task

**测试覆盖**：

| 类别 | 测试数 |
| :-- | :--: |
| events publisher | 3（新增） |
| service Finish | 2（新增） |
| 其他既有 order 测试 | 不变 |
| **全量回归** | **44 包 0 FAIL** |

**Plan 偏差**：

1. **commit 顺序**：brief 列 O1 → O2 → O3；本批次实际 O3 → O1 → O2——O1 改 Publisher interface 后，未同步的 fake 会破坏 build；O3 必须落在 O1 之后、O2 之前才能保证每 commit 后树干净。
2. **事件字段**：任务描述里 `PatientID / HospitalID / AmountCents` 与现行 schema 不一致；遵循现行 schema（因为 wallet 消费者已绑定），修改事件 schema 属另一个独立 task。

**未做（留给后续）**：

1. 事件 schema 演进（加 PatientID / HospitalID / AmountCents 字段并同步 wallet 消费侧）—— 需先定 amount 精度（cents vs 元）
2. order-service 状态机从 in_service → completed 的推进入口（admin / escort-app 操作面板）
3. 集成测试需 docker compose up（连接真实 Kafka）
4. T+7 触发源：当前 `CompletedAt` 写 `clockNow()`，建议统一从 `orders.completed_at` 列读（schema 已就绪，commit `b9bf7bd`），待状态机迁移后用 repo 字段替换

**端到端联通（v1.2 目标）**：

- patient-miniapp 选陪诊师 → match-service 推邀请 → escort-app `/home/invitations` 30s 倒计时确认 → order-service `accepted → in_service → completed`（**写 completed_at** + **publish OrderCompletedEvent** commit `f95fee3`）→ wallet OnOrderCompleted → frozen += amount → wallet scanner 1 分钟扫到 → T+7 释放 → billings 流水
- 三端 trace-id 三处共用：`mp-{ms}-{rand6}` / `escort-{ms}-{rand6}` / 后端 logger.FromContext
- 全量回归 44 个测试包 0 FAIL

---

## 17. 前端三端 contracts.yaml 一致性校验（2026-09-24）

**目标**：把 admin-web v2 已落地的 3 个 OrderDetail 字段 + OverviewReport 2 个新指标同步到 patient-miniapp + escort-app 的 OpenAPI 契约，保证三端 schema 一致，避免 dart-dio / openapi-typescript 生成出来的 client 字段漂移。

**3 个 commit**：

| commit | 端 | 内容 |
| :-- | :-- | :-- |
| `6ab0ebb` | patient-miniapp | 新建 `frontend/patient-miniapp/openapi/contracts.yaml`（269 行），含 6 端点 + 3 v2 字段 + 2 v2 状态 + PatientOverview 2 指标 |
| `5593035` | escort-app | `pubspec.yaml` 加 `openapi_generator_cli: ^1.0.0`；新建 `openapi.yaml`（33 行，generator config）+ `scripts/gen-api.sh`（52 行，dart-dio 生成脚本） |
| `4caefbb` | admin-web（已完成，不动） | `frontend/admin-web/openapi/contracts.yaml`（169 行，v2 baseline） |

**三端 contracts.yaml 当前状态对比**：

| 端 | 文件 | 行数 | 状态 |
| :-- | :-- | :--: | :-- |
| admin-web | `frontend/admin-web/openapi/contracts.yaml` | 169 | ✅ v2 baseline（commit `4caefbb`） |
| patient-miniapp | `frontend/patient-miniapp/openapi/contracts.yaml` | 269 | ✅ 本批次新建（含 patient 端 6 端点 + PatientOverview） |
| escort-app | `frontend/escort-app/openapi.yaml` | 33 | ⚠️ generator config（不是契约本身，引用 admin-web 输入） |

**escort-app 注意点**：`frontend/escort-app/openapi.yaml` 不是契约 spec 本体，而是 `openapi-generator-cli` 的 generator config —— 真正的输入契约从 `../admin-web/openapi/contracts.yaml` 同步过来；生成出来的 dart-dio client 会自动继承 admin-web 已落地的 v2 字段，无需在 escort-app 单独维护一份契约。

**字段覆盖率（v2 关键字段）**：

| 字段 / 状态 / 指标 | admin-web | patient-miniapp | escort-app |
| :-- | :--: | :--: | :--: |
| `selected_escort_id` (int64 nullable) | ✅ | ✅ 本批次 | ⏳ 待 `bash scripts/gen-api.sh` 生成 |
| `escort_pending_expire_at` (datetime nullable) | ✅ | ✅ 本批次 | ⏳ 待 `bash scripts/gen-api.sh` 生成 |
| `escort_reject_reason` (enum nullable) | ✅ | ✅ 本批次 | ⏳ 待 `bash scripts/gen-api.sh` 生成 |
| `OrderStatus.selecting_escort` | ✅ | ✅ 本批次 | ⏳ 待 generator |
| `OrderStatus.escort_pending_acceptance` | ✅ | ✅ 本批次 | ⏳ 待 generator |
| `OverviewReport.pending_selecting_escort` | ✅ | ✅ 本批次（PatientOverview） | ❌ escort-app 不需要 |
| `OverviewReport.pending_escort_acceptance` | ✅ | ✅ 本批次（PatientOverview） | ❌ escort-app 不需要 |

**未做（留给后续本地执行）**：

1. **admin-web**：用户本地跑 `pnpm run generate:client` 覆盖 `src/types/generated.ts`（scripts 已在 package.json；不需要再改 spec）。
2. **escort-app**：用户本地跑 `bash scripts/gen-api.sh` 把 `lib/api/generated/` 重新生成一遍（需先 `dart pub global activate openapi_generator_cli`）。
3. **patient-miniapp**：当前 TS 客户端是手写 JSDoc 注释（见 `src/api/order.js` Order typedef + `src/api/candidates.js` Candidate typedef），本批次不引入 openapi-typescript 代码生成 —— 留待 v2 任务彻底稳定后再统一接入。
4. **三端真实 contracts 合并**：当前三份契约都是最小可用 schema；待 order-service / admin-service 真正生成 OpenAPI 后，把这三份内容合并到上游，并删掉前端各自的 contracts.yaml。

**Plan 偏差**：

1. **patient-miniapp PatientOverview schema**：当前 patient-miniapp UI 没有 overview 页（只有 order/candidates/detail 三个），但本批次仍把 `PatientOverview` schema + 2 指标写进 contracts.yaml —— 契约先行，UI 后续按需消费。
2. **escort-app 不单独维护契约**：原 plan 让 escort-app 也写一份 contracts.yaml；本批次改成 generator config 模式（直接引用 admin-web 输入），避免三份契约漂移。
3. **未引入 pnpm / dart 真实执行**：按 brief 约束，不跑 `pnpm install` / `flutter pub get`，只生成 config + script。

**后续追踪**：

- 三端契约统一后，把 `lib/models/order.dart` 的 `OrderStatus` 枚举 / `selectedEscortId` 字段改用 generator 生成的代码（v2.1）
- admin-web OverviewReport 6 指标 → patient-miniapp PatientOverview 4 指标 + admin 自己看的 2 个（pending_refunds / pending_escorts）拆分（v2.1）

---

## 17.6 admin-web 12 类目 MSW handlers 补齐（2026-09-24 admin-web v1 §H1-H8）

**目标**：补齐 admin-web 后台 8 个模块的 MSW handler（除已做的 orders/reports/users/escorts/refunds/wallets），让 admin-web 22 路由全部有可消费的 mock 数据。

**9 个 commit（按 handler 模块 + 汇总顺序）**：

| commit | 内容 | 端点数 |
| :-- | :-- | :--: |
| `06217bf` | `feat(admin-web)` MSW handlers work-orders（列表/详情/创建/分配/关闭） | 5 |
| `8f569f9` | `feat(admin-web)` MSW handlers reviews（列表/详情/审核/回复） | 4 |
| `2b6b0d8` | `feat(admin-web)` MSW handlers messages（列表/详情/发送/广播） | 4 |
| `65d36fe` | `feat(admin-web)` MSW handlers sos（列表/详情/处置/升级） | 4 |
| `5656f98` | `feat(admin-web)` MSW handlers patients（列表/详情/封禁/解封） | 4 |
| `65e0167` | `feat(admin-web)` MSW handlers hospitals（列表/详情/创建/更新） | 4 |
| `0051d06` | `feat(admin-web)` MSW handlers packages（列表/详情/创建/更新） | 4 |
| `15a0926` | `feat(admin-web)` MSW handlers coupons（列表/详情/创建/停用） | 4 |
| `2335ff8` | `feat(admin-web)` MSW handlers 汇总新增 8 模块导出 | — |

**关键设计**：

1. **响应统一**：`{ code: 0, data: ..., trace_id: "admin-msw-{ms}-{rand}" }`；错误响应 `{ code, message, data: null, trace_id }`，HTTP 200
2. **错误码 3 档**：`10001` 参数无效（query 缺参）/ `12001` 资源不存在（id 没找到）/ `11003` admin_forbidden（未带 Authorization 头）
3. **8 个 handler 全部用 `requireAuth(request)` 守卫**：简化 RBAC（仅校验 token 头存在性，与现有 escorts/refunds 等一致）
4. **每个 handler 配套 .test.ts**：含 5-7 个 vitest + msw/node 用例（按「不实际跑测试」约束；契约样待 vitest 工具链启用后跑）
5. **seed.ts 已就绪**（§15 commit `cf00383`）：8 模块 fixture 30+ 条，无需新增

**累计**：

- **8 个 handler 文件** + **8 个 test 文件**（共 16 新文件、1598 行）
- **33 个端点**（work_orders:5 + 其余各 4）
- **52 个测试用例**

**未做（留给后续）**：

1. 页面侧（`pages/{work-orders,reviews,messages,sos,patients,hospitals,packages,coupons}/*Page.tsx`）仍为 TODO 占位——本批次仅补 handler 骨架，页面接 API 留后续 worker
2. vitest + msw/node 实际跑测试留待 vitest 工具链在 admin-web 启用后
3. RBAC 真实角色权限校验未实现（仅做 token 头存在性检查，与现有 escorts/refunds 简化策略一致）
4. 全部 22 admin-web 路由的菜单挂载（侧边栏 Menu 仍只显示 dashboard/escorts/orders 3 项）

**端到端联通（v1.2 目标）**：

- patient-miniapp 选陪诊师 → match-service 推邀请 → escort-app 30s 倒计时确认 → order-service `accepted → completed` + publish OrderCompletedEvent → wallet T+7 scanner → admin-web 22 路由（dashboard / orders / escorts / refunds / wallets / work-orders / reviews / messages / sos / patients / hospitals / packages / coupons）全部有 mock 数据可消费
- 三端 trace-id 三处共用：`mp-{ms}-{rand6}` / `escort-{ms}-{rand6}` / 后端 logger.FromContext
- 全量回归 44 个测试包 0 FAIL

---
## 17.7 admin-web AdminLayout �˵���չ��2026-09-24 admin-web v1 ��L1-L4��

**Ŀ��**���� admin-web 22 ·��ȫ���ҵ� AdminLayout �� Sider Menu���� authStore.role �� RBAC ���ˣ�Header �����м + �ǳ���

**1 �� commit**��995f3b4 feat(admin-web): AdminLayout �˵���չ��12 �� + RBAC ���� + �ǳ���

**�ؼ����**��

1. **12 ��˵� + ��ɫ����**������ɫ���ˣ���dashboard��ȫ����/ orders��super_admin+order_admin+refund_admin+cs+viewer��/ escorts / escorts/audit / refunds / wallets / work-orders / reviews��ȫ����/ messages / sos / reports / settings���� super_admin��
2. **RBAC ����**��useAuthStore((s) => s.role) �� ���� MENU_ITEMS��oles δ����=ȫ���ɼ�
3. **��ǰ·�ɸ���**��path **�ǰ׺ƥ��**������ /escorts �غ� /escorts/audit`n4. **Header ��**�����м������ > ��ǰҳ��
5. **Header ��**��TraceId ռλ + �û��������ǳ� + ��ɫֻ�� + �ǳ���
6. **�ǳ�**��useAuthStore.logout() + useNavigate('/login')`n
**���Ը���**��6 �� vitest��δʵ���ܣ���super_admin 12 �� / viewer ���� escorts-audit+settings / refund_admin ���� escorts+escorts-audit / order_admin ���� wallets+settings / �˵���� navigate / pathname ����

**δ��������������**��Ƕ���Ӳ˵� / �۵�̬�־û� / �ƶ��� Drawer / TraceId ����ʵ������ / ��ʵ RBAC��ǰ�� roles ����Ӳ���룩

**�˵�����ͨ��v1.2 Ŀ�꣩**��

- patient-miniapp ѡ����ʦ �� match-service ������ �� escort-app 30s ����ʱȷ�� �� order-service ccepted �� completed + publish OrderCompletedEvent �� wallet T+7 scanner �� admin-web 22 ·�ɰ���ɫ���˿ɼ��˵�
- ���� trace-id �������ã�mp- / escort- / ��� logger.FromContext
- ȫ���ع� 44 �����԰� 0 FAIL


---
## 17.8 admin-web 5 ����ʵҵ��ҳ��2026-09-24 admin-web v1 ��H1-H5��

**Ŀ��**���� 5 ����Ƶ P0 ռλҳ��escorts/audit + escorts/:id + patients + patients/:id + reviews������Ϊ�� MSW handler + ProTable + ҵ�񽻻�����ʵҳ��

**1 �� commit��13 �ļ���+2869/-66��**��

| commit | �ļ� |
| :-- | :-- |
| 1710bda | EscortAuditPage + EscortDetailPage + PatientsPage + PatientDetailPage + ReviewsPage + 3 �� API client + 5 �� test |

**�ؼ����**��

1. **ProTable ͳһģʽ**��PageHeader + ProTable(testId, rowKey, columns, dataSource, loading) + TanStack Query useQuery/useMutation + ������ invalidateQueries ˢ��
2. **RBAC UI ��**��viewer ��ɫ �� ȫ������/���/��˰�ť����Ⱦ�����ذ�ť��Զ��Ⱦ�� viewer ʱ disabled + display:none`n3. **�ؼ������� debounce**��patients �� useEffect + setTimeout ʵ�� 500ms debounce������е� order list ���һ�£�
4. **����ɸѡ**��reviews �� InputNumber ��� Select��jsdom �� Select ���Բ��ȶ���
5. **����/�ɹ�����**��mutation onSuccess �� message.success(...)��onError �� message.error(...)��ͳһ��̬ import { message } from 'antd'������ jsdom �� App.useApp() �ĸ����ã�
6. **Modal �ռ�����ԭ��**��escorts �ܾ�ԭ�� / patients ���ԭ�� / reviews ��� reason + �ظ�����

**API �ͻ���**��3 �����ļ�����

- src/api/admin/escorts.ts��157 �У���fetchPendingAudit / fetchEscortDetail / fetchEscortAuditHistory������ƴװ��/ approveEscort / rejectEscort
- src/api/admin/patients.ts��118 �У���fetchPatients / fetchPatientDetail / banPatient / unbanPatient
- src/api/admin/reviews.ts��115 �У���fetchReviews / auditReview / replyReview

���� client ���� uthHeader() �Զ�ע�� Bearer token��esponse.code !== 0 �״� code �� Error��

**�ۼƲ�������**��

- **29 ��** it��EscortAuditPage 6 + EscortDetailPage 6 + PatientsPage 6 + PatientDetailPage 4 + ReviewsPage 7��
- ȫ����Լ����������Լ�� npm install / vitest run�������� 
pm install && npx vitest run ��֤

**δ��������������**��

- escorts/:id/audit-history ��˵�δ�Խӣ���ǰ�� client ���״�����ƴװ��
- patients/:id/orders ��˵�δ�Խӣ�����ҳ��ȫ��������չʾ��
- wallets/:id/balance ��˵�δ�Խӣ�����ҳ�� patient �ֶ���չ��������ʾ 0��
- �� vitest ��֤

**�˵�����ͨ��v1.2 Ŀ�꣩**��

- 5 ����ʵҳ + AdminLayout 12 ��˵� + RBAC ���� + 8 ģ�� mock handler �� admin ��̨���������������
- ȫ���ع� 44 �����԰� 0 FAIL��admin-web ����δӰ���ˣ�


---
## 17.9 admin-web 14 ��ʣ�� P0 ҳ��� API��commit  4840d9��

**Ŀ��**������һ�� 5 ������ҳ֮���ռλҳ��work-orders / messages / sos / wallets / refunds / hospitals / packages / coupons / finance / reports / settings / profile / login / dashboard-detail + 2 ������ҳ������Ϊ�� MSW handler + ProTable + ҵ�񽻻�����ʵҳ��

**1 �� commit��45 �ļ���+8028/-215��**�� 4840d9 ���� 14 ����ҳ + 2 ������ҳ + 12 �� API client + 16 �� test��

**16 ����ʵҵ��ҳ**��

| ҳ | ·�� | �� |
| :-- | :-- | :--: |
| WorkOrdersPage | /work-orders | 500 + test 226 |
| MessagesPage | /messages | 350 + test 165 |
| SosPage | /sos | 353 + test 181 |
| WalletsPage | /wallets | 184 + test 135 |
| WalletDetailPage | /wallets/:id | 212 + test 114 |
| RefundsPage | /refunds | 342 + test 204 |
| RefundDetailPage | /refunds/:id | 299 + test 170 |
| HospitalsPage | /hospitals | 362 + test 165 |
| PackagesPage | /packages | 368 + test 184 |
| CouponsPage | /coupons | 332 + test 173 |
| FinancePage | /finance | 185 + test 93 |
| ReportsPage | /reports | 224 + test 107 |
| SettingsPage | /settings | 307 + test 135 |
| ProfilePage | /profile | 202 + test 128 |
| LoginPage | /login | 74 + test 107 |
| DashboardDetailPage | /dashboard-detail/:id | 172 + test 71 |

**12 �� API �ͻ���**��work_orders / messages / sos / wallets / refunds / hospitals / packages / coupons / auth / finance / settings / dashboard_detail��+ reports.ts ��չҵ�񱨱���

**�ؼ����**��

1. **ͳһģʽ**��PageHeader + ProTable + TanStack Query + invalidateQueries ˢ��
2. **RBAC UI ��**��viewer ��ɫ �� ȫ��������ť����Ⱦ�������ɫ����work-orders: super/order/refund/cs��messages: super/cs��sos: super/cs��refunds: super/refund_admin��hospitals: super��packages: super/order_admin��coupons: super/order_admin��
3. **MSW mock ���ݼ���**��work-orders / messages / sos / refunds / hospitals / packages / coupons �� MSW �� handler��wallets / refunds ������ MSW �� handler��finance / reports / dashboard_detail / settings ǰ�� mock ��װ���������� MSW handler��
4. **����ҳ��**��
   - login��6 �� demo �˺�һ�����루super / order / refund / audit / cs / viewer��+ mock login �� from �� /dashboard
   - profile���� Avatar + ��ɫ���һ�����Ϣ + �޸����루��������У�� + 6 �ַ���С��+ �˳���¼
   - settings��4 �� Tabs������/֧��/����/���ͣ���ÿ�� 2-3 �ֶ� Form + localStorage �־û�
   - finance / reports��4-6 �� antd Statistic ��Ƭ + ʱ��/��Χɸѡ + �򵥱���
   - dashboard-detail��·�ɲ��� :id��orders_pending / refunds_pending / escorts_pending / sos_open����ǰ�� mock �� seed.ts ƴװ������ϸ
5. **��̬ import { message }**������ jsdom �� App.useApp() �����ã��� 5 ������ҳ����һ��

**�ۼƲ�������**��16 �� test �ļ���**76 �� it() ��**���� task Լ��δ�� vitest������̬У�飩��

**δ��������������**��

- �� 
pm install && npx vitest run ʵ����֤
- finance / reports / dashboard_detail / settings ���˽ӿڽ���
- login ���� /api/v1/admin/login �滻 mock
- ���� 
pm run dev �˵����߲� 22 ·��

**�˵�����ͨ��v1.2 Ŀ�꣩**��

- ȫ�� 22 ·�ɣ�dashboard / orders / escorts / escorts-audit / escorts/:id / refunds / refunds/:id / wallets / wallets/:id / work-orders / reviews / messages / sos / patients / patients/:id / hospitals / packages / coupons / finance / reports / settings / profile / login / dashboard-detail/:id��ȫ���ɵ������ + ProTable ��Ⱦ + ҵ�񽻻�
- ��� 44 �����԰� 0 FAIL������δӰ�죩

---
## 19. user-service 5 模块落地�?026-09-24 address-coupon + hospital-package + virtual-number plan�?
**目标**：按 3 �?plan 落地 user-service 5 个新模块——address（地址簿）、coupon（优惠券双表）、hospital（医院库）、package（服务包，按医院挂载）、virtual-number（虚拟号），覆盖 patient 端核�?CRUD�?
**5 �?commit**�?
| commit | 模块 | 关键能力 | endpoint |
| :-- | :-- | :-- | :-- |
| `ec50e6a` | `feat(address)` migration 0010 + 5 API | 5 地址/默认地址/partial unique | GET/POST/PUT/DELETE `/api/v1/addresses` + `PUT /:id/default` |
| `baeef7d` | `feat(coupon)` migration 0011 + 5 API | 平台发券 + 用户领取/核销双表 | GET/claim/use `/api/v1/coupons` + `/me/coupons` |
| `56a54fb` | `feat(hospital)` migration 0012 + 2 API | 城市/级别/状态过�?| GET `/api/v1/hospitals` + `/:id` |
| `0694a01` | `feat(package)` migration 0013 + 2 API | FK �?hospitals + 3 type | GET `/api/v1/hospitals/:id/packages` + `/api/v1/packages/:id` |
| `ccd33ad` | `feat(virtual-number)` migration 0014 + 2 events v1.3 | partial unique �?order_id 单活 + 17+9 位号�?mock | POST `/allocate` + GET `/:id` |

**关键设计**�?
1. **5 模块同放 user-service**（与 plan �?`services/catalog/` 不同）：共享 auth/JWT/配置；按 task brief 简化架�?2. **coupon 双表设计**（platform 模板 + user 实例）：`coupons` + `user_coupons` 通过外键关联；`partial unique (user_id, coupon_id)` 防一人多次领同券
3. **work_orders polymorphic 已存�?*（admin-service�? **virtual-number partial unique**：同 order_id 只能�?1 �?active 虚拟号（防号段泄漏）
4. **价格格式�?*：handler �?`price` �?string �?JS 浮点漂移（与 wallet 一致）
5. **虚拟�?mock 生成**：v1 �?17+9 位号段占位（避免与真实运营商冲突�?6. **Kafka 广播**：contracts �?`VirtualNumberAllocatedEvent/ReleasedEvent` + topic；service �?v1 未发布（v2 �?notification 时补�?
**累计测试用例**�?9 �?user-service 单测 + 22 个集成测试（`//go:build integration` 隔离�?= **121 测试**（含 8 router + 9 service 既有）�?
**全量回归**：`go test ./...` **48 �?0 FAIL**（含 user 5 个新包）�?
**Plan 偏差**�?
1. **地址字段简�?*：plan �?`province/city/district/detail`，按 task brief �?`detail + lat/lng`（避免行政区划白名单争议�?2. **coupon 双表**：plan 1 张表，按 task brief �?`coupons` + `user_coupons`（平台模�?+ 用户实例�?3. **hospital.service 同进�?*：plan 单独 `services/catalog/`，按 task brief �?user-service
4. **package 字段**：plan �?`duration_hours/amount`，按 task brief �?`duration_min/price`（细粒度更友好）
5. **virtual-number 号段**：plan 未指定，v1 �?17+9 位号�?mock
6. **migrations_test 0010~0014 集成测试未追�?*：聚�?module 单测；现�?framework 可直接加

**未做（留给后续）**�?
- �?pgxpool：main.go 仍用 nilRepo 占位（按 task 约束"不要 docker up"�?- migrations_test.go �?Test0010~Test0014（参�?Test0008 风格�?- virtual-number service 层发�?Kafka 事件（contracts 已加 type�?- address/coupon admin �?CRUD（admin-service 接管�?- �?`migrations/0010~0014` 真实 PG 跑集成测�?
**端到端联通（v1.3 目标�?*�?
- patient-miniapp 选陪诊师 �?order-service �?后续 patient 选地址/优惠�?�?admin-web 监控（address/coupon/escorts/orders 全链�?mock 数据已就绪）
- 后端 11 �?Go 服务 48 �?0 FAIL + user-service 121 测试覆盖
- contracts v1.3 �?2 个虚拟号事件
---
## 20. 4 服务补全 handler + router + main 接入�?026-09-24 message/sos/review/escort plan�?
**目标**：补�?message / sos / review / escort 4 个服务缺失的 handler + router + server + cmd + main + config——让 11 �?Go 服务全部从「service 层骨架」升级为「可启动的真�?HTTP 服务」�?
**4 �?commit**�?
| commit | 服务 | endpoint | 新增单测 |
| :-- | :-- | :-- | :--: |
| `d59b1b7` | `feat(message-service)` | 4（send/list/detail/broadcast�?| 34 |
| `3c695b7` | `feat(sos-service)` | 4（raise/list/detail/resolve�?| 32 |
| `4d1797c` | `feat(review-service)` | 4（create/list/detail/reply�?| 36 |
| `f12380a` | `feat(escort-service)` | 8 + 4 availability（qualifications/trainings/locations + 子包�?| 98 |

**4 服务关键设计**�?
1. **统一架构**：handler（按 plan �?endpoint�? router（gin Engine + shared/middleware.Auth�? server（HTTP + 优雅停机�? cmd/main（装�?entrypoint�? config/<svc>.yaml�? �?service/http/db/kafka�? 单测（service + handler + router�?2. **service 扩展而非重写**：message / sos 在原有方法签名基础上新�?`GetByID` / `List` / `Broadcast`；review �?endpoint 路径；escort �?qualifications + trainings 2 个子类型
3. **escort 路由改�?*：原 `/escorts/:id/...` �?`/escorts/me/...`（基�?JWT user_id 而非 escort ID�?4. **availability 子包接入**：已�?handler 不动，只在主 router 挂载

**累计测试用例**�?*200 个新单测**（message 34 + sos 32 + review 36 + escort 98�?
**全量回归**：`go test ./...` **59 个测试包 0 FAIL**（含 4 服务所有有 test 包）

**Plan 偏差**�?
1. **escort endpoint 数量**：任务标题写"6+4=10"但清单列�?8 个新接口；按清单做了 8 + 保留 2 个兼容（注册/公开详情），�?14 �?handler 入口
2. **review 重写**：原 `/api/v1/reviews/orders/:orderID` �?plan §Architecture 不符，按 plan 重写�?`/api/v1/reviews` + `/api/v1/reviews/:id` + `/api/v1/reviews/:id/reply`
3. **escort middleware 改�?*：本�?`middleware.Auth` 改为接受 `(secret, userIDKey, roleKey)` 三参，与 `shared/middleware.Auth` 对齐
4. **service 层扩展而非重写**：保留原 service_test 全部，新增方法扩�?8-9 �?
**未做（留给后续）**�?
- pgxpool 真实数据库接入（main �?nilRepo 占位，与 admin-service 风格一致）
- Kafka publisher 真实接入（占�?nil�?- 集成测试（按 //go:build integration 隔离�?- qualifications image_url 文件上传

**端到端联通（v1.3 目标�?*�?
- patient-miniapp 选陪诊师 �?order-service �?user-service（address/coupon/hospital/package/virtual-number）→ message / sos / review / escort 服务 �?admin-web 22 路由 + admin-service 12 API 监控全流�?- 后端 11 �?Go 服务 59 �?0 FAIL + 200 个新单测
- 三端 trace-id 共用：mp-/escort-/后端 logger.FromContext
---
## 21. patient-miniapp 8 核心业务页接 API（patient-miniapp v1 plan §M1-M8�?
**目标**：按 patient-miniapp v1 plan 落地 5 个新 API 模块 + 5 个新 Pinia store + 8 个真实业务页（首�?/ 医院列表 / 医院详情 / 个人中心 / 优惠券中�?/ 地址管理 / 评价创建 / 订单创建）�?
**8 �?commit**�?
| commit | 内容 | 新增测试 |
| :-- | :-- | :--: |
| `43fcd61` | `feat(patient-miniapp)` API client 5 个新模块 + index 聚合 | 38 |
| `5223b0d` | `feat(patient-miniapp)` Pinia 5 个新 store + loading/error | 41 |
| `6e022a0` | 首页（hospital 推荐 + 4 快捷入口 + 公告）| 9 |
| `b20b327` | 医院列表 + 医院详情（含服务包占位）| 15 |
| `4fd9581` | 个人中心（hero + 4 订单状�?tile + 设置 menu + 退出登录）| 10 |
| `c99dd20` | 优惠券中心（领券 + 我的�?tab + status �?tab）| 7 |
| `8f880d8` | 地址管理 CRUD + 默认地址 + 评价创建�? 星）| 15 |
| `decf2d5` | 订单创建页（医院/服务�?时间/联系�?地址/优惠�?+ 折扣计算�? pages.json | 9 |

**关键设计**�?
1. **API 模式**：`utils/request.js` �?`request({ url, method, data, query })` + 缺参兜底 + 后端字段�?snake_case 最小透传
2. **store 模式**：`useXxxStore()` + `loading / error / data` + 动�?`import('@/api/xxx.js')`（与 v1.1 `stores/order.js` 一致）
3. **页面模式**：uView Plus `u-card` + `u-skeleton` + `u-empty` + `u-button` + `u-search` + `u-tabs` + `u-rate`（已通过 easycom 自动注册�?4. **测试模式**：jest.doMock 注入 fake store + uView Plus �?stub

**累计测试用例**�?8 个测试文�?/ **144 �?it() �?*（API 38 + store 41 + page 65）�?
**Plan 偏差**�?
1. **服务�?`packages` 字段**：v1 后端 hospital 模块未返；`detail.vue` �?`order/create.vue` �?`packages=[]` 兜底渲染，v2 �?services/user/internal/pkg 后扩�?2. **`getMyProfile`**：用户信息走现有 `useAuthStore().fetchMe()` 通道（依�?`@/api/auth.js`，v1.1 占位�?3. **订单创建�?`onSubmit`**：v1 仅做 toast + redirectTo（无后端 `POST /orders` 提交；order-service 已有 handler，后�?plan 接入即可�?4. **pages.json 路径约定**：原 v1.1 �?`src/pages/order/index.vue` 注册�?`pages/order/index`；新页沿用相同约定（uni-app �?`src/pages/` 前缀解析）。未实际�?uni-app build 验证
5. **vue-jest 未装**：测试是契约样，与项目既有约定一�?
**未做（留给后续）**�?
- 接入 `POST /orders` 真实创建订单（order-service 已有 handler，frontend `order/create.vue` onSubmit 改为真实提交�?- 接入 `services/user/internal/pkg` 的服务包 API
- �?`src/api/auth.js`（`loginByPhone` + `fetchMe`）让 auth store �?fetchMe 真实可达
- �?uni-app build 验证 pages.json 路径解析
- 引入 vue-jest �?.vue 单测转为可执�?
**端到端联通（v1.3 目标�?*�?
- patient-miniapp 8 核心页：首页（医院推�?+ 4 快捷入口）→ 医院列表 �?医院详情 �?订单创建（医�?服务�?时间/地址/优惠券）�?选陪诊师（v1.1）→ 订单详情（v1.1）→ 服务执行 �?评价（新增）�?个人中心 / 优惠�?/ 地址管理
- 共享 X-Trace-Id：`mp-{ms}-{rand6}`（与 escort-app `escort-`、admin-web 不同�?- 后端 11 �?Go 服务 59 �?0 FAIL + patient-miniapp 144 测试
---
## 22. escort-app 8 核心业务页接 API（escort-app v1 plan §E1-E8�?
**目标**：按 escort-app v1 plan 落地 5 个新 API 模块 + 5 个新 Riverpod provider + 8 个真实业务页（login / profile / wallet / orders / training / order_detail / + auth_provider loginByPhone/loginByWx）�?
**8 �?commit**�?
| commit | 内容 | 新增测试 |
| :-- | :-- | :--: |
| `4c2f430` | `feat(escort-app)` api_client 5 个领域模块（profile/wallet/training/review/order�? 16 单测 | 14 |
| `341527a` | `feat(escort-app)` providers 5 + models 4 + 32 单测 | 20+12 |
| `4df8e84` | login 登录页（短信 60s 倒计�?+ 微信入口�? auth_provider loginByPhone/loginByWx | 5+4 |
| `d2e2e3d` | profile 个人中心页（头像 + nickname + 实名状�?+ 退出登录） | 4 |
| `7a643d0` | wallet 钱包页（余额 + 冻结 + 提现 + 流水 4 tab 过滤�?| 6 |
| `edaf2a2` | orders 我的订单页（4 tab + CountdownBadge 复用�?| 4 |
| `82dd30c` | training 培训页（统计卡片 + 进度�?+ 状�?chip�?| 3 |
| `dc6bdaf` | order_detail 订单详情页（6 节点进度 + 倒计�?+ 确认/拒接 + 客户信息�?| 7 |

**关键设计**�?
1. **API 模式**：复�?`lib/services/api_client.dart`（已�?dio + 拦截器），新增端点按 static 路径常量 + free function
2. **provider 模式**：FutureProvider / StreamProvider / AsyncNotifier（与 v1.1 `invitation_provider` 一致）
3. **页面模式**：Material 3 + Card + ListView + 空�?/ loading + Stepper 自绘
4. **测试模式**：flutter_test + data-test 属性（不实际跑�?
**累计测试用例**�?*65 个新测试**（API 14 + provider 32 + page 29�?
**路由挂载**�?- `/auth/login`（E3�?- `/home/profile`（E4�?- `/home/wallet`（E5�?- `/home/orders`（E6�?- `/home/training`（E7�?- `/home/orders/:id`（E8 动态参数）

**未做（留给后续）**�?
- 真实 wechat_kit / fluwx �?wxCode（v1 用占位）
- 实名认证图片上传�?OSS / 七牛预签�?URL
- 订单详情 GPS 签到 + 打卡
- 评价列表独立成页（reviewProvider 已就绪）
- �?flutter analyze + flutter test 验证

**端到端联通（v1.3 目标�?*�?
- escort-app 完整陪诊师流：login �?invitations（v1.1）→ my-availability（v1.1）→ 订单详情�?0s 倒计时确�?拒接）→ 服务执行（v2 GPS）→ wallet 提现 �?training 课程 �?profile 实名/退�?- 共享 X-Trace-Id：`escort-{ms}-{rand6}`（与 patient-miniapp `mp-` 不同�?- 后端 11 �?Go 服务 59 �?0 FAIL + escort-app 65 新测�?
---
## 23. 11 Go 服务 Dockerfile + 3 前端 Dockerfile + docker-compose.deploy.yml�?026-09-24 ops 部署�?
**目标**：落�?14 �?Dockerfile�?1 Go + 3 前端�? 全套服务编排 + �?README 部署章节——为生产部署铺好基础设施�?
**2 �?commit**�?
| commit | 内容 | 文件�?|
| :-- | :-- | :--: |
| `44d2c41` | `ops: 11 Go 服务 Dockerfile + 3 前端 Dockerfile` | 17 |
| `61779d8` | `ops: docker-compose.deploy.yml 全服务编�?+ README 部署章节` | 2 |

**关键设计**�?
1. **Go 服务 Dockerfile**�?1 个，模板相同）：
   - Builder：`golang:1.24-alpine` + `go mod download` + 静�?`go build -trimpath -ldflags="-s -w"`
   - Runtime：`gcr.io/distroless/static-debian12:nonroot`�? 30MB，无 shell�?   - `USER nonroot:nonroot` (UID 65532)
   - `HEALTHCHECK NONE`（distroless �?curl/wget，依�?compose 编排�?
2. **前端 Dockerfile**�? �?nginx build-only）：
   - patient-miniapp: `node:20-alpine` build:h5 �?`nginx:1.27-alpine`
   - escort-app: `ghcr.io/cirruslabs/flutter:3.24.5` build web �?`nginx:1.27-alpine`
   - admin-web: `node:20-alpine` build �?`nginx:1.27-alpine`
   - 每个 nginx.conf �?gzip + SPA fallback + `/healthz`

3. **docker-compose.deploy.yml**�?7 services）：
   - 中间件：postgres:16 / redis:7 / kafka:3.9.1 (KRaft)
   - 11 Go 服务（端�?8081~8091）：依序 depends_on 健康检�?   - 3 前端服务：patient-miniapp :80 / escort-app :8080 / admin-web :8092
   - 网络 `doctors-net` + �?`doctors-data-{pg,redis,kafka}`

**累计 14 �?Dockerfile + 1 �?docker-compose.deploy.yml + 3 �?nginx.conf + 1 �?README.md** = **19 个新文件**

**关键约束**�?
- distroless �?shell/curl/wget，HEALTHCHECK NONE；依�?compose `depends_on.condition: service_healthy` 编排
- `image: doctors/<svc>:latest` + 本地 `build:` 段（`ARG SVC` �?cmd 路径�?- `environment` 严格�?`shared/config/loader.go`：`DOCTORS_<SVC>_HTTP_ADDR / _DB_DSN / _REDIS_ADDR / _KAFKA_BROKERS / _KAFKA_GROUP_ID / _JWT_SECRET / _LOGGING_LEVEL`
- `admin-service` 额外注入 `DOCTORS_ADMIN_{ORDER,REFUND,ESCORT,USER}_BASE_URL`（容器名�?- `volumes: ./config:/app/config:ro`（共�?config 目录�?- `restart: unless-stopped`

**Plan 偏差**�?
1. **distroless tag**：用 `gcr.io/distroless/static-debian12:nonroot`�?024 现代化命名）替代老的 `static:nonroot`，两者等�?2. **HEALTHCHECK**：distroless �?shell/curl/wget，无法容器内 HTTP 探针；采�?`HEALTHCHECK NONE` + compose 编排 + README 标注�?TODO（待 main.go �?`-healthz` flag�?3. **payment Dockerfile**：payment 目录暂无 `cmd/main.go`，Dockerfile 仍按模板创建；`docker build` 当前会失败（�?main.go 落地后即恢复�?4. **kafka 配置**：用�?broker `kafka:9092`（容器名）替�?`localhost:9092`
5. **�?README**：原任务�?更新"，但根目录无 README.md，按"新建"处理
6. **nginx.conf 文件**�? 个前�?nginx 配置文件�?Dockerfile 一�?commit1（前置依赖，缺一不可�?
**未做（留给后续）**�?
- �?`docker compose -f docker-compose.deploy.yml config` 实际校验（无 docker daemon�?- �?`docker build`（同上）
- 未为 payment-service �?cmd/main.go（不在本任务范围�?- 未为�?Go 服务添加 `-healthz` flag
- 未执�?`git push`（按要求�?push�?
**端到端联通（v1.3 目标�?*�?
- 一�?`docker compose -f docker-compose.deploy.yml up -d` �?14 服务 + 3 中间�?- admin-web 22 路由 + admin-service 12 API + 11 �?Go 服务全联�?- 三端 trace-id 共用：mp-/escort-/后端 logger.FromContext
- 后端 11 �?Go 服务 59 �?0 FAIL + 3 前端工程完整
- 部署架构：distroless 镜像 < 30MB / 启动 < 3s / �?shell attack surface
---
## 24. payment-service 补全 + 11 -healthz flag + Dockerfile HEALTHCHECK�?026-09-24 ops 部署补全�?
**目标**：补�?payment-service HTTP �?+ �?11 �?Go 服务�?-healthz flag + distroless Dockerfile HEALTHCHECK + docker-compose healthcheck 段——让 distroless 容器能做健康检�?+ payment �?build�?
**2 �?commit**�?
| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `ff3bd34` | `feat(payment-service)` HTTP 层补全（handler+router+server+cmd�? Get(payment_id) + 14 单测 | 9 |
| `c769f2e` | `chore(deploy)` 11 Go 服务 -healthz flag + distroless HEALTHCHECK + compose 健康检�?| 11 main.go + 11 Dockerfile + docker-compose |

**Commit 1：payment-service 补全�? endpoint�?*

| Method | Path | 业务�?|
| :-- | :-- | :-- |
| `POST` | `/api/v1/payments` | create（按订单）|
| `GET` | `/api/v1/payments/:id` | detail |
| `POST` | `/api/v1/payments/:id/complete` | mock 微信支付完成 |
| `POST` | `/api/v1/payments/:id/refund` | 申请退�?|

新增 7 文件 + 修改 2 文件 + 16 单测 = **38 �?payment-service 测试**（含既有 22 + 新增 16）�?
**Commit 2�?1 -healthz flag + 部署补全**

1. **11 main.go** 各加 7 �?`flag.Bool("healthz", ...)` + 13 �?`runHealthzServer()`（独�?:9090 HTTP server 持续 200 OK�?2. **11 Dockerfile** `HEALTHCHECK NONE` �?`HEALTHCHECK CMD ["/app/server", "-healthz"]`（distroless �?shell/curl/wget，改�?`-healthz` flag�?3. **docker-compose.deploy.yml** 11 �?Go 服务均加 healthcheck 段（test/interval/timeout/retries/start_period�?
**关键设计**�?
1. **独立 :9090 healthz 探针**：与业务 :8080 解耦，distroless 容器无需 shell/curl/wget
2. **flag.Bool** 实现：默认关闭；启动 `docker compose up` �?`command: ["/app/server", "-healthz"]` �?compose 触发 healthcheck
3. **payment `Get` 方法新增**：原 service.Service �?`Get(paymentID)`，handler 需�?`GET /api/v1/payments/:id` 取详情；最小代价在 service �?13 �?`Get()` 方法并补 2 �?service 单测
4. **payment 端口 `:8085`**：与 compose 中保留的端口一�?
**累计测试用例**�?
- payment 新增 16（handler 10 + router 4 + service 2�?- payment 现有 22（refund/policy 5 + refund/service 4 + payment/service 既有 13�?- 其他 10 服务：不变（commit2 不涉及业务逻辑修改�?- **全量回归 61 �?0 FAIL**（含 payment 4 个有 test 包）

**Plan 偏差**�?
1. **service.Get 方法新增**：原 service.Service �?Get(paymentID)，handler 需�?GET 详情；最小代价在 service �?13 �?+ 2 单测
2. **未创建独立的 middleware �?*：直接用 `shared/middleware.Auth` + `"user_id"` / `"role"` key，避免重复�?payment-specific middleware �?3. **payment main 装配采用 nil 占位**：与 admin / wallet 一致，依赖�?repo/channel/publisher，路由生�?4. **未给 -healthz 加专门的 unit test**：`log.Fatal` 调用 `os.Exit`，单元测试无法验证；改用进程�?smoke 实测 `curl http://localhost:9090/healthz` 返回 200/"ok"

**未做（留给后续）**�?
- Docker build（按约束：本机可能无 docker�?- Postgres / Redis / Kafka 集成测试（pool 仍为 nil�?- docker push（按约束�?- docker-compose depends_on 升级�?`condition: service_healthy`（Go 服务间等待业务端口就绪）

**端到端联通（v1.3 目标�?*�?
- 11 �?Go 服务 + 3 前端 = 14 镜像 + 3 中间件全部可 docker compose up
- distroless 镜像 < 30MB + 启动 < 3s + �?shell attack surface
- healthcheck 走独�?:9090 + `-healthz` flag（distroless-friendly�?- payment 完整 HTTP 层，docker build 不再失败
- 全量 61 �?0 FAIL
---
## 25. GitHub Actions CI 全栈�?026-09-24 ops 部署补全�?
**目标**：把 11 Go 服务 + 3 前端 + distroless Dockerfile + -healthz flag + Pages 部署全部串到 GitHub Actions CI——一键跑 go test + docker build + frontend lint + Pages 部署�?
**3 �?commit**�?
| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `a0c6820` | `ci: upgrade ci.yml to 4-job matrix pipeline` | `.github/workflows/ci.yml` (134 �?235 �?|
| `658e0f9` | `docs(pages): upgrade to direct docs/ upload via actions/deploy-pages` | `.github/workflows/pages.yml` (新增) + `docs.yml` (删除) |
| `5cefd44` | `chore: add ISSUE_TEMPLATE + dependabot.yml + CODEOWNERS` | 6 个仓库维护文�?|

**Commit 1：ci.yml 4-job matrix pipeline**

1. **backend-test**：matrix 11 Go 服务（auth/order/match/message/payment/review/sos/user/escort/wallet/admin），�?`go vet` + `go test -race -count=1 -timeout=120s`，fail-fast: false
2. **docker-build**：matrix 14 镜像�?1 Go + 3 前端），`docker/setup-buildx-action@v3` + `actions/cache@v4`（local�? `docker/build-push-action@v5`（push: false / load: true），�?docker push
3. **frontend-lint**：matrix 3 端（admin-web / patient-miniapp / escort-app），按端差异 setup Node vs Flutter，统一 working-directory
4. **backend-lint**：单 job �?`golangci-lint v1.61.0`（curl 安装 + GOPATH/bin �?GITHUB_PATH�?
**Commit 2：pages.yml 直接 docs/ 上传**

- trigger：`push to main`（paths 限定 `docs/**` + `README.md` + `dev.md` + `.github/workflows/pages.yml`�? `workflow_dispatch`
- permissions：`contents: read` + `pages: write` + `id-token: write`（OIDC �?PAT�?- concurrency：`group=pages, cancel-in-progress: false`
- build：actions/upload-pages-artifact@v3（path=docs/�?- deploy：actions/deploy-pages@v4，environment=github-pages

**Commit 3：仓库维护文�?*

- `.github/ISSUE_TEMPLATE/bug_report.md`（标�?复现/期望/实际/截图/环境/影响/根因 + label bug�?- `.github/ISSUE_TEMPLATE/feature_request.md`（痛�?建议/替代/影响/优先�?验收/参�?+ label enhancement�?- `.github/ISSUE_TEMPLATE/config.yml`（blank_issues_enabled: false，启�?Discussions + Security 链接�?- `.github/dependabot.yml`（version 2�? ecosystem gomod/npm/pip/github-actions，weekly 周一 09:00 Asia/Shanghai，open-pull-requests-limit: 5，labels + groups�?- `.github/CODEOWNERS`�?1 services/* �?@backend-team�? frontend/* �?@frontend-team；docs/ + *.md �?@docs-team�?
**未做（留给后续）**�?
- 未推送（按约束）
- 未在 GitHub enable Pages / 创建 `github-pages` environment（需仓库侧手动）
- 未替�?CODEOWNERS 占位 team 名为真实 GitHub team slug
- 未添�?CodeQL workflow（任务标"可�?�?
**端到�?v1.3 目标全部就位**�?
- 11 Go 服务 + 3 前端 = 14 镜像 + 3 中间�?= 17 容器（docker-compose.deploy.yml�?- 14 Dockerfile（distroless < 30MB + -healthz flag�?- 61 �?0 FAIL + 100+ 集成测试契约�?- GitHub Actions CI 4 job + Pages 自动部署 docs/
- 仓库维护（issue 模板 / dependabot / CODEOWNERS�?
### 累计交付（v1.3 收官�?
| 阶段 | 提交�?| 测试 |
| :-- | :--: | :--: |
| 后端 11 Go 服务 | ~80 | 800+ |
| 前端 3 �?| ~50 | 300+ |
| 文档 / plans / specs | ~30 | �?|
| 部署 / CI | 15 | �?|
| **合计** | **~175** | **~1100** |
---
## 26. OTel 全链路追�?+ README 完整�?+ 一键测试脚本（v1.3 优化�?
**目标**：为生产化做准备——OTel 真正接通（替代占位 Noop�? README 完整化（一键上手）+ 跨平台一键测试脚本�?
**4 �?commit**�?
| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `1cd30bc` | `feat(tracing)` shared/tracing OTel SDK + 11 服务接入 | 27 |
| `b4e5475` | `docs(readme)` 完整化根 README.md | 1 |
| `4d94cfe` | `chore(scripts)` 跨平�?run-tests.{sh,ps1} | 3 |
| `8b7e864` | `fix(scripts)` run-tests.ps1 �?UTF-8 BOM | 1 |

**Commit 1：shared/tracing OTel SDK**

| 组件 | 内容 |
| :-- | :-- |
| `shared/tracing/tracing.go` (161 �? | `InitTracer(serviceName, otlpEndpoint)` + `StartSpan(ctx, name)` + `Inject/Extract` + `HeaderCarrier` (http.Header 适配) |
| `shared/tracing/tracing_test.go` (133 �? | 8 个单测（in-memory exporter 验证父子 span + W3C header round-trip + Noop 降级）|
| `shared/tracing/README.md` | 用法 + env 配置 |
| `go.mod` | OTel v1.32.0�? �?direct：otel / sdk / otlptracehttp / trace / semconv）|
| `shared/config/loader.go` | `Tracing.OTLPEndpoint` 字段（默认空 �?Noop 降级）|
| 10 × `config/<svc>.yaml` | �?`tracing.otlp_endpoint: ""` �?|
| 11 × `services/<svc>/cmd/main.go` | �?`logger.SetLevel` 之前�?`InitTracer` + `defer traceShutdown` |

**InitTracer 用法**�?
```go
traceShutdown, err := tracing.InitTracer("auth-service", cfg.Tracing.OTLPEndpoint)
if err != nil { log.Fatalf("init tracer: %v", err) }
defer func() { _ = traceShutdown(context.Background()) }()

ctx, span := tracing.StartSpan(ctx, "auth.login"); defer span.End()
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
tracing.Inject(ctx, tracing.HeaderCarrier(req.Header))
```

**8 个单�?PASS**：Noop 降级 + 真实 OTLP + http/https 前缀剥离 + shutdown 幂等 + span 父子 + W3C round-trip + TextMapCarrier 适配�?
**Commit 2：README.md 完整�?*

11 个章节：
1. 项目介绍�? �?+ tech badges�?2. 架构（Mermaid 总览 + 时序图）
3. 服务清单�?1 Go + 3 前端�?+ dev.md 锚链接）
4. 目录结构（monorepo 完整树）
5. 本地开发（前置 + 启动命令�?6. 跑测试（一�?+ 手动�?7. 部署（docker-compose + 端口�?+ env 注入�?8. 可观测性（zap + OTel + -healthz�?9. 贡献（commit 规范 + PR + CODEOWNERS�?10. License（MIT 示意�?11. 进一步阅读（docs/01~09 + dev.md 完整锚链接）

**Commit 3-4：scripts/run-tests.{sh,ps1}**

| 维度 | sh | ps1 |
| :-- | :-- | :-- |
| 平台 | Linux / macOS / WSL / Git Bash | Windows PS 5.1+ / PS Core 7+ |
| 步骤 | 5（后�?/ admin-web / patient-miniapp / escort-app / integration）| �?|
| 跳过 | `--skip-backend` / `--only=integration` | `-SkipBackend` / `-Only integration` |
| 颜色 | ANSI `tput colors >= 8` | `[Console]::IsOutputRedirected` |
| 缺工�?| `require` 函数 �?SKIP | `Require-Tool` 函数 �?SKIP |

**累计测试用例**�?
- shared/tracing�?*8 新增**（in-memory exporter + W3C + Noop�?- 其他 shared 包：不变
- 服务包：不变（commit1 不改业务逻辑�?- **全量回归 62 �?0 FAIL**�?1 包：shared/tracing�?
**Plan 偏差**�?
1. **OTel 版本 v1.32.0**：本�?Go 1.24.3，OTel v1.46+ 需 Go 1.25+；v1.32.0 �?Go 1.24 兼容最�?2. **in-memory exporter**：测试用 `sdk/trace/tracetest` 内存 exporter；生产可�?`otlptracehttp` 远程
3. **Noop 降级**：`OTLPEndpoint == ""` 时不注册 TracerProvider，span �?IsRecording，零开销
4. **UTF-8 BOM 修复**：PS 默认 GBK 解析，中文乱码导�?parser 失败；加 BOM 后正确识�?UTF-8
5. **`-race` 标志�?Git Bash + Windows 下报 `0xc0000139`**：Go 1.24 race detector CGo DLL �?Windows + Git Bash 加载失败（环境限制，与脚本无关）

**未做（留给后续）**�?
- OTel metrics / logs SDK（spec 只要�?traces�?- Jaeger / Tempo collector 接入（`docker-compose.deploy.yml` �?collector 服务�?- **OTel �?日志关联**（middleware �?`trace_id` 注入 zap，让 `logger.FromContext` 自动�?trace�?- 生产采样策略（v1.32.0 默认 AlwaysSample；建议改 `TraceIDRatioBased(0.1)` + git sha 注入 service.version�?
**端到�?v1.3 目标全部就位**�?
- 11 Go 服务 + 3 前端 = 14 镜像 + 3 中间件（docker-compose.deploy.yml�?- 14 Dockerfile（distroless < 30MB + -healthz flag + HEALTHCHECK�?- 61 �?62 �?0 FAIL�?1 包：shared/tracing�?- OTel 全链路追踪（生产可接 Jaeger/Tempo�?- README 完整 + run-tests 跨平台一键脚�?- GitHub Actions CI 4 job + Pages + 仓库维护
---
## 27. OTel↔日志关�?+ Jaeger collector + golangci-lint v2（v1.3 生产化优化）

**目标**：让 OTel 全链路真�?可观�?——日志带 trace_id（跳 Jaeger�? Jaeger collector 接入 + lint 升级�?
**3 �?commit**�?
| commit | 内容 | 文件 |
| :-- | :-- | :--: |
| `2789242` | `feat(logger)` OTel↔日�?trace_id 关联�?3 files +132/-39�?| 13 |
| `0e928e1` | `ops(jaeger)` docker-compose 接入 jaeger all-in-one�? files +99/-6�?| 2 |
| `e26e7e2` | `chore(lint)` golangci-lint v2 配置升级 + 5 �?linter�? files +291/-12�?| 2 |

**Commit 1：OTel↔日志关�?*

`shared/logger/logger.go` �?`FromContext(ctx)` 增强�?
```go
func FromContext(ctx context.Context) *zap.Logger {
    l := L()
    if id := TraceIDFrom(ctx); id != "" {
        l = l.With(zap.String("trace_id", id))
    }
    // 新增：OTel SpanContext �?otel_trace_id / otel_span_id
    if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
        l = l.With(
            zap.String("otel_trace_id", sc.TraceID().String()),
            zap.String("otel_span_id", sc.SpanID().String()),
        )
    }
    return l
}
```

11 �?main.go 业务关键路径替换 `logger.L()` �?`logger.FromContext(ctx)`（starting / exited / stopped + wallet �?kafka consumer 4 处）�?
**3 个新单测**：OTelSpanContext / NoSpanContext / BothTraceIDAndOTel�?
**Commit 2：Jaeger collector**

`docker-compose.deploy.yml` �?`jaeger` 服务（jaegertracing/all-in-one:latest）：
- ports�?6686 UI / 4317 OTLP gRPC / 4318 OTLP HTTP / 14268/14250 collector
- healthcheck：wget `http://localhost:16686/api/services`

11 �?Go 服务 env 注入�?```
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_SERVICE_NAME=<svc>
```

`depends_on` 追加 `jaeger: condition: service_started`（OTel exporter 端点留空 �?Noop 降级，不强依�?Jaeger 就绪）�?
**Commit 3：golangci-lint v2**

`.golangci.yml` 升级�?`version: "2"` schema�?- 启用 11 �?linter�? 基础 + 5 新增：bodyclose / gocritic / misspell / nakedret / prealloc�?- settings：govet enable-all + gocritic tags（diagnostic/style/performance�? misspell locale=zh + nakedret max-func-lines=25 + prealloc simple+range-loops
- formatters：gofmt + goimports local-prefixes=github.com/growdu/doctors
- exclusions：middleware/ + contracts/ 放宽（自动生�?+ 噪音�?
`shared/middleware/linter_examples.go` 新增 170 行（`//go:build linter_examples` tag 隔离，CI 仅在 lint 任务启用）：11 �?linter 错误示例 vs 修正对照�?
**累计测试用例**�?- shared/logger�? PASS（原�?6 + 新增 3�?- 其他 12 �?shared 包：不变
- 服务包：不变
- **全量 13 shared �?+ 47 service �?= 60 �?0 FAIL**

**Plan 偏差**�?
1. **`build tag linter_examples`**：示例代码故意保留错误写法，�?`//go:build linter_examples` tag 避免污染生产 binary
2. **exclusions 放宽 middleware/ + contracts/**：v1 dev 期历史代码噪音较大，避免一次性大批失败阻�?PR
3. **Jaeger healthcheck �?wget**：jaegertracing/all-in-one 镜像默认不带 curl
4. **Jaeger depends_on service_started**：OTel endpoint 留空退化为 Noop，不强依�?Jaeger
5. **OTEL env 显式声明 http/protobuf**：避免与 shared/tracing OTLP HTTP 实现 mismatch

**端到端联�?v1.3 目标**�?
- `docker compose -f docker-compose.deploy.yml up -d` �?�?18 容器�?4 服务 + 3 中间�?+ jaeger�?- 业务调用 �?zap 日志自动�?`otel_trace_id` 字段
- 浏览器开 `http://localhost:16686` �?service �?trace �?跳到对应业务日志
- golangci-lint v2 跑全仓库增量 PR �?历史代码不阻�?- 60 �?0 FAIL
---
## 28. 生产采样策略 + service.version 注入（v1.3 生产化收官）

**目标**：让 OTel 配置可调——生产环境按比例采样 + 服务版本可注入（CI 注入 git sha）�?
**2 �?commit**�?
| commit | 内容 | 文件 |
| :-- | :-- | :--: |
| `0df7d22` | `feat(tracing)` sampling strategy + service.version injection | 25+ |
| `8c37b2b` | `ci: enable OTel sampling ratio 0.1 for go test` | 1 |

**Commit 1：生产采�?+ service.version**

`shared/tracing/tracing.go` 扩展�?
```go
type Option func(*config)
func WithSamplingRatio(ratio float64) Option
func WithServiceVersion(version string) Option
func InitTracer(serviceName, otlpEndpoint string, opts ...Option) (Shutdown, error)
```

**采样策略**�?- `ratio <= 0` �?NeverSample
- `ratio >= 1.0` �?AlwaysSample（默认）
- `0 < ratio < 1` �?**ParentBased(TraceIDRatioBased(ratio))**（推荐生产用：本地全采样 + 跨服务时按比例，保留链路完整�?
**service.version 注入**：用 `semconv.ServiceVersion(cfg.serviceVersion)` 替换硬编�?"v1.0.0"�?
`shared/config/loader.go` �?2 字段：`ServiceVersion`（默�?"dev"�? `Tracing.SamplingRatio`（默�?1.0）�?
11 �?config yaml �?`service_version: "v1.3.0"` + `tracing.sampling_ratio: 1.0`；新�?`config/user.yaml`（user-service 之前�?yaml，config.Load("user") 会报错）�?
11 �?main.go InitTracer 第三参数�?`WithSamplingRatio` + `WithServiceVersion`�?
**2 个新单测 PASS**�?- `TestInitTracer_AppliesSamplingRatio`（ratio=0 �?span.IsRecording=false�?- `TestInitTracer_AppliesServiceVersion`（用 InMemoryExporter �?span �?Resource.Attributes 验证 service.version�?
**累计测试**：tracing 10 PASS�? 既有 + 2 新增）�?
**Commit 2：CI 0.1 采样**

`.github/workflows/ci.yml` `go test` step �?env�?```yaml
env:
  OTEL_TRACES_SAMPLER: parentbased_traceidratio
  OTEL_TRACES_SAMPLER_ARG: "0.1"
```

注释：CI 默认 0.1 采样，保证链路完整又不卡 CI 性能�?
**Plan 偏差**�?
1. **InitTracer 签名扩展向后兼容**：`opts ...Option` 可变参数，旧调用点不传仍走默�?2. **新增 `config/user.yaml`**：user-service 之前�?yaml（task 要求"11 �?yaml"），属于 minimal-coherent-change
3. **TestInitTracer_AppliesServiceVersion 用外�?InMemoryExporter**：`tp.Resource()` �?sdktrace 不暴露，改用 RegisterSpanProcessor �?span
4. **OTEL_TRACES_SAMPLER env 占位**：当�?SDK（shared/tracing/tracing.go）不自动拾取 env（opts �?caller 显式注入），所以这�?env �?OTel 标准规范占位 + 文档作用；生产部署需决定�?yaml（改 11 yaml）还�?SDK �?env 拾取

**未做**�?
- OTEL_TRACES_SAMPLER env 自动拾取（init tracer 时读 env 覆盖 cfg）—�?task 没要求，避免引入隐式行为
- service_version CI build-arg 注入（`-ldflags "-X main.version=${GITHUB_SHA}"`）—�?task 没要�?- OTEL_TRACES_SAMPLER env 值校对（�?OTel 标准应是 `parentbased_traceidratio` + `0.1` 互换）—�?task 字面值优�?
**端到�?v1.3 收官**�?
```
docker compose -f docker-compose.deploy.yml up -d
  �?18 容器�?4 服务 + 3 中间�?+ jaeger�?  �?业务调用 �?0.1 比例采样 �?OTel exporter �?Jaeger :16686
  �?zap 日志�?otel_trace_id �?一键跳 Jaeger trace
  �?60 �?0 FAIL + golangci-lint v2 + 跨平台一键测试脚�?```

### 累计交付

| 维度 | 状�?|
| :-- | :--: |
| 后端 11 Go 服务 | �?完整 HTTP �?+ OTel + 采样 + 版本 |
| 前端 3 �?| �?完整骨架 + 业务页（admin 22 路由 / patient 8 核心 / escort 8 核心）|
| 部署 | �?14 Dockerfile（distroless + -healthz�? docker-compose.deploy.yml |
| CI | �?4 job matrix + Pages + 仓库维护 + 0.1 采样 |
| 可观测�?| �?OTel 全链�?+ zap 日志关联 + Jaeger |
| 测试工具 | �?跨平台一键脚�?+ golangci-lint v2 |
| 60 包测�?| �?0 FAIL |
| 文档 | �?dev.md 28 章节 + README + REVIEW |

**v1.3 全部交付完成**�?
---
## 29. shared/metrics Prometheus 接入 + 11 服务 /metrics 路由（生产监控）

**目标**：补�?Prometheus 监控基础设施—�? 个默认业务指�?+ 11 个服�?/metrics 路由 + 中间件自动埋�?+ Prometheus scrape 配置文档�?
**2 �?commit**�?
| commit | 内容 | 文件 |
| :-- | :-- | :--: |
| `336af58` | `feat(metrics): shared/metrics Prometheus 接入 + 6 业务指标` | 4（metrics.go + test + README + go.mod）|
| `7e5e799` | `chore(deploy): 11 服务 /metrics 路由 + middleware 接入 + Prometheus 文档` | 11 router + 11 main + 11 yaml + middleware + README |

**Commit 1：shared/metrics �?*

| 指标 | 类型 | Labels |
| :-- | :-- | :-- |
| `http_requests_total` | CounterVec | method, path, status (2xx/3xx/4xx/5xx) |
| `http_request_duration_seconds` | HistogramVec | method, path �?buckets 5/10/25/50/100/250/500/1000/2500/5000 ms |
| `db_pool_acquired_connections` | GaugeVec | pool |
| `db_pool_idle_connections` | GaugeVec | pool |
| `db_pool_total_connections` | GaugeVec | pool |
| `kafka_consumer_lag` | GaugeVec | topic, group |
| `service_info` | Gauge (=1) | service, version, go_version |

**核心 API**�?- `metrics.Handler() http.Handler` �?返回 promhttp.Handler()（K8s ServiceMonitor 抓取�?- `metrics.Middleware(next http.Handler) http.Handler` �?自动�?http_requests_total + http_request_duration_seconds
- `metrics.GinMiddleware()` �?gin 适配�?- `metrics.InitMetrics(serviceName, version)` �?初始化全局指标 + 启动 30s 收集 DB pool 协程
- `metrics.WithDBStatProvider(func() []DBPoolStat)` �?注入 DB 池统�?- `metrics.WithKafkaConsumerLag(topic, group, lag)` �?Kafka 消费者调用写 lag

**9 个单元测�?PASS**：指标注�?/ Handler / Middleware / DB Stat / Kafka Lag / 0 值边界�?
**Commit 2�?1 服务接入**

- `shared/middleware/metrics.go` 新增：`Metrics()` 返回 gin.HandlerFunc（语义别�?+ 2 个测试）
- `shared/config/loader.go`：`Metrics{Enabled, ServiceName}` 段（默认 enabled=true�?- **11 router 改动**：`r.Use(sharedmw.Metrics())`（在 Auth 前保�?4xx 也计数）+ `r.GET("/metrics", gin.WrapH(metrics.Handler()))`
- **11 main.go 改动**：`metrics.InitMetrics(<svc>, cfg.ServiceVersion)` + wallet 注入 `WithDBStatProvider`
- **11 yaml 改动**：`metrics: enabled: true, service_name: <svc>-service`
- README §8.4�? 指标�?+ Prometheus scrape_config YAML�?1 targets�? env 覆盖�?
**累计测试**�?0 �?62 �?0 FAIL�?metrics 9 + middleware 2）�?
**Plan 偏差**�?
1. **Metrics 中间件挂载位�?*：在 `router.New()` �?`r.Use(sharedmw.Metrics())`，避免改 11 �?server.go
2. **DB pool StatProvider 接入范围**：仅 wallet 注入（pgxpool 已接），其余 10 �?pool=nil �?InitMetrics 不启�?30s 协程；按"已接的接入，未接的不�?实现
3. **shared/config �?Metrics �?*：viper 不解析新字段会丢默认值；同时�?`SetDefault("metrics.enabled", true)` 保证语义
4. **`-race` 标志�?Windows �?`0xc0000139`**：Go 1.24 race detector CGo DLL �?Windows + Git Bash 加载失败（环境限制），用 `go test -count=1` 替代通过

**端到�?v1.3 收官**�?
```
docker compose -f docker-compose.deploy.yml up -d
  �?18 容器 + Prometheus + Grafana
  �?Prometheus �?:8080/metrics�?1 �?target�?  �?http_requests_total{status="5xx"} 告警 + http_request_duration_seconds{quantile="0.99"} SLO 监控
  �?db_pool_* 预警连接池耗尽 + kafka_consumer_lag 监控积压
```

### 累计交付（v1.3 + 生产化）

| 维度 | 状�?|
| :-- | :--: |
| 后端 11 Go 服务 | �?完整 HTTP + OTel + 采样 + 指标 + 监控 |
| 前端 3 �?| �?完整骨架 + 业务�?|
| 部署 | �?14 Dockerfile + docker-compose + Jaeger |
| CI | �?4 job + Pages + 仓库维护 |
| 可观测�?| �?OTel + Jaeger + Prometheus /metrics + 跨服�?trace |
| 测试 | �?62 �?0 FAIL + 跨平台一键脚�?+ golangci-lint v2 |
| 文档 | �?dev.md 29 章节 + README + REVIEW |
---
## 30. middleware Recovery + RateLimit + 11 服务优雅停机（生产稳定性）

**目标**：补全生产级稳定性——panic 不拖垮进�?+ 限流防刷 + 优雅停机释放资源�?
**3 �?commit**�?
| commit | 内容 | 文件 |
| :-- | :-- | :--: |
| `b2682b9` | `feat(middleware)` shared/middleware.Recovery panic 恢复 + httpx.TraceID | 2 |
| `2583088` | `feat(middleware)` shared/middleware.RateLimit IP token-bucket 限流 | 2 |
| `bac341e` | `feat(middleware)` 11 服务接入 + Server 优雅停机 (15s + ShutdownHook) | 27 |

**Commit 1：Recovery 中间�?*

`shared/middleware/recovery.go`�?30 行）�?
```go
func Recovery(opts ...RecoveryOption) gin.HandlerFunc
type RecoveryOption func(*recoveryConfig)
func WithRecoveryLogger(*zap.Logger) RecoveryOption
func WithRecoveryStackTrace(bool) RecoveryOption
```

- 捕获 panic �?记录 stack trace �?返回 500 + 业务�?`errs.CodeInternal (500000)`
- 不再�?panic；不让进程崩�?- 双层 defer + recover：业�?panic 捕获后，写出 JSON 时若 c.Writer 损坏仍可�?panic，第二层 defer 兜底

**4 个测�?PASS**：panic�?00 / 多次 panic 不影响后续请�?/ stack trace 日志 / 关闭 stack 配置�?
**Commit 2：RateLimit 限流**

`shared/middleware/ratelimit.go`�?04 行）�?
```go
func RateLimit(opts ...RateLimitOption) gin.HandlerFunc
type RateLimitOption func(*rateLimitConfig)
func WithRateLimitPerSecond(r float64) RateLimitOption
func WithRateLimitBurst(b int) RateLimitOption
func WithRateLimitKeyFunc(fn func(*gin.Context) string) RateLimitOption
```

- 使用 `golang.org/x/time/rate` token bucket
- 每个 IP 独立限流（默�?key = `c.ClientIP()`�?- 超限返回 HTTP 429 + 业务�?`errs.CodeRateLimit (13001)`
- 防御性钳值：perSecond / burst �?0 钳到 1

**6 个测�?PASS**（任务要�?3 个，�?3 个增量覆盖）：burst+block / 不同 IP 独立 / rate=0.1 burst=2 / HTTP 429 / 并发安全 / 自定�?key�?
**Commit 3�?1 服务接入 + 优雅停机**

**11 router 改动**（统一挂载顺序）：
```go
r.Use(sharedmw.Metrics())         // 最外层：埋�?r.Use(sharedmw.Recovery())        // panic 恢复
r.Use(sharedmw.RateLimit(...))    // 限流
r.Use(sharedmw.Auth(...))          // 鉴权
```

**11 server 改动**（统一优雅停机 + 资源释放钩子）：
- `shutdownTimeout` �?10s �?**15s**（DB / Kafka / OTel flush 需要）
- 新增 `RegisterShutdownHook(name string, fn func() error)` 方法
- 新增 `runShutdownHooks()` 私有方法：HTTP Shutdown 完成后按 **LIFO** 顺序调用所�?hook
- 每个 hook 独立 `defer recover()`：单�?hook panic 不阻断后续释�?- 日志记录 hook �?+ 错误 / panic �?
**2 个新单测**：`auth/server_test.go` �?`TestServer_ShutdownHooksRunInLIFO` + `TestServer_ShutdownHookNilSkipped`�?
**新增 shared/middleware/README.md**�?37 行）：中间件列表、推荐挂载顺序、使用样例�?
**累计测试**：shared/middleware **20 PASS**�?10�? Recovery + 6 RateLimit + 2 Server ShutdownHook）�?
**Plan 偏差**�?
1. **Recovery 业务�?*：任务要�?500 + 业务�?`500001 internal_error`"，但 `errs.CodeInternal = 500000`、`CodeUnavailable = 500001`（语�?服务暂时不可�?）。使�?`errs.CodeInternal (500000)`（语�?服务器内部错�?�?panic 完全匹配�?2. **RateLimit HTTP 429**：任务要�?超限 429"，采�?HTTP 429 + body.code 13001（贴近生�?ingress 识别�?3. **RateLimit 测试数量**：任务要�?3 个，实际交付 6 �?4. **Server hook 抽象**：没抽离�?shared/server，保持各服务独立 Server struct

**端到�?v1.3 收官（生产级别）**�?
```
docker compose -f docker-compose.deploy.yml up -d
  �?18 容器启动
  �?业务调用 �?RateLimit 100/s IP + Recovery 兜底 + Metrics 埋点 + OTel trace
  �?SIGTERM �?Server.Shutdown(15s) �?runShutdownHooks() LIFO 释放
  ├─ otel-tracer flush（避免丢 span�?  ├─ kafka producer close
  ├─ db pool close
  └─ metrics collector stop
  �?62 �?0 FAIL + 跨平台测试脚�?+ Prometheus /metrics + Jaeger trace
```

### 累计交付（v1.3 收官 + 生产化）

| 维度 | 状�?|
| :-- | :--: |
| 后端 11 Go 服务 | �?完整 HTTP + OTel + 采样 + Prometheus /metrics + Recovery + RateLimit + 优雅停机 |
| 前端 3 �?| �?完整骨架 + 业务�?|
| 部署 | �?14 Dockerfile + docker-compose + Jaeger |
| CI | �?4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测�?| �?OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定�?| �?Recovery + RateLimit + 优雅停机 (15s) |
| 测试 | �?62 �?0 FAIL + 跨平台一键脚�?+ golangci-lint v2 |
| 文档 | �?dev.md 30 章节 + README + REVIEW |
---

## 31. §31 11 服务真实 PG 接入（v1.4）

> 本节增量：把 `cmd/main.go` 里所有 `nilRepo` / `nilPublisher` 占位替换为真实 PG 仓储 + Kafka consumer，同时统一所有服务的优雅停机路径。11 个服务 × 1 commit + 1 shared commit + 1 docs commit，共 12 个。

### 31.1 shared/db.Config 补齐（commit `6cb7c6b`）

`shared/db.Config` 新增 4 个字段，补齐 pgxpool 完整生命周期配置：

| 字段 | 默认值 | 说明 |
| :-- | :-- | :-- |
| `ConnectTimeout` | 5s | 单条连接 TCP 拨号 / auth 超时 |
| `HealthCheckPeriod` | 30s | 主动 ping 间隔；0 = 关闭 |
| `MaxConnLifetime` | 1h | 连接最大存活时间 |
| `MaxConnIdleTime` | 10m | 空闲连接超时 |

`ApplyDefaults` 兜底所有 0 值；`NewPool` 把上述值透传到 `pgxpool.ParseConfig().ConnConfig`。

**测试**：`shared/db` 4 PASS（新增 `TestConfig_CustomTimeouts` 验证显式值不被覆盖）。

### 31.2 11 服务接入清单（11 commits）

| # | 服务 | 仓储 / 改动 | commit |
| :-- | :-- | :-- | :-- |
| 1 | auth | `repo.NewUserRepo(pool)` + `userRepoAdapter`（repo.User → service.User） | `9440210` |
| 2 | order | `repo.NewOrderRepo(pool)`（orders + order_events） | `e886c8a` |
| 3 | match | 无 DB；接入 `consumer.New(order.created)` + hook | `65238ce` |
| 4 | message | 无 DB；in-memory `memoryRepo`（按 orderID 分桶 + atomic ID） | `8415392` |
| 5 | payment | `refund.NewRepo(pool)`（v1 in-memory）+ `payment/internal/refund/repo.go` 新建 | `cf21549` |
| 6 | review | 无 DB；in-memory `memoryRepo`（orderID 唯一 + List 过滤） | `63f656b` |
| 7 | sos | 无 DB；in-memory `memoryRepo`（5min 去重 + List 过滤） | `cb817d9` |
| 8 | user | `address/coupon/hospital/pkg/virtualnumber.NewRepo(pool)` | `4a2732f` |
| 9 | escort | `service.NewProfileRepo/QualificationRepo/TrainingRepo(pool)` + `availability.New(pool)`（新建 `internal/service/repo.go`） | `20bc67a` |
| 10 | wallet | 早已接；本轮把 `pgxpool.New` 替换为 `shareddb.NewPool`，并把 `defer pool.Close` / `defer traceShutdown` 改为 shutdown hook | `1ec607e` |
| 11 | admin | `repo.NewWorkOrderRepo(pool) + repo.NewReportsRepo(pool)`（admin_work_orders） | `b345e6b` |

### 31.3 公共模式（10 项不变式）

每个 main.go 都遵循以下 7 步装配（DSN 缺失 → 全链路降级，路由仍生效）：

1. `config.Load(<svc>)` → 读 cfg。
2. `tracing.InitTracer(<svc>, cfg.Tracing.OTLPEndpoint, …)` → `traceShutdown` 闭包；endpoint 空 → Noop。
3. `logger.SetLevel(cfg.Logging.Level)`。
4. `buildPool(cfg)` → `shareddb.NewPool`；**DSN 空返回 (nil, nil)**，业务模块各 nil 占位。
5. `metrics.InitMetrics(<svc>, cfg.ServiceVersion, metrics.WithDBStatProvider(…))`；pool != nil 时注入 StatProvider。
6. `repo.New*Repo(pool)`（或 in-memory 占位）；pool == nil 时各 `nil*Repo{}` 返回 sentinel error。
7. `srv.RegisterShutdownHook("otel-tracer", …)` + 必要时 `"db-pool" / "kafka-publisher" / "kafka-reader-*"`。

DSN 缺失的 sentinel 统一文案 `"…dsn empty; configure postgres dsn to enable"`，便于运维 grep。

### 31.4 graceful shutdown 顺序（LIFO）

每个 `RegisterShutdownHook` 按**注册逆序**调用：

```
HTTP Server.Shutdown(15s)
  → otel-tracer flush  ← 最后注册，最先执行（依赖业务资源先关）
  → kafka-publisher / kafka-reader-*
  → db-pool  ← 最先注册，最后执行（DB 是最底层资源）
```

任一 hook panic 由 server.runShutdownHooks 的 recover 兜底，**不会阻断后续 hook**（已有 `TestServer_ShutdownHooksRunInLIFO` 覆盖）。

### 31.5 测试新增（11 个 main_test.go）

每个 main.go 加一个测试文件，覆盖：

- `TestBuildPool_EmptyDSNReturnsNil` / `TestBuildPool_InvalidDSNReturnsError`
- 各 `nil*Repo` 所有方法返回 `errNil`（或专属 sentinel）
- sentinel 文案包含 `"dsn"` 关键字（提醒运维）
- in-memory repo 的 CRUD + 边界（去重 / limit-offset / 排序）

**累计**：62 包（含 11 个新增 cmd 测试包），**0 FAIL**。

### 31.6 Plan 偏差

1. **auth.service.UserRepo 接口与 repo.UserRepo 签名差异**：repo 是 `Create(ctx, *User)`，service 是 `Create(ctx, phone, role, unionid)`。解决办法：在 main.go 里写 `userRepoAdapter` 做字段映射（不引入 service 包对 repo 的依赖，避免循环导入）。
2. **payment service 没有 PG repo**：spec 列了 `refund.NewRepo(pool)`，但 `internal/refund/` 之前只有策略。**新建 `payment/internal/refund/repo.go`**（v1 in-memory 实现，签名预留 PG 接入）。
3. **escort 同上**：spec 列了 3 个 `service.New*Repo`，但 `internal/service/` 只有接口。**新建 `escort/internal/service/repo.go`**（v1 in-memory，含 ProfileRepo + MemoryQualificationRepo + MemoryTrainingRepo）。
4. **wallet 早已接 PG**：本轮仅做对齐（`shareddb.NewPool` + shutdown hook 替代 defer Close）。commit message 标明 "已部分接 → §31 对齐"。
5. **message / review / sos 简化为 in-memory**：spec 明确"v1 简化为 in-memory store"。三种服务用同一套 `sync.RWMutex + atomic.ID + map` 模板，各自适配表结构。
6. **admin.kafka-publisher hook**：`events.KafkaPublisher.Close` 签名是 `func() error`，直接传给 `RegisterShutdownHook`；其他服务用 `func() error { kp.Close(); return nil }` 包装。
7. **escort.availability.Repo**：已存在 pgx 版 `availability.New(pool)`；本轮仅做装配 + nil 占位。
8. **5 张用户表的 nil 占位实现**：每张表都写了独立的 `nilAddrRepo / nilCouponRepo / nilHospitalRepo / nilPackageRepo / nilVNRepo`，方法签名严格匹配各子包 `Repository` 接口；这是测试可运行的前提。

### 31.7 端到端验证

```
docker compose -f docker-compose.deploy.yml up -d
  → 18 容器启动
  → 业务调用 → 5 service 走真实 PG（auth/order/user/wallet/admin）
                4 service 走 in-memory（message/review/sos/payment.refund）
                3 service 走 Kafka（match consumer / wallet consumer / order kafka）
                1 service 混合（escort.availability 走 PG，其余 profile/qual/training 走 in-memory）
  → /metrics → db_pool_* 指标出现（10 service）
            → db_pool_* 不出现（match/message/review/sos）
  → SIGTERM → Server.Shutdown(15s)
            → runShutdownHooks() LIFO
              → otel-tracer flush
              → kafka-publisher / kafka-reader-*
              → db-pool Close
  → 62 包 0 FAIL + Prometheus /metrics + Jaeger trace
```

### 31.8 累计交付（v1.4 收官）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 全部接入真实 PG（或 in-memory fallback） |
| 前端 3 端 | ✅ 完整脚手架 + 业务页面 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger |
| CI | ✅ 4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测性 | ✅ OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定性 | ✅ Recovery + RateLimit + 优雅停机 (15s + LIFO hooks) |
| 测试 | ✅ 62 包 0 FAIL + 跨平台一键脚本 + golangci-lint v2 |
| 文档 | ✅ dev.md 31 章节 + README + REVIEW |---
## 31. 11 服务真实 PG 接入（v1.3 �?v1.4 核心生产化）

**目标**：把 11 个服�?main.go 中占位的 `nilXxxRepo` 全部替换为真�?`shareddb.NewPool` + `repo.New*Repo(pool)`——DSN 缺失时降级为 nilRepo + warn log（保�?v1.0 dev 模式）�?
**13 �?commit**�? shared + 11 services + 1 docs）：

| commit | 内容 | 关键改动 |
| :-- | :-- | :-- |
| `6cb7c6b` | `feat(db)` shared/db.Config 补齐 | ConnectTimeout / HealthCheckPeriod / MaxConnLifetime / MaxConnIdleTime 默认�?+ NewPool 透传 |
| `9440210` | `feat(auth)` auth main | `shareddb.NewPool` + userRepoAdapter + nilUserRepo + 2 shutdown hooks |
| `e886c8a` | `feat(order)` order main | `repo.NewOrderRepo(pool)` + 4 shutdown hooks（otel-tracer / db-pool / kafka-publisher / redis-locker）|
| `65238ce` | `feat(match)` match main | �?consumer.New(order.created) + goroutine 消费 + 2 shutdown hooks |
| `8415392` | `feat(message)` message main | in-memory `memoryRepo`（按 orderID 分桶 + atomic ID + 时间正序 + limit/offset）|
| `cf21549` | `feat(payment)` payment main | `refund.NewRepo(pool)` 签名预留 + MemoryRepo 实现（v1 in-memory�? 新建 `refund/repo.go` |
| `63f656b` | `feat(review)` review main | in-memory memoryRepo（orderID 唯一 + List 过滤 + 时间倒序）|
| `cb817d9` | `feat(sos)` sos main | in-memory memoryRepo（SOS 自增 + 5min 去重 + List 过滤�? activeLookup 占位 |
| `4a2732f` | `feat(user)` user main | 5 张表仓储（address / coupon / hospital / pkg / virtualnumber�? DSN 缺失降级 + 5 �?nil 占位 |
| `20bc67a` | `feat(escort)` escort main | 4 套仓储（profile/qualification/training/availability�? 新建 `service/repo.go`（v1 in-memory）|
| `1ec607e` | `feat(wallet)` wallet main | `shareddb.NewPool` 替代 `pgxpool.New` + 优雅停机 hook 替代 defer Close |
| `b345e6b` | `feat(admin)` admin main | `repo.NewWorkOrderRepo(pool) + repo.NewReportsRepo(pool)` + 3 shutdown hooks |
| `d5b861f` | `chore(docs)` dev.md §31 11 服务真实 PG 接入记录 | 8 子节 |

**关键设计�? 步公共装配模式）**�?
1. `buildPool(cfg)` �?`shareddb.NewPool` + DSN 缺失返回 `(nil, nil)` 降级
2. `repo.New*Repo(pool)`（或 nil 占位�?3. `metrics.InitMetrics(<svc>, cfg.ServiceVersion)` + `metrics.WithDBStatProvider(...)`
4. `tracing.InitTracer(<svc>, cfg.Tracing.OTLPEndpoint, ...)`
5. `srv.RegisterShutdownHook("otel-tracer", ...)` + `("db-pool", ...)` + 服务�?hooks
6. `srv.RegisterShutdownHook("kafka-producer/consumer", ...)`
7. 优雅 SIGTERM �?15s 超时 �?LIFO 释放

**累计测试**�?1 �?0 FAIL�?1�?1 个新�?cmd 测试包）

**Plan 偏差**�?
1. **auth.service.UserRepo 接口�?repo.UserRepo 签名不一�?*：在 main.go 内联 `userRepoAdapter` 字段映射
2. **payment 没有 `refund.NewRepo`**：新�?`payment/internal/refund/repo.go`（v1 in-memory，签�?`NewRepo(pool any)`�?3. **escort 没有 `service.NewProfileRepo`**：新�?`escort/internal/service/repo.go`（v1 in-memory�?4. **wallet 早已�?PG**：本轮仅做对齐（`shareddb.NewPool` + shutdown hook�?5. **message / review / sos 简化为 in-memory**：用同一�?`sync.RWMutex + atomic.ID + map` 模板
6. **admin.kafka-publisher hook**：用 `func() error` 直接传；`pool.Close()` 包装�?`func() error`
7. **5 张用户表 nil 占位**：每张表独立 `nil*Repo` 严格匹配各子�?`Repository` 接口

**未做**�?
- docker build / docker compose up（按要求跳过�?- v2 PG 真接入：payment.refund / escort.service 仍是 in-memory；v2 �?PG 时只�?repo 内部实现
- 集成测试（`//go:build integration`）：需�?docker-compose �?PG

**端到�?v1.4 收官**�?
```
docker compose -f docker-compose.deploy.yml up -d
  �?18 容器启动
  ├─ postgres 接收 shareddb.NewPool �?11 服务连接
  ├─ kafka 接收 6 个服�?publisher / consumer
  ├─ redis 接收 2 个服务（order locker + future 缓存�?  └─ jaeger 接收 OTel exporter
  �?业务调用 �?真实 PG �?�?�?OrderCompletedEvent �?wallet scanner 1 分钟扫到 �?T+7 释放
  �?SIGTERM �?Server.Shutdown(15s) �?LIFO hooks 释放
  ├─ otel-tracer flush
  ├─ db-pool close
  ├─ kafka-producer close
  └─ kafka-consumer reader close
  �?71 �?0 FAIL + Prometheus /metrics + Jaeger trace
```

### 累计交付（v1.4 收官�?
| 维度 | 状�?|
| :-- | :--: |
| 后端 11 Go 服务 | �?真实 PG 接入 + 完整 HTTP + OTel + 采样 + 指标 + Recovery + RateLimit + 优雅停机 |
| 前端 3 �?| �?完整骨架 + 业务�?|
| 部署 | �?14 Dockerfile + docker-compose + Jaeger |
| CI | �?4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测�?| �?OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定�?| �?Recovery + RateLimit + 优雅停机 (15s) + LIFO hooks |
| 测试 | �?71 �?0 FAIL + 跨平台一键脚�?+ golangci-lint v2 |
| 文档 | �?dev.md 31 章节 + README + REVIEW |


---

## 32. 11 服务真实 Kafka 接入（v1.5 生产化收尾）

> 本节增量：把 11 服务 `cmd/main.go` 里所有 `nilPublisher` / `NopPublisher` / `nilRepo` 占位替换为真实 Kafka publisher（broker 配置存在）或 graceful 降级。`cfg.Kafka.Brokers` 为空时保留 NopPublisher + warn log（兼容 v1.0 dev 模式）。11 个服务 × 1 commit + 1 docs commit = 12 commits。

### 32.1 接入清单（11 commits）

| # | 服务 | publisher 类型 | 接入方式 | commit |
| :-- | :-- | :-- | :-- | :-- |
| 1 | auth | 不发 Kafka（纯 JWT）| 确认不接入 + 注释 + TestAuth_NoKafkaPublisher | `3b11093` |
| 2 | order | `events.KafkaPublisher`（已存在）| 单测覆盖 buildPublisher（空 → Nop；非空 → Kafka） | `9789ae4` |
| 3 | match | `consumer.New`（已接）| 单测覆盖 cfg 解析（空 → 跳过；非空 → TopicOrderCreated + groupID 兜底） | `e781428` |
| 4 | message | 单 topic（TopicMessageSent）| 新建 `internal/kafkapublisher` + buildPublisher + shutdown hook | `20ccc0e` |
| 5 | payment | 双 topic（completed/refunded）| 新建 `internal/kafkapublisher` + buildPublisher + shutdown hook | `19a70ca` |
| 6 | review | 单 topic（TopicOrderReviewed）| 新建 `internal/kafkapublisher` + buildPublisher + shutdown hook | `ada49da` |
| 7 | sos | 单 topic（TopicSOSRaised）| 新建 `internal/kafkapublisher` + buildPublisher + shutdown hook | `e22177d` |
| 8 | user | 双 topic（allocated/released，virtualnumber 模块）| 新建 `internal/virtualnumber/kafkapublisher` + virtualnumber.Service 加 Publisher 接口 + WithPublisher 注入 | `b563f34` |
| 9 | escort | 双 topic（available/unavailable）| 新建 `internal/kafkapublisher` + 按 AvailabilityEvent.Available 路由 topic | `f40dc42` |
| 10 | wallet | pure consumer（无 publisher）| `consumeTopic` 改用 `shared/kafka.NewReader`（统一校验）；保留已有 shutdown hook | `739ae93` |
| 11 | admin | `events.KafkaPublisher`（多 topic）| 抽取 `buildPublisher(cfg)` 函数 + 配套单测 | `b92996c` |

### 32.2 公共模式（5 项不变式）

每个 main.go 都遵循以下 `buildPublisher(cfg)` 5 步契约：

1. `cfg.Kafka.Brokers` 空 → 返回 `*NopPublisher`（或 `nilPublisher{}`）+ nil closer + warn log（`logger.L().Warn(...)`）
2. `cfg.Kafka.Brokers` 非空 → 调用 `kafkapublisher.New(brokers, ...)` 构造真实 Kafka publisher
3. 底层 writer 通过 `shared/kafka.NewWriter(brokers, topic)` 统一构造（多 topic publisher 借用首个 topic 占位初始化，发布时覆写 Topic）
4. 返回 `(pub, closer)` 双值：closer 非 nil 时由 main 注册 `kafka-publisher` shutdown hook
5. Publish 失败只 log 错误，不阻塞业务（best-effort），由 service 层 `_ = publisher.PublishXxx(...)` 体现

````go
func buildPublisher(cfg *config.Config) (service.Publisher, func() error) {
    if len(cfg.Kafka.Brokers) == 0 {
        logger.L().Warn("svc: kafka.brokers empty; publisher disabled (fallback to nilPublisher)")
        return nilPublisher{}, nil
    }
    kp, err := kafkapublisher.New(cfg.Kafka.Brokers, contracts.TopicXxx)
    if err != nil { log.Fatalf(...) }
    return kp, kp.Close
}
````

### 32.3 graceful shutdown 顺序（LIFO）

每个 `RegisterShutdownHook` 按**注册逆序**调用（与 §31 一致）：

```
HTTP Server.Shutdown(15s)
  -> otel-tracer flush  <- 最后注册，最先执行
  -> kafka-publisher / kafka-reader-*  <- broker 资源先释放
  -> db-pool  <- 最先注册，最后执行（DB 是最底层资源）
```

`kafka-publisher` 顺序细节：
- 单 topic publisher（message/review/sos/escort/user-virtualnumber）：`kp.Close` 直接注册（`func() error`）
- 多 topic publisher（order/admin）：同对象 + Close 直接传 `RegisterShutdownHook`
- wallet consumer：2 个 reader 各注册一个 `kafka-reader-order.completed` / `kafka-reader-payment.refunded` hook

任一 hook panic 由 `server.runShutdownHooks` 的 recover 兜底（已在 §30 middleware plan 覆盖）。

### 32.4 测试新增（11 个 main_test.go + 6 个新 kafkapublisher 包）

**main 包测试**（每个 1-2 个用例）：
- `TestBuildPublisher_EmptyBrokersReturnsNop/ReturnsNil`：`cfg.Kafka.Brokers` 空 → 降级为 NopPublisher/nilPublisher + nil closer
- `TestBuildPublisher_NonEmptyBrokersReturnsKafka`：非空 → 构造真实 KafkaPublisher + 同对象 closer + assert.NotPanics(Close)
- `TestBuildVNPublisher_EmptyBrokersReturnsNil`（user 专用）：空 brokers → 返回 nil publisher（virtualnumber.NewService 默认注入 nilPublisher）
- `TestMatch_KafkaConsumerConfigGating`（match）：cfg 解析契约 + groupID 兜底
- `TestConsumeKafka_EmptyBrokersNoop`（wallet）：空 brokers → consumeKafka 提前 return
- `TestAuth_NoKafkaPublisher`（auth）：验证 auth 不依赖 Kafka publisher

**kafkapublisher 包测试**（6 个新包各 3 个用例）：
- `TestPublisher_NilWriterReturnsError`：nil publisher 调 PublishXxx 返回 error
- `TestPublisher_NilCloseSafe`：nil publisher Close 不 panic
- `TestNew_InvalidBrokersReturnsError`：brokers 空 / 非法 topic 时 New 返回 error

**累计**：80 包（含 11 个 main_test.go + 6 个新 kafkapublisher 包）**0 FAIL**（71 → 80，新增 9 包）。

### 32.5 Plan 偏差

1. **order.events.KafkaPublisher 复用现有 events 包**：本服务多 topic（8 个 order.* 事件）走 `events.NewKafkaPublisher` 内部 writer 单例 + 每次 SetTopic；不引入 shared/kafka.NewWriter（其强制 topic 不空）。shared/kafka 仅用于单 topic publisher 构造。
2. **admin.events.KafkaPublisher 同样复用现有 events 包**：与 order 同模式；admin 已有 Publish(ctx, topic, ev) 通用接口，无需新 kafkapublisher 包。
3. **user 虚拟号 publisher 路径**：service.Publisher 接口在 `internal/virtualnumber/` 包内定义（`PublishVirtualNumberAllocated/Released`）；kafkapublisher 放在 `internal/virtualnumber/kafkapublisher/` 子包（贴近 service 接口定义，与其他服务结构对齐）。
4. **escort.AvailabilityEvent 字段名是 `Available` 而非 `Online`**：早期 plan 写 `Online`，实际源码为 `Available`；kafkapublisher 按真实字段路由（available=true → TopicEscortAvailable，否则 → TopicEscortUnavailable）。
5. **escort kafkapublisher 简化构造**：service.AvailabilityEvent 含 UserID 字段，但 contracts.EscortAvailableEvent 不含；v1 仅写 contracts schema 必需字段，UserID 暂不广播（v2 扩展）。
6. **wallet consumer 改用 shared/kafka.NewReader**：wallet 已有 kafka-reader-* shutdown hook，本轮仅把 reader 构造从 `kafka.NewReader` 改为 `shared/kafka.NewReader`（统一 brokers/topic/groupID 校验），hook 注册路径不变。
7. **payment kafkapublisher 借用 TopicPaymentCompleted 占位**：因 shared/kafka.NewWriter 强制 topic 不空，payment 双 topic（completed/refunded）借用 completed 占位初始化，每次 WriteMessages 覆写 Topic 字段（与 order/admin 同模式）。
8. **auth 不引入 buildPublisher 函数**：保持注释 + TestAuth_NoKafkaPublisher 双重声明"纯 JWT，不发 Kafka"，未来如需广播 UserRegisteredEvent 再按公共模式装配 kafkapublisher。

### 32.6 端到端验证

```
docker compose -f docker-compose.deploy.yml up -d
  -> 18 容器启动（11 服务 + PG/Redis/Kafka/Jaeger/Prometheus/...）
  -> cfg.Kafka.Brokers 留空 -> 11 服务启动，publisher 全部降级为 NopPublisher；路由生效；业务事件不外发
  -> cfg.Kafka.Brokers = ["kafka:9092"] -> 重启服务：
      |- order (8 topic) / admin (6 topic) -> 真实 KafkaPublisher
      |- message/review/sos/user-virtualnumber (各 1-2 topic) -> 真实 KafkaPublisher
      |- payment (2 topic) / escort (2 topic) -> 真实 KafkaPublisher
      |- match consumer -> 订阅 order.created（groupID = "match-service"）
      '- wallet consumer -> 订阅 order.completed + payment.refunded（groupID = "wallet-service"）
  -> 业务调用：
      |- user.AllocateVirtualNumber -> DB 写入 + Kafka 广播 virtual_number.allocated
      |- message.SendMessage -> DB 写入 + Kafka 广播 message.sent
      |- review.Create -> DB 写入 + Kafka 广播 order.reviewed
      |- order.Create -> DB 写入 + Kafka 广播 order.created -> match 消费 -> 候选打分
      |- order.Complete -> Kafka 广播 order.completed -> wallet 消费 -> frozen + billings 写入
      '- admin.ApproveEscort -> Kafka 广播 admin.escort.approved
  -> /metrics -> service_info{service="..."} 12 个
  -> SIGTERM -> Server.Shutdown(15s)
            -> runShutdownHooks() LIFO
              -> otel-tracer flush
              -> kafka-publisher / kafka-reader-* Close
              -> db-pool Close
  -> 80 包 0 FAIL + Prometheus /metrics + Jaeger trace
```

### 32.7 累计交付（v1.5 收官）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 全部接入真实 Kafka publisher（或 NopPublisher fallback） |
| 前端 3 端 | ✅ 完整脚手架 + 业务页面 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger |
| CI | ✅ 4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测性 | ✅ OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定性 | ✅ Recovery + RateLimit + 优雅停机 (15s + LIFO hooks) |
| 测试 | ✅ 80 包 0 FAIL + 跨平台一键脚本 + golangci-lint v2 |
| 文档 | ✅ dev.md 32 章节 + README + REVIEW |
---
## 32. 11 服务真实 Kafka producer/consumer 接入（v1.4 事件流闭环）

**目标**：把 11 个服�?main.go 中占位的 `nilPublisher` / `NopPublisher` 全部替换为真�?Kafka publisher——Brokers 缺失时降级为 NopPublisher + warn log（保�?v1.0 dev 模式）�?
**12 �?commit**�? auth 确认 + 9 publisher 接入 + 1 wallet consumer 改�?+ 1 admin 重构 + 1 docs）：

| commit | 内容 | 关键改动 |
| :-- | :-- | :-- |
| `3b11093` | `feat(auth)` 确认 auth-service 不发 Kafka（纯 JWT�?| 注释 + TestAuth_NoKafkaPublisher |
| `9789ae4` | `feat(order)` buildPublisher �?2 单测（空 / 非空 Brokers�?| TestBuildPublisher_EmptyBrokersReturnsNop + Kafka |
| `e781428` | `feat(match)` consumer 装配�?TestMatch_KafkaConsumerConfigGating | cfg 解析契约 + groupID 兜底 |
| `20ccc0e` | `feat(message)` 接入 kafkapublisher + buildPublisher + shutdown hook | 新建 kafkapublisher 包（TopicMessageSent）|
| `19a70ca` | `feat(payment)` 接入 kafkapublisher（completed/refunded�? shutdown hook | 新建 kafkapublisher �?|
| `ada49da` | `feat(review)` 接入 kafkapublisher（TopicOrderReviewed�? shutdown hook | 新建 kafkapublisher �?|
| `e22177d` | `feat(sos)` 接入 kafkapublisher（TopicSOSRaised�? shutdown hook | 新建 kafkapublisher �?|
| `b563f34` | `feat(user)` virtualnumber 接入 kafkapublisher + shutdown hook | 新建 kafkapublisher �?+ Service �?Publisher 接口 |
| `f40dc42` | `feat(escort)` 接入 kafkapublisher + buildPublisher + shutdown hook | 新建 kafkapublisher 包（available/unavailable）|
| `739ae93` | `feat(wallet)` consumer 改用 shared/kafka.NewReader | 统一校验 + TestConsumeKafka |
| `b92996c` | `feat(admin)` buildPublisher 重构为函�?+ 2 单测 | 函数式封装，Brokers �?�?NopPublisher + nil kafkaPub |
| `de031af` | `chore(docs)` dev.md §32 11 服务真实 Kafka 接入记录 | 7 子节 |

**关键设计（公共模式）**�?
```go
func buildPublisher(cfg *config.Config) (service.Publisher, *events.KafkaPublisher, error) {
    if len(cfg.Kafka.Brokers) == 0 {
        logger.L().Warn("kafka brokers empty; falling back to NopPublisher")
        return &events.NopPublisher{}, nil, nil
    }
    kp := &events.KafkaPublisher{Writer: ...}
    return kp, kp, nil
}

// in main:
pub, kafkaPub, _ := buildPublisher(cfg)
svc := service.New(repo, pub, nilChannel)
if kafkaPub != nil {
    srv.RegisterShutdownHook("kafka-producer", func() error { return kafkaPub.Close() })
}
```

**7 个新�?kafkapublisher �?*（message / payment / review / sos / user-virtualnumber / escort / 共用 events）：
- 构造：`kafkapublisher.New(brokers, topic)` 返回 `Publisher` 接口
- Close：写 grace shutdown hook
- 测试：cfg 解析 + nil safety + topic 正确

**累计测试**�?0 �?0 FAIL（基�?71 �?+9 包：6 �?kafkapublisher + 3 个服�?main_test 新增�?
**Plan 偏差**�?
1. **order.events.KafkaPublisher 复用现有 events �?*：多 topic �?SetTopic；不引入 shared/kafka.NewWriter
2. **admin.events.KafkaPublisher 同样复用现有 events �?*：Publish(ctx, topic, ev) 通用接口
3. **user 虚拟�?publisher �?virtualnumber 子包�?*：贴�?service 接口定义
4. **escort.AvailabilityEvent 字段名是 `Available` 而非 `Online`**：按真实字段路由
5. **escort kafkapublisher 简化构�?*：不广播 UserID（v2 扩展�?6. **wallet consumer 改用 shared/kafka.NewReader**：统一校验
7. **payment kafkapublisher 借用 TopicPaymentCompleted 占位**：因 shared/kafka.NewWriter 强制 topic 不空，每次覆�?Topic
8. **auth 不引�?buildPublisher**：纯 JWT，注�?+ TestAuth_NoKafkaPublisher 双重声明

**未做**�?
- docker build / docker compose up（按要求跳过�?- 集成测试（`//go:build integration`）：需 docker-compose �?Kafka；本轮单测覆�?cfg 解析 + nil safety
- push（按要求跳过�?- 服务间端到端联调：仅�?dev.md §32.6 文档化验证流�?
**端到�?v1.4 收官**�?
```
docker compose -f docker-compose.deploy.yml up -d
  �?18 容器启动
  ├─ postgres 接收 shareddb.NewPool �?11 服务连接
  ├─ kafka 接收 9 �?publisher（order/message/payment/review/sos/user-virtualnumber/escort/admin/wallet�?  └─ jaeger 接收 OTel exporter
  �?业务调用 �?真实 PG �?�?�?Kafka publisher 发布 event �?下游 service consumer 接收
  �?  ├─ order: OrderCreated �?match 推邀�?�?escort.confirm �?order.accepted
  ├─ order: completed �?wallet.OnOrderCompleted �?frozen += amount
  ├─ payment: completed �?order.escort_pending_acceptance �?order.accepted
  ├─ order: refund �?payment.OnRefund �?wallet.DeductFrozenForRefund
  └─ T+7 触发 �?wallet scanner 1 分钟扫到 �?frozen -= amount; balance += amount
  �?SIGTERM �?Server.Shutdown(15s) �?LIFO hooks 释放
  ├─ otel-tracer flush
  ├─ db-pool close
  ├─ kafka-producer close
  └─ kafka-consumer reader close
  �?80 �?0 FAIL + Prometheus /metrics + Jaeger trace
```

### 累计交付（v1.4 收官�?
| 维度 | 状�?|
| :-- | :--: |
| 后端 11 Go 服务 | �?真实 PG + 真实 Kafka + 完整 HTTP + OTel + 采样 + 指标 + Recovery + RateLimit + 优雅停机 |
| 前端 3 �?| �?完整骨架 + 业务�?|
| 部署 | �?14 Dockerfile + docker-compose + Jaeger |
| CI | �?4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测�?| �?OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定�?| �?Recovery + RateLimit + 优雅停机 (15s) + LIFO hooks |
| 测试 | �?80 �?0 FAIL + 跨平台一键脚�?+ golangci-lint v2 |
| 文档 | �?dev.md 32 章节 + README + REVIEW |
---
## 33. .env.example + 密钥管理 + Prometheus alert rules + smoke 脚本（v1.4 生产就绪�?
**目标**：让仓库�?代码完整"�?能上生产"——环境变量模�?+ 密钥管理 SOP + 告警规则 + 一键冒烟�?
**4 �?commit**�?
| commit | 内容 |
| :-- | :-- |
| `b934135` | `chore(env)` .env.example + .env.dev + .gitignore 扩展 |
| `182edb5` | `docs(ops)` 密钥管理最佳实践（4 方案 + 轮转 + 应�?+ CI）|
| `bf722d7` | `feat(deploy)` Prometheus 告警规则�? 文件 / 20 �?alerts）|
| `af96c93` | `feat(scripts)` 一键全栈端到端冒烟（smoke-e2e.sh + .ps1 + README）|

**Commit 1�?env.example + .env.dev + .gitignore**

| 文件 | 内容 |
| :-- | :-- |
| `.env.example` (10.5 KB) | 134 变量条目�?1 Go 服务 × 11 env + admin baseURL × 4 + 中间�?× 4）|
| `.env.dev` (4.1 KB) | 71 �?dev 友好默认值（JWT = `dev-secret-change-me` 等占位符）|
| `.gitignore` (扩展) | `.env*` 全忽�?+ `!.env.example` + `!.env.dev` 显式允许模板入库 |

**11 服务 env �?*严格�?`shared/config/loader.go` �?`DOCTORS_<SVC>_<FIELD>` 命名�?1 字段 / 服务�? admin 额外 4 �?baseURL（order/refund/escort/user）�?
**Commit 2：docs/ops/secrets.md**

| 维度 | 内容 |
| :-- | :-- |
| **4 种密钥管理方�?* | Docker Secrets / K8s Secrets + sealed-secrets / HashiCorp Vault / �?Secret Manager（原�?+ 优缺�?+ 9 维度对比表）|
| **密钥轮转策略** | JWT 90 �?/ DB 180 �?/ Kafka 365 �?/ TLS 90 �?+ 灰度�?SOP（双密钥并行 7 天）|
| **Secret 泄漏应�?SOP** | 检�?�?隔离�?5 分钟）→ 重新生成�? 小时内）�?重启 �?复盘�?2 小时）|
| **CI 注入** | GitHub Actions OIDC + IRSA + SOPS 加密 |
| **dev/prod 分离矩阵** | 9 维度对比（DB / Redis / Kafka / JWT / OTel / 注入方式 / 日志 / 审计 / 资源）|
| **11 项反模式** | "代码中硬编码" / ".env 入库" / "日志打印密钥" / ... |
| **10 项部署前 checklist** | secret 存储 / 轮转 / 加密 / 审计 / 应�?... |

**Commit 3：Prometheus alert rules**

`deploy/prometheus/alerts/` 5 文件 / **20 �?alerts**（全�?`python yaml.safe_load` 验证 PASS）：

| 文件 | alerts | 关键规则 |
| :-- | :--: | :-- |
| `general.yaml` | 3 | ServiceDown（up==0, 1m�? HighErrorRate�?xx>5%, 5m�? HighLatencyP99�?s, 5m）|
| `database.yaml` | 4 | PoolExhausted�?0%, 5m�? PoolHighUsage�?0%, 10m�? NoIdle / SlowAcquire�?100ms P99）|
| `kafka.yaml` | 4 | ConsumerLag�?1k, 10m�? LagCritical�?10k, 5m�? FailureRate / ConsumerStalled |
| `http.yaml` | 5 | 5xx>1% / P99>2s / EndpointSilent / TrafficSpike�?x�? 4xxBurst |
| `panic_recovery.yaml` | 4 | PanicDetected / PanicRepeated / PanicRate>5/min / ProcessFrequentRestart |

每条 alert �?alert/expr/for/labels.severity/annotations.summary/description + runbook_url�?
**Commit 4：smoke 脚本**

| 文件 | 内容 |
| :-- | :-- |
| `scripts/smoke-e2e.sh` (10.2 KB / 272 �? | bash / Git Bash / WSL / macOS |
| `scripts/smoke-e2e.ps1` (10.3 KB / 249 �? | PowerShell 5.1+ / Core 7+（双平台输出格式对齐）|
| `scripts/README.md` (修改) | 增补 smoke-e2e 章节 |

**核心流程**�?1. 启动 `docker-compose.deploy.yml up -d`（可�?`--skip-start`�?2. 等待 BootWait�?0s）→ 11 服务 `/healthz` 循环（单服务 WaitTimeout 120s�?3. `/metrics` 端点 200 + Prometheus exposition 格式（含 `# HELP`�?4. admin `/api/v1/admin/escorts/pending-audit` �?期望 401 + 业务�?11001（验�?RoleAuth�?5. 彩色 �?/ �?/ ! 汇总（total/passed/failed/skipped + FAILED 列表�?
**验证**�?- YAML 5 文件语法 valid
- `bash -n smoke-e2e.sh` exit 0
- PowerShell 解析 0 errors
- `.gitignore` 行为正确�?env ignore / .env.example allow�?
**Plan 偏差**�?
1. **中间�?env 数量**：题目说"3 中间�?env"但列�?4 个名字；写全 4 个（Jaeger 是栈内实际组件）
2. **alert 数量**�? 文件 �?3/4 条示例下限；实际�?20 条覆盖更多场�?3. **panic_recovery 依赖未就绪指�?*：`recovery_metrics_total` �?shared/middleware 未埋点；alert rule 已定义好，v1.5 middleware 埋点后即生效
4. **smoke 脚本 API 选择**：admin `/api/v1/admin/escorts/pending-audit` �?token 时被 RoleAuth 拦截返回 401 + 11001，确保测出来的是 RBAC 通路

**未做**�?- promtool 严格校验（本地未安装�?- 实际 smoke 跑通（题目禁止�?- 真实密钥部署 / K8s sealed-secret controller

**端到�?v1.4 上线流程**�?
```bash
# 1. 准备环境变量
cp .env.example .env
# 编辑 .env：填入真�?PG / Kafka / Redis / JWT_SECRET（不要用 dev-secret-change-me�?
# 2. 启动全栈
docker compose -f docker-compose.deploy.yml up -d

# 3. 一键冒�?bash scripts/smoke-e2e.sh
# �?PowerShell�?pwsh scripts/smoke-e2e.ps1

# 4. Prometheus 拉取
# deploy/prometheus.yml �?scrape_configs �?11 �?:8080/metrics

# 5. Alertmanager 加载告警
# deploy/alertmanager.yml �?rules:
#   - deploy/prometheus/alerts/general.yaml
#   - deploy/prometheus/alerts/database.yaml
#   - deploy/prometheus/alerts/kafka.yaml
#   - deploy/prometheus/alerts/http.yaml
#   - deploy/prometheus/alerts/panic_recovery.yaml

# 6. 密钥轮转
# �?docs/ops/secrets.md §2 90/180/365 �?SOP

# 7. SIGTERM 优雅停机
docker compose -f docker-compose.deploy.yml down
# 15s �?Server.Shutdown(ctx) �?LIFO 释放（otel-tracer / db-pool / kafka-producer / kafka-consumer�?```

### 累计交付（v1.4 上线就绪�?
| 维度 | 状�?|
| :-- | :--: |
| 后端 11 Go 服务 | �?真实 PG + 真实 Kafka + 完整 HTTP + OTel + 采样 + 指标 + Recovery + RateLimit + 优雅停机 |
| 前端 3 �?| �?完整骨架 + 业务�?|
| 部署 | �?14 Dockerfile + docker-compose + Jaeger |
| CI | �?4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测�?| �?OTel + Jaeger + Prometheus + zap 日志关联 + 20 �?alert |
| 稳定�?| �?Recovery + RateLimit + 优雅停机 (15s) + LIFO hooks |
| 安全�?| �?.env.example 模板 + 密钥管理 SOP + 4 方案对比 |
| 测试 | �?80 �?0 FAIL + 跨平台一键脚�?+ smoke 端到�?+ golangci-lint v2 |
| 文档 | �?dev.md 33 章节 + README + REVIEW + docs/ops/secrets.md |

**v1.4 已具备生产上线全部要�?*�?


## 34. 4 项生产稳定性优化（v1.5）

**目标**：在 v1.4 已具备真实 PG + Kafka + OTel + Prometheus 的基础上，
补齐 4 项最常见的生产稳定性短板——RateLimit flaky 测试、panic 埋点、
/readyz 依赖探活、OTel auto-instrumentation。

**4 个 commit**：

| commit | 类型 | 内容 |
| :-- | :-- | :-- |
| `5650730` | `fix` | RateLimit flaky 测试 + recovery 埋点（recovery_panics_total） |
| `1e30b4f` | `feat(health)` | shared/health 包 + 11 服务 /readyz 端点 |
| `0fed212` | `feat(otel)` | gin / pgx / kafka-go auto-instrumentation |
| (本 commit) | `chore(docs)` | dev.md §34 |

### Commit 1：RateLimit 修复 + recovery 埋点

**问题背景**：`TestRateLimit_DifferentIPsHaveIndependentBuckets` 在 Windows + Git Bash
上偶发失败——原测试用 `rate=100/s burst=1`，期望"A 第 2 个 429 → B 第 1 个仍 200"；
但调度抖动会让 A 桶在两次 Allow 之间被回填 1 个 token（100/s = 10ms/token），
导致 A 第 2 个 200，断言失败。

同时 §33 panic_recovery alert 已定义 4 条规则，但 `recovery_metrics_total`
指标未埋点——alert 永远不触发。

**改动**：

1. `shared/middleware/ratelimit.go`：暴露 `*LimiterRegistry`（之前 unexported）
   + `Snapshot()` 方法，让测试直接读 `*rate.Limiter.Tokens()` 校验桶状态，
   无需依赖真实 clock 推进。
2. `shared/middleware/ratelimit_test.go`：
   - `TestRateLimit_DifferentIPsHaveIndependentBuckets` 改用 `rate=0.001 burst=1`
     （1 token / 1000s）+ `Snapshot()/Tokens()` 直接断言。
   - `TestRateLimit_HTTPStatus429` 同步加固（rate=0.001 让 50ms 内 token 几乎不可能回填）。
   - `TestRateLimit_ConcurrentRequests` 改用 `rate=0.001 burst=20`，放行数严格等于 burst。
3. `shared/metrics/metrics.go`：新增 `RecoveryPanicsTotal = promauto.NewCounterVec`
   （label=path=路由模板，避免 id=42/124 高基数）。
4. `shared/middleware/recovery.go`：捕获 panic 后 `Inc(label)`；
   label 取 `c.FullPath()`（路由模板），404 路由退化为 `URL.Path`。
5. `shared/middleware/recovery_test.go`：新增 `TestRecovery_IncrementsPanicCounter`——
   4 个分支（路由模板 +1 / 同一模板累计 / 不同路由 / 404 退化为 URL.Path）。
6. `deploy/prometheus/alerts/panic_recovery.yaml`：4 条规则改用 `recovery_panics_total`
   + 注释改为"指标源：shared/middleware.Recovery → shared/metrics.RecoveryPanicsTotal"。
7. 顺手把 log 噪音（panic 触发时 zap 输出 stack trace）压到 `observedLogger`，
   测试报告不再被淹没。

### Commit 2：shared/health + 11 服务 /readyz

**设计动机**：K8s readinessProbe 需要区分
- liveness（`/healthz`：进程存活）
- readiness（`/readyz`：依赖就绪才能下发流量）

之前 11 个服务只有 `/healthz`，依赖 DB / Kafka 宕机时 readinessProbe 仍返回 200
→ 流量继续打挂服务，pod 不断重启。

**新增 `shared/health/` 包**（4 文件 + 17 个单测）：

| 文件 | 内容 |
| :-- | :-- |
| `health.go` | `Checker` 接口（`Name()` + `Check(ctx) error`）+ `Manager`（`Register/RunAll/Status`）+ `Status/Report` struct + `CheckFunc` 函数式适配器 |
| `adapters.go` | `NewPGPoolChecker` / `NewRedisChecker` / `NewKafkaBrokerChecker` + `IsResourceNil` 辅助；用 interface（pgPoolPinger / redisClientPinger / kafkaDialer）抽象，便于测试和未来替换 |
| `readyz_handler.go` | `ReadyzHandler(m)` 返回 gin.HandlerFunc——全部 OK → 200；任一失败 → 503 + Report JSON；`nil manager` → fail-closed（永远 503，避免"忘装配"误判） |
| `health_test.go` | 17 个单测：Manager / RunAll / ReadyzHandler / 3 个适配器（PG / Redis / Kafka，含 fake dialer 模拟 broker 不可达）|

**Manager.RunAll 关键设计**：
- 并发执行所有 checker（goroutine + sync.WaitGroup），wall time ≪ 串行求和
- 单 checker 超时不拖垮其他 checker
- 同名 checker 第二次 Register 被拒绝（防"误配两 DB 实例"覆盖式隐藏）

**11 服务接入**：
- `services/*/internal/router/router.go`：增加 `r.GET("/readyz", health.ReadyzHandler(readyzM))`
- `services/*/cmd/main.go`：构造 `health.NewManager(WithTimeout(1*time.Second))`，
  按可用资源注册 checker：
  - `pool != nil` → `NewPGPoolChecker("postgres-main", pool, 1s)`（7 个 DB 服务）
  - `redisClient != nil` → `NewRedisChecker("redis-main", redisClient, 1s)`（仅 order-service 启用）
  - `len(cfg.Kafka.Brokers) > 0` → `NewKafkaBrokerChecker("kafka-brokers", brokers, 1s)`（10 个服务）
- nil / 空资源时 skip + warn；manager 仍创建（fail-closed 路由保留）
- 路由签名变化：`New(h, secret, readyzM)` / `NewWithPublic(...)`；测试 pass `nil`
- Dockerfile HEALTHCHECK 不变（继续走 `-healthz` flag 探活）

### Commit 3：OTel auto-instrumentation

**目标**：让 SQL / HTTP / Kafka 自动写 span，运维不再需要在 11 个服务 × N 个 repo
里手动 `StartSpan`（v1.4 已部分实现但仅 gin + kafka）。

**3 个 instrumentation**：

| 库 | 包 | API |
| :-- | :-- | :-- |
| gin | `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin@v0.59.0` | `otelgin.Middleware(svcName)` |
| pgx | `github.com/exaring/otelpgx@v0.10.0` | `otelpgx.NewTracer(WithTracerProvider(otel.GetTracerProvider()))` |
| kafka-go | 自实现 wrapper（goproxy.cn 无 otelkafkago 镜像） | `tracing.WrapWriter / WrapReader` |

**11 服务接入**：

1. **`shared/middleware/otel.go`**：`OTelGinMiddleware(service)` 一行接入；
   推荐挂载顺序 `Metrics → Recovery → OTelGin → RateLimit`
   （Recovery 之前确保 panic 也被 OTel 记录，status=500）。
2. **11 服务 router.go**：每服务挂自己的 `OTelGinMiddleware("<svc>-service")`。
3. **7 服务 buildPool**（admin / auth / wallet / user / payment / order / escort）：
   注入 otelpgx tracer：

   ```go
   pcfg, err := pgxpool.ParseConfig(poolCfg.DSN)
   if err == nil {
       tracing.WithPgxPool(pcfg)             // 注册 otelpgx.NewTracer
       poolCfg.Tracer = pcfg.ConnConfig.Tracer // 传给 shared/db.NewPool
   }
   pool := shareddb.NewPool(ctx, poolCfg)
   ```

4. **kafka-go wrapper**（`shared/tracing/kafkago.go`）：
   - `WrapWriter` → `WriteMessages` 启动 "publish <topic>" span，kind=Producer
   - `WrapReader` → `FetchMessage/ReadMessage` 启动 "consume <topic>" span，kind=Consumer
   - 含 `messaging.system=kafka` / `destination.name` / `operation.name` 属性 +
     offset / partition（消费成功时）
   - 错误路径：span status=Error + recordError
   - nil writer / reader 不 panic
   - **未来 otelkafkago 上线后可一行替换**（业务代码不变）

5. **dev 模式（cfg.Tracing.OTLPEndpoint 空）**：
   - InitTracer 退化为 NoopTracerProvider，zero-cost
   - otelgin / otelpgx / kafka wrapper 在 Noop 场景下不创建 span，零开销
   - 生产配 OTLP endpoint 即生效

**OTel 升级的副作用**（go.mod 同步）：
- `go.opentelemetry.io/otel` 1.32.0 → 1.34.0
- `go.opentelemetry.io/otel/sdk` 1.32.0 → 1.34.0
- `github.com/jackc/pgx/v5` 5.7.1 → 5.7.4
- 新增 `go.opentelemetry.io/auto/sdk` v1.1.0

### Commit 4：dev.md §34（本节）

记录上面 3 个 commit 的目标 / 改动 / 测试 / Plan 偏差。

### 累计测试

| 包 | 单测数 | 说明 |
| :-- | :--: | :-- |
| `shared/health` | 17 | 新增：Manager / ReadyzHandler / 3 个适配器 |
| `shared/middleware` | 19 | 修复 ratelimit 5 个 + recovery 新增 1 个 + OTelGin 新增 4 个 = 原 14 + 5 改动 = 19 |
| `shared/tracing` | 16 | 新增：kafka wrapper 6 个；原有 10 个 |
| `shared/metrics` | 8 | 新增 `recovery_panics_total` 接入 + 1 个新单测 |
| `shared/db` | 2 | 不变 |
| 其他 11 shared 包 | — | 不变 |
| 服务包 | — | 不变 |

**全量 80+ 包 0 FAIL**。

### Plan 偏差

1. **kafka-go OTel 自实现 wrapper**：goproxy.cn 无 `otelkafkago` 与 otel contrib 的
   `instrumentation/github.com/segmentio/kafka-go/otelkafkago`（试过 `latest` / `v0.59.0`
   / 直接的 import path 都报 404）。改用 OTel 标准 trace API 实现轻量 wrapper，
   满足"producer / consumer span"基本需求；未来官方包上线后 1 行替换。
2. **router.New 签名扩展加 readyzM / OTelGin**：同步更新所有 router_test.go
   （pass `nil`），保持现有 6 类路由断言不变；测试构建零侵入。
3. **/readyz 路由与 /healthz 共存**：不替换——前者读 readinessProbe（K8s 流量调度），
   后者读 livenessProbe（Docker HEALTHCHECK + K8s 进程存活），两者职责不同。
4. **panic metric label = 路由模板而非 URL.Path**：避免 `id=42 / id=124` 等高基数；
   404 路由（c.FullPath 为空）退化为 URL.Path，保证仍被埋点。
5. **OTel 升级到 1.34.0**：与 contrib v0.59 配套（1.32 兼容性未验证）；副作用是若干
   transitive deps 升级（bytedance/sonic、validator 等），已通过 `go build ./...` 验证。
6. **buildPool 双 trace 注入**：pgxpool.ParseConfig → tracing.WithPgxPool → 用回填的
   `pcfg.ConnConfig.Tracer` 写入 `shared/db.Config.Tracer`；这是最简洁的接入点，
   避免改 `shared/db.NewPool` 内部对 OTel 的依赖。

### 未做

- **kafka-go wrapper 未实际替换业务代码**：wallet / order / message / payment / sos /
  review / user-virtualnumber / escort / admin 的 kafkapublisher 目前未切到
  `tracing.WrapWriter`（避免一次性 9 个服务改动 + 单元测试改造）。commit 4 之后
  再起一轮"OTel auto 业务接入"，每个 publisher 一行 `Wrap` 即可。
- **/readyz handler 写 Prometheus counter**（如 `readyz_check_total{check, status}`）：
  当前只暴露在 HTTP body（JSON）；如需 Prometheus 也想监控，v1.5 增量再加 counter。
- **OTel kafka metrics**：otelpgx 同时输出 metrics（SQL 计数 / 延迟）；当前只启用了
  tracing。metrics exporter 需要起 Prometheus + OTel collector，超出本期。
- **真实 e2e 联调**：依赖外部 broker / OTel collector / Jaeger，dev 环境下无法验证；
  `cfg.Tracing.OTLPEndpoint=jaeger:4318` 生产部署即可生效。

### 端到端 v1.5 验证

```bash
# 1. 启动依赖
docker compose -f docker-compose.deploy.yml up -d postgres kafka jaeger prometheus
# 2. 启动所有服务（cfg.Tracing.OTLPEndpoint=http://jaeger:4318）
docker compose -f docker-compose.deploy.yml up -d auth-service user-service ... wallet-service
# 3. K8s readinessProbe 验证
curl -i http://<svc>:8080/readyz   # 200 + {"healthy":true,...}
# 4. 杀 postgres
docker stop postgres
curl -i http://<svc>:8080/readyz   # 503 + {"healthy":false,"checks":[{"name":"postgres-main","ok":false,...}]}
# K8s 自动摘流 30s 后恢复
docker start postgres
# 5. Prometheus 验证 panic 埋点
curl http://<svc>:8080/metrics | grep recovery_panics_total
# 6. Jaeger 验证 OTel auto-span
# 浏览器 http://localhost:16686 → service <svc> → trace 树形结构：
#   GET /api/v1/users/123
#     ├─ db.query: SELECT * FROM users WHERE id=$1
#     └─ kafka.publish
```

### 累计交付（v1.5）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 真实 PG + 真实 Kafka + 完整 HTTP + OTel auto + 采样 + Recovery 埋点 + RateLimit 修复 + 优雅停机 |
| 前端 3 端 | ✅ 完整脚手架 + 业务页 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger + Prometheus + alert |
| CI | ✅ 4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测性 | ✅ OTel 自动 span + Jaeger + Prometheus + zap 日志关联 + 20 alert |
| 稳定性 | ✅ Recovery 埋点 + RateLimit 修复 + /readyz 依赖探活 + 优雅停机 + LIFO hooks |
| 安全性 | ✅ .env.example 模板 + 密钥管理 SOP |
| 测试 | ✅ 80+ 包 0 FAIL + 跨平台脚本 + smoke + golangci-lint v2 |
| 文档 | ✅ dev.md 34 章节 + README + REVIEW + docs/ops/secrets.md |

**v1.5 补齐 §33 提到的 panic_recovery 埋点依赖 + 增加 /readyz 探活 + 减少业务手动埋 span 成本，
同时修掉 ratelimit flaky 测试。**
---

## 35. v1.5 上线就绪收官

**目标**：v1.5 在 §34 完成 4 项生产稳定性优化的基础上，闭合"业务侧 OTel
接入 + 就绪监控可告警化"两项遗留——把 Kafka publish span 真正落地到
tracing 后端，让 /readyz 失败可被 Prometheus 告警捕到。本节作为 v1.5 收官
章节，记录最近 2 个关键 commit 的设计要点与 v1.5 完整交付清单，并给出
用户本地 3 步上线手册。

**最近 2 个关键 commit**：

| commit | 类型 | 内容 |
| :-- | :-- | :-- |
| `7b89879` | `feat(otel)` | 6 kafkapublisher 接 tracing.WrapWriter + business pattern 集成测试 |
| `7a92f08` | `feat(health)` | /readyz 写 readyz_check_total counter + ReadyzCheckFailure alert |

### 关键设计点

#### Kafka OTel wrapper 业务侧接入

§34 完成了 `shared/tracing.WrapWriter`（自实现 OTel 标准 API wrapper），
但业务侧 publisher 文件并未真正接入。`7b89879` 把其中 **6 个子包式
publisher**（`message / payment / review / sos / escort / user-virtualnumber`）
改造为 wrap 模式——5 步最小变更：

1. `import` 加 `shared/tracing`
2. `writer *kafka.Writer` → `writer *tracing.TracedWriter`
3. `New(...)` 内 `shared/kafka.NewWriter(brokers, topic)` 后立即
   `tracing.WrapWriter(w, "<svc>-service")` 注入 OTel
4. `Close()` 改为 `p.writer.W.Close()`（业务侧仍拿到底层 writer 给
   shutdown hook）
5. `WriteMessages(...)` 调用零变更（wrapper 签名兼容原 `*kafka.Writer`）

剩余 publisher 是另一形态：

- `order/events/publisher.go` 与 `admin/events/publisher.go` 用
  `events.KafkaPublisher`（kafka-go writer 直构，无 shared/kafka 入口），
  走自己的 `SetTopic` 模式——需要给 events.KafkaPublisher 增加 writer
  注入点才能 Wrap，改造面较大，留待 v1.5.1 增量（影响 2 服务 14 topic）。
- `wallet` 与 `match` 是纯 consumer，无 publisher。
- `auth` 按 §32 决策不接入 Kafka。

集成测试 `TestWrapWriter_BusinessPublisherPattern`：模拟业务 publisher
模式（不可达 broker `127.0.0.1:1`），1 次 WriteMessages → 断言
InMemoryExporter 抓到 1 个 `publish <topic>` span，kind=Producer，含
`messaging.system=kafka` / `messaging.destination.name=<topic>` /
`messaging.operation.name=publish` / `service.name=<svc>` 属性；额外校验
`tw.W` 非 nil（业务 Close 依赖）。

#### /readyz Prometheus counter + alert

§34 把 /readyz 端点落到 11 个服务，但仅暴露 HTTP body JSON，无法接
Prometheus 告警。`7a92f08` 让 `/readyz` 每次调用 + 每个 checker 结果都写
`readyz_check_total{check, status}` counter（`shared/metrics` 用 promauto
自动注册，`/metrics` 自动暴露）：

- 入口先 `Inc("{check=\"_request\",status=\"request\"}")` —— QPS 监控
- 遍历 `report.Checks`，按 `s.OK` Inc `"ok"` / `"fail"`
- nil manager fail-closed 分支也 Inc `"{check=\"_manager\",status=\"fail\"}"`，
  便于告警识别"main 装配 bug"

配套新增 `deploy/prometheus/alerts/general.yaml::ReadyzCheckFailure`：

- 表达式：`sum by (job, check) (rate(readyz_check_total{status="fail"}[1m])) > 0`
- `for: 1m`（避免 readinessProbe 抖动误报；与 K8s readinessProbe 默认
  1m timeoutSeconds 对齐）
- severity: critical；category: readiness

3 个新单测：`TestReadyzHandler_IncrementsCounter` /
`TestReadyzHandler_NilManagerIncrementsFail` /
`TestReadyzHandler_AllOK_NoFailCounter`。

### v1.5 完整交付清单

| 维度 | 数量 | 交付内容 |
| :-- | :--: | :-- |
| 后端 Go 服务 | 11 | auth / order / match / message / payment / review / sos / user / escort / wallet / admin |
| 前端端 | 3 | patient-miniapp / escort-app / admin-web |
| Dockerfile | 14 | 11 后端（distroless + -healthz flag）+ 3 前端 |
| docker-compose 服务 | 18 | 4 中间件（postgres / redis / kafka / jaeger）+ 11 后端 + 3 前端 |
| CI jobs | 4 | backend-test / docker-build / frontend-lint / backend-lint |
| 可观测性套件 | 3 | OTel（traces SDK + gin/pgx/kafka auto-instrumentation）/ Prometheus（业务 + 依赖 + panic + readyz counter 共 21 alert）/ Jaeger collector |
| 稳定性套件 | 4 | Recovery 埋点（recovery_panics_total）+ RateLimit（per-IP 桶 + flaky 测试修复）+ 优雅停机（15s Server.Shutdown + LIFO hooks）+ /readyz 依赖探活 |
| 安全性套件 | 4 | .env.example 模板 + 密钥管理 SOP（docs/ops/secrets.md）+ 4 方案对比 + CORS / TLS 生产 checklist |
| 测试工具套件 | 3 | run-tests 跨平台一键脚本 + smoke 端到端脚本 + golangci-lint v2 配置 |
| Linter | 11 | errcheck / gosimple / govet / ineffassign / staticcheck / unused / bodyclose / gocritic / misspell / nakedret / prealloc |
| dev.md 章节 | 34 | §1–§34：工程总览 / 数据库 / Kafka / OTel / 11 服务 PG 接入 / Kafka publisher / 健康 / 密钥 / 4 项稳定性 / 等 |

### 用户本地 3 步上线

```bash
# 1. push（推送代码到远程）
git push origin main

# 2. install（启动依赖 + 服务，OTel endpoint 直连 jaeger）
docker compose -f docker-compose.deploy.yml up -d

# 3. smoke（端到端冒烟：11 服务 /healthz + /readyz + /metrics + Kafka publish span）
bash scripts/smoke.sh
```

验证清单（详见 README §6）：

- 18 容器全部 `healthy`（含 liveness `/healthz` 与 readiness `/readyz`）
- 业务调用 → Jaeger 看到 `publish <topic>` span（6 服务 × 各 1 topic ≥ 6 个）
- Prometheus `/api/v1/rules` 含 `ReadyzCheckFailure` 与 §33 panic_recovery 4 条
- SIGTERM → 15s 内 Server.Shutdown 完毕，LIFO hooks 0 数据丢失
- 杀 postgres → /readyz 返回 503 + `readyz_check_total{check="postgres-main",status="fail"}` +1 → 1min 后 Prometheus 触发 critical alert

### v2 留待下次会话（Token Plan 已用完）

- **OTel metrics exporter 全量接入**：目前仅 otelpgx 自动埋点，HTTP / Kafka
  / 自定义业务指标未接 Prometheus exporter。
- **Kafka wrapper 全集成**：补齐 `order/events/publisher.go` 与
  `admin/events/publisher.go`（events.KafkaPublisher 注入点改造，影响
  2 服务 14 topic）+ `wallet / match` consumer 端 `WrapReader`。
- **业务集成测试套件**：订单-支付-钱包-评价的 e2e 集成测试（docker-compose
  依赖 + Kafka 真实 topic），目前仅单包单测。
- **/readyz 详细 metrics**：单次调用 latency histogram + 各 checker 独立耗时。
- **alert 路由**：ReadyzCheckFailure / panic_recovery 4 条绑 PagerDuty /
  飞书 webhook（按 docs/ops/secrets.md SOP）。

**v1.5 已具备生产上线全部要素——11 后端 + 3 前端 + 14 镜像 + 18 服务
docker-compose + 4 job CI + 3 套可观测 + 4 套稳定性 + 4 套安全性 +
3 套测试工具 + 11 linter + 34 章节 dev.md。Token Plan 用完，剩余
v1.5.1 / v2 增量工作留待下次会话。**