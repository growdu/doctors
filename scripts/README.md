# scripts/ · 运维与测试脚本

`scripts/` 下分两类脚本：**smoke**（单服务冒烟，按服务名 `smoke-<svc>.sh`）和 **run-tests**（跨平台一键全栈测试）。

## 一键全栈测试（`run-tests.sh` / `run-tests.ps1`）

### 用法

```bash
# Linux / macOS / WSL / Git Bash
bash scripts/run-tests.sh
bash scripts/run-tests.sh --skip-frontend       # 只跑后端
bash scripts/run-tests.sh --only=integration    # 只跑集成

# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1
powershell -File scripts/run-tests.ps1 -SkipFrontend
powershell -File scripts/run-tests.ps1 -Only integration
```

### 步骤

1. **后端单元测试**：`go test -race -count=1 -timeout=180s ./...`（约 200+ 用例）。
2. **admin-web**（Vitest）：`pnpm install && npx vitest run`。
3. **patient-miniapp**（Jest）：`npm install && npm test`。
4. **escort-app**（flutter test）：`flutter pub get && flutter test`。
5. **集成测试**：`go test -tags=integration ./migrations/... ./services/...`（需 `docker compose up -d`）。

### 输出格式

- 每个步骤带耗时（秒）+ 状态（`✓ PASS` / `✗ FAIL` / `- SKIP`）。
- 末尾汇总：total / passed / failed / skipped + FAILED 列表。
- 退出码：`0` = 通过或可跳过失败；`1` = 后端 / 集成 FAIL（CI 阻塞）。

## 跨平台注意

| 维度       | `run-tests.sh`                                 | `run-tests.ps1`                                |
| ---------- | ---------------------------------------------- | ---------------------------------------------- |
| 平台       | Linux / macOS / WSL / Git Bash                 | Windows PowerShell 5.1+ / PowerShell Core 7+  |
| 颜色       | ANSI（自动检测 `tput colors` ≥ 8）             | `[Console]::IsOutputRedirected` / `$Host.UI.SupportsVirtualTerminal` |
| 路径分隔符 | `/`                                            | 自动适配 `/` 或 `\`                            |
| Shell 特性 | `set -u`、`$(())` 算数、`[[ ]]`               | `[CmdletBinding()]` + `param()`                |
| 跳过标记   | `--skip-backend` `--only=integration`          | `-SkipBackend` `-Only integration`（无 `--`） |

## smoke-* 脚本（按服务冒烟）

5 个服务的端到端冒烟脚本（`scripts/smoke-{auth,order,match,wallet,admin}.sh`），用于单个服务 build + 启动 + 关键路由验证。
每个脚本独立运行；本地需先 `go build ./services/<svc>/cmd` 通过。

## smoke-e2e — 一键全栈端到端冒烟（11 服务）

> 跨平台入口 `scripts/smoke-e2e.sh`（bash / Git Bash / WSL）和 `scripts/smoke-e2e.ps1`（PowerShell 5.1+），输出格式对齐。

### 用法

```bash
# Linux / macOS / WSL / Git Bash
bash scripts/smoke-e2e.sh                    # 全流程（自动 docker compose up）
bash scripts/smoke-e2e.sh --skip-start       # 假设已起，只做端点巡检
bash scripts/smoke-e2e.sh --skip-api         # 跳过 API 调用（只跑 healthz/metrics）
bash scripts/smoke-e2e.sh --timeout=180      # 自定义单服务等待秒

# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts/smoke-e2e.ps1
powershell -File scripts/smoke-e2e.ps1 -SkipStart
powershell -File scripts/smoke-e2e.ps1 -SkipApi
powershell -File scripts/smoke-e2e.ps1 -WaitTimeout 180
```

### 流程

1. **启动 docker-compose**：默认执行 `docker compose -f docker-compose.deploy.yml up -d`；接着 sleep `BootWait`（默认 30s）。
2. **探测 /healthz**：对 11 服务（8081–8091）的 `/healthz` 端点循环探测，单服务等待上限 `WaitTimeout`（默认 120s）。
3. **探测 /metrics**：对 `/healthz` 已通的服务探测 `/metrics` 端点，确认返回 200 + Prometheus exposition 格式（含 `# HELP`）。
4. **API 烟测**：调用 admin `/api/v1/admin/escorts/pending-audit`（无 token 应得 401 + 业务码 11001），验证 RBAC 中间件挂载正确。
5. **汇总输出**：彩色 ✓ / ✗ / ! + 总计 / 通过 / 失败 / 跳过 + FAILED 列表。

退出码：`0` = 全通过；`1` = 任一 FAIL（容器**保持运行**便于排查）。

## 增量

| 脚本                          | 阶段    | 用途                                |
| ----------------------------- | ------- | ----------------------------------- |
| `run-tests.sh` / `.ps1`       | §26     | 一键全栈测试（后端 + 3 前端 + 集成）|
| `smoke-*.sh`                  | §20+    | 单服务 build + 启动 + 路由验证      |
| `smoke-e2e.sh` / `.ps1`       | §34+    | 一键全栈端到端冒烟（11 服务 healthz + metrics + API 鉴权）|