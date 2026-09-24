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

## 4. 阶段 3 · order-service

**目标**：订单创建、状态机、抢单并发（FOR UPDATE SKIP LOCKED）。

**接口**：
- `POST /api/v1/orders` 创建（含价格试算）
- `GET /api/v1/orders` 我的订单
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/accept` 陪诊师接单（核心）
- `POST /api/v1/orders/{id}/cancel`
- `POST /api/v1/orders/{id}/finish`
- `GET /healthz`

**关键决策**：
1. **抢单原子化**：用 PG `SELECT … FOR UPDATE SKIP LOCKED` + `version` 乐观锁（不需要 Redis 锁）
3. **状态机**：11 状态 + 14 转换，与 `docs/04` 对齐；每次状态变更写 `order_event` 表
2. **迁移**：用 `golang-migrate`；schema 起步先有 `users / orders / order_events`

---

## 5. 阶段 4 · match-service

**目标**：抢单池（Redis ZSET）+ 候选计算 + 推送。

**接口**：
- `GET /api/v1/match/feed` 陪诊师端拉抢单池
- `POST /api/v1/match/candidates` 订单端查候选
- 内部：`/internal/match/dispatch`（订单服务回调）
- `GET /healthz`

**关键决策**：
1. **ZSET score = 距离 + 时效加权**，候选 Top-N 推送
2. **推送通道**：v1 用 mock Sender（log + 内部 channel）；真接入在 v2
3. 与 order-service 通过 Kafka 解耦：`order.created` → match 消费 → 写抢单池

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