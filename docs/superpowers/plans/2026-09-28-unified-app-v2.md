# Unified App v2 — 3 端合并为 1 个 App（多角色 + SSO）

**作者**：Mavis（用户决策记录在 §0）
**日期**：2026-09-28
**对应 docs/superpowers/specs**：2026-09-24-roadmap-design.md（roadmap v1 → v2）

---

## 0. 决策记录（2026-09-28 用户会话）

| 决策点 | 选项 | 用户选择 | 备注 |
| ------ | ---- | -------- | ---- |
| 合并范围 | miniapp+escort / 全部 3 端 / miniapp only / 暂不合并 | **全部 3 端合并为 1 个 App** | admin-web + patient-miniapp + escort-app 全合并 |
| 技术栈 | uni-app all / Flutter all / Monorepo / 稍后定 | **稍后再定** | 本 plan 留技术栈选型章节；选型有 4 个候选，优劣 §3.1 |
| 推进策略 | design_only / backend_first / frontend_first / **full_rework** | **一次性完整重构** | 后端 + 全部前端 + 单测 + e2e 一次到位 |

**关键非决策**（用户明确表态）：
- 端不绑角色：3 端 UI 应该是同一份代码，登录后根据 token roles 切换
- 多角色并发：1 个用户可同时是 patient + escort + admin
- SSO：同账号一次登录 3 端都能用

---

## 1. 现状分析（v1 架构）

### 1.1 后端（11 Go service + 1 shared）

| 服务 | 端口 | 状态 | role 依赖 |
| ---- | ---- | ---- | -------- |
| auth-service | 8081 | ✅ 真跑 | JWT `Role string` 单字段（`shared/auth/jwt.go:22`） |
| order-service | 8082 | ✅ 真跑 | dev mode 用 devFakeUsers 注入 |
| match-service | 8083 | ✅ 真跑 | dev mode 用 devFakeEscortLoader |
| message / payment / review / sos / user / wallet / admin | 8084-8091 | 骨架齐未启 | 待 §35+ |

### 1.2 前端（3 端 + 3 技术栈）

| 端 | 项目 | 技术栈 | 文件数 | 主用场景 | role 绑定 |
| ---- | ---- | ------ | ------ | -------- | -------- |
| **admin-web** | `frontend/admin-web/` | Vite + React 18 + AntD 5 | 51 pages + 8 stores + 14 mocks handlers | PC 后台管理 | **硬绑定 6 admin role**（LoginPage 硬编码 username 前缀 → role） |
| **patient-miniapp** | `frontend/patient-miniapp/` | uni-app + Vue 3 + uView Plus + Pinia | 28 pages + 9 stores | 微信小程序 + H5 + Android + iOS 下单 | auth-service 注册时 role=patient（写死） |
| **escort-app** | `frontend/escort-app/` | Flutter 3.24+ | 79 dart files（含 test） | Flutter web + Android + iOS 接单 | auth-service 注册时 role=escort；profile 注释"陪诊师端恒为 escort" |

### 1.3 关键缺陷（用户提出的问题）

1. **角色锁端**：3 个 app 各自独立登录；1 个用户多角色要分别登录 3 次
2. **端能力未统一**：admin-web 仅 PC web；patient-miniapp 4 端；escort-app 3 端（无小程序）
3. **token 不共享**：3 端各自 localStorage / Provider 存 token；改密码要 3 处改
4. **role 单值**：后端 JWT 是单字符串 role，无法表达多角色并发
5. **角色切换无 UI**：登录后无"切换我是 patient 还是 escort"路径

### 1.4 当前真实价值 vs 重构价值

- ✅ 当前已实现：380+ commits、admin-web 单测 130/162、3 端 build、e2e 链路、PG/Redis/Kafka 真联通
- ❌ 缺失：端统一、角色切换、SSO、多角色并发
- 重构收益：用户体验提升（1 个账号 1 个 app 全场景）；DX 提升（3 个独立 codebase → 1 个）

---

## 2. 目标架构（Unified App v2）

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    Unified App（单 codebase）                     │
│  ┌───────────────┬───────────────┬───────────────┐               │
│  │   Admin 域   │  Patient 域   │  Escort 域    │               │
│  │ (PC 后台为主) │ (移动端为主)  │ (移动端为主)  │               │
│  │  admin-web   │ patient-mini- │ escort-app   │               │
│  │   视图       │ app 视图      │  视图        │               │
│  └───────┬───────┴───────┬───────┴───────┬───────┘               │
│          └───────────────┼───────────────┘                       │
│                  共享层（auth / api / store / theme）              │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼ JWT (roles []string + uid + unionid)
┌─────────────────────────────────────────────────────────────────┐
│              auth-service（核心升级：multi-role + SSO）             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│  │ /sms     │ │ /login   │ │ /switch  │ │ /me      │           │
│  │ /send    │ │ /refresh │ │ /role    │ │          │           │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘           │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
              11 个 Go service（auth 适配层 + role gate 中间件）
```

### 2.2 数据模型（user + roles）

```sql
-- users 表（既有 v1，单 role）
ALTER TABLE users ADD COLUMN roles JSONB NOT NULL DEFAULT '[]';
-- 兼容：v1.role 字段保留，新代码读 roles 数组
```

```go
// shared/auth/jwt.go
type Claims struct {
    UserID  int64    `json:"uid"`
    Roles   []string `json:"roles"`           // ← v2 新增：多角色并发
    Active  string   `json:"active"`          // ← v2 新增：当前激活角色
    UnionID string   `json:"unionid,omitempty"`
    jwt.RegisteredClaims
}
```

### 2.3 端能力矩阵（v2 目标）

| 端 | 编译目标 | admin 域 | patient 域 | escort 域 |
| ---- | -------- | -------- | ---------- | --------- |
| **Unified App**（技术栈待定，§3.1）| h5 + 微信小程序 + Android + iOS + PC web | ✅（PC web 完整）| ✅（移动端完整）| ✅（移动端完整）|

- 每个域的 UI 按目标设备差异化渲染：
  - admin 域：PC web 完整布局（侧边栏 + 多列表格）；移动端简化（折叠菜单 + 单列卡片）
  - patient 域：移动端优先（首页瀑布流 / 下单流程）；PC web 调整布局（多列卡片）
  - escort 域：移动端优先（接单列表 / 任务详情）；PC web 简化（仅订单管理）

### 2.4 角色切换流程

```
用户登录 → 拿到 JWT (roles=[patient, escort])
   ↓
进入 unified app 首页（home shell）
   ↓
侧边栏 / 顶部 tab 显示 "当前角色：patient"
   ↓
点切换 → 弹角色选择 modal → 选 escort
   ↓
调 POST /api/v1/auth/switch-role { active: "escort" }
   ↓
后端签发新 JWT（active=escort），返回新 token
   ↓
前端存新 token，路由跳 escort 域首页
```

### 2.5 路由守卫（v2）

```ts
// 统一守卫：检查 token.active 在路由要求的 roles 列表里
function RoleGuard({ requiredRoles, children }) {
  const active = useAuthStore(s => s.user?.active_role);
  if (!requiredRoles.includes(active)) return <Forbidden/>;
  return children;
}

// admin 域路由
<Route path="/admin/*" element={<RoleGuard requiredRoles={['super_admin','order_admin',...]}><AdminLayout/></RoleGuard>} />
// patient 域路由
<Route path="/patient/*" element={<RoleGuard requiredRoles={['patient']}><PatientLayout/></RoleGuard>} />
// escort 域路由
<Route path="/escort/*" element={<RoleGuard requiredRoles={['escort']}><EscortLayout/></RoleGuard>} />
```

---

## 3. 设计决策点（待用户最终确认）

### 3.1 技术栈选型（最关键）

| 方案 | 优势 | 劣势 | 估时 |
| ---- | ---- | ---- | ---- |
| **A · uni-app all** | 已编译 4 端（h5+小程序+android+ios）；admin-web 端用 uni-app 重写时 PC 体验中规中矩（uView 是移动端组件库） | admin PC 体验可能下降；escort-app Flutter 代码（79 文件）全部要重写 | 4-6 周 |
| **B · Flutter all** | escort-app 已用 Flutter，Flutter Web/Android/iOS 3 端齐全；admin 域用 Flutter Web（Flutter for Web 在 PC 体验尚可） | 微信小程序端 Flutter 不支持（需用 web-view 包一层）；patient-miniapp Vue 代码（28 pages）全部重写 | 5-7 周 |
| **C · Monorepo 多技术栈** | 保留各端最优技术栈（React for admin、uni-app for miniapp、Flutter for escort）；放 monorepo + 共享 contracts + design tokens | 不是真"统一 app"，是 monorepo 概念统一；用户可能不接受 | 2-3 周 |
| **D · React Native + Next.js** | RN 编译 android+ios；Next.js 编译 h5+小程序（有限）；admin-web 已是 React | miniapp 支持弱；RN 在中国生态不如 uni-app | 4-6 周 |

**推荐**：方案 A（uni-app all），原因：
- patient-miniapp 已用 uni-app 4 端齐全，迁移成本最低
- 微信小程序是陪诊业务主入口（patient 端），uni-app 是国内生态最成熟方案
- admin 域可降级为移动端简化版（PC web 用 antd-mobile 或自研 PC layout）

⚠️ 最终方案需用户决策后再开始实施。

### 3.2 旧项目处置策略

| 选项 | 含义 |
| ---- | ---- |
| **D1 · 完全废弃** | 3 个独立项目 git mv 到 `frontend/_archive_v1/`；新建 `frontend/unified-app/` |
| **D2 · 保留 import 路径** | 新项目 `frontend/unified-app/`；v1 代码保留作为参考但不再开发 |

推荐 D2（保留参考，避免 git history 丢失）。

### 3.3 后端迁移策略

| 步骤 | 说明 |
| ---- | ---- |
| **B1 · JWT Claims 加字段** | Claims 加 `Roles []string` 和 `Active string`；保留 `Role string` 字段兼容 |
| **B2 · auth handler 加 /switch-role** | 新接口，调一次返回新 token（active 切换） |
| **B3 · service 层 RoleGate** | 中间件 `RequireRole(role string)` 校验 token.active == role；OrderService 等可加 |
| **B4 · users.roles JSONB 列 + 数据迁移** | 一次性 SQL：把 v1.role 复制到 roles 数组 |
| **B5 · 旧 Role 字段降级** | v1.1+ 后移除 Role 字段 |

B1-B3 是 P0，B4-B5 是收尾。

---

## 4. Task 列表（每个 Task = 1 commit + 单测 + e2e）

### Phase 1 · 后端 + 协议升级（P0 必做）

| Task | Commit prefix | 估时 | 内容 |
| ---- | ------------- | ---- | ---- |
| 1.1 | `feat(auth): Claims 加 Roles []string + Active string` | 0.5 天 | 兼容旧 Role 字段 |
| 1.2 | `feat(auth): POST /api/v1/auth/switch-role 接口 + handler + 单测` | 0.5 天 | 接受 active=escort，返回新 token |
| 1.3 | `feat(shared): RequireRole(role string) Gin 中间件` | 0.5 天 | 复用 auth-service 当前 Auth 中间件 |
| 1.4 | `feat(auth): migration 0016 加 users.roles JSONB` | 0.5 天 | up + down + 单测 |
| 1.5 | `test(auth): 切换角色全链路集成测试` | 1 天 | Redis + JWT 真跑 |
| 1.6 | `feat(order): handler 加 RequireRole("patient")` | 0.5 天 | patient-only endpoint |
| 1.7 | `feat(match): handler 加 RequireRole("escort")` | 0.5 天 | escort-only endpoint |
| 1.8 | `feat(admin): handler 加 RequireRole("super_admin"\|"order_admin"\|...)` | 0.5 天 | admin-only endpoint |
| 1.9 | `docs: contracts.yaml 加 Roles/Active 字段 + 切换角色事件 schema` | 0.5 天 | 跨服务契约同步 |
| 1.10 | `chore: 移除 v1 Claims.Role 字段（v1.1+）` | 0.5 天 | 配套 B5 |

### Phase 2 · 前端 unified app 骨架（技术栈确定后启动）

| Task | Commit prefix | 估时 | 内容 |
| ---- | ------------- | ---- | ---- |
| 2.1 | `chore: 新建 frontend/unified-app/ 项目骨架` | 0.5 天 | （技术栈决定后）|
| 2.2 | `feat: shared/auth 包：JWT 解码 + multi-role state` | 1 天 | 适配后端 Roles[] |
| 2.3 | `feat: shared/api 包：HTTP client + auth interceptor` | 1 天 | 单一入口 |
| 2.4 | `feat: shared/store/authStore`：token + roles + active_role + switchRole action | 1 天 | 多角色管理 |
| 2.5 | `feat: shared/router`：RoleGuard + 域分组路由（admin/patient/escort）| 1 天 | 路由守卫 |
| 2.6 | `feat: home shell + 角色切换 modal` | 1 天 | 顶层导航 + 切换 UI |
| 2.7 | `feat: 设计系统共享（design tokens + 主题）` | 0.5 天 | 3 域一致 |

### Phase 3 · 业务模块迁移（按域分批）

#### admin 域（admin-web → unified-app）

| Task | Commit | 估时 | 内容 |
| ---- | ------ | ---- | ---- |
| 3A.1 | `feat: 迁移 admin 域布局 + 9 个核心页面` | 3 天 | 仪表盘 / 订单管理 / 用户管理 / 退款 / 钱包 / 评价 / 客服 / 设置 / 数据看板 |
| 3A.2 | `feat: 迁移 admin MSW handlers` | 1 天 | 14 handler → unified MSW |
| 3A.3 | `test: admin 域 E2E（Playwright）` | 1 天 | 列表 / 详情 / 状态变更 |

#### patient 域（patient-miniapp → unified-app）

| Task | Commit | 估时 | 内容 |
| ---- | ------ | ---- | ---- |
| 3P.1 | `feat: 迁移 patient 域布局 + 28 个页面` | 3 天 | 首页 / 医院 / 套餐 / 下单 / 订单 / 钱包 / 消息 / SOS / 评价 / 设置 / 个人中心 |
| 3P.2 | `feat: 迁移 patient stores（9 个）` | 1 天 | pinia stores |
| 3P.3 | `feat: 迁移 patient uni-app 4 端 manifest + easycom` | 0.5 天 | h5+小程序+android+ios |
| 3P.4 | `test: patient 域 E2E（uni-app + playwright）` | 1 天 | 下单 / 支付 / 退款 |

#### escort 域（escort-app → unified-app）

| Task | Commit | 估时 | 内容 |
| ---- | ------ | ---- | ---- |
| 3E.1 | `feat: 迁移 escort 域布局 + 16 个页面` | 3 天 | 抢单池 / 任务详情 / 我的订单 / 钱包 / 培训 / 消息 / 个人中心 |
| 3E.2 | `feat: 迁移 escort Riverpod providers（8 个）` | 1 天 | state notifier providers |
| 3E.3 | `feat: 迁移 escort 端到端调用（feed / reject / accept）` | 1 天 | 业务流 |
| 3E.4 | `test: escort 域 E2E（flutter drive 或 integration_test）` | 1 天 | 接单 / 拒单 / 完成 |

### Phase 4 · 收尾

| Task | Commit | 估时 | 内容 |
| ---- | ------ | ---- | ---- |
| 4.1 | `chore: 旧 v1 项目 git mv 到 frontend/_archive_v1/` | 0.5 天 | 不真删，保留参考 |
| 4.2 | `chore: 统一 pnpm workspace + 共享 types` | 1 天 | monorepo |
| 4.3 | `ci: GitHub Actions 三端统一构建矩阵` | 0.5 天 | 6 build job |
| 4.4 | `docs: §42 unified-app-v2 完整记录` | 1 天 | dev.md |
| 4.5 | `docs: 新 spec unified-app-design.md` | 1 天 | 架构决策 + 端能力矩阵 |
| 4.6 | `release: v2.0.0 tag` | 0.5 天 | release note |

**总估时**：Phase 1（5 天）+ Phase 2（6 天）+ Phase 3（14 天）+ Phase 4（5 天）= **约 30 天**

---

## 5. 验证清单（v2 release 前必须全过）

- [ ] 后端：JWT multi-role + /switch-role 集成测试全过
- [ ] 后端：3 域 RoleGate 中间件单元测试全过
- [ ] 前端：unified-app 编译 4 端通过（h5+小程序+android+ios）
- [ ] 前端：admin 域 Playwright e2e 全过
- [ ] 前端：patient 域 e2e 全过（uni-app 模式）
- [ ] 前端：escort 域 e2e 全过（flutter integration_test 或 drive）
- [ ] E2E：登录 patient → 切 escort → 接单 → 切 admin → 审批 完整链路
- [ ] 性能：4 端 bundle size 不超 v1 各端总和 1.5x
- [ ] 文档：dev.md §42 + spec unified-app-design.md + release notes

---

## 6. 风险与回退点

| 风险 | 缓解 |
| ---- | ---- |
| 技术栈选型失误（全栈替换返工） | Phase 1（后端）独立完成；前端选型前先做 1 周 spike（admin 域 1 个页面 + patient 域 1 个页面 + escort 域 1 个页面用候选技术栈搭 demo）|
| 后端 Claims 字段变更破坏现有 v1 客户端 | Claims 兼容（保留 Role 单字段 + 加 Roles 数组）；v1 客户端读 Role 仍工作 |
| 角色切换后旧 token 仍可用（安全） | /switch-role 把旧 token 加黑名单（Redis 5min TTL）；refresh 强制重新登录 |
| uni-app 在 admin 域 PC 体验差 | admin 域用 antd-mobile + 自研 PC layout 双轨；或选 Flutter all |

---

## 7. 推送策略

- 每 Task 完成后 `git push origin main`
- 推送前 `go test ./...` + `pnpm test` + e2e 全过
- 失败立刻停下来诊断，不掩盖

---

## 8. 状态追踪

| 日期 | 进度 | 阻塞 |
| ---- | ---- | ---- |
| 2026-09-28 | 本 plan 创建；技术栈待用户最终决策 | 用户需回复 §3.1 选项 A/B/C/D |