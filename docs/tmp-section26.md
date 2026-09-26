---
## 26. OTel 全链路追踪 + README 完整化 + 一键测试脚本（v1.3 优化）

**目标**：为生产化做准备——OTel 真正接通（替代占位 Noop）+ README 完整化（一键上手）+ 跨平台一键测试脚本。

**4 个 commit**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `1cd30bc` | `feat(tracing)` shared/tracing OTel SDK + 11 服务接入 | 27 |
| `b4e5475` | `docs(readme)` 完整化根 README.md | 1 |
| `4d94cfe` | `chore(scripts)` 跨平台 run-tests.{sh,ps1} | 3 |
| `8b7e864` | `fix(scripts)` run-tests.ps1 加 UTF-8 BOM | 1 |

**Commit 1：shared/tracing OTel SDK**

| 组件 | 内容 |
| :-- | :-- |
| `shared/tracing/tracing.go` (161 行) | `InitTracer(serviceName, otlpEndpoint)` + `StartSpan(ctx, name)` + `Inject/Extract` + `HeaderCarrier` (http.Header 适配) |
| `shared/tracing/tracing_test.go` (133 行) | 8 个单测（in-memory exporter 验证父子 span + W3C header round-trip + Noop 降级）|
| `shared/tracing/README.md` | 用法 + env 配置 |
| `go.mod` | OTel v1.32.0（5 个 direct：otel / sdk / otlptracehttp / trace / semconv）|
| `shared/config/loader.go` | `Tracing.OTLPEndpoint` 字段（默认空 → Noop 降级）|
| 10 × `config/<svc>.yaml` | 加 `tracing.otlp_endpoint: ""` 段 |
| 11 × `services/<svc>/cmd/main.go` | 在 `logger.SetLevel` 之前调 `InitTracer` + `defer traceShutdown` |

**InitTracer 用法**：

```go
traceShutdown, err := tracing.InitTracer("auth-service", cfg.Tracing.OTLPEndpoint)
if err != nil { log.Fatalf("init tracer: %v", err) }
defer func() { _ = traceShutdown(context.Background()) }()

ctx, span := tracing.StartSpan(ctx, "auth.login"); defer span.End()
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
tracing.Inject(ctx, tracing.HeaderCarrier(req.Header))
```

**8 个单测 PASS**：Noop 降级 + 真实 OTLP + http/https 前缀剥离 + shutdown 幂等 + span 父子 + W3C round-trip + TextMapCarrier 适配。

**Commit 2：README.md 完整化**

11 个章节：
1. 项目介绍（1 段 + tech badges）
2. 架构（Mermaid 总览 + 时序图）
3. 服务清单（11 Go + 3 前端表 + dev.md 锚链接）
4. 目录结构（monorepo 完整树）
5. 本地开发（前置 + 启动命令）
6. 跑测试（一键 + 手动）
7. 部署（docker-compose + 端口表 + env 注入）
8. 可观测性（zap + OTel + -healthz）
9. 贡献（commit 规范 + PR + CODEOWNERS）
10. License（MIT 示意）
11. 进一步阅读（docs/01~09 + dev.md 完整锚链接）

**Commit 3-4：scripts/run-tests.{sh,ps1}**

| 维度 | sh | ps1 |
| :-- | :-- | :-- |
| 平台 | Linux / macOS / WSL / Git Bash | Windows PS 5.1+ / PS Core 7+ |
| 步骤 | 5（后端 / admin-web / patient-miniapp / escort-app / integration）| 同 |
| 跳过 | `--skip-backend` / `--only=integration` | `-SkipBackend` / `-Only integration` |
| 颜色 | ANSI `tput colors >= 8` | `[Console]::IsOutputRedirected` |
| 缺工具 | `require` 函数 → SKIP | `Require-Tool` 函数 → SKIP |

**累计测试用例**：

- shared/tracing：**8 新增**（in-memory exporter + W3C + Noop）
- 其他 shared 包：不变
- 服务包：不变（commit1 不改业务逻辑）
- **全量回归 62 包 0 FAIL**（+1 包：shared/tracing）

**Plan 偏差**：

1. **OTel 版本 v1.32.0**：本机 Go 1.24.3，OTel v1.46+ 需 Go 1.25+；v1.32.0 是 Go 1.24 兼容最新
2. **in-memory exporter**：测试用 `sdk/trace/tracetest` 内存 exporter；生产可换 `otlptracehttp` 远程
3. **Noop 降级**：`OTLPEndpoint == ""` 时不注册 TracerProvider，span 不 IsRecording，零开销
4. **UTF-8 BOM 修复**：PS 默认 GBK 解析，中文乱码导致 parser 失败；加 BOM 后正确识别 UTF-8
5. **`-race` 标志在 Git Bash + Windows 下报 `0xc0000139`**：Go 1.24 race detector CGo DLL 在 Windows + Git Bash 加载失败（环境限制，与脚本无关）

**未做（留给后续）**：

- OTel metrics / logs SDK（spec 只要求 traces）
- Jaeger / Tempo collector 接入（`docker-compose.deploy.yml` 缺 collector 服务）
- **OTel ↔ 日志关联**（middleware 把 `trace_id` 注入 zap，让 `logger.FromContext` 自动带 trace）
- 生产采样策略（v1.32.0 默认 AlwaysSample；建议改 `TraceIDRatioBased(0.1)` + git sha 注入 service.version）

**端到端 v1.3 目标全部就位**：

- 11 Go 服务 + 3 前端 = 14 镜像 + 3 中间件（docker-compose.deploy.yml）
- 14 Dockerfile（distroless < 30MB + -healthz flag + HEALTHCHECK）
- 61 → 62 包 0 FAIL（+1 包：shared/tracing）
- OTel 全链路追踪（生产可接 Jaeger/Tempo）
- README 完整 + run-tests 跨平台一键脚本
- GitHub Actions CI 4 job + Pages + 仓库维护
