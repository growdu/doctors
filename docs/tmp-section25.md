---
## 25. GitHub Actions CI 全栈（2026-09-24 ops 部署补全）

**目标**：把 11 Go 服务 + 3 前端 + distroless Dockerfile + -healthz flag + Pages 部署全部串到 GitHub Actions CI——一键跑 go test + docker build + frontend lint + Pages 部署。

**3 个 commit**：

| commit | 内容 | 文件 |
| :-- | :-- | :-- |
| `a0c6820` | `ci: upgrade ci.yml to 4-job matrix pipeline` | `.github/workflows/ci.yml` (134 → 235 行)|
| `658e0f9` | `docs(pages): upgrade to direct docs/ upload via actions/deploy-pages` | `.github/workflows/pages.yml` (新增) + `docs.yml` (删除) |
| `5cefd44` | `chore: add ISSUE_TEMPLATE + dependabot.yml + CODEOWNERS` | 6 个仓库维护文件 |

**Commit 1：ci.yml 4-job matrix pipeline**

1. **backend-test**：matrix 11 Go 服务（auth/order/match/message/payment/review/sos/user/escort/wallet/admin），跑 `go vet` + `go test -race -count=1 -timeout=120s`，fail-fast: false
2. **docker-build**：matrix 14 镜像（11 Go + 3 前端），`docker/setup-buildx-action@v3` + `actions/cache@v4`（local）+ `docker/build-push-action@v5`（push: false / load: true），无 docker push
3. **frontend-lint**：matrix 3 端（admin-web / patient-miniapp / escort-app），按端差异 setup Node vs Flutter，统一 working-directory
4. **backend-lint**：单 job 跑 `golangci-lint v1.61.0`（curl 安装 + GOPATH/bin → GITHUB_PATH）

**Commit 2：pages.yml 直接 docs/ 上传**

- trigger：`push to main`（paths 限定 `docs/**` + `README.md` + `dev.md` + `.github/workflows/pages.yml`）+ `workflow_dispatch`
- permissions：`contents: read` + `pages: write` + `id-token: write`（OIDC 无 PAT）
- concurrency：`group=pages, cancel-in-progress: false`
- build：actions/upload-pages-artifact@v3（path=docs/）
- deploy：actions/deploy-pages@v4，environment=github-pages

**Commit 3：仓库维护文件**

- `.github/ISSUE_TEMPLATE/bug_report.md`（标题/复现/期望/实际/截图/环境/影响/根因 + label bug）
- `.github/ISSUE_TEMPLATE/feature_request.md`（痛点/建议/替代/影响/优先级/验收/参考 + label enhancement）
- `.github/ISSUE_TEMPLATE/config.yml`（blank_issues_enabled: false，启用 Discussions + Security 链接）
- `.github/dependabot.yml`（version 2，4 ecosystem gomod/npm/pip/github-actions，weekly 周一 09:00 Asia/Shanghai，open-pull-requests-limit: 5，labels + groups）
- `.github/CODEOWNERS`（11 services/* → @backend-team；3 frontend/* → @frontend-team；docs/ + *.md → @docs-team）

**未做（留给后续）**：

- 未推送（按约束）
- 未在 GitHub enable Pages / 创建 `github-pages` environment（需仓库侧手动）
- 未替换 CODEOWNERS 占位 team 名为真实 GitHub team slug
- 未添加 CodeQL workflow（任务标"可选"）

**端到端 v1.3 目标全部就位**：

- 11 Go 服务 + 3 前端 = 14 镜像 + 3 中间件 = 17 容器（docker-compose.deploy.yml）
- 14 Dockerfile（distroless < 30MB + -healthz flag）
- 61 包 0 FAIL + 100+ 集成测试契约样
- GitHub Actions CI 4 job + Pages 自动部署 docs/
- 仓库维护（issue 模板 / dependabot / CODEOWNERS）

### 累计交付（v1.3 收官）

| 阶段 | 提交数 | 测试 |
| :-- | :--: | :--: |
| 后端 11 Go 服务 | ~80 | 800+ |
| 前端 3 端 | ~50 | 300+ |
| 文档 / plans / specs | ~30 | — |
| 部署 / CI | 15 | — |
| **合计** | **~175** | **~1100** |
