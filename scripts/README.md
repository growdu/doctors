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

## 增量

| 脚本                       | 阶段    | 用途                                |
| -------------------------- | ------- | ----------------------------------- |
| `run-tests.sh` / `.ps1`    | §26     | 一键全栈测试（后端 + 3 前端 + 集成）|
| `smoke-*.sh`               | §20+    | 单服务 build + 启动 + 路由验证      |