# 陪诊师平台 · Doctors

> 面向"陪诊师 + 患者"双边市场的预约 / 匹配 / 支付 / 评价 / 钱包 / SOS 一体化平台。
> 本仓库为 monorepo，统一 Go 后端（11 个微服务）+ 3 个前端应用（小程序 / Flutter / 后台 SPA）。

![status](https://img.shields.io/badge/status-v1.0-blue) ![go](https://img.shields.io/badge/go-1.24%2B-00ADD8) ![node](https://img.shields.io/badge/node-20%2B-339933) ![flutter](https://img.shields.io/badge/flutter-3.24%2B-02569B) ![license](https://img.shields.io/badge/license-MIT-green)

---

## 1. 项目介绍

**Doctors** 是一个完整的陪诊服务平台：患者通过小程序下单，陪诊师在 Flutter App 上接单，
后台管理员通过 React SPA 监管全流程。整套系统由 **11 个 Go 微服务** 编排、**PostgreSQL + Redis + Kafka** 三件套基础设施支撑；
前端覆盖患者端（uni-app 小程序）、陪诊师端（Flutter 跨端）、后台端（Vite + React）。设计目标：

- **高内聚**：每服务只负责一个业务域（订单 / 支付 / 钱包 / 评价 …），内部用 gin + 自研 handler 模式。
- **可观测**：全链路 OpenTelemetry（OTLP HTTP）+ zap 结构化日志 + W3C `traceparent` HTTP 透传。
- **可演进**：monorepo + 单一 go module；前后端共享 `contracts/` 事件契约 + OpenAPI yaml。

完整设计稿见 [`docs/`](./docs/)；增量开发日志见 [`dev.md`](./dev.md)。

---

## 2. 架构

### 2.1 系统总览

```mermaid
flowchart LR
    subgraph 用户
        P[患者]
        E[陪诊师]
        A[管理员]
    end

    subgraph 客户端
        MP[patient-miniapp<br/>uni-app 小程序]
        FA[escort-app<br/>Flutter 跨端]
        AW[admin-web<br/>Vite + React SPA]
    end

    subgraph API网关
        NG[nginx / Envoy<br/>:80 / :8092]
    end

    subgraph "Go 微服务 (11 个)"
        AS[auth-service<br/>:8081]
        OS[order-service<br/>:8082]
        MS[match-service<br/>:8083]
        MES[message-service<br/>:8084]
        PS[payment-service<br/>:8085]
        RS[review-service<br/>:8086]
        SS[sos-service<br/>:8087]
        US[user-service<br/>:8088]
        ES[escort-service<br/>:8089]
        WS[wallet-service<br/>:8090]
        ADS[admin-service<br/>:8091]
    end

    subgraph "基础设施"
        PG[(PostgreSQL 16<br/>:5432)]
        RD[(Redis 7<br/>:6379)]
        KF[Kafka<br/>:9092]
        OTEL[OTel Collector<br/>:4318]
    end

    P --> MP
    E --> FA
    A --> AW
    MP --> NG
    FA --> NG
    AW --> NG
    NG --> AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS --> PG
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS --> RD
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS --> KF
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS -.OTLP.-> OTEL
```

### 2.2 关键交互流（患者下单）

```
patient-miniapp
  └─► auth-service        登录 / 拿 JWT
  └─► user-service        拉地址 / 优惠券 / 医院 / 套餐
  └─► order-service       创建订单（pgx SKIP LOCKED）
        └─► Kafka OrderCreated
              ├─► match-service    候选打分 + 抢单池（Redis SETNX）
              └─► message-service  站内信推送
        └─► payment-service     调起支付（mock channel）
              └─► Kafka PaymentCompleted
                    ├─► wallet-service  T+7 结算（scanner 1 min）
                    └─► review-service  待评价（订单完成 +24h）
        └─► sos-service         一键紧急联系（紧急通道）
admin-web → admin-service      工单 / 报表 / 内部服务代理
```

---

## 3. 服务清单

### 3.1 后端微服务（11 个 Go 服务）

| 服务             | 端口 | 职责                                                                 | Plan                                                                                         |
| ---------------- | ---- | -------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `auth-service`   | 8081 | 手机号 / 微信登录 / 实名认证 / JWT 签发                              | [dev.md §3](./dev.md#3-阶段-2--auth-service--已完成)                                          |
| `order-service`  | 8082 | 订单创建 / 抢单（SKIP LOCKED）/ 锁单（Redis SETNX）/ 状态机 / 调度  | [dev.md §4](./dev.md#4-阶段-3--order-service--已完成)                                        |
| `match-service`  | 8083 | 候选打分 / 抢单池（NopPool / RedisPool）                            | [dev.md §5](./dev.md#5-阶段-4--match-service--已完成)                                        |
| `message-service`| 8084 | 站内信 / WebSocket / Kafka publish                                  | [dev.md §20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan) |
| `payment-service`| 8085 | 支付下单 / 退款 / 退款策略 / Channel 抽象                          | [dev.md §24](./dev.md#24-payment-service-补全--11--healthz-flag--dockerfile-healthcheck2026-09-24-ops-部署补全) |
| `review-service` | 8086 | 双向评价 / 标签 / 商家回复                                          | [dev.md §20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan) |
| `sos-service`    | 8087 | 一键呼叫 / 位置上报 / 紧急联系 / 30s 频率去重                       | [dev.md §20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan) |
| `user-service`   | 8088 | 资料 / 实名 / 地址 / 优惠券 / 医院 / 套餐 / 虚拟号                  | [dev.md §14](./dev.md#14-patient-miniapp-v11-选人模式增量2026-09-24-patient-miniapp-setup-plan-p1-p7)  |
| `escort-service` | 8089 | 陪诊师注册 / 实名 / 接单设置 / 排班 / 可用性（availability 子包）    | [dev.md §13](./dev.md#13-escort-app-v11-选人模式增量2026-09-24-escort-app-setup-plan-a1-a7)         |
| `wallet-service` | 8090 | 余额 / 冻结 / 提现 / 流水 / T+7 scanner                              | [dev.md §16](./dev.md#16-后端-wallet-t7-结算服务2026-09-24-wallet-plan-w1-w5)                       |
| `admin-service`  | 8091 | 后台：工单 / 报表 / 内部服务代理（order / review / escort / user）   | [dev.md §10](./dev.md#10-模块解耦与新骨架服务2026-09-24)                                          |

### 3.2 前端应用（3 端）

| 前端                | 端口 | 技术栈                                | Plan                                                                  |
| ------------------- | ---- | ------------------------------------- | --------------------------------------------------------------------- |
| `patient-miniapp`   | 80   | uni-app + Vue 3 + uView Plus + Pinia | [dev.md §14](./dev.md#14-patient-miniapp-v11-选人模式增量2026-09-24-patient-miniapp-setup-plan-p1-p7) |
| `escort-app`        | 8080 | Flutter 3.24+（web / iOS / Android） | [dev.md §13](./dev.md#13-escort-app-v11-选人模式增量2026-09-24-escort-app-setup-plan-a1-a7)         |
| `admin-web`         | 8092 | Vite + React 18 + TypeScript + antd  | [dev.md §15](./dev.md#15-admin-web-v1-缺失骨架补齐2026-09-24-admin-web-setup-plan-a1-a11)         |

---

## 4. 目录结构

```
doctors/
├── services/                  # 11 个 Go 后端微服务
│   ├── auth/      cmd/main.go + internal/{handler,router,server,service,realname,sms,wxlogin}
│   ├── order/     cmd/main.go + internal/{handler,router,server,service,repo,events,scheduler,lock}
│   ├── match/     cmd/main.go + internal/{handler,router,server,service,pool}
│   ├── message/   cmd/main.go + internal/{handler,router,server,service}
│   ├── payment/   cmd/main.go + internal/{handler,router,server,service}
│   ├── review/    cmd/main.go + internal/{handler,router,server,service}
│   ├── sos/       cmd/main.go + internal/{handler,router,server,service}
│   ├── user/      cmd/main.go + internal/{handler,router,server,service,address,coupon,hospital,pkg,virtualnumber}
│   ├── escort/    cmd/main.go + internal/{handler,router,server,service,availability}
│   ├── wallet/    cmd/main.go + internal/{handler,router,server,service,repo}
│   └── admin/     cmd/main.go + internal/{handler,router,server,service,repo,clients,events}
│
├── shared/                     # 跨服务通用包（11 个）
│   ├── config/                  # viper 配置加载（DOCTORS_<SVC>_* env override）
│   ├── logger/                  # zap 结构化日志 + trace_id 上下文传递
│   ├── tracing/                 # OTel 全链路追踪（OTLP HTTP + W3C TraceContext）
│   ├── metrics/                 # Prometheus 业务指标（HTTP / DB / Kafka）+ /metrics 抓取端
│   ├── auth/                    # JWT 签发 / 校验
│   ├── httpx/                   # 统一响应 / 错误码
│   ├── middleware/              # Gin 中间件（Auth / RoleAuth / Metrics / Cors / Logging）
│   ├── contracts/               # 共享事件契约（OrderCompleted / PaymentRefunded / ...）
│   ├── db/                      # pgxpool 封装 + 健康检查
│   ├── redis/                   # go-redis 封装
│   ├── kafka/                   # kafka-go 封装（Producer / Consumer）
│   ├── idempotency/             # 幂等键
│   ├── lock/                    # Redis SETNX 分布式锁
│   └── errs/                    # 业务错误码表
│
├── migrations/                 # 数据库迁移（14 份 SQL：0001~0014）
│   ├── 0001_users.up.sql / down.sql
│   ├── 0002_orders.up.sql / down.sql
│   ├── 0003_orders_state.up.sql / down.sql
│   ├── 0004_refunds.up.sql / down.sql
│   ├── 0007_wallets.up.sql / down.sql
│   ├── 0008_admin_work_orders.up.sql / down.sql
│   ├── 0009_select_escort.up.sql / down.sql
│   ├── 0010_addresses.up.sql / down.sql
│   ├── 0011_coupons.up.sql / down.sql
│   ├── 0012_hospitals.up.sql / down.sql
│   ├── 0013_packages.up.sql / down.sql
│   ├── 0014_virtual_numbers.up.sql / down.sql
│   └── migrations_test.go       # golang-migrate 集成测试
│
├── frontend/                   # 3 个前端工程
│   ├── admin-web/              # Vite + React 18 + TS + antd
│   ├── patient-miniapp/        # uni-app + Vue 3 + uView Plus
│   └── escort-app/             # Flutter 3.24+
│
├── config/                     # 11 个服务 yaml 配置（DOCTORS_<SVC>_* env override）
│   ├── auth.yaml order.yaml match.yaml ...
│   └── wallet.yaml             # 每个文件含 http / db / redis / kafka / auth / logging / tracing
│
├── docs/                       # 需求文档 + 架构图（9 份 markdown + diagrams/）
│   ├── 01-项目概述.md
│   ├── 02-角色与权限.md
│   ├── 03-功能需求.md
│   ├── 04-业务流程.md
│   ├── 05-非功能需求.md
│   ├── 06-系统架构.md
│   ├── 07-数据模型.md
│   ├── 08-接口需求.md
│   ├── 09-验收与发布.md
│   └── REVIEW-REPORT.md
│
├── scripts/                    # 运维 / 测试 / 部署脚本
│   ├── smoke-*.sh              # 5 个服务 smoke 测试
│   ├── run-tests.sh / .ps1     # 一键跑全套测试（§26 增量，跨平台）
│   └── README.md
│
├── deploy/                     # K8s manifests（按需启用，本期未生成）
├── .github/workflows/ci.yml    # GitHub Actions CI（4 job：lint / test / build / docker）
├── docker-compose.yml          # 本地 dev 中间件（PG + Redis + Kafka）
├── docker-compose.deploy.yml   # 全量部署编排（11 Go + 3 前端 + 3 中间件）
├── Makefile                    # 顶层命令（test / build / docker-up / down）
├── go.mod / go.sum             # 单一 go module
├── mkdocs.yml                  # docs/ 站点配置
├── dev.md                      # 增量开发日志（§0~§25）
└── README.md                   # 本文件
```

---

## 5. 本地开发

### 5.1 前置条件

| 工具         | 版本        | 说明                                            |
| ------------ | ----------- | ----------------------------------------------- |
| Go           | **1.24+**   | toolchain go1.24.3；项目 go.mod 锁定             |
| Node.js      | **20+**     | admin-web / patient-miniapp dev / build         |
| pnpm         | 9+          | admin-web 包管理                                |
| Flutter      | **3.24+**   | escort-app；Dart SDK >= 3.5                     |
| Docker       | 24+         | 本地 PG / Redis / Kafka + 部署编排              |
| Docker Compose v2 | -    | `docker compose`（非 `docker-compose`）         |
| 端口空闲     | -           | 5432 / 6379 / 9092 + 8081~8091 + 80 / 8080 / 8092 |

### 5.2 启动中间件

```bash
docker compose up -d              # PG + Redis + Kafka
# 或：make docker-up
```

### 5.3 应用数据库迁移

```bash
for f in migrations/*.up.sql; do
  psql "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable" -f "$f"
done
```

### 5.4 启动单个 Go 服务

```bash
go run ./services/auth/cmd
# 或：make run-auth
```

环境变量覆盖 yaml（与 `shared/config/loader.go` 兼容）：

```bash
export DOCTORS_AUTH_HTTP_ADDR=:8081
export DOCTORS_AUTH_DB_DSN=postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable
export DOCTORS_AUTH_KAFKA_BROKERS=127.0.0.1:9092
export DOCTORS_AUTH_JWT_SECRET=dev-secret-change-me
export DOCTORS_AUTH_TRACING_OTLP_ENDPOINT=otel-collector:4318   # §26：留空 → Noop
```

### 5.5 启动前端开发服务器

```bash
cd frontend/admin-web
pnpm install && pnpm dev          # → http://localhost:5173

cd frontend/patient-miniapp
npm install && npm run dev:h5     # → http://localhost:8080

cd frontend/escort-app
flutter pub get && flutter run -d chrome  # → http://localhost:8080
```

---

## 6. 跑测试

### 6.1 一键脚本（推荐）

```bash
# Linux / macOS / WSL
bash scripts/run-tests.sh

# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1
```

脚本自动跑：

1. **后端单元测试**：`go test -race -count=1 -timeout=180s ./...`（200+ 测试用例）
2. **前端单元测试**：
   - admin-web：`pnpm install && npx vitest run`
   - patient-miniapp：`npm install && npm test`
   - escort-app：`flutter pub get && flutter test`
3. **集成测试**：`go test -tags=integration ./migrations/... ./services/...`（需 `docker compose up -d`）

每个步骤独立 try/skip，缺失工具不阻塞整体；输出彩色汇总（绿/红）+ 每步耗时。

### 6.2 手动命令

```bash
make test                 # 后端单元测试
make test-integration     # 集成测试（需 docker compose up）

# 单独跑某包
go test -race -count=1 ./services/auth/...
go test -race -count=1 ./shared/tracing/...
```

---

## 7. 部署

### 7.1 Docker 全栈（推荐）

> 入口：`docker-compose.deploy.yml`
> 编排范围：3 中间件 + 11 Go 后端 + 3 前端 + 1 网络 + 3 持久卷

#### 前置条件

- Docker Engine ≥ 24 + Docker Compose v2
- 宿主机空闲端口：5432 / 6379 / 9092（中间件）+ 8081~8091（Go 服务）+ 80 / 8080 / 8092（前端）

#### 一键构建 + 启动

```bash
docker compose -f docker-compose.deploy.yml build           # 构建 14 个镜像
docker compose -f docker-compose.deploy.yml up -d           # 后台启动
docker compose -f docker-compose.deploy.yml ps              # 查看运行状态
```

启动顺序由 `depends_on.condition: service_healthy` 保证：

```
postgres ─┐
redis    ─┼─→ 11 Go 服务 ─→ admin-service（依赖 order / review / escort / user）
kafka    ─┘
                └→ 3 前端服务（nginx，无外部依赖）
```

#### 常用运维

```bash
# 查看日志
docker compose -f docker-compose.deploy.yml logs -f --tail=200 auth-service

# 重启单个服务
docker compose -f docker-compose.deploy.yml restart order-service

# 滚动更新
docker compose -f docker-compose.deploy.yml build auth-service
docker compose -f docker-compose.deploy.yml up -d auth-service

# 停止 + 清理（保留数据）
docker compose -f docker-compose.deploy.yml down

# 清空所有数据
docker compose -f docker-compose.deploy.yml down -v
```

### 7.2 端口清单

| 端口  | 用途                    | 端口  | 用途                  |
| ----- | ----------------------- | ----- | --------------------- |
| 5432  | PostgreSQL              | 8084  | message-service       |
| 6379  | Redis                   | 8085  | payment-service       |
| 9092  | Kafka                   | 8086  | review-service        |
| 4318  | OTel → Jaeger (OTLP HTTP) | 8087  | sos-service           |
| 16686 | **Jaeger UI**           | 8088  | user-service          |
| 80    | patient-miniapp H5      | 8089  | escort-service        |
| 8080  | escort-app (Flutter web)| 8090  | wallet-service        |
| 8081  | auth-service            | 8091  | admin-service         |
| 8082  | order-service           | 8092  | admin-web             |
| 8083  | match-service           |       |                       |

### 7.3 配置注入

每个 Go 服务通过 `DOCTORS_<UPPER_SVC>_*` 环境变量注入；yaml（`config/<svc>.yaml`）通过 `./config:/app/config:ro` 挂载作为默认值。

| 环境变量                                | 含义                              | 示例                                                |
| --------------------------------------- | --------------------------------- | --------------------------------------------------- |
| `DOCTORS_<SVC>_HTTP_ADDR`               | HTTP 监听地址                     | `:8080`                                             |
| `DOCTORS_<SVC>_DB_DSN`                  | PostgreSQL DSN                    | `postgres://doctors:doctors@postgres:5432/doctors`  |
| `DOCTORS_<SVC>_REDIS_ADDR`              | Redis 地址                        | `redis:6379`                                        |
| `DOCTORS_<SVC>_KAFKA_BROKERS`           | Kafka broker 列表                 | `kafka:9092`                                        |
| `DOCTORS_<SVC>_JWT_SECRET`              | JWT 签名密钥（**生产必须改**）    | `dev-secret-change-me`                              |
| `DOCTORS_<SVC>_LOGGING_LEVEL`           | zap 日志级别                      | `debug` / `info` / `warn` / `error`                 |
| `DOCTORS_<SVC>_TRACING_OTLP_ENDPOINT`   | OTel OTLP 接收端（留空 → Noop）   | `otel-collector:4318`                               |
| `DOCTORS_<SVC>_METRICS_ENABLED`        | Prometheus 指标开关               | `true` / `false`                                    |
| `DOCTORS_<SVC>_METRICS_SERVICE_NAME`   | 覆盖 `service_info{service=}`     | `auth-service`                                      |

### 7.4 镜像与体积

- **Go 服务**：`gcr.io/distroless/static-debian12:nonroot`（< 30MB，无 shell，UID 65532）
- **前端服务**：`nginx:1.27-alpine`（含 wget，用于容器内健康探针）
- 镜像 tag：`doctors/<svc>:latest`

---

## 8. 可观测性

### 8.1 日志（zap）

- 全局 logger：`shared/logger.L()`，JSON 格式输出到 stdout。
- trace_id 传递：`logger.WithTrace(ctx, id)` + `logger.FromContext(ctx).Info(...)` 无侵入注入。
- 日志级别：`config.<svc>.yaml.logging.level` 或 env `DOCTORS_<SVC>_LOGGING_LEVEL`。

### 8.2 全链路追踪（OTel · §26 增量）

- 包：`shared/tracing` —— `InitTracer` / `StartSpan` / `Inject` / `Extract` / `HeaderCarrier`。
- 协议：OTLP HTTP（4318 端口），默认 `insecure`；HTTPS 改 `https://` 前缀即生效。
- Propagator：W3C `TraceContext` + `Baggage`（HTTP header `traceparent` 透传）。
- endpoint 留空 → 退化 `NoopTracerProvider`，零开销、不导出。
- 接入：11 个服务 `cmd/main.go` 已在 `logger.SetLevel` 之前调 `tracing.InitTracer(<svc>, cfg.Tracing.OTLPEndpoint)`。
- 日志关联：`logger.FromContext(ctx)` 自动读 `trace.SpanContextFromContext`，注入 `otel_trace_id` / `otel_span_id` 字段（Jaeger UI 可按 traceId 检索）。
- Collector：`docker-compose.deploy.yml` 内置 `jaeger` 服务（jaegertracing/all-in-one）；Go 服务通过 `OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318` 导出 span。
- **Jaeger UI 访问**：[http://localhost:16686](http://localhost:16686) — 选 service（如 `auth-service`）→ Search → 点 trace 看瀑布图。
- 详细用法：[shared/tracing/README.md](./shared/tracing/README.md)

### 8.3 健康检查

- 中间件（PG / Redis / Kafka）由官方镜像内置 healthcheck，`depends_on.condition: service_healthy` 等待其就绪。
- 3 个前端 nginx 在 Dockerfile 内置 `HEALTHCHECK CMD wget -q --spider http://127.0.0.1/index.html`。
- 11 个 Go 服务运行在 distroless（无 shell / 无 curl / 无 wget）；容器内 HTTP 探针通过 `-healthz` flag 启独立 :9090 server 返回 200 OK。

### 8.4 Prometheus 业务指标（§29 metrics plan）

- 包：[`shared/metrics`](./shared/metrics) + [`shared/middleware/metrics.go`](./shared/middleware/metrics.go)
- 抓取端：11 个 Go 服务在主端口（:8080）暴露 `GET /metrics`（标准 prom 文本格式），与 `/healthz` 共用同一 HTTP server。
- 中间件：router 最先挂 `middleware.Metrics()`，自动记录 `http_requests_total{method,path,status}` + `http_request_duration_seconds{method,path}`。
- 6 个默认业务指标：

| 指标名                              | 类型        | 标签                       | 含义                                                  |
| ----------------------------------- | ----------- | -------------------------- | ----------------------------------------------------- |
| `http_requests_total`               | CounterVec  | `method`,`path`,`status`   | HTTP 请求累计数；`status` 归桶到 2xx/3xx/4xx/5xx。   |
| `http_request_duration_seconds`     | HistogramVec| `method`,`path`            | HTTP 请求耗时分布（buckets 5ms~5s）。                |
| `db_pool_acquired_connections`      | GaugeVec    | `pool`                     | DB 连接池占用连接数。                                |
| `db_pool_idle_connections`          | GaugeVec    | `pool`                     | DB 连接池空闲连接数。                                |
| `db_pool_total_connections`         | GaugeVec    | `pool`                     | DB 连接池总连接数。                                  |
| `kafka_consumer_lag`                | GaugeVec    | `topic`,`group`            | Kafka 消费者滞后。                                   |
| `service_info`                      | Gauge (=1)  | `service`,`version`,`go_version` | 服务构建元信息（单序列）。                        |

- DB pool 自动采集：接入 `metrics.WithDBStatProvider(func() []DBPoolStat{...})` 后启动 30s 一次的后台 goroutine。`wallet-service` 已接 pgxpool；其余服务 pool=nil 时该指标暂不出现。
- **Prometheus scrape 配置示例**：

  ```yaml
  scrape_configs:
    - job_name: doctors-services
      metrics_path: /metrics
      scrape_interval: 15s
      static_configs:
        - targets:
          - auth-service:8080
          - order-service:8080
          - match-service:8080
          - message-service:8080
          - payment-service:8080
          - review-service:8080
          - sos-service:8080
          - user-service:8080
          - escort-service:8080
          - wallet-service:8080
          - admin-service:8080
  ```

- 配置项（`config/<svc>.yaml`）：

  ```yaml
  metrics:
    enabled: true            # false → router 不挂 metrics 中间件
    service_name: auth-service  # 覆盖 service_info{service="..."}
  ```

- 关停 HEALTHCHECK：`/metrics` 由 Prometheus Operator / kube-prometheus-stack 的 ServiceMonitor 拉取，不进 Docker HEALTHCHECK（K8s 仍走 `-healthz` 探 liveness）。

---

## 9. 贡献

### 9.1 Commit 规范

```
<scope>(<module>): <description>

[optional body]

[optional footer]
```

`<scope>` 取值：`feat` / `fix` / `refactor` / `docs` / `test` / `ops` / `chore`。
`<module>`：受影响的子目录（`tracing` / `auth` / `admin-web` / `dev.md` / `ci` 等）。

示例：

```
feat(tracing): shared/tracing OTel SDK + 11 服务接入
fix(order): race condition on accept with SKIP LOCKED
docs(dev.md): §26 OTel 全链路追踪 + 一键测试脚本
chore(scripts): 跨平台 run-tests.sh + run-tests.ps1
```

### 9.2 PR 流程

1. Fork 仓库 → 创建特性分支（`<scope>/<short-desc>`）。
2. 本地跑 `bash scripts/run-tests.sh`（或 PowerShell 版本）确保全绿。
3. 提交 commit 遵循上述规范；开 PR 时附：
   - **Why**（业务背景 / 关联 issue）
   - **What**（改动摘要）
   - **How to test**（验证步骤）
   - **Screenshots**（前端 UI 变更）
4. CI 通过 + 1 位 reviewer 批准后合入 `main`。

### 9.3 CODEOWNERS

按目录自动分配 reviewer（[.github/CODEOWNERS](./.github/CODEOWNERS)）：

| 路径                              | Owner          |
| --------------------------------- | -------------- |
| `services/**`                     | `@backend-team` |
| `frontend/**`                     | `@frontend-team` |
| `docs/**`、`*.md`                 | `@docs-team`   |
| `docker-compose*.yml` / `Makefile`| `@backend-team` |

---

## 10. License

本仓库使用 **MIT License**（示意，正式 LICENSE 文件待 §26 收尾时落地）：

```
MIT License

Copyright (c) 2026 Doctors Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## 11. 进一步阅读

### 11.1 设计文档

- [`docs/01-项目概述.md`](./docs/01-项目概述.md) ~ [`docs/09-验收与发布.md`](./docs/09-验收与发布.md)
- [`docs/REVIEW-REPORT.md`](./docs/REVIEW-REPORT.md)

### 11.2 增量开发日志（dev.md 章节）

| 章节                                                                                                       | 内容                                  |
| ---------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| [§0](./dev.md#0-全局设计思路)                                                                              | 全局设计思路 / 选型 / 仓库结构        |
| [§3](./dev.md#3-阶段-2--auth-service--已完成)                                                              | auth-service 完成                     |
| [§4](./dev.md#4-阶段-3--order-service--已完成)                                                             | order-service 完成                    |
| [§5](./dev.md#5-阶段-4--match-service--已完成)                                                             | match-service 完成                    |
| [§10](./dev.md#10-模块解耦与新骨架服务2026-09-24)                                                          | 11 服务骨架解耦                       |
| [§11](./dev.md#11-前端三端骨架2026-09-24-三-worker-并行)                                                   | 前端三端骨架                          |
| [§13](./dev.md#13-escort-app-v11-选人模式增量2026-09-24-escort-app-setup-plan-a1-a7)                       | escort-app v1.1 增量                  |
| [§14](./dev.md#14-patient-miniapp-v11-选人模式增量2026-09-24-patient-miniapp-setup-plan-p1-p7)             | patient-miniapp v1.1 增量             |
| [§15](./dev.md#15-admin-web-v1-缺失骨架补齐2026-09-24-admin-web-setup-plan-a1-a11)                         | admin-web v1 补齐                     |
| [§16](./dev.md#16-后端-wallet-t7-结算服务2026-09-24-wallet-plan-w1-w5)                                     | wallet T+7 结算服务                   |
| [§17](./dev.md#17-前端三端-contractsyaml-一致性校验2026-09-24)                                             | contracts.yaml 一致性校验             |
| [§20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan)             | 4 服务补全（message / sos / review / escort） |
| [§21](./dev.md#21-patient-miniapp-8-核心业务页接-api2026-09-24-patient-miniapp-v1-plan-m1-m8)              | patient-miniapp 8 业务页接 API         |
| [§22](./dev.md#22-escort-app-8-核心业务页接-api2026-09-24-escort-app-v1-plan-e1-e8)                        | escort-app 8 业务页接 API             |
| [§23](./dev.md#23-11-go-服务-dockerfile--3-前端-dockerfile--docker-composedeployyml2026-09-24-ops-部署)   | 11 Go + 3 前端 Dockerfile             |
| [§24](./dev.md#24-payment-service-补全--11--healthz-flag--dockerfile-healthcheck2026-09-24-ops-部署补全) | payment-service 补全 + -healthz flag |
| [§25](./dev.md#25-github-actions-ci-全栈2026-09-24-ops-部署补全)                                            | GitHub Actions CI 全栈                |

### 11.3 仓库命令与配置

- [`Makefile`](./Makefile) — 本地常用 target
- [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) — CI 配置
- [`.golangci.yml`](./.golangci.yml) — Lint 规则
- [`shared/tracing/README.md`](./shared/tracing/README.md) — OTel 全链路追踪用法
- [`scripts/README.md`](./scripts/README.md) — 一键测试脚本