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

## 1. 阶段 0 · 项目骨架

**目标**：可被 `make test` / `make docker-up` 正常执行的最简 monorepo。

**关键决策**：

1. **monorepo 单 go module** 而非 multi-module：11 个微服务在编译时共享 `shared/` 包，sqlc、proto 复用；缺点是大，但 v1 服务不多，可接受。
2. **顶层 Makefile** 暴露所有命令；`make help` 必须列出全部 target。
3. **docker-compose** 只起 PG + Redis + Kafka 三个外部依赖；服务本身由 `make run-<svc>` 拉起。
4. **`shared/httpx`** 第一版就定下统一响应格式：

   ```json
   {"code": 0, "message": "ok", "data": {}, "trace_id": "..."}
   ```

   错误码规范参见 `docs/08-接口需求.md` 8.8。

**步骤**：
- [ ] 0.1 go.mod（Go 1.22）
- [ ] 0.2 .gitignore
- [ ] 0.3 .golangci.yml
- [ ] 0.4 顶层 Makefile
- [ ] 0.5 docker-compose.yml（PG + Redis + Kafka）
- [ ] 0.6 shared/httpx 包（含响应/错误码 + 单测）
- [ ] 0.7 顶层 smoke：`make help` + `go test ./...`

---

## 2. 阶段 1 · shared 基础设施

**目标**：所有服务可直接 import 的通用包：config、logger、db（pgxpool）、redis、kafka、auth（JWT）。

**关键决策**：

1. **config**：Viper + 多源（环境变量 > 本地 yaml）；每个服务 `config.Load("auth")` 自动选 `config/auth.yaml`
2. **logger**：zap + 全局 `logger.L()`；强制 JSON 输出；含 trace_id 注入
3. **db**：pgxpool + 健康检查 + `WithTx` 事务辅助；连接池参数从 config 读
4. **redis**：go-redis v9 + `ClusterClient` 支持
5. **kafka**：kafka-go（segmentio）；生产者 / 消费者接口统一
6. **auth**：HS256 JWT；claim 含 `user_id / role / unionid / exp`

每个包：先写 `*_test.go`，再写 `*.go`，最后 `commit`。

---

## 3. 阶段 2 · auth-service

**目标**：可独立启动的认证服务，含手机验证码登录 + 微信登录 + JWT 签发。

**接口**：
- `POST /api/v1/auth/sms/send`
- `POST /api/v1/auth/login` (sms / wx)
- `POST /api/v1/auth/refresh`
- `POST /api/v1/users/real-name/auth`
- `GET /api/v1/users/me`
- `GET /healthz`

**关键决策**：
1. 短信发送用 mock 接口（v1 不接真通道，留 Sender 接口）
2. 微信登录用 `wxlogin.Client` 接口，默认实现 mock；真接入由 config 切真渠道
3. 实名信息只存哈希 + 末四位
4. unionid + role + openid_mini/openid_app 三段式存储（呼应评审 C-07）

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