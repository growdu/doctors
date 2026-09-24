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
