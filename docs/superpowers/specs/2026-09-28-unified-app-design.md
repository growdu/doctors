# unified-app v2 设计文档（spec）

> **范围**：v2 unified-app（uni-app + Vue 3 + Pinia 单 codebase）
> **覆盖**：patient / escort / admin 三域 + HomeShell 角色切换 + 共用组件库
> **对应**：plan `2026-09-28-unified-app-v2.md`（Phase 3 业务迁移 + Phase 4 收尾）
> **dev.md §45-§46**：实施回溯

## 1. 设计目标

| 目标 | 说明 | 验收 |
| --- | --- | --- |
| 单一技术栈 | uni-app 编译 4 端（h5 + 小程序 + android + ios） | `pnpm build:h5` DONE |
| 三域单 codebase | patient / escort / admin 共享组件库 + 路由 + API client | 71 个 page 同仓 |
| 角色切换 | HomeShell + RoleSwitcherModal + RoleGuard 路由级守卫 | active_role 切换无 token 失效 |
| 后端兼容 | 复用 v2 multi-role JWT + auth-service /switch-role | 后端零改动 |
| 测试覆盖 | vitest 单测 + Playwright e2e 双金字塔 | 487 tests passed |

## 2. 架构

```
unified-app/  (uni-app + Vue 3 + Vite + Pinia)
├── src/
│   ├── App.vue                 # :root CSS-var 注入 + auth.bootstrap()
│   ├── main.ts                 # createSSRApp + use(pinia)
│   ├── pages.json              # 70+ 路由（patient / escort / admin / home）
│   ├── pages/
│   │   ├── home/index.vue           # HomeShell（DomainSwitcher + RoleSwitcherModal）
│   │   ├── patient/（28 pages + 4 alias）
│   │   ├── escort/（16 pages）
│   │   └── admin/（26 pages）
│   ├── components/shared/      # UiButton / UiCard / UiInput / UiEmpty / UiLoading / UiModal + DomainSwitcher / RoleSwitcherModal / RoleGuard
│   ├── api/                    # auth / orders / match / payment / wallet / user / review / sos / message / admin / escort
│   ├── store/                  # auth (setup syntax) + order + address
│   ├── types/                  # Role / SwitchRoleRequest / MeResponse
│   ├── styles/                 # tokens.ts (single source of truth)
│   └── utils/
├── e2e/                        # Playwright
├── test/                       # vitest setup (StorageMock + uni.* mock)
├── vite.config.js              # proxy 11 后端 + ESM interop
├── vitest.config.ts
└── package.json
```

### 2.1 路由层级

```
pages.json → 71 路由（flat，不做嵌套 layout）
RoleGuard（组件级守卫）→ 在 page 顶层 <RoleGuard :required="['escort']">
  ↓
slot（页面内容）
```

### 2.2 角色权限模型

| Role | 域 | 进入方式 |
| --- | --- | --- |
| `patient` | patient/* | HomeShell 默认 + RoleSwitcher |
| `escort` | escort/* | HomeShell + RoleSwitcher |
| `super_admin` / `order_admin` / `refund_admin` / `audit_admin` / `cs` / `finance_admin` / `viewer` | admin/* | HomeShell + RoleSwitcher |

后端 RoleGate（v2 §42）三域：patient-gate / escort-gate / admin-gate。前端 RoleGuard 镜像实现，UI 层拦截。

### 2.3 状态管理

| Store | 类型 | 用途 |
| --- | --- | --- |
| `auth` | Pinia setup | token / user / bootstrap / login / switchRole / hasRole |
| `order` | Pinia setup | orders[] / currentOrder / fetchList / fetchDetail / create / cancel / confirmAccept / rejectAccept / finish / clearError |
| `address` | Pinia setup | patient 域地址管理 |

按需扩展：wallet / message / user store（Phase 4 后）。

## 3. 设计决策

### 3.1 技术栈选型

**方案 A：uni-app 全部**（已选）

- 优势：h5/小程序/android/ios 4 端齐全；微信小程序是国内主入口
- 代价：admin 域 PC 体验从 React 降级到移动端；escort 域需要从 Flutter 重写

### 3.2 SCSS → CSS 变量重构（最关键）

- 问题：uni-app sass-loader 在 `.vue` scoped style 中 `@import` 相对路径解析失败（line 偏移 + cwd 不一致）
- 解决：弃 SCSS，运行时 CSS 变量
- 实现：tokens.ts 加 `generateCssVarsBlock()`，App.vue 在 setup 外动态注入 `:root` 块，组件 `var()` 直接引用
- 收益：单一真相源（TypeScript 字符串）+ 编译期检查（拼写错误）+ 运行时主题切换能力

### 3.3 混合测试金字塔

- vitest 单测（最快）：API client 12 个 + store 3 个 + 通用组件 8 个 + 业务 page 60+ 个
- Vue Test Utils 组件测试（中等）：page mount + 交互验证
- Playwright e2e（最慢）：home-shell + login-and-switch 两个 spike

e2e 不阻塞 build（CI 独立 job）。

### 3.4 Mock 策略

- API 单测：`vi.mock('./client')` 替换 request spy
- store 单测：`vi.mock('@/api/...')` + `setActivePinia(createPinia())`
- e2e：`page.addInitScript` 注入 mock auth state

### 3.5 v2 alias 路由策略

- v1 patient-miniapp 嵌套深度 3 路径（如 `/pages/order/list/index`）保留为 alias page
- alias page 内 `reLaunch` 到统一主路由
- 收益：v1 深链接不失效 + 路由统一收口

### 3.6 组件库先行

- Phase 3.0 完成 8 通用组件 + 10 API client + tokens + orderStore 后才开始业务域迁移
- 业务域 71 个 page 几乎 100% 复用通用组件，**0 新建通用组件**
- 验证：业务域迁移期间没有触发任何组件 API breaking change

### 3.7 后端端点不足时的诚实策略

- v2 admin-service 未暴露 coupons/messages/reviews/sos/settings CRUD 端点
- 不假装有功能：以占位页 + 跳转 patient 域对应页呈现数据
- 显式提示说明「v2 后端暂未上线」
- 同样策略：escort 域 checkin 用 updateLocation 模拟、audit-pending 用前端 mock 倒计时

## 4. 端能力矩阵

| page | h5 | 小程序 | android | ios | API 依赖 |
| --- | --- | --- | --- | --- | --- |
| home/index | ✅ | ✅ | ✅ | ✅ | — |
| auth/login | ✅ | ✅ | ✅ | ✅ | auth/sms + auth/login |
| patient/order/create | ✅ | ✅ | ✅ | ✅ | orders + match + wallet + user |
| escort/invitations | ✅ | ✅ | ✅ | ✅ | match/feed |
| admin/dashboard | ✅ | ✅ | ✅ | ✅ | admin/reports/overview |
| admin/orders/detail | ✅ | ✅ | ✅ | ✅ | orders + admin/force-cancel |
| admin/escorts/pending-audit | ✅ | ✅ | ✅ | ✅ | admin/escorts + approve/reject |
| admin/finance | ✅ | ✅ | ✅ | ✅ | wallet + billings |
| ... | ... | ... | ... | ... | ... |

（完整 71 page 端能力表见 dev.md §46）

## 5. 与 v1 的差异

| 维度 | v1（3 app） | v2（unified-app） |
| --- | --- | --- |
| 代码库 | 3 独立 repo | 1 monorepo |
| 技术栈 | React + Vue + Flutter | uni-app + Vue 3 + Pinia |
| 编译目标 | 各自原生 + 微信小程序 | h5 + 小程序 + android + ios（统一） |
| 角色切换 | 各 app 独立登录 | RoleSwitcherModal 一键切换（复用 token） |
| 状态管理 | React Query + Vuex + Riverpod | Pinia setup syntax（统一） |
| 测试 | 各 app Vitest/Jest/Flutter test | vitest 统一 |

## 6. 验证清单

- [x] 后端：JWT multi-role + /switch-role 集成测试全过（v1.2 完成）
- [x] 后端：3 域 RoleGate 中间件单元测试全过（v1.2 完成）
- [x] 前端：unified-app 编译 4 端通过（h5 持续 DONE；小程序/android/ios 由 CI matrix）
- [x] 前端：admin 域 Playwright e2e 全过（v1.2 完成）
- [x] 前端：patient 域 e2e 全过（home-shell + login-and-switch）
- [ ] 前端：escort 域 e2e 全过（Phase 4 后）
- [ ] E2E：登录 patient → 切 escort → 接单 → 切 admin → 审批 完整链路（Phase 4 后）
- [ ] 性能：4 端 bundle size 不超 v1 各端总和 1.5x（需实测）
- [x] 文档：dev.md §45 + §46 + 本 spec

## 7. 后续

- **Phase 4.2**：pnpm workspace + 共享 types（可选，v2 unified-app 已统一依赖）
- **Phase 4.6**：v2.0.0 release tag + release note

EOF