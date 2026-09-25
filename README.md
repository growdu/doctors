# 陪诊师平台 · Doctors

面向"陪诊师 + 患者"双边市场的预约 / 匹配 / 支付 / 评价 / 钱包 / SOS 一体化平台。
本仓库为 monorepo，统一 Go 后端 + 3 个前端应用。

> 项目完整设计稿见 [`docs/`](./docs/)；开发进度跟踪见 [`dev.md`](./dev.md)。

---

## 1. 项目结构

```
.
├── services/             # 11 个 Go 后端微服务（auth/order/match/message/payment/...）
│   └── <svc>/
│       ├── cmd/main.go   # 服务入口
│       └── internal/     # handler / router / server / service / repo / ...
├── shared/               # 跨服务公共包（config / logger / httpx / auth / db / redis / kafka / contracts / middleware）
├── migrations/           # 数据库迁移（0001~0014）
├── frontend/             # 3 个前端工程（patient-miniapp / escort-app / admin-web）
├── config/               # 各服务 yaml 配置（DOCTORS_<SVC>_* 环境变量可覆盖）
├── docker-compose.yml           # 本地 dev 中间件（PG / Redis / Kafka）
└── docker-compose.deploy.yml    # 全量部署编排（11 Go + 3 前端 + 3 中间件）
```

---

## 2. 服务清单

| 服务             | 端口 | 说明                                                 |
| ---------------- | ---- | ---------------------------------------------------- |
| auth-service     | 8081 | 鉴权：手机号 / 微信登录 / 实名认证 / JWT              |
| order-service    | 8082 | 订单：创建 / 抢单 / 锁单 / 状态机 / 调度              |
| match-service    | 8083 | 匹配：候选打分 / 抢单池                               |
| message-service  | 8084 | 消息：站内信 / WebSocket                              |
| payment-service  | 8085 | 支付：下单 / 退款 / 退款策略                          |
| review-service   | 8086 | 评价：双向评价 / 标签                                 |
| sos-service      | 8087 | SOS：一键呼叫 / 位置上报 / 紧急联系                   |
| user-service     | 8088 | 用户：资料 / 实名 / 健康档案                          |
| escort-service   | 8089 | 陪诊师：注册 / 实名 / 接单设置 / 排班                 |
| wallet-service   | 8090 | 钱包：余额 / 冻结 / 提现 / 流水                       |
| admin-service    | 8091 | 后台：工单 / 报表 / 内部服务代理                      |

| 前端                | 端口 | 技术栈                          |
| ------------------- | ---- | ------------------------------- |
| patient-miniapp     | 80   | uni-app + Vue 3 + uView Plus    |
| escort-app          | 8080 | Flutter 3.24（web bundle）      |
| admin-web           | 8092 | Vite + React 18 + TypeScript    |

---

## 3. 本地开发

### 3.1 启动中间件

```bash
docker compose up -d              # PG + Redis + Kafka
# 或：make docker-up
```

### 3.2 应用数据库迁移

```bash
for f in migrations/*.up.sql; do
  psql "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable" -f "$f"
done
```

### 3.3 启动单个 Go 服务

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
```

### 3.4 单元 / 集成测试

```bash
make test                   # 单元测试（需 PG health）
make test-integration       # 集成测试（需 docker compose up）
```

---

## 4. 部署（Docker 全栈）

> 入口：`docker-compose.deploy.yml`
> 编排范围：3 中间件 + 11 Go 后端 + 3 前端 + 1 网络 + 3 持久卷

### 4.1 前置条件

- Docker Engine ≥ 24
- Docker Compose v2（`docker compose` 而非 `docker-compose`）
- 宿主机空闲端口：`5432 6379 9092`（中间件） + `8081~8091`（Go 服务） + `80 8080 8092`（前端）

### 4.2 一键构建 + 启动

```bash
# 在仓库根目录
docker compose -f docker-compose.deploy.yml build           # 构建 14 个镜像（首次较慢）
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

### 4.3 常用运维命令

```bash
# 查看日志
docker compose -f docker-compose.deploy.yml logs -f --tail=200 auth-service

# 重启单个服务
docker compose -f docker-compose.deploy.yml restart order-service

# 进入容器调试
docker compose -f docker-compose.deploy.yml exec auth-service /app/server -h

# 滚动更新（修改代码后）
docker compose -f docker-compose.deploy.yml build auth-service
docker compose -f docker-compose.deploy.yml up -d auth-service

# 停止 + 删除容器（保留数据）
docker compose -f docker-compose.deploy.yml down

# 停止 + 删除容器 + 卷（清空所有数据）
docker compose -f docker-compose.deploy.yml down -v
```

### 4.4 配置注入

每个 Go 服务通过 `DOCTORS_<UPPER_SVC>_*` 环境变量注入；yaml（`config/<svc>.yaml`）通过 `./config:/app/config:ro` 挂载作为默认值：

| 环境变量                           | 含义                              | 示例                                                |
| ---------------------------------- | --------------------------------- | --------------------------------------------------- |
| `DOCTORS_<SVC>_HTTP_ADDR`          | HTTP 监听地址                     | `:8080`                                             |
| `DOCTORS_<SVC>_DB_DSN`             | PostgreSQL DSN                    | `postgres://doctors:doctors@postgres:5432/doctors`  |
| `DOCTORS_<SVC>_REDIS_ADDR`         | Redis 地址                        | `redis:6379`                                        |
| `DOCTORS_<SVC>_KAFKA_BROKERS`      | Kafka broker 列表（逗号分隔）      | `kafka:9092`                                        |
| `DOCTORS_<SVC>_KAFKA_GROUP_ID`     | 消费者组 ID                       | `auth-service`                                      |
| `DOCTORS_<SVC>_JWT_SECRET`         | JWT 签名密钥（**生产必须改**）    | `dev-secret-change-me`                              |
| `DOCTORS_<SVC>_LOGGING_LEVEL`      | zap 日志级别                      | `debug` / `info` / `warn` / `error`                 |

> 修改 `config/<svc>.yaml` 后需要重启对应服务（`./config` 是 bind mount，文件改动即时生效，但服务需 reload）。

### 4.5 镜像与体积

- **Go 服务**：`gcr.io/distroless/static-debian12:nonroot` 运行时（< 30MB，无 shell，UID 65532）
- **前端服务**：`nginx:1.27-alpine`（含 wget，用于容器内健康探针）
- 镜像 tag：`doctors/<svc>:latest`，本地 build，可直接 push 到私有仓库后改 `image:` 段

### 4.6 健康检查说明

- 中间件（PG / Redis / Kafka）由官方镜像内置 healthcheck，`depends_on.condition: service_healthy` 等待其就绪
- 3 个前端 nginx 在 Dockerfile 内置 `HEALTHCHECK CMD wget -q --spider http://127.0.0.1/index.html`
- 11 个 Go 服务运行在 distroless（无 shell / 无 curl / 无 wget）；容器内 HTTP 探针需二进制支持 `-healthz` flag（**TODO**：下一阶段为各服务添加 `-healthz` 标志；当前依赖 `depends_on.condition: service_started` 在启动顺序上保守）

### 4.7 端口清单

| 端口  | 用途                |
| ----- | ------------------- |
| 5432  | PostgreSQL          |
| 6379  | Redis               |
| 9092  | Kafka               |
| 80    | patient-miniapp H5  |
| 8080  | escort-app (Flutter web) |
| 8081  | auth-service        |
| 8082  | order-service       |
| 8083  | match-service       |
| 8084  | message-service     |
| 8085  | payment-service     |
| 8086  | review-service      |
| 8087  | sos-service         |
| 8088  | user-service        |
| 8089  | escort-service      |
| 8090  | wallet-service      |
| 8091  | admin-service       |
| 8092  | admin-web           |

---

## 5. 仓库约定

- Go 1.24，module `github.com/growdu/doctors`
- 提交规范：`<scope>: <description>`（feat / fix / refactor / docs / test / ops / chore）
- 代码风格：`gofmt` + `go vet` + `golangci-lint`
- 测试：单元测试与生产代码同包（`xxx_test.go`）；集成测试 `-tags=integration`
- 提交前必跑：`make fmt vet test`

---

## 6. 进一步阅读

- [`docs/01-项目概述.md`](./docs/01-项目概述.md) ~ [`docs/09-验收与发布.md`](./docs/09-验收与发布.md)
- [`docs/REVIEW-REPORT.md`](./docs/REVIEW-REPORT.md)
- [`dev.md`](./dev.md) — 增量开发日志
- [`Makefile`](./Makefile) — 本地常用 target
- [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) — CI 配置