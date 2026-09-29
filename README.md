# 陪诊师平台 · Doctors

> 面向"陪诊师 + 患者"双边市场的预约 / 匹配 / 支付 / 评价 / 钱包 / SOS 一体化平台。
> 本仓库为 monorepo，统一 Go 后端（11 个微服务）+ 4 个前端工程（v1 三端 + v2 unified-app 合并目标）。

![status](https://img.shields.io/badge/status-v2.0--sp2-blue) ![go](https://img.shields.io/badge/go-1.24%2B-00ADD8) ![node](https://img.shields.io/badge/node-20%2B-339933) ![flutter](https://img.shields.io/badge/flutter-3.24%2B-02569B) ![license](https://img.shields.io/badge/license-MIT-green)

---

## 1. 项目介绍

**Doctors** 是一个完整的陪诊服务平台。整套系统由 **11 个 Go 微服务** 编排、**PostgreSQL + Redis + Kafka** 三件套基础设施支撑。

### 1.1 架构演进：v1 → v2

| 版本 | 状态 | 前端架构 |
| --- | --- | --- |
| **v1** | ✅ 已完成（§0–§44 384 commits） | 3 端独立 codebase（patient-miniapp uni-app / escort-app Flutter / admin-web React） |
| **v2 unified-app** | 🚧 进行中（§42–§45 后端 + 基础设施完成，业务域迁移中） | 1 个 codebase = 3 域（uni-app + Vue 3 + Pinia + 多角色 + SSO） |

**v2 关键设计**（[plan 2026-09-28-unified-app-v2.md](./docs/superpowers/plans/2026-09-28-unified-app-v2.md)）：
- **3 域统一**：同一份代码登录后按 `active_role` 切换访问 patient / escort / admin 域
- **JWT 多角色**：后端 Claims 从单字段 `role string` → `roles []string + active string`
- **/switch-role 接口**：`POST /api/v1/auth/switch-role` 换 active 角色返回新 JWT
- **SSO**：同账号一次登录，3 域 token 通用
- **域快捷入口**：DomainSwitcher + RoleSwitcherModal 完整 UI 闭环

### 1.2 设计目标

- **高内聚**：每服务只负责一个业务域（订单 / 支付 / 钱包 / 评价 …），内部用 gin + 自研 handler 模式。
- **可观测**：全链路 OpenTelemetry（OTLP HTTP）+ zap 结构化日志 + W3C `traceparent` HTTP 透传。
- **可演进**：monorepo + 单一 go module；前后端共享 `contracts/` 事件契约 + v2 unified-app 单 codebase。
- **统一体验**：v2 一个 App 覆盖 4 端输出（h5 + 微信小程序 + Android + iOS）+ PC 后台。

完整设计稿见 [`docs/`](./docs/)；增量开发日志见 [`dev.md`](./dev.md)。

---

## 2. 架构

### 2.1 v2 统一架构（当前目标）

```mermaid
flowchart TB
    U["Unified App<br/>uni-app + Vue 3 + Pinia<br/>单 codebase · 多端输出"]
    
    subgraph "Unified App 内部"
      AD["Admin 域<br/>PC 后台"]
      PD["Patient 域<br/>小程序 + H5 + APP"]
      ED["Escort 域<br/>移动端为主"]
      SH["共享层<br/>auth / api / store / router / theme"]
    end
    
    subgraph "API 网关"
      NG["nginx / Envoy<br/>:80 / :8092"]
    end
    
    subgraph "Go 微服务 (11 个 · v2 多角色守卫已挂)"
      AS["auth-service<br/>:8081<br/>JWT roles[]+active · /switch-role"]
      OS["order-service<br/>:8082<br/>patient RoleAuthWithKey"]
      MS["match-service<br/>:8083<br/>escort RoleAuthWithKey"]
      MES["message-service<br/>:8084"]
      PS["payment-service<br/>:8085"]
      RS["review-service<br/>:8086"]
      SS["sos-service<br/>:8087"]
      US["user-service<br/>:8088"]
      ES["escort-service<br/>:8089"]
      WS["wallet-service<br/>:8090"]
      ADS["admin-service<br/>:8091<br/>v2 multi-role gate"]
    end
    
    subgraph "基础设施"
      PG[("PostgreSQL 16<br/>users.roles JSONB<br/>:5432")]
      RD[("Redis 7<br/>:6379")]
      KF["Kafka<br/>:9092"]
      OTEL["OTel Collector<br/>:4318"]
    end
    
    U --> SH
    SH --> AS
    AD --> ADS
    PD --> OS & PS & WS & SS & RS
    ED --> MS & ES & WS
    SH -.v2 multi-role.-> AS
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS --> PG
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS --> RD
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS --> KF
    AS & OS & MS & MES & PS & RS & SS & US & ES & WS & ADS -.OTLP.-> OTEL
```

### 2.2 关键交互流（v2 patient 下单 + escort 接单）

```
unified-app (patient 域 active)
  └─► auth-service              SMS 登录 / 拿 JWT (active=patient)
  └─► user-service              拉地址 / 优惠券 / 医院 / 套餐
  └─► order-service             创建订单（pgx SKIP LOCKED）
        └─► Kafka OrderCreated
              ├─► match-service      候选打分 + 抢单池（Redis SETNX）
              └─► message-service    站内信推送
        └─► payment-service        调起支付（mock channel）
              └─► Kafka PaymentCompleted
                    ├─► wallet-service   T+7 结算（scanner 1 min）
                    └─► review-service   待评价（订单完成 +24h）
        └─► sos-service            一键紧急联系（紧急通道）

unified-app (escort 域 active)
  └─► /api/v1/auth/switch-role  active=escort → 新 JWT
  └─► match-service             GET /match/feed（escort-only gate）
  └─► order-service             POST /orders/:id/confirm-accept
  └─► wallet-service            GET /escorts/me/wallet

unified-app (admin 域 active)
  └─► /api/v1/auth/switch-role  active=order_admin → 新 JWT
  └─► admin-service             GET /admin/orders（admin RoleAuthWithKey）
  └─► wallet-service            POST /admin/wallet/withdrawals/:id/approve
```

### 2.3 v1 → v2 迁移路线图

| 阶段 | 状态 | 工作 |
| --- | :-: | --- |
| 阶段 0–4：后端骨架 + auth + order + match | ✅ | §0–§4 |
| v1.3–v1.5：生产化（OTel / Prometheus / 部署） | ✅ | §26–§35 |
| v1.6：三端 e2e 链路完整化 | ✅ | §36–§39 |
| v2 Phase 1：后端 multi-role + /switch-role | ✅ | §42（9 commits） |
| v2 Phase 2 spike：unified-app 骨架 + e2e | ✅ | §43（3 commits） |
| **v2 Phase 3.0：unified-app 基础设施建设** | ✅ | **§45（16 commits）** |
| v2 Phase 3.1：patient 域 28 pages 迁移 | ⏳ 待启动 | dev.md §45.5 |
| v2 Phase 3.2：admin 域 51 pages 重写 | ⏳ | plan §3.2 |
| v2 Phase 3.3：escort 域 20 pages 重写 | ⏳ | plan §3.3 |
| v2 Phase 4：旧项目归档 + 移除 v1 Claims.Role | ⏳ | plan §3.3 |

---

## 3. 服务清单

### 3.1 后端微服务（11 个 Go 服务 · v2 多角色守卫）

| 服务             | 端口 | 职责                                                                 | 角色守卫（v2） | Plan                                                                                         |
| ---------------- | ---- | -------------------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------- |
| `auth-service`   | 8081 | 手机号 / 微信登录 / 实名认证 / JWT 签发 / **/switch-role / multi-role** | —（签发方）  | [dev.md §3](./dev.md#3-阶段-2--auth-service--已完成) + [§42](./dev.md#42-unified-app-v2--phase-1-后端--跨端契约同步完成9-commits) |
| `order-service`  | 8082 | 订单创建 / 抢单（SKIP LOCKED）/ 锁单（Redis SETNX）/ 状态机 / 调度  | patient/escort | [dev.md §4](./dev.md#4-阶段-3--order-service--已完成) + [§42](./dev.md#42-unified-app-v2--phase-1-后端--跨端契约同步完成9-commits) |
| `match-service`  | 8083 | 候选打分 / 抢单池（NopPool / RedisPool）                            | escort | [dev.md §5](./dev.md#5-阶段-4--match-service--已完成) + [§41](./dev.md#41-match-service-端到端跑通-阶段-4-task-41-46-收尾) |
| `message-service`| 8084 | 站内信 / WebSocket / Kafka publish                                  | — | [dev.md §20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan) |
| `payment-service`| 8085 | 支付下单 / 退款 / 退款策略 / Channel 抽象                          | — | [dev.md §24](./dev.md#24-payment-service-补全--11--healthz-flag--dockerfile-healthcheck2026-09-24-ops-部署补全) |
| `review-service` | 8086 | 双向评价 / 标签 / 商家回复                                          | — | [dev.md §20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan) |
| `sos-service`    | 8087 | 一键呼叫 / 位置上报 / 紧急联系 / 30s 频率去重                       | — | [dev.md §20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan) |
| `user-service`   | 8088 | 资料 / 实名 / 地址 / 优惠券 / 医院 / 套餐 / 虚拟号                  | — | [dev.md §19](./dev.md#19-user-service-5-模块落地2026-09-24-address-coupon--hospital-package--virtual-number-plan) |
| `escort-service` | 8089 | 陪诊师注册 / 实名 / 接单设置 / 排班 / 可用性（availability 子包）    | escort | [dev.md §20](./dev.md#20-4-服务补全-handler--router--main-接入2026-09-24-messagesosreviewescort-plan) |
| `wallet-service` | 8090 | 余额 / 冻结 / 提现 / 流水 / T+7 scanner                              | patient/escort/admin | [dev.md §16](./dev.md#16-后端-wallet-t7-结算服务2026-09-24-wallet-plan-w1-w5) |
| `admin-service`  | 8091 | 后台：工单 / 报表 / 内部服务代理（order / review / escort / user）   | super_admin + 5 admin_* | [dev.md §10](./dev.md#10-模块解耦与新骨架服务2026-09-24) + [§42](./dev.md#42-unified-app-v2--phase-1-后端--跨端契约同步完成9-commits) |

### 3.2 前端应用（v1 三端 + v2 unified-app 合并目标）

| 前端                | 版本 | 端口 | 技术栈 | 状态 |
| ------------------- | ---: | ---- | ------ | --- |
| `patient-miniapp`   | v1   | 80   | uni-app + Vue 3 + uView Plus + Pinia | ✅ 已完成（v1.1 + multi-platform）→ 待迁移至 unified-app |
| `escort-app`        | v1   | 8080 | Flutter 3.24+（web / iOS / Android） | ✅ 已完成（12 pages + Flutter Web PWA）→ 待重写为 Vue |
| `admin-web`         | v1   | 8092 | Vite + React 18 + TypeScript + antd | ✅ 已完成（51 pages + MSW + Playwright e2e）→ 待重写为 Vue |
| **`unified-app`**   | **v2** | **5174** | **uni-app + Vue 3 + Pinia + uView Plus** | 🚧 **§45 Phase 3.0 完成（基础设施 + HomeShell）；Phase 3.1 patient 域迁移待启动** |

**unified-app 当前进度（Phase 3.0 收官）**：
- 测试基础设施：Vitest + jsdom + Vue Test Utils（**194 tests passed**）
- 设计系统：58 design tokens（TS 单一真相源 + CSS 变量运行时注入）
- 通用组件：UiButton / UiCard / UiInput / UiEmpty / UiLoading / UiModal（**6 个**组件 + 单测）
- 域组件：DomainSwitcher / RoleSwitcherModal（**2 个**组件 + 单测）
- API 扩展层：10 client × **60 端点** × 75 TS 类型
- 业务 store：orderStore（fetchList / fetchDetail / create / cancel / confirm / reject / finish + upsert）
- e2e spike：3 个 Playwright 测试（HomeShell 渲染 + 角色切换弹层 + 域可用性）
- 重大决策：**SCSS → CSS 变量重构**（uni-app sass-loader 集成 issue）

详细：[dev.md §45](./dev.md#45-unified-app-phase-30-基础设施建设收官2026-09-29)

---

## 4. 目录结构

```
doctors/
├── services/                  # 11 个 Go 后端微服务
│   ├── auth/      cmd/main.go + internal/{handler,router,server,service,realname,sms,wxlogin,repo,middleware}
│   ├── order/     cmd/main.go + internal/{handler,router,server,service,repo,events,scheduler,state,middleware}
│   ├── match/     cmd/main.go + internal/{handler,router,server,service,pool,consumer,scorer,middleware}
│   ├── message/   cmd/main.go + internal/{handler,router,server,service}
│   ├── payment/   cmd/main.go + internal/{handler,router,server,service}
│   ├── review/    cmd/main.go + internal/{handler,router,server,service}
│   ├── sos/       cmd/main.go + internal/{handler,router,server,service}
│   ├── user/      cmd/main.go + internal/{handler,address,coupon,hospital,pkg,virtualnumber,service}
│   ├── escort/    cmd/main.go + internal/{handler,availability,service}
│   ├── wallet/    cmd/main.go + internal/{handler,service,repo}
│   └── admin/     cmd/main.go + internal/{handler,service,repo,clients,events}
│
├── shared/                     # 跨服务通用包（13 个）
│   ├── config/                  # viper 配置加载（DOCTORS_<SVC>_* env override）
│   ├── logger/                  # zap 结构化日志 + trace_id 上下文传递
│   ├── tracing/                 # OTel 全链路追踪（OTLP HTTP + W3C TraceContext）
│   ├── metrics/                 # Prometheus 业务指标（HTTP / DB / Kafka）+ /metrics 抓取端
│   ├── health/                  # 健康检查 + readiness 探针（PG / Redis / Kafka）
│   ├── auth/                    # JWT 签发 / 校验（含 v2 multi-role：Roles[] + Active）
│   ├── httpx/                   # 统一响应 / 错误码
│   ├── middleware/              # Gin 中间件（Auth / RoleAuthWithKey / Metrics / Cors / Logging / Recovery / RateLimit）
│   ├── contracts/               # 共享事件契约（OrderCompleted / PaymentRefunded / ...）
│   ├── db/                      # pgxpool 封装 + 健康检查 + WithTx
│   ├── redis/                   # go-redis 封装 + 分布式锁
│   ├── kafka/                   # kafka-go 封装（Producer / Consumer）
│   ├── idempotency/             # 幂等键（Store 接口 + 规范化）
│   └── errs/                    # 业务错误码表
│
├── migrations/                 # 数据库迁移（16 份 SQL · 0001~0016）
│   ├── 0001_users.up.sql / down.sql
│   ├── ...（中间 14 份订单 / 钱包 / 地址 / 优惠券 / 医院 / 套餐 / 虚拟号）
│   ├── 0015_dev_seed.up.sql     # dev 环境种子数据（13 users + 医院 + 套餐）
│   ├── 0016_users_roles_jsonb.up.sql   # v2 multi-role：roles JSONB + GIN 索引
│   └── migrations_test.go       # golang-migrate 集成测试
│
├── frontend/                   # 4 个前端工程
│   ├── unified-app/            # v2 单 codebase 合并目标（Phase 3.0 完成）
│   │   ├── src/
│   │   │   ├── api/            # 10 client × 60 端点 × 75 类型（auth/orders/match/payment/wallet/user/review/sos/message/admin/escort）
│   │   │   ├── components/shared/  # UiButton/UiCard/UiInput/UiEmpty/UiLoading/UiModal + DomainSwitcher/RoleSwitcherModal/RoleGuard
│   │   │   ├── pages/          # home（HomeShell 已完成） + patient/escort/admin（待迁移）
│   │   │   ├── store/          # auth（§43） + order（§3.0.4）
│   │   │   ├── styles/         # tokens.ts（58 项 + generateCssVarsBlock） + tokens.scss + global.scss
│   │   │   ├── types/auth.ts   # Role 枚举 + Me/Login/SwitchRole response
│   │   │   ├── test/           # smoke.spec.ts
│   │   │   ├── App.vue         # 动态注入 :root CSS 变量 + auth.bootstrap
│   │   │   ├── main.ts         # Pinia 装配
│   │   │   ├── pages.json      # uni-app 路由
│   │   │   └── uni.scss        # uni-app 主题色
│   │   ├── e2e/                # Playwright spike（HomeShell + 角色切换 + 域可用性）
│   │   ├── test/setup.ts       # StorageMock + uni.* 全局 mock
│   │   ├── vitest.config.ts
│   │   ├── vite.config.js      # ESM interop + 18 条 proxy 规则
│   │   └── package.json
│   ├── admin-web/              # Vite + React 18 + TS + antd（v1 · 待重写为 Vue）
│   ├── patient-miniapp/        # uni-app + Vue 3 + uView Plus（v1 · 待迁移）
│   └── escort-app/             # Flutter 3.24+（v1 · 待重写为 Vue）
│
├── config/                     # 11 个服务 yaml 配置（DOCTORS_<SVC>_* env override）
├── docs/                       # 9 份需求文档 + 17 plan + 6 spec（mkdocs-material 部署）
├── scripts/                    # 运维 / 测试 / 部署脚本
├── deploy/                     # K8s manifests（按需启用，本期未生成）
├── .github/workflows/ci.yml    # GitHub Actions CI（5 jobs：backend-test / docker-build / frontend-lint / backend-lint / docs-build）
├── pages.yml                    # GitHub Actions Pages（mkdocs-material → growdu.github.io/doctors/）
├── docker-compose.yml          # 本地 dev 中间件（PG + Redis + Kafka）
├── docker-compose.deploy.yml   # 全量部署编排（11 Go + 3 前端 + 3 中间件 + jaeger + otel）
├── Makefile                    # 顶层命令（test / build / docker-up / down）
├── go.mod / go.sum             # 单一 go module
├── mkdocs.yml                  # docs/ 站点配置（--strict）
├── dev.md                      # 增量开发日志（§0~§45 · 4233 行）
└── README.md                   # 本文件
```

---

## 5. 本地开发

### 5.1 前置条件

| 工具         | 版本        | 说明                                            |
| ------------ | ----------- | ----------------------------------------------- |
| Go           | **1.24+**   | toolchain go1.24.3；项目 go.mod 锁定             |
| Node.js      | **20+**     | admin-web / patient-miniapp / **unified-app** dev |
| pnpm         | 9+          | admin-web + **unified-app** 包管理（corepack 自动启用） |
| Flutter      | **3.24+**   | escort-app；Dart SDK >= 3.5（v1 only） |
| Docker       | 24+         | 本地 PG / Redis / Kafka + 部署编排              |
| Docker Compose v2 | -    | `docker compose`（非 `docker-compose`）         |
| 端口空闲     | -           | 5432 / 6379 / 9092 + 8081~8091 + 80 / 8080 / 8092 / 5174 |

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

#### v2 unified-app（合并目标）

```bash
cd frontend/unified-app
corepack enable && corepack prepare pnpm@8.15.9 --activate   # §44 fix：corepack 优于 pnpm/action-setup
pnpm install
pnpm dev:h5                  # → http://localhost:5174（h5 dev server）
pnpm typecheck               # vue-tsc --noEmit
pnpm test                    # vitest run（194 tests）
pnpm test:watch              # 监听模式
pnpm build:h5                # 生产构建（dist/build/h5/）
```

**dev:h5 跨域**：vite.config.js 自动 proxy 11 后端服务（`/api/v1/*` → `:8081~:8091`）

#### v1 三端（迁移期维护）

```bash
cd frontend/admin-web
pnpm install && pnpm dev          # → http://localhost:5173（React）

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
2. **unified-app 单元 + 组件测试**：
   ```bash
   cd frontend/unified-app
   pnpm install && pnpm test       # → vitest run（194 tests）
   ```
3. **admin-web 单测**：`cd frontend/admin-web && pnpm install && npx vitest run`
4. **patient-miniapp 单测**：`cd frontend/patient-miniapp && npm install && npm test`
5. **escort-app 单测**：`cd frontend/escort-app && flutter pub get && flutter test`
6. **集成测试**：`go test -tags=integration ./migrations/... ./services/...`（需 `docker compose up -d`）

每个步骤独立 try/skip，缺失工具不阻塞整体；输出彩色汇总（绿/红）+ 每步耗时。

### 6.2 手动命令

```bash
make test                 # 后端单元测试
make test-integration     # 集成测试（需 docker compose up）

# 单独跑某包
go test -race -count=1 ./services/auth/...
go test -race -count=1 ./shared/tracing/...

# unified-app 单独跑
cd frontend/unified-app
pnpm test                # vitest run（默认 CI 模式）
pnpm test:watch          # 开发模式
pnpm test:coverage       # 覆盖率报告（@vitest/coverage-v8）
```

### 6.3 unified-app 测试覆盖

| 类别 | 用例数 |
| | ---: |
| 测试基础设施 smoke | 5 |
| design tokens 验证 | 18 |
| SCSS ↔ TS 同步 | 8 |
| UiButton / UiCard / UiInput / UiEmpty / UiLoading / UiModal | 60 |
| DomainSwitcher / RoleSwitcherModal | 15 |
| api/*（orders / match / payment / wallet / user / review / sos / message / admin / escort） | 70 |
| orderStore | 14 |
| **总计** | **194** |

---

## 7. 部署

### 7.1 Docker 全栈（推荐）

> 入口：`docker-compose.deploy.yml`
> 编排范围：3 中间件 + 11 Go 后端 + 3 前端（v1）+ 1 网络 + 3 持久卷

> ⚠️ **v2 unified-app 暂未集成部署**：Phase 3.0 仅完成基础设施，业务域迁移未完成（Phase 3.1+）。
> 部署前需完成 Phase 4 旧项目归档 + `frontend/_archive_v1/` 整理。

#### 前置条件

- Docker Engine ≥ 24 + Docker Compose v2
- 宿主机空闲端口：5432 / 6379 / 9092（中间件）+ 8081~8091（Go 服务）+ 80 / 8080 / 8092（v1 前端）

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
                └→ 3 v1 前端服务（nginx，无外部依赖）
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
| 80    | patient-miniapp H5（v1）| 8089  | escort-service        |
| 8080  | escort-app (Flutter web · v1) | 8090  | wallet-service |
| 8081  | auth-service            | 8091  | admin-service         |
| 8082  | order-service           | 8092  | admin-web（v1）       |
| 8083  | match-service           | **5174** | **unified-app dev:h5（v2）** |
| 8084  | message-service（重复）  | **5173** | admin-web dev（v1） |

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

---

## 9. v2 角色权限模型（§42）

### 9.1 JWT Claims 演进

```go
// v1（已弃用，待 Phase 4 移除）
type Claims struct {
    UserID int64
    Role   string  // 单字段
    UnionID string
    exp, iat, iss
}

// v2（当前）
type Claims struct {
    UserID   int64
    Role     string  // v1 兼容：取 Active
    Roles   []string // 多角色数组
    Active   string  // 当前激活角色
    UnionID  string
    exp, iat, iss
}
```

### 9.2 角色枚举

| 角色 | 描述 | 域 |
| --- | --- | --- |
| `patient` | 患者 | patient 域 |
| `escort` | 陪诊师 | escort 域 |
| `super_admin` | 超级管理员 | admin 域（全部权限） |
| `order_admin` | 订单管理员 | admin 域（订单/退款） |
| `refund_admin` | 退款管理员 | admin 域（退款审批） |
| `cs` | 客服 | admin 域（工单） |
| `audit_admin` | 审核管理员 | admin 域（陪诊师审核） |
| `viewer` | 只读账号 | admin 域（只读） |

### 9.3 角色守卫（RoleAuthWithKey）

```go
// shared/middleware/role.go
// 三级回退：v2 严格 → v2 兼容 → v1 fallback
func RoleAuthWithKey(roleKey string, allowedRoles ...string) gin.HandlerFunc
```

- **v2 严格**：Claims.Active 非空 → 仅 active ∈ allowedRoles 通过
- **v2 兼容**：Active 空 → 回退 v1 路径（ctx[roleKey] 或 cl.Role）
- **v1 fallback**：无 Claims → ctx[roleKey] 单 role

各服务用自己的 ctx-key：
- `services/auth/internal/middleware`：`key = 'auth_role'`
- `services/admin/internal/router`：`key = 'role'`（shared 默认）
- `services/order/internal/middleware`：`key = 'order_role'`
- `services/match/internal/middleware`：`key = 'match_role'`

### 9.4 切换角色（/api/v1/auth/switch-role）

```http
POST /api/v1/auth/switch-role
Authorization: Bearer <multi-role-token>
Content-Type: application/json

{ "active": "escort" }
```

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "token": "<new-token-with-active=escort>",
    "user_id": 1,
    "active": "escort",
    "roles": ["patient", "escort"]
  },
}
```

### 9.5 数据模型（migration 0016）

```sql
-- users.roles JSONB 列 + GIN 索引
ALTER TABLE users ADD COLUMN roles JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE INDEX idx_users_roles_gin ON users USING gin (roles);

-- AddRole SQL（UNION ALL + jsonb_agg(DISTINCT v) 去重追加）
UPDATE users SET roles = jsonb_build_array(role) WHERE roles = '[]'::jsonb;
```

### 9.6 完整切换链路

```bash
# 1. SMS 登录拿 v1 token
curl -X POST :8081/api/v1/auth/sms/send -d '{"phone":"139xxx"}'
curl -X POST :8081/api/v1/auth/login -d '{"type":"sms","phone":"139xxx","code":"xxx"}'
# → token1 (v1 单 role)

# 2. DB 加 escort 角色（运营审核通过场景）
psql -c "UPDATE users SET roles = roles || '[\"escort\"]'::jsonb WHERE id = 12;"

# 3. 用 multi-role token 切换 active
TOK_MULTI=$(go run tools/sign-token/main.go -uid 12 -roles patient,escort -active patient)
curl -X POST :8081/api/v1/auth/switch-role \
  -H "Authorization: Bearer $TOK_MULTI" -d '{"active":"escort"}'
# → token2 active=escort, roles=[patient,escort]

# 4. 前端切到 escort 域：unified-app home shell → /match/feed 可见
curl -H "Authorization: Bearer $token2" :8083/api/v1/match/feed?order_id=1
# → 200 escort 候选列表
```

---

## 10. 文档站

`docs/` + `mkdocs.yml` 配置 mkdocs-material 站点（[§44](./dev.md#44-ci--github-pages-完善2026-09-28)）。

- 在线：[https://growdu.github.io/doctors/](https://growdu.github.io/doctors/)
- 包含：9 章设计文档（`docs/01-项目概述.md` ~ `docs/09-验收与发布.md`）+ 17 plan + 6 spec + 图表
- `--strict` 模式：broken nav / missing anchor 立即失败（0-tolerance 维护纪律）
- 触发：push 到 `main` 时 `.github/workflows/pages.yml` 自动 build + deploy

---

## 11. CI/CD（§44 + §45）

### 11.1 GitHub Actions（`ci.yml`）

5 jobs：
- `backend-test`：9/11 服务（auth/admin/user/sos/escort/message/review/payment/wallet）✅ success
- `backend-test (order/match)`：依赖 PG/Redis/Kafka 真跑，pre-existing fail
- `backend-lint`：golangci-lint v1.61 vs 项目 rule 不匹配，pre-existing fail
- `frontend-lint (unified-app)`：corepack + pnpm + typecheck + build:h5 ✅ success（§44.2.1 修复 4 commits）
- `frontend-lint (admin-web/patient-miniapp/escort-app)`：npm/pnpm/Flutter 链路差异，pre-existing fail
- `docs-build`：mkdocs --strict ✅ success（§44 新增）
- `docker-build`：多阶段 cache-from 兼容性问题，pre-existing fail（14 镜像 11 fail）

### 11.2 GitHub Pages（`pages.yml`）

mkdocs-material 构建 `site/` → [https://growdu.github.io/doctors/](https://growdu.github.io/doctors/)

§44.2.1 修复 4 commits：
- pymdown-extensions 10.6.1 → 10.7.1（PyPI typo）
- matrix `setup: node-pnpm` + corepack（link: 协议 + PATH 继承）
- `goTo(path)` 替代 `uni.navigateTo` inline expression（vue-tsc 不识别 `uni` 全局）
- 去掉 `--no-audit --no-fund`（npm 专属）

### 11.3 unified-app CI 设计（§44.6.4）

- matrix.setup 双值 `node` / `node-pnpm`：清晰区分两类前端包管理器
- corepack 优于 pnpm/action-setup：避免 step 上下文 PATH 继承问题
- pnpm 8.15.9 是 admin-web 已锁定的兼容版本（packageManager 字段）

---

## 12. 持续集成 + 测试覆盖率目标

| 维度 | 当前 | 目标 |
| --- | :-: | :-: |
| 后端服务数 | 11/11 | 11/11 ✅ |
| 后端单测覆盖 | ~80% | ≥ 80% |
| 后端集成测试（PG/Redis/Kafka 真跑） | ✅（§40-41） | 全服务覆盖 |
| unified-app 组件库测试 | 194 ✅ | ≥ 200（Phase 3.1 后） |
| unified-app e2e（Playwright） | spike ✅ | 全域覆盖（Phase 3.1 后） |
| admin-web 单测 | 130/162 | ≥ 150 |
| admin-web e2e | ✅ Playwright | CI 集成待办（§44.7） |
| patient-miniapp 单测 | ✅ | CI 集成待办（§44.7） |
| escort-app 单测 | ✅ | CI 集成待办（§44.7） |
| CI 后端 test（auth/）） | ✅ success | 全服务 |
| CI backend-test order/match | ❌ fail | 修复 dev fake 模式 |
| CI docker-build | ❌ fail | cache-from 兼容 |
| 文档站 | ✅ Pages 部署 | mkdocs --strict |

---

## 13. 许可

MIT License — 详见 [LICENSE](./LICENSE)。