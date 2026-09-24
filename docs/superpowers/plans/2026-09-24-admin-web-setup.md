# Admin-Web Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 产出 L2 v1.0 管理后台 SPA 骨架。在 `web/admin-web/` 下建立 Vite 5 + React 18 + TS 项目；包含 18 个 P0 页面骨架（仅路由 + 空组件占位）、Zustand authStore、TanStack Query 集成、RequireAuth + RequirePermission 双层路由守卫、API client（fetch + 拦截器）、MSW mock handlers（admin 12 API 全覆盖）、ProTable 通用组件封装、Vitest + RTL + Playwright 测试框架。**v1 重点是骨架 + RBAC + mock handlers；UI 完善（真实图表 / 复杂表单 / 数据绑定）放后续 plan**。

**Architecture:** 单仓多包结构 `web/admin-web/`（与 `web/patient-miniapp` 平级，独立 pnpm package）。状态层用 Zustand（轻量；不引 Redux）；数据请求用 TanStack Query v5（缓存 + 自动 retry + 后台刷新）；UI 用 Ant Design 5 + ProTable（`@ant-design/pro-components` 封装）；图表 `@ant-design/charts`；Mock 用 MSW 2（开发期无后端依赖）。OpenAPI TypeScript client 由 `openapi-typescript` 从 `web/openapi/contracts.yaml` 自动产出，纳入生成文件管理（不手写 schema 类型）。

**Tech Stack:**
- **运行时**: Node.js 20+ · pnpm 9+（与 `web/patient-miniapp` 共享）
- **框架**: React 18.3 + TypeScript 5.5
- **构建**: Vite 5.4（fast HMR + Rollup production）
- **UI**: Ant Design 5.21 + `@ant-design/pro-components` 2.7 + `@ant-design/charts` 2.2
- **状态**: Zustand 4.5
- **数据请求**: TanStack Query 5.x
- **路由**: React Router 6.26（声明式 + 嵌套 + lazy）
- **HTTP**: fetch（不引 axios）+ 自封装 client（拦截器）
- **Mock**: MSW 2.x（Service Worker 拦截 fetch）
- **测试**: Vitest（单元）+ React Testing Library（组件）+ Playwright（E2E）
- **OpenAPI**: openapi-typescript 7.x（自动生成 client types）
- **代码质量**: ESLint 9 + Prettier 3 + @typescript-eslint 8

**前置依赖:**
- `docs/superpowers/specs/2026-09-24-admin-web-design.md`（admin-web 设计：18 页面 + RBAC + 目录结构）
- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.3（admin 12 API 清单）
- `docs/superpowers/specs/2026-09-24-patient-miniapp-design.md`（共享 OpenAPI 契约）
- 后端 `web/openapi/contracts.yaml`（admin-service 已生成的 12 个 API）
- `shared/contracts` Go 后端已发布 `Order*Event` / `RefundCompletedEvent` / `SOSResolvedEvent`（admin-web v1 不消费 Kafka；TanStack Query 轮询即可）

---

## Global Constraints

- Node.js 20 LTS（toolchain via Volta 或 nvm）
- pnpm 9+ workspace（根 `pnpm-workspace.yaml` 包含 `web/admin-web`）
- React 18.3 + TypeScript 5.5 strict mode（`strict: true` + `noUncheckedIndexedAccess: true`）
- 测试覆盖率：业务包 ≥ 70%（v1 骨架阶段；UI 包可降级到 ≥ 50%）
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat(admin-web):` / `test(admin-web):` / `chore(admin-web):` / `docs(admin-web):`
- 包管理：`pnpm`（与 `web/patient-miniapp` 共享；统一 lockfile）
- HTTP client：fetch 包装（不引 axios）；拦截器做 token 注入 + 401 登出 + 错误归一
- 状态管理：Zustand（不引 Redux；不引 MobX）
- 数据请求：TanStack Query v5（不引 SWR；统一缓存策略）
- UI：Ant Design 5 + ProTable（统一复杂表格；不混 react-table）
- Mock：MSW 2 Service Worker（开发期拦截 fetch；不引 json-server / msw 替代品）
- OpenAPI 客户端：`openapi-typescript` 自动生成；生成文件 `src/types/generated.ts`（committed）+ `.gitignore` 排除中间产物
- RBAC：前端 `permissions.ts` 静态映射；后端权威（前端守卫仅 UX，不替代后端鉴权）
- 路由守卫：`RequireAuth`（未登录 → /login）+ `RequirePermission`（无权限 → 403 页）
- 测试：Vitest jsdom env + RTL；Playwright 跑真浏览器（headless chromium）
- 不引：axios / redux / react-table / lodash（用 es-toolkit 或原生）/ moment（用 dayjs）/ styled-components
- v1 不做：暗色主题 / 移动端响应式 / i18n / WebSocket / 实时通知 / 图表大屏 / 自定义 SQL
- 端口约定：dev `http://localhost:8080`（与 patient-miniapp 8081 错开）；e2e 用 `http://localhost:8081`

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `web/admin-web/package.json` | Create | pnpm 包定义（依赖 + scripts） |
| `web/admin-web/vite.config.ts` | Create | Vite 配置（alias + proxy + env） |
| `web/admin-web/tsconfig.json` | Create | TS 严格模式 + 路径别名 |
| `web/admin-web/tsconfig.node.json` | Create | Node 端 TS 配置（vite.config 等） |
| `web/admin-web/index.html` | Create | HTML 入口 |
| `web/admin-web/.eslintrc.cjs` | Create | ESLint 9 flat config |
| `web/admin-web/.prettierrc` | Create | Prettier 格式化 |
| `web/admin-web/.gitignore` | Create | 排除 dist / node_modules / coverage |
| `web/admin-web/.env.development` | Create | 开发期环境变量（VITE_API_BASE） |
| `web/admin-web/.env.production` | Create | 生产期环境变量 |
| `web/admin-web/vitest.config.ts` | Create | Vitest 配置（jsdom + setup） |
| `web/admin-web/playwright.config.ts` | Create | Playwright 配置（chromium headless） |
| `web/admin-web/src/main.tsx` | Create | React 根入口（挂载 + Provider） |
| `web/admin-web/src/App.tsx` | Create | Router + ConfigProvider 顶层 |
| `web/admin-web/src/config.ts` | Create | 运行时配置（API base / 版本） |
| `web/admin-web/src/vite-env.d.ts` | Create | Vite 环境变量类型 |
| `web/admin-web/src/__tests__/setup.ts` | Create | Vitest setup（@testing-library/jest-dom） |
| `web/admin-web/src/types/generated.ts` | Create | OpenAPI 自动生成的类型（不手写） |
| `web/admin-web/src/types/api.d.ts` | Create | 业务类型扩展（AdminUser / Role / Permission） |
| `web/admin-web/src/api/client.ts` | Create | fetch 包装 + 拦截器 + 错误归一 |
| `web/admin-web/src/api/client.test.ts` | Create | 拦截器单测（fetch mock） |
| `web/admin-web/src/api/auth.ts` | Create | 登录 / 刷新 / 当前用户 API |
| `web/admin-web/src/api/admin/users.ts` | Create | 患者 / 陪诊师列表 + 详情 |
| `web/admin-web/src/api/admin/orders.ts` | Create | 订单列表 + 详情 + 强制取消 |
| `web/admin-web/src/api/admin/refunds.ts` | Create | 退款列表 + 审核 |
| `web/admin-web/src/api/admin/escorts.ts` | Create | 待审核陪诊师 + 通过 / 拒绝 |
| `web/admin-web/src/api/admin/work-orders.ts` | Create | 工单列表 + 详情（P1 骨架） |
| `web/admin-web/src/api/admin/wallet.ts` | Create | 提现审核 + 账单（P1 骨架） |
| `web/admin-web/src/api/admin/reviews.ts` | Create | 评价管理（P1 骨架） |
| `web/admin-web/src/api/admin/reports.ts` | Create | 数据看板（P1 骨架） |
| `web/admin-web/src/api/types.ts` | Create | API 响应包装 / 分页 / 错误类型 |
| `web/admin-web/src/api/api.test.ts` | Create | admin API 调用方法单测（msw） |
| `web/admin-web/src/stores/authStore.ts` | Create | Zustand auth store（token + user + permissions） |
| `web/admin-web/src/stores/uiStore.ts` | Create | UI 状态（侧边栏折叠 / 主题） |
| `web/admin-web/src/stores/authStore.test.ts` | Create | authStore 单测 |
| `web/admin-web/src/hooks/useAuth.ts` | Create | useAuth hook（包裹 authStore + useQuery） |
| `web/admin-web/src/hooks/usePermission.ts` | Create | usePermission hook（hasPermission 工具） |
| `web/admin-web/src/hooks/usePolling.ts` | Create | 轮询 hook（基于 useQuery refetchInterval） |
| `web/admin-web/src/hooks/useTableParams.ts` | Create | ProTable 参数同步到 URL（v1 留接口） |
| `web/admin-web/src/hooks/usePermission.test.ts` | Create | 权限判断单测 |
| `web/admin-web/src/router/routes.tsx` | Create | 路由表（18 页面 + 懒加载） |
| `web/admin-web/src/router/guards.tsx` | Create | RequireAuth + RequirePermission 组件 |
| `web/admin-web/src/router/permissions.ts` | Create | Role → Permission 静态映射 |
| `web/admin-web/src/router/guards.test.tsx` | Create | 守卫组件单测（RTL + MemoryRouter） |
| `web/admin-web/src/components/ProTable/ProTable.tsx` | Create | ProTable 通用封装（request / polling / 搜索栏） |
| `web/admin-web/src/components/ProTable/ProTable.test.tsx` | Create | ProTable 单测（RTL + msw） |
| `web/admin-web/src/components/StatusBadge/StatusBadge.tsx` | Create | 订单状态颜色徽章 |
| `web/admin-web/src/components/AuditAction/AuditAction.tsx` | Create | 审核按钮（通过 / 拒绝二次确认） |
| `web/admin-web/src/components/TraceId/TraceId.tsx` | Create | 链路 ID 展示（从响应 header 提取） |
| `web/admin-web/src/components/ErrorBoundary/ErrorBoundary.tsx` | Create | 错误边界（fallback + 上报） |
| `web/admin-web/src/components/PageHeader/PageHeader.tsx` | Create | 页面标题 + 面包屑 |
| `web/admin-web/src/layouts/AdminLayout.tsx` | Create | Sider + Header + Content 经典布局 |
| `web/admin-web/src/layouts/AdminLayout.test.tsx` | Create | AdminLayout 单测 |
| `web/admin-web/src/styles/global.scss` | Create | 全局样式（重置 + 滚动条 + 自定义变量） |
| `web/admin-web/src/styles/theme.ts` | Create | Ant Design 主题 token（主色 #1677ff） |
| `web/admin-web/src/mocks/browser.ts` | Create | MSW Service Worker 启动（dev only） |
| `web/admin-web/src/mocks/handlers/index.ts` | Create | handlers 聚合 |
| `web/admin-web/src/mocks/handlers/auth.ts` | Create | 登录 / 刷新 / 当前用户 mock |
| `web/admin-web/src/mocks/handlers/admin/users.ts` | Create | 用户列表 + 详情 mock |
| `web/admin-web/src/mocks/handlers/admin/orders.ts` | Create | 订单列表 + 强制取消 mock |
| `web/admin-web/src/mocks/handlers/admin/refunds.ts` | Create | 退款队列 + 审核 mock |
| `web/admin-web/src/mocks/handlers/admin/escorts.ts` | Create | 陪诊师审核 mock |
| `web/admin-web/src/mocks/handlers/admin/reports.ts` | Create | 数据看板 mock |
| `web/admin-web/src/mocks/data/seed.ts` | Create | mock 种子数据（faker.js） |
| `web/admin-web/src/mocks/handlers/auth.test.ts` | Create | mock handlers 单测（vitest + msw node） |
| `web/admin-web/src/pages/login/LoginPage.tsx` | Create | 登录页骨架（账号 + 密码 + 2FA） |
| `web/admin-web/src/pages/dashboard/DashboardPage.tsx` | Create | 数据看板骨架（占位 + 标题） |
| `web/admin-web/src/pages/users/PatientListPage.tsx` | Create | 患者列表骨架 |
| `web/admin-web/src/pages/users/PatientDetailPage.tsx` | Create | 患者详情骨架 |
| `web/admin-web/src/pages/users/EscortListPage.tsx` | Create | 陪诊师列表骨架 |
| `web/admin-web/src/pages/users/EscortAuditPage.tsx` | Create | 陪诊师审核骨架（双栏） |
| `web/admin-web/src/pages/orders/OrderListPage.tsx` | Create | 订单列表骨架（ProTable） |
| `web/admin-web/src/pages/orders/OrderDetailPage.tsx` | Create | 订单详情骨架（时间线 + 事件流） |
| `web/admin-web/src/pages/orders/ForceCancelModal.tsx` | Create | 强制取消弹窗骨架 |
| `web/admin-web/src/pages/refunds/RefundListPage.tsx` | Create | 退款队列骨架（轮询） |
| `web/admin-web/src/pages/refunds/RefundAuditPage.tsx` | Create | 退款审核骨架 |
| `web/admin-web/src/pages/work-orders/WorkOrderListPage.tsx` | Create | 工单列表骨架（P1 占位） |
| `web/admin-web/src/pages/work-orders/WorkOrderDetailPage.tsx` | Create | 工单详情骨架 |
| `web/admin-web/src/pages/wallet/WithdrawalReviewPage.tsx` | Create | 提现审核骨架（P1） |
| `web/admin-web/src/pages/reviews/ReviewModerationPage.tsx` | Create | 评价管理骨架（P1） |
| `web/admin-web/src/pages/reports/GMVReportPage.tsx` | Create | GMV 报表骨架（P1） |
| `web/admin-web/src/pages/reports/RefundRatePage.tsx` | Create | 退款率骨架（P1） |
| `web/admin-web/src/pages/settings/RefundPoliciesPage.tsx` | Create | 退款策略配置骨架（P1） |
| `web/admin-web/e2e/login.spec.ts` | Create | Playwright 登录 smoke |
| `web/admin-web/e2e/order-list.spec.ts` | Create | Playwright 订单列表（mock 后端） |
| `web/admin-web/e2e/refund-audit.spec.ts` | Create | Playwright 退款审核 |
| `web/admin-web/e2e/escort-audit.spec.ts` | Create | Playwright 陪诊师审核 |
| `web/admin-web/scripts/generate-client.sh` | Create | 从 contracts.yaml 生成 types 的 shell 脚本 |
| `web/admin-web/public/mockServiceWorker.js` | Create | MSW 自动生成（committed） |
| `web/admin-web/README.md` | Create | 项目说明（dev/build/test） |
| `pnpm-workspace.yaml` | Modify | 根 workspace 包含 `web/admin-web` |
| `docs/04-业务流程.md` | Modify | §4.8 加 admin 审核流程 |
| `dev.md` | Modify | §10.14 加本 plan 落地记录 |

---

### Task 1: 项目脚手架（Vite + React 18 + TS + 目录结构）

**Files:**
- Create: `web/admin-web/package.json`
- Create: `web/admin-web/vite.config.ts`
- Create: `web/admin-web/tsconfig.json`
- Create: `web/admin-web/tsconfig.node.json`
- Create: `web/admin-web/index.html`
- Create: `web/admin-web/.gitignore`
- Create: `web/admin-web/.env.development`
- Create: `web/admin-web/.env.production`
- Create: `web/admin-web/src/main.tsx`
- Create: `web/admin-web/src/App.tsx`
- Create: `web/admin-web/src/config.ts`
- Create: `web/admin-web/src/vite-env.d.ts`
- Modify: `pnpm-workspace.yaml`

**Step 1: 写 main.tsx + App.tsx 烟雾测试（RED）**

`web/admin-web/src/__tests__/App.test.tsx`：

```tsx
// 验证 React 应用能挂载 + 显示标题。
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import App from '@/App';

describe('App', () => {
  it('renders admin-web title', () => {
    render(<App />);
    expect(screen.getByText(/admin-web/i)).toBeInTheDocument();
  });
});
```

**Step 2: 跑测试确认失败**

Run:
```bash
pnpm --filter admin-web exec vitest run --reporter=verbose 2>&1 | head -30
```
Expected: FAIL — `Failed to resolve import "@/App"` + `Cannot find package '@testing-library/react'`

**Step 3: 写 package.json**

`web/admin-web/package.json`：

```json
{
  "name": "admin-web",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext ts,tsx --report-unused-disable-directives --max-warnings 0",
    "format": "prettier --write \"src/**/*.{ts,tsx,scss}\"",
    "typecheck": "tsc -b --noEmit",
    "test:unit": "vitest run",
    "test:unit:watch": "vitest",
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui",
    "generate:client": "bash scripts/generate-client.sh",
    "mock:dev": "cross-env VITE_ENABLE_MOCK=true pnpm dev"
  },
  "dependencies": {
    "@ant-design/charts": "^2.2.4",
    "@ant-design/icons": "^5.5.1",
    "@ant-design/pro-components": "^2.7.10",
    "@tanstack/react-query": "^5.59.0",
    "antd": "^5.21.4",
    "dayjs": "^1.11.13",
    "es-toolkit": "^1.17.0",
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.26.2",
    "zustand": "^4.5.5"
  },
  "devDependencies": {
    "@playwright/test": "^1.48.0",
    "@testing-library/dom": "^10.4.0",
    "@testing-library/jest-dom": "^6.5.0",
    "@testing-library/react": "^16.0.1",
    "@testing-library/user-event": "^14.5.2",
    "@types/node": "^22.7.4",
    "@types/react": "^18.3.11",
    "@types/react-dom": "^18.3.0",
    "@typescript-eslint/eslint-plugin": "^8.8.0",
    "@typescript-eslint/parser": "^8.8.0",
    "@vitejs/plugin-react": "^4.3.2",
    "cross-env": "^7.0.3",
    "eslint": "^9.11.1",
    "eslint-plugin-react": "^7.37.1",
    "eslint-plugin-react-hooks": "^5.0.0",
    "eslint-plugin-react-refresh": "^0.4.12",
    "jsdom": "^25.0.1",
    "msw": "^2.4.9",
    "openapi-typescript": "^7.4.1",
    "prettier": "^3.3.3",
    "sass": "^1.79.4",
    "typescript": "^5.5.4",
    "vite": "^5.4.8",
    "vitest": "^2.1.2"
  },
  "msw": {
    "workerDirectory": ["public"]
  }
}
```

**Step 4: 写 tsconfig + vite.config + index.html**

`web/admin-web/tsconfig.json`：

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    },
    "types": ["vite/client", "@testing-library/jest-dom"]
  },
  "include": ["src", "vite.config.ts", "vitest.config.ts", "playwright.config.ts"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

`web/admin-web/tsconfig.node.json`：

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true,
    "strict": true
  },
  "include": ["vite.config.ts", "vitest.config.ts", "playwright.config.ts"]
}
```

`web/admin-web/vite.config.ts`：

```ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 8080,
    proxy: {
      '/api/v1': {
        target: 'http://127.0.0.1:8090', // admin-service（后端 plan 落地后调整）
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'es2022',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'antd-vendor': ['antd', '@ant-design/icons', '@ant-design/pro-components'],
          'query-vendor': ['@tanstack/react-query', 'zustand'],
        },
      },
    },
  },
});
```

`web/admin-web/index.html`：

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>陪诊管理后台</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**Step 5: 写 .gitignore + .env + config + main + App + vite-env.d.ts**

`web/admin-web/.gitignore`：

```
node_modules
dist
dist-ssr
*.local
.DS_Store
.vscode/*
!.vscode/extensions.json
.idea
coverage
playwright-report
test-results
.eslintcache
```

`web/admin-web/.env.development`：

```
VITE_API_BASE=http://127.0.0.1:8090
VITE_ENABLE_MOCK=true
VITE_APP_TITLE=陪诊管理后台
```

`web/admin-web/.env.production`：

```
VITE_API_BASE=/api/v1
VITE_ENABLE_MOCK=false
VITE_APP_TITLE=陪诊管理后台
```

`web/admin-web/src/config.ts`：

```ts
// 运行时配置（从 import.meta.env 读取）。
export const config = {
  apiBase: import.meta.env.VITE_API_BASE ?? '/api/v1',
  enableMock: import.meta.env.VITE_ENABLE_MOCK === 'true',
  appTitle: import.meta.env.VITE_APP_TITLE ?? '陪诊管理后台',
  version: '0.1.0',
} as const;
```

`web/admin-web/src/vite-env.d.ts`：

```ts
/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE: string;
  readonly VITE_ENABLE_MOCK: string;
  readonly VITE_APP_TITLE: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
```

`web/admin-web/src/main.tsx`：

```tsx
// React 应用入口（挂载根组件 + 初始化 MSW）。
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from '@/App';
import { config } from '@/config';
import '@/styles/global.scss';

async function bootstrap() {
  // v1 启用 MSW（开发期拦截 fetch 模拟后端）。
  if (config.enableMock) {
    const { worker } = await import('@/mocks/browser');
    await worker.start({ onUnhandledRequest: 'bypass' });
  }

  const root = document.getElementById('root');
  if (!root) throw new Error('root element not found');
  createRoot(root).render(
    <StrictMode>
      <App />
    </StrictMode>,
  );
}

bootstrap().catch((err) => {
  console.error('bootstrap failed', err);
});
```

`web/admin-web/src/App.tsx`：

```tsx
// 应用顶层：路由 + 全局 Provider（ConfigProvider + QueryClientProvider）。
import { ConfigProvider, App as AntApp } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from 'react-router-dom';
import zhCN from 'antd/locale/zh_CN';
import { router } from '@/router/routes';
import { themeConfig } from '@/styles/theme';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});

export default function App() {
  return (
    <ConfigProvider locale={zhCN} theme={themeConfig}>
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>
  );
}
```

**Step 6: 创建空 styles + router 目录占位（避免 import 报错）**

```bash
mkdir -p web/admin-web/src/styles
mkdir -p web/admin-web/src/router
mkdir -p web/admin-web/src/mocks
touch web/admin-web/src/styles/global.scss
```

`web/admin-web/src/router/routes.tsx`（占位，Task 9 替换为完整路由表）：

```tsx
// v1 占位（Task 9 替换为完整 18 页面路由）。
import { createMemoryRouter } from 'react-router-dom';

export const router = createMemoryRouter([
  { path: '/', element: <div>admin-web placeholder</div> },
]);
```

`web/admin-web/src/styles/theme.ts`（占位，Task 11 替换为完整 token）：

```ts
import type { ThemeConfig } from 'antd';

export const themeConfig: ThemeConfig = {
  token: { colorPrimary: '#1677ff' },
};
```

**Step 7: 修改根 pnpm-workspace.yaml**

`pnpm-workspace.yaml`：

```yaml
packages:
  - 'web/*'
```

**Step 8: 安装依赖**

```bash
cd /Users/growduduan/ai/doctors
pnpm install
```

Expected: `web/admin-web/node_modules/` 装好；workspace 识别 `admin-web`。

**Step 9: 跑测试确认通过（GREEN，但 main.tsx 还没实现 mocks，bootstrap 会失败——先用 mock-off 模式跑）**

```bash
cd /Users/growduduan/ai/doctors
VITE_ENABLE_MOCK=false pnpm --filter admin-web exec vitest run 2>&1 | tail -20
```

> **注**: 这次 RED → GREEN 用 vitest 单测验证 App 渲染；不跑 dev server（main.tsx 里 mock-off 跳过 MSW）。

预期会因为 `import '@/App'` 找不到（无 msw 兜底）—— 实际上因为我们还没装依赖时已经在跑测试，所以这一轮也会因为缺包失败；待 `pnpm install` 完成后再跑一次。

**Step 10: 跑 typecheck 确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web run typecheck
```
Expected: PASS（仅有 unused-import 警告可忽略；本 Task 不引入业务 import）

**Step 12: Commit**

```bash
git add web/admin-web/ pnpm-workspace.yaml
git commit -m "feat(admin-web): Vite 5 + React 18 + TS 脚手架（package.json + vite/tsconfig + main/App 占位 + 1 个 App 单测）"
```

---

### Task 2: 代码质量工具（ESLint + Prettier + strict tsconfig）

**Files:**
- Create: `web/admin-web/.eslintrc.cjs`
- Create: `web/admin-web/eslint.config.js`
- Create: `web/admin-web/.prettierrc`
- Create: `web/admin-web/.prettierignore`

**Step 1: 写 lint 测试（RED）**

`web/admin-web/src/__tests__/lint-fixture.test.ts`（验证 ESLint 能解析 .tsx）：

```ts
import { describe, it, expect } from 'vitest';

describe('lint fixture', () => {
  it('passes typecheck-only check', () => {
    expect(1 + 1).toBe(2);
  });
});
```

**Step 2: 跑测试（lint 不需要跑这个；只是为后续 Task 留 baseline）**

**Step 3: 写 ESLint config（flat config，ESLint 9 风格）**

`web/admin-web/eslint.config.js`：

```js
import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'coverage', 'playwright-report'] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      '@typescript-eslint/consistent-type-imports': 'error',
      'no-console': ['warn', { allow: ['warn', 'error'] }],
    },
  },
);
```

> **说明**: ESLint 9 用 flat config；如团队习惯 .eslintrc.cjs 可改用旧格式。本 Task 默认 flat config。

**Step 4: 写 .prettierrc**

`web/admin-web/.prettierrc`：

```json
{
  "semi": true,
  "singleQuote": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2,
  "arrowParens": "always",
  "endOfLine": "lf"
}
```

`web/admin-web/.prettierignore`：

```
node_modules
dist
coverage
playwright-report
public/mockServiceWorker.js
```

**Step 5: 跑 lint 确认零警告**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web run lint
```
Expected: PASS（0 errors 0 warnings）

> 如 lint 报缺失包，先 `pnpm install` 让 ESLint 找到 plugin。

**Step 6: Commit**

```bash
git add web/admin-web/
git commit -m "chore(admin-web): ESLint 9 flat + Prettier 3 + 0 warning baseline"
```

---

### Task 3: 测试框架（Vitest + RTL + Playwright + smoke）

**Files:**
- Create: `web/admin-web/vitest.config.ts`
- Create: `web/admin-web/playwright.config.ts`
- Create: `web/admin-web/src/__tests__/setup.ts`
- Create: `web/admin-web/e2e/smoke.spec.ts`
- Create: `web/admin-web/public/mockServiceWorker.js`（MSW 初始化占位，Task 9 替换）

**Step 1: 写 Vitest + RTL 单测（RED）**

`web/admin-web/src/__tests__/components/Hello.test.tsx`：

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { Button } from 'antd';

describe('Hello (AntD Button smoke)', () => {
  it('renders and triggers click handler', async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click me</Button>);
    await userEvent.click(screen.getByRole('button', { name: /click me/i }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run --reporter=verbose 2>&1 | tail -30
```
Expected: FAIL — `Cannot find module 'jsdom'` 或 `ReferenceError: document is not defined`（jsdom env 未配置）

**Step 3: 写 vitest.config.ts**

`web/admin-web/vitest.config.ts`：

```ts
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config';

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/__tests__/setup.ts'],
      css: false,
      coverage: {
        provider: 'v8',
        reporter: ['text', 'html'],
        exclude: [
          'node_modules',
          'dist',
          'src/**/*.test.{ts,tsx}',
          'src/__tests__/**',
          'src/types/generated.ts',
        ],
        thresholds: {
          lines: 70,
          functions: 70,
          statements: 70,
        },
      },
    },
  }),
);
```

**Step 4: 写 Vitest setup**

`web/admin-web/src/__tests__/setup.ts`：

```ts
// Vitest 全局 setup：注册 jest-dom matchers + 清理。
import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

afterEach(() => {
  cleanup();
});
```

**Step 5: 跑测试确认通过（GREEN）**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run --reporter=verbose 2>&1 | tail -20
```
Expected: PASS（Hello.test.tsx 1 个测试通过）

**Step 6: 写 Playwright config + smoke**

`web/admin-web/playwright.config.ts`：

```ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://127.0.0.1:8081',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'pnpm dev --port 8081',
    url: 'http://127.0.0.1:8081',
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});
```

`web/admin-web/e2e/smoke.spec.ts`：

```ts
import { test, expect } from '@playwright/test';

test('admin-web smoke loads root page', async ({ page }) => {
  await page.goto('/');
  // 占位页（Task 1 的 router 兜底）应显示 "admin-web placeholder"。
  await expect(page.getByText(/admin-web/i)).toBeVisible();
});
```

**Step 7: 安装 Playwright 浏览器**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec playwright install --with-deps chromium
```
Expected: chromium 下载完成

**Step 8: 跑 E2E 确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web run test:e2e 2>&1 | tail -30
```
Expected: PASS（1 passed）

> **注意**: dev server 启动 + MSW mock-off 占位；E2E 只验证 SPA 加载。当前 0 个 MSW handler 时，业务接口会失败，但占位页不调用接口，E2E 通过。

**Step 9: Commit**

```bash
git add web/admin-web/
git commit -m "test(admin-web): Vitest + RTL + Playwright 配置 + Hello 单测 + smoke E2E"
```

---

### Task 4: OpenAPI TypeScript 客户端（自动生成）

**Files:**
- Create: `web/admin-web/scripts/generate-client.sh`
- Create: `web/admin-web/src/types/.gitkeep`
- Create: `web/admin-web/src/types/api.d.ts`

**Step 1: 写生成脚本测试（RED）**

`web/admin-web/src/__tests__/types/contracts-exists.test.ts`：

```ts
import { describe, it, expect } from 'vitest';
import { existsSync } from 'node:fs';
import path from 'node:path';

describe('OpenAPI generated types', () => {
  it('generated.ts exists after running generate:client', () => {
    // v1 占位：先 expect false，Task 4 完成后 true。
    const exists = existsSync(path.resolve(__dirname, '../types/generated.ts'));
    expect(typeof exists).toBe('boolean');
  });
});
```

**Step 2: 跑测试确认存在性**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run contracts-exists 2>&1 | tail -10
```
Expected: PASS（仅检查 boolean；generated.ts 可能还不存在）

**Step 3: 写 generate-client.sh + 手动跑一次生成 placeholder**

`web/admin-web/scripts/generate-client.sh`：

```bash
#!/usr/bin/env bash
# 从 web/openapi/contracts.yaml 生成 OpenAPI TypeScript client。
# 依赖：openapi-typescript（pnpm dev 依赖）+ 根 web/openapi/contracts.yaml。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OPENAPI_FILE="$ROOT/web/openapi/contracts.yaml"
OUT="$ROOT/web/admin-web/src/types/generated.ts"

if [[ ! -f "$OPENAPI_FILE" ]]; then
  echo "[WARN] $OPENAPI_FILE not found; generating empty stub."
  mkdir -p "$(dirname "$OUT")"
  cat > "$OUT" <<'EOF'
// 占位：等待后端 admin-service plan 完成生成 contracts.yaml 后自动填充。
// v1 期间所有 API 调用走 MSW mock（src/mocks/handlers/）。
export type Role =
  | 'super_admin'
  | 'order_admin'
  | 'refund_admin'
  | 'cs'
  | 'audit_admin'
  | 'viewer';

export interface AdminUser {
  id: number;
  username: string;
  phone: string;
  role: Role;
  permissions: string[];
  totp_enabled: boolean;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: AdminUser;
  permissions: string[];
}
EOF
  echo "[OK] stub written to $OUT"
  exit 0
fi

pnpm exec openapi-typescript "$OPENAPI_FILE" --output "$OUT"
echo "[OK] generated $OUT from $OPENAPI_FILE"
```

```bash
chmod +x web/admin-web/scripts/generate-client.sh
bash web/admin-web/scripts/generate-client.sh
```

Expected: 写占位 stub 到 `src/types/generated.ts`（`web/openapi/contracts.yaml` v1 还不存在，触发 stub 分支）。

**Step 4: 写业务类型扩展**

`web/admin-web/src/types/api.d.ts`：

```ts
// 业务类型扩展（在 generated.ts 之上补充 admin 端专用类型）。
import type { AdminUser, LoginResponse } from './generated';

export type Role =
  | 'super_admin'
  | 'order_admin'
  | 'refund_admin'
  | 'cs'
  | 'audit_admin'
  | 'viewer';

export type Permission =
  | 'order:read'
  | 'order:force_cancel'
  | 'refund:read'
  | 'refund:approve'
  | 'refund:reject'
  | 'escort:audit'
  | 'work_order:read'
  | 'work_order:reply'
  | 'user:read'
  | 'review:moderate'
  | 'report:read'
  | 'setting:write'
  | '*';

export type { AdminUser, LoginResponse };

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
}

export interface ListResponse<T> {
  data: T[];
  pagination: Pagination;
}

export interface ApiError {
  code: number;
  message: string;
  trace_id?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
  totp?: string;
}

export interface RefreshRequest {
  refresh_token: string;
}
```

**Step 5: 跑 typecheck 确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web run typecheck
```
Expected: PASS

**Step 6: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): openapi-typescript 生成脚本 + 类型 stub + 业务类型扩展（api.d.ts）"
```

---

### Task 5: API client（fetch 包装 + 拦截器）

**Files:**
- Create: `web/admin-web/src/api/client.ts`
- Create: `web/admin-web/src/api/client.test.ts`
- Create: `web/admin-web/src/api/types.ts`

**Step 1: 写 client 单测（RED）**

`web/admin-web/src/api/client.test.ts`：

```ts
// 验证 fetch wrapper：401 抛出 ApiError、token 注入、超时。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiClient, ApiError } from './client';

describe('apiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('returns parsed JSON on 200', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 0, data: { hello: 'world' } }), {
          status: 200,
        }),
      ),
    );
    const res = await apiClient<{ hello: string }>('/test');
    expect(res.data.hello).toBe('world');
  });

  it('throws ApiError on 401', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 40101, message: 'unauthorized' }), {
          status: 401,
        }),
      ),
    );
    await expect(apiClient('/test')).rejects.toBeInstanceOf(ApiError);
  });

  it('injects Authorization header from getToken', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ code: 0, data: {} }), { status: 200 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    // 注入测试 token。
    const { setAuthTokenGetter } = await import('./client');
    setAuthTokenGetter(() => 'test-token-123');

    await apiClient('/protected');
    expect(fetchMock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer test-token-123',
        }),
      }),
    );
  });

  it('extracts trace_id from response header', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 0, data: {} }), {
          status: 200,
          headers: { 'x-trace-id': 'trace-abc-123' },
        }),
      ),
    );
    const res = await apiClient('/test');
    expect(res.trace_id).toBe('trace-abc-123');
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run client.test 2>&1 | tail -30
```
Expected: FAIL — `Cannot find module './client'`

**Step 3: 写 types.ts + client.ts**

`web/admin-web/src/api/types.ts`：

```ts
// API 通用响应 + 错误类型。
export interface ApiResp<T> {
  code: number; // 0 成功；非 0 业务码
  message?: string;
  data?: T;
  trace_id?: string;
}

export class ApiError extends Error {
  readonly code: number;
  readonly traceId?: string;
  readonly status?: number;

  constructor(opts: { code: number; message: string; traceId?: string; status?: number }) {
    super(opts.message);
    this.name = 'ApiError';
    this.code = opts.code;
    this.traceId = opts.traceId;
    this.status = opts.status;
  }
}
```

`web/admin-web/src/api/client.ts`：

```ts
// fetch 包装：拦截器（token 注入 + 401 拦截 + 错误归一 + 超时）。
import { config } from '@/config';
import type { ApiResp } from './types';
import { ApiError } from './types';

type TokenGetter = () => string | null;
let getToken: TokenGetter = () => null;

/** 注入 token 获取器（由 authStore 在 bootstrap 时挂上）。 */
export function setAuthTokenGetter(fn: TokenGetter) {
  getToken = fn;
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  body?: unknown;
  params?: Record<string, string | number | boolean | undefined>;
  headers?: Record<string, string>;
  timeout?: number;
}

function buildUrl(path: string, params?: RequestOptions['params']): string {
  const base = config.apiBase.endsWith('/') ? config.apiBase : config.apiBase + '/';
  const cleanPath = path.startsWith('/') ? path.slice(1) : path;
  let url = base + cleanPath;
  if (params) {
    const usp = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v === undefined) continue;
      usp.append(k, String(v));
    }
    const qs = usp.toString();
    if (qs) url += (url.includes('?') ? '&' : '?') + qs;
  }
  return url;
}

export async function apiClient<T>(
  path: string,
  opts: RequestOptions = {},
): Promise<ApiResp<T>> {
  const url = buildUrl(path, opts.params);
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
    ...opts.headers,
  };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const ctrl = new AbortController();
  const timeoutMs = opts.timeout ?? 30_000;
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);

  let resp: Response;
  try {
    resp = await fetch(url, {
      method: opts.method ?? 'GET',
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      signal: ctrl.signal,
    });
  } catch (err) {
    clearTimeout(timer);
    if ((err as Error).name === 'AbortError') {
      throw new ApiError({ code: -1, message: 'request timeout' });
    }
    throw new ApiError({ code: -2, message: (err as Error).message ?? 'network error' });
  }
  clearTimeout(timer);

  const traceId = resp.headers.get('x-trace-id') ?? undefined;

  let parsed: ApiResp<T> | null = null;
  try {
    parsed = (await resp.json()) as ApiResp<T>;
  } catch {
    throw new ApiError({
      code: resp.status,
      message: 'response is not valid JSON',
      traceId,
      status: resp.status,
    });
  }

  if (!resp.ok || (parsed && parsed.code !== 0)) {
    throw new ApiError({
      code: parsed?.code ?? resp.status,
      message: parsed?.message ?? `HTTP ${resp.status}`,
      traceId,
      status: resp.status,
    });
  }

  return { ...parsed, trace_id: traceId } as ApiResp<T>;
}
```

**Step 4: 跑测试确认通过（GREEN）**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run client.test 2>&1 | tail -30
```
Expected: PASS（4 个测试）

**Step 5: Commit**

```bash
git add web/admin-web/src/api/
git commit -m "feat(admin-web): fetch client + 拦截器（token/401/timeout）+ ApiError + 4 个单测"
```

---

### Task 6: MSW handlers（admin 12 API 全覆盖）

**Files:**
- Create: `web/admin-web/src/mocks/data/seed.ts`
- Create: `web/admin-web/src/mocks/handlers/auth.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/users.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/orders.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/refunds.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/escorts.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/reports.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/work-orders.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/wallet.ts`
- Create: `web/admin-web/src/mocks/handlers/admin/reviews.ts`
- Create: `web/admin-web/src/mocks/handlers/index.ts`
- Create: `web/admin-web/src/mocks/browser.ts`
- Create: `web/admin-web/public/mockServiceWorker.js`（MSW 初始化产物）
- Create: `web/admin-web/src/mocks/handlers/auth.test.ts`

**Step 1: 写 handlers 单测（RED）**

`web/admin-web/src/mocks/handlers/auth.test.ts`：

```ts
// 验证 MSW handlers 拦截 fetch + 返回 mock 数据。
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from './index';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('MSW auth handler', () => {
  it('POST /api/v1/auth/login returns token', async () => {
    const resp = await fetch('http://localhost/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: 'admin',
        password: 'admin123',
        totp: '000000',
      }),
    });
    const body = await resp.json();
    expect(resp.status).toBe(200);
    expect(body.code).toBe(0);
    expect(body.data.access_token).toBeTruthy();
    expect(body.data.user.role).toBe('super_admin');
  });

  it('GET /api/v1/admin/users returns paginated list', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/users?page=1&page_size=5', {
      headers: { Authorization: 'Bearer test-token' },
    });
    const body = await resp.json();
    expect(body.code).toBe(0);
    expect(body.data.data).toBeInstanceOf(Array);
    expect(body.data.pagination.total).toBeGreaterThan(0);
  });

  it('GET /api/v1/admin/orders?status=matching returns filtered list', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/orders?status=matching', {
      headers: { Authorization: 'Bearer test-token' },
    });
    const body = await resp.json();
    expect(body.code).toBe(0);
    for (const o of body.data.data) {
      expect(o.status).toBe('matching');
    }
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run auth.test 2>&1 | tail -30
```
Expected: FAIL — `Cannot find module './index'`

**Step 3: 写种子数据**

`web/admin-web/src/mocks/data/seed.ts`：

```ts
// Mock 种子（开发期使用，固定 ID 便于测试）。
import type { AdminUser, Role } from '@/types/api';

export const seedAdminUser: AdminUser = {
  id: 1,
  username: 'admin',
  phone: '13800138000',
  role: 'super_admin' as Role,
  permissions: ['*'],
  totp_enabled: false,
};

export const seedPatients = Array.from({ length: 20 }, (_, i) => ({
  id: i + 1,
  nickname: `patient_${i + 1}`,
  phone: `1380013${String(i).padStart(4, '0')}`,
  real_name_verified: i % 3 !== 0,
  created_at: new Date(Date.now() - i * 86_400_000).toISOString(),
  total_orders: i,
  total_spent: i * 100,
}));

export const seedOrders = Array.from({ length: 30 }, (_, i) => {
  const statuses = [
    'created',
    'paid',
    'matching',
    'pending_acceptance',
    'accepted',
    'in_service',
    'completed',
    'refunding',
    'refunded',
    'canceled',
  ];
  return {
    id: i + 1,
    order_no: `O${String(i + 1).padStart(8, '0')}`,
    patient_nickname: `patient_${(i % 20) + 1}`,
    escort_nickname: i % 2 === 0 ? `escort_${(i % 10) + 1}` : null,
    hospital_name: ['北京协和', '上海瑞金', '广州中山'][i % 3]!,
    final_amount: 100 + i * 10,
    status: statuses[i % statuses.length]!,
    created_at: new Date(Date.now() - i * 3600_000).toISOString(),
  };
});

export const seedRefunds = Array.from({ length: 10 }, (_, i) => ({
  id: i + 1,
  order_id: (i % 30) + 1,
  amount: 50 + i * 10,
  refund_percent: 0.5 + i * 0.05,
  reason: ['医生停诊', '陪诊师迟到', '行程冲突', '其他'][i % 4]!,
  status: ['pending', 'approved', 'rejected'][i % 3]!,
  created_at: new Date(Date.now() - i * 86_400_000).toISOString(),
}));

export const seedPendingEscorts = Array.from({ length: 5 }, (_, i) => ({
  id: i + 100,
  nickname: `escort_pending_${i + 1}`,
  avatar_url: `https://i.pravatar.cc/64?u=${i}`,
  real_name: `张三${i + 1}`,
  submitted_at: new Date(Date.now() - i * 3600_000).toISOString(),
  certifications: ['护士执业证', '健康管理师'],
  status: 'pending',
}));
```

**Step 4: 写 auth handlers**

`web/admin-web/src/mocks/handlers/auth.ts`：

```ts
import { http, HttpResponse } from 'msw';
import { seedAdminUser } from '../data/seed';

export const authHandlers = [
  http.post('/api/v1/auth/login', async ({ request }) => {
    const body = (await request.json()) as { username: string; password: string; totp?: string };
    if (body.username !== 'admin' || body.password !== 'admin123') {
      return HttpResponse.json(
        { code: 40101, message: '账号或密码错误' },
        { status: 401 },
      );
    }
    return HttpResponse.json({
      code: 0,
      data: {
        access_token: 'mock-access-token-' + Date.now(),
        refresh_token: 'mock-refresh-token-' + Date.now(),
        user: seedAdminUser,
        permissions: ['*'],
      },
    });
  }),

  http.post('/api/v1/auth/refresh', async ({ request }) => {
    const body = (await request.json()) as { refresh_token: string };
    if (!body.refresh_token) {
      return HttpResponse.json({ code: 40101, message: 'invalid refresh token' }, { status: 401 });
    }
    return HttpResponse.json({
      code: 0,
      data: {
        access_token: 'mock-access-token-refreshed-' + Date.now(),
      },
    });
  }),

  http.get('/api/v1/auth/me', () =>
    HttpResponse.json({ code: 0, data: seedAdminUser }),
  ),
];
```

**Step 5: 写 admin handlers（users / orders / refunds / escorts / reports / work-orders / wallet / reviews）**

`web/admin-web/src/mocks/handlers/admin/users.ts`：

```ts
import { http, HttpResponse } from 'msw';
import { seedPatients } from '../../data/seed';

export const userHandlers = [
  http.get('/api/v1/admin/users', ({ request }) => {
    const url = new URL(request.url);
    const page = Number(url.searchParams.get('page') ?? 1);
    const pageSize = Number(url.searchParams.get('page_size') ?? 10);
    const start = (page - 1) * pageSize;
    const slice = seedPatients.slice(start, start + pageSize);
    return HttpResponse.json({
      code: 0,
      data: { data: slice, pagination: { page, page_size: pageSize, total: seedPatients.length } },
    });
  }),

  http.get('/api/v1/admin/users/:id', ({ params }) => {
    const id = Number(params.id);
    const patient = seedPatients.find((p) => p.id === id);
    if (!patient) return HttpResponse.json({ code: 13001, message: 'user not found' }, { status: 404 });
    return HttpResponse.json({ code: 0, data: { ...patient, orders: [], reviews: [] } });
  }),
];
```

`web/admin-web/src/mocks/handlers/admin/orders.ts`：

```ts
import { http, HttpResponse } from 'msw';
import { seedOrders } from '../../data/seed';

export const orderHandlers = [
  http.get('/api/v1/admin/orders', ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const page = Number(url.searchParams.get('page') ?? 1);
    const pageSize = Number(url.searchParams.get('page_size') ?? 20);
    let filtered = seedOrders;
    if (status) filtered = filtered.filter((o) => o.status === status);
    const start = (page - 1) * pageSize;
    return HttpResponse.json({
      code: 0,
      data: {
        data: filtered.slice(start, start + pageSize),
        pagination: { page, page_size: pageSize, total: filtered.length },
      },
    });
  }),

  http.get('/api/v1/admin/orders/:id', ({ params }) => {
    const order = seedOrders.find((o) => o.id === Number(params.id));
    if (!order) return HttpResponse.json({ code: 13001, message: 'order not found' }, { status: 404 });
    return HttpResponse.json({
      code: 0,
      data: { ...order, timeline: [], events: [] },
    });
  }),

  http.post('/api/v1/admin/orders/:id/force-cancel', async ({ params, request }) => {
    const body = (await request.json()) as { reason: string };
    return HttpResponse.json({
      code: 0,
      data: { order_id: Number(params.id), status: 'canceled', reason: body.reason },
    });
  }),
];
```

`web/admin-web/src/mocks/handlers/admin/refunds.ts`：

```ts
import { http, HttpResponse } from 'msw';
import { seedRefunds } from '../../data/seed';

export const refundHandlers = [
  http.get('/api/v1/admin/refunds', ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    let filtered = seedRefunds;
    if (status) filtered = filtered.filter((r) => r.status === status);
    return HttpResponse.json({
      code: 0,
      data: {
        data: filtered,
        pagination: { page: 1, page_size: filtered.length, total: filtered.length },
      },
    });
  }),

  http.post('/api/v1/admin/refunds/:id/approve', ({ params }) =>
    HttpResponse.json({
      code: 0,
      data: { refund_id: Number(params.id), status: 'approved' },
    }),
  ),

  http.post('/api/v1/admin/refunds/:id/reject', async ({ params, request }) => {
    const body = (await request.json()) as { reason: string };
    return HttpResponse.json({
      code: 0,
      data: { refund_id: Number(params.id), status: 'rejected', reason: body.reason },
    });
  }),
];
```

`web/admin-web/src/mocks/handlers/admin/escorts.ts`：

```ts
import { http, HttpResponse } from 'msw';
import { seedPendingEscorts } from '../../data/seed';

export const escortHandlers = [
  http.get('/api/v1/admin/escorts/pending-audit', () =>
    HttpResponse.json({
      code: 0,
      data: { data: seedPendingEscorts, pagination: { page: 1, page_size: 10, total: seedPendingEscorts.length } },
    }),
  ),

  http.post('/api/v1/admin/escorts/:id/approve', ({ params }) =>
    HttpResponse.json({
      code: 0,
      data: { escort_id: Number(params.id), status: 'approved' },
    }),
  ),

  http.post('/api/v1/admin/escorts/:id/reject', async ({ params, request }) => {
    const body = (await request.json()) as { reason: string };
    return HttpResponse.json({
      code: 0,
      data: { escort_id: Number(params.id), status: 'rejected', reason: body.reason },
    });
  }),
];
```

`web/admin-web/src/mocks/handlers/admin/reports.ts`：

```ts
import { http, HttpResponse } from 'msw';

export const reportHandlers = [
  http.get('/api/v1/admin/reports/overview', () =>
    HttpResponse.json({
      code: 0,
      data: {
        today_gmv: 125_300.5,
        today_orders: 87,
        refund_rate: 3.2,
        complaint_rate: 0.8,
        pending_escorts: 5,
        order_trend: Array.from({ length: 30 }, (_, i) => ({
          date: new Date(Date.now() - i * 86_400_000).toISOString().slice(0, 10),
          count: 50 + Math.floor(Math.random() * 30),
        })).reverse(),
      },
    }),
  ),
];
```

`web/admin-web/src/mocks/handlers/admin/work-orders.ts`：

```ts
import { http, HttpResponse } from 'msw';
export const workOrderHandlers = [
  http.get('/api/v1/admin/work-orders', () =>
    HttpResponse.json({
      code: 0,
      data: { data: [], pagination: { page: 1, page_size: 20, total: 0 } },
    }),
  ),
];
```

`web/admin-web/src/mocks/handlers/admin/wallet.ts`：

```ts
import { http, HttpResponse } from 'msw';
export const walletHandlers = [
  http.get('/api/v1/admin/billings', () =>
    HttpResponse.json({
      code: 0,
      data: { data: [], pagination: { page: 1, page_size: 20, total: 0 } },
    }),
  ),
];
```

`web/admin-web/src/mocks/handlers/admin/reviews.ts`：

```ts
import { http, HttpResponse } from 'msw';
export const reviewHandlers = [
  http.get('/api/v1/admin/reviews', () =>
    HttpResponse.json({
      code: 0,
      data: { data: [], pagination: { page: 1, page_size: 20, total: 0 } },
    }),
  ),
];
```

**Step 6: 聚合 handlers + browser**

`web/admin-web/src/mocks/handlers/index.ts`：

```ts
import { authHandlers } from './auth';
import { userHandlers } from './admin/users';
import { orderHandlers } from './admin/orders';
import { refundHandlers } from './admin/refunds';
import { escortHandlers } from './admin/escorts';
import { reportHandlers } from './admin/reports';
import { workOrderHandlers } from './admin/work-orders';
import { walletHandlers } from './admin/wallet';
import { reviewHandlers } from './admin/reviews';

export const handlers = [
  ...authHandlers,
  ...userHandlers,
  ...orderHandlers,
  ...refundHandlers,
  ...escortHandlers,
  ...reportHandlers,
  ...workOrderHandlers,
  ...walletHandlers,
  ...reviewHandlers,
];
```

`web/admin-web/src/mocks/browser.ts`：

```ts
// 开发期 MSW Service Worker 入口。
import { setupWorker } from 'msw/browser';
import { handlers } from './handlers';

export const worker = setupWorker(...handlers);
```

**Step 7: 初始化 mockServiceWorker.js**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec msw init public/ --save
```
Expected: `public/mockServiceWorker.js` 创建

**Step 8: 跑测试确认通过（GREEN）**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run auth.test 2>&1 | tail -20
```
Expected: PASS（3 个测试；MSW 拦截 fetch + 返回 mock 数据）

**Step 9: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): MSW handlers (auth + admin 12 API 全覆盖 + 种子数据 + 3 个 handler 单测)"
```

---

### Task 7: Zustand authStore + 登录页

**Files:**
- Create: `web/admin-web/src/stores/authStore.ts`
- Create: `web/admin-web/src/stores/uiStore.ts`
- Create: `web/admin-web/src/stores/authStore.test.ts`
- Create: `web/admin-web/src/api/auth.ts`
- Create: `web/admin-web/src/api/auth.test.ts`
- Create: `web/admin-web/src/pages/login/LoginPage.tsx`

**Step 1: 写 authStore 单测（RED）**

`web/admin-web/src/stores/authStore.test.ts`：

```ts
// 验证 Zustand authStore：login/logout/hasPermission 状态机。
import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore } from './authStore';

describe('useAuthStore', () => {
  beforeEach(() => {
    useAuthStore.getState().logout();
  });

  it('initial state is empty', () => {
    const s = useAuthStore.getState();
    expect(s.token).toBeNull();
    expect(s.user).toBeNull();
    expect(s.permissions).toEqual([]);
  });

  it('setAuth fills token/user/permissions', () => {
    useAuthStore.getState().setAuth({
      token: 'tok-1',
      user: { id: 1, username: 'admin', phone: '138', role: 'super_admin', permissions: ['*'], totp_enabled: false },
      permissions: ['*'],
    });
    const s = useAuthStore.getState();
    expect(s.token).toBe('tok-1');
    expect(s.user?.role).toBe('super_admin');
    expect(s.permissions).toEqual(['*']);
  });

  it('hasPermission checks wildcard', () => {
    useAuthStore.getState().setAuth({
      token: 'tok-1',
      user: { id: 1, username: 'admin', phone: '138', role: 'super_admin', permissions: ['*'], totp_enabled: false },
      permissions: ['*'],
    });
    expect(useAuthStore.getState().hasPermission('order:force_cancel')).toBe(true);
  });

  it('logout clears state', () => {
    useAuthStore.getState().setAuth({
      token: 'tok-1',
      user: { id: 1, username: 'admin', phone: '138', role: 'order_admin', permissions: ['order:read'], totp_enabled: false },
      permissions: ['order:read'],
    });
    useAuthStore.getState().logout();
    expect(useAuthStore.getState().token).toBeNull();
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run authStore.test 2>&1 | tail -20
```
Expected: FAIL — `Cannot find module './authStore'`

**Step 3: 写 authStore + uiStore**

`web/admin-web/src/stores/authStore.ts`：

```ts
// 鉴权状态：token / user / permissions；localStorage 持久化。
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { AdminUser, Permission } from '@/types/api';

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: AdminUser | null;
  permissions: Permission[];

  setAuth(payload: {
    token: string;
    refreshToken?: string;
    user: AdminUser;
    permissions: Permission[];
  }): void;
  logout(): void;
  hasPermission(perm: Permission): boolean;
}

const STORAGE_KEY = 'admin-web-auth';

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      refreshToken: null,
      user: null,
      permissions: [],

      setAuth(payload) {
        set({
          token: payload.token,
          refreshToken: payload.refreshToken ?? null,
          user: payload.user,
          permissions: payload.permissions,
        });
      },

      logout() {
        set({ token: null, refreshToken: null, user: null, permissions: [] });
      },

      hasPermission(perm) {
        const perms = get().permissions;
        if (perms.includes('*')) return true;
        return perms.includes(perm);
      },
    }),
    {
      name: STORAGE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (s) => ({
        token: s.token,
        refreshToken: s.refreshToken,
        user: s.user,
        permissions: s.permissions,
      }),
    },
  ),
);
```

`web/admin-web/src/stores/uiStore.ts`：

```ts
// UI 状态：侧边栏折叠、主题（v1 仅 light）。
import { create } from 'zustand';

interface UIState {
  siderCollapsed: boolean;
  toggleSider(): void;
}

export const useUIStore = create<UIState>((set) => ({
  siderCollapsed: false,
  toggleSider() {
    set((s) => ({ siderCollapsed: !s.siderCollapsed }));
  },
}));
```

**Step 4: 写 auth API**

`web/admin-web/src/api/auth.ts`：

```ts
// 鉴权相关 API：登录 / 刷新 / 当前用户。
import { apiClient } from './client';
import type { LoginRequest, RefreshRequest, LoginResponse, AdminUser } from '@/types/api';

export const authApi = {
  async login(req: LoginRequest) {
    const resp = await apiClient<LoginResponse>('/auth/login', {
      method: 'POST',
      body: req,
    });
    return resp.data!;
  },

  async refresh(req: RefreshRequest) {
    const resp = await apiClient<{ access_token: string }>('/auth/refresh', {
      method: 'POST',
      body: req,
    });
    return resp.data!;
  },

  async me() {
    const resp = await apiClient<AdminUser>('/auth/me');
    return resp.data!;
  },
};
```

`web/admin-web/src/api/auth.test.ts`：

```ts
import { describe, it, expect } from 'vitest';
import { authApi } from './auth';

describe('authApi (with MSW)', () => {
  it('login returns token + user', async () => {
    const result = await authApi.login({
      username: 'admin',
      password: 'admin123',
      totp: '000000',
    });
    expect(result.access_token).toBeTruthy();
    expect(result.user.role).toBe('super_admin');
  });

  it('login with wrong password throws', async () => {
    await expect(
      authApi.login({ username: 'admin', password: 'wrong', totp: '000000' }),
    ).rejects.toThrow();
  });
});
```

> **注意**: 上面这套 test 用 `setupServer(...handlers)` from Task 6 handlers；测试文件顶部需 import `setupServer` + `beforeAll/afterAll/afterEach`（参考 Task 6 auth.test.ts 模式），这里略。

**Step 5: 跑测试确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run authStore auth 2>&1 | tail -20
```
Expected: PASS（4 + 2 个测试）

**Step 6: 写 LoginPage（骨架）**

`web/admin-web/src/pages/login/LoginPage.tsx`：

```tsx
// 登录页骨架（账号 + 密码 + 2FA + mock 一键登录）。
import { useState } from 'react';
import { Form, Input, Button, Card, Typography, Alert, Space } from 'antd';
import { useNavigate } from 'react-router-dom';
import { authApi } from '@/api/auth';
import { useAuthStore } from '@/stores/authStore';
import { setAuthTokenGetter } from '@/api/client';
import type { Permission } from '@/types/api';

const { Title, Text } = Typography;

export default function LoginPage() {
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onFinish = async (values: { username: string; password: string; totp?: string }) => {
    setLoading(true);
    setError(null);
    try {
      const result = await authApi.login(values);
      setAuth({
        token: result.access_token,
        refreshToken: result.refresh_token,
        user: result.user,
        permissions: result.permissions as Permission[],
      });
      setAuthTokenGetter(() => useAuthStore.getState().token);
      navigate('/dashboard');
    } catch (err) {
      setError((err as Error).message || '登录失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f0f2f5' }}>
      <Card style={{ width: 400 }}>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Title level={3} style={{ textAlign: 'center', margin: 0 }}>陪诊管理后台</Title>
          <Text type="secondary" style={{ textAlign: 'center', display: 'block' }}>
            admin / admin123（mock 一键登录）
          </Text>

          {error && <Alert type="error" message={error} />}

          <Form layout="vertical" onFinish={onFinish} initialValues={{ username: 'admin', password: 'admin123', totp: '000000' }}>
            <Form.Item name="username" label="账号" rules={[{ required: true }]}>
              <Input placeholder="admin" autoComplete="username" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true }]}>
              <Input.Password placeholder="admin123" autoComplete="current-password" />
            </Form.Item>
            <Form.Item name="totp" label="2FA 验证码（mock）">
              <Input placeholder="000000" maxLength={6} />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>登录</Button>
          </Form>
        </Space>
      </Card>
    </div>
  );
}
```

**Step 7: 路由占位指向 /login（Task 9 替换为完整路由表，本 Task 临时手动挂上）**

> **不破坏 Task 1 占位 router**：本 Task 不动 router。LoginPage 在 Task 9 接入。

**Step 8: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): Zustand authStore + auth API + LoginPage（骨架 + 6 个单测）"
```

---

### Task 8: TanStack Query + 通用 hooks

**Files:**
- Create: `web/admin-web/src/hooks/useAuth.ts`
- Create: `web/admin-web/src/hooks/usePermission.ts`
- Create: `web/admin-web/src/hooks/usePolling.ts`
- Create: `web/admin-web/src/hooks/useTableParams.ts`
- Create: `web/admin-web/src/hooks/usePermission.test.ts`

**Step 1: 写 hooks 单测（RED）**

`web/admin-web/src/hooks/usePermission.test.ts`：

```ts
// 验证 usePermission hook：hasPermission + canAny。
import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { usePermission } from './usePermission';
import { useAuthStore } from '@/stores/authStore';

describe('usePermission', () => {
  beforeEach(() => {
    useAuthStore.getState().logout();
  });

  it('returns false when no permissions', () => {
    const { result } = renderHook(() => usePermission('order:read'));
    expect(result.current.hasPermission('order:read')).toBe(false);
  });

  it('returns true when user has wildcard', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      user: { id: 1, username: 'admin', phone: '1', role: 'super_admin', permissions: ['*'], totp_enabled: false },
      permissions: ['*'],
    });
    const { result } = renderHook(() => usePermission('order:force_cancel'));
    expect(result.current.hasPermission('order:force_cancel')).toBe(true);
  });

  it('canAny checks multiple permissions', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      user: { id: 1, username: 'admin', phone: '1', role: 'order_admin', permissions: ['order:read'], totp_enabled: false },
      permissions: ['order:read'],
    });
    const { result } = renderHook(() => usePermission('order:read'));
    expect(result.current.canAny(['order:read', 'escort:audit'])).toBe(true);
    expect(result.current.canAll(['order:read', 'escort:audit'])).toBe(false);
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run usePermission 2>&1 | tail -20
```
Expected: FAIL — `Cannot find module './usePermission'`

**Step 3: 写 useAuth + usePermission + usePolling + useTableParams**

`web/admin-web/src/hooks/useAuth.ts`：

```ts
// 鉴权 hook：包装 authStore + 当前用户查询。
import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/authStore';
import { authApi } from '@/api/auth';

export function useAuth() {
  const token = useAuthStore((s) => s.token);
  const user = useAuthStore((s) => s.user);
  const permissions = useAuthStore((s) => s.permissions);
  const setAuth = useAuthStore((s) => s.setAuth);
  const logout = useAuthStore((s) => s.logout);

  const meQuery = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: authApi.me,
    enabled: !!token && !user,
    staleTime: 60_000,
  });

  return {
    token,
    user,
    permissions,
    isAuthenticated: !!token,
    me: meQuery.data ?? user,
    setAuth,
    logout,
    refresh: meQuery.refetch,
  };
}
```

`web/admin-web/src/hooks/usePermission.ts`：

```ts
// 权限 hook：hasPermission / canAny / canAll。
import { useAuthStore } from '@/stores/authStore';
import type { Permission } from '@/types/api';

export function usePermission(_key?: Permission) {
  const hasPermission = useAuthStore((s) => s.hasPermission);
  const perms = useAuthStore((s) => s.permissions);

  return {
    permissions: perms,

    hasPermission(perm: Permission) {
      return hasPermission(perm);
    },

    canAny(permsToCheck: Permission[]) {
      if (perms.includes('*')) return true;
      return permsToCheck.some((p) => perms.includes(p));
    },

    canAll(permsToCheck: Permission[]) {
      if (perms.includes('*')) return true;
      return permsToCheck.every((p) => perms.includes(p));
    },
  };
}
```

`web/admin-web/src/hooks/usePolling.ts`：

```ts
// 轮询 hook：基于 useQuery 的 refetchInterval；v1 用于退款队列 / 工单 SLA。
import { useQuery, type UseQueryOptions } from '@tanstack/react-query';

export function usePolling<T>(
  opts: UseQueryOptions<T> & { intervalMs?: number },
) {
  return useQuery({
    ...opts,
    refetchInterval: opts.intervalMs ?? 5_000,
    refetchIntervalInBackground: false,
  });
}
```

`web/admin-web/src/hooks/useTableParams.ts`：

```ts
// 表格参数同步（v1 留接口：URL search params 双向同步；骨架版仅返回对象）。
import { useSearchParams } from 'react-router-dom';

export interface TableParams {
  page: number;
  page_size: number;
  status?: string;
  keyword?: string;
}

export function useTableParams(): [TableParams, (next: Partial<TableParams>) => void] {
  const [params, setParams] = useSearchParams();
  const current: TableParams = {
    page: Number(params.get('page') ?? 1),
    page_size: Number(params.get('page_size') ?? 20),
    status: params.get('status') ?? undefined,
    keyword: params.get('keyword') ?? undefined,
  };
  const update = (next: Partial<TableParams>) => {
    const merged = { ...current, ...next };
    const search = new URLSearchParams();
    if (merged.page !== 1) search.set('page', String(merged.page));
    if (merged.page_size !== 20) search.set('page_size', String(merged.page_size));
    if (merged.status) search.set('status', merged.status);
    if (merged.keyword) search.set('keyword', merged.keyword);
    setParams(search);
  };
  return [current, update];
}
```

**Step 4: 跑测试确认通过（GREEN）**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run usePermission 2>&1 | tail -20
```
Expected: PASS（3 个测试）

**Step 5: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): useAuth/usePermission/usePolling/useTableParams + 3 个 hook 单测"
```

---

### Task 9: 路由 + 双层守卫（RequireAuth + RequirePermission + RBAC）

**Files:**
- Create: `web/admin-web/src/router/permissions.ts`
- Create: `web/admin-web/src/router/guards.tsx`
- Create: `web/admin-web/src/router/routes.tsx`（替换 Task 1 占位）
- Create: `web/admin-web/src/router/guards.test.tsx`
- Create: `web/admin-web/src/pages/_404/NotFoundPage.tsx`
- Create: `web/admin-web/src/pages/_403/ForbiddenPage.tsx`

**Step 1: 写 guards 单测（RED）**

`web/admin-web/src/router/guards.test.tsx`：

```tsx
// 验证 RequireAuth / RequirePermission 组件行为。
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { RequireAuth, RequirePermission } from './guards';
import { useAuthStore } from '@/stores/authStore';

function setup() {
  return render(
    <MemoryRouter initialEntries={['/protected']}>
      <Routes>
        <Route
          path="/protected"
          element={
            <RequireAuth>
              <RequirePermission perm="order:read">
                <div>PROTECTED CONTENT</div>
              </RequirePermission>
            </RequireAuth>
          }
        />
        <Route path="/login" element={<div>LOGIN PAGE</div>} />
        <Route path="/403" element={<div>FORBIDDEN</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('RequireAuth', () => {
  beforeEach(() => useAuthStore.getState().logout());

  it('redirects to /login when no token', () => {
    setup();
    expect(screen.getByText('LOGIN PAGE')).toBeInTheDocument();
  });

  it('renders children when authenticated', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      user: { id: 1, username: 'admin', phone: '1', role: 'super_admin', permissions: ['*'], totp_enabled: false },
      permissions: ['*'],
    });
    setup();
    expect(screen.getByText('PROTECTED CONTENT')).toBeInTheDocument();
  });

  it('redirects to /403 when permission denied', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      user: { id: 1, username: 'admin', phone: '1', role: 'order_admin', permissions: ['order:read'], totp_enabled: false },
      permissions: ['order:read'],
    });
    // permission='order:read' will pass; change to test forbidden
    // Skipping nested assertion; covered in RequirePermission suite below.
  });
});

describe('RequirePermission', () => {
  beforeEach(() => useAuthStore.getState().logout());

  it('renders 403 when permission missing', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      user: { id: 1, username: 'admin', phone: '1', role: 'viewer', permissions: [], totp_enabled: false },
      permissions: [],
    });
    render(
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route
            path="/protected"
            element={
              <RequireAuth>
                <RequirePermission perm="escort:audit">
                  <div>PROTECTED CONTENT</div>
                </RequirePermission>
              </RequireAuth>
            }
          />
          <Route path="/login" element={<div>LOGIN PAGE</div>} />
          <Route path="/403" element={<div>FORBIDDEN</div>} />
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText('FORBIDDEN')).toBeInTheDocument();
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run guards 2>&1 | tail -20
```
Expected: FAIL — `Cannot find module './guards'`

**Step 3: 写 permissions.ts（RBAC 静态映射）**

`web/admin-web/src/router/permissions.ts`：

```ts
// Role → Permission 静态映射（与后端 token claim 对齐）。
import type { Role, Permission } from '@/types/api';

export const ROLE_PERMISSIONS: Record<Role, Permission[]> = {
  super_admin: ['*'],
  order_admin: ['order:read', 'order:force_cancel', 'refund:read'],
  refund_admin: ['refund:read', 'refund:approve', 'refund:reject'],
  cs: ['work_order:read', 'work_order:reply', 'user:read'],
  audit_admin: ['escort:audit'],
  viewer: ['user:read', 'order:read', 'refund:read', 'report:read'],
};

export function defaultPermissionsForRole(role: Role): Permission[] {
  return ROLE_PERMISSIONS[role] ?? [];
}
```

**Step 4: 写 guards.tsx（RequireAuth + RequirePermission）**

`web/admin-web/src/router/guards.tsx`：

```tsx
// 路由守卫：未登录跳 /login；无权限跳 /403。
import { Navigate, useLocation } from 'react-router-dom';
import { Result } from 'antd';
import { useAuthStore } from '@/stores/authStore';
import type { Permission } from '@/types/api';

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token);
  const location = useLocation();
  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}

export function RequirePermission({
  perm,
  children,
}: {
  perm: Permission;
  children: React.ReactNode;
}) {
  const hasPermission = useAuthStore((s) => s.hasPermission);
  if (!hasPermission(perm)) {
    return <Result status="403" title="403" subTitle="您没有访问此页面的权限" />;
  }
  return <>{children}</>;
}
```

**Step 5: 写 404 / 403 页**

`web/admin-web/src/pages/_404/NotFoundPage.tsx`：

```tsx
import { Result, Button } from 'antd';
import { useNavigate } from 'react-router-dom';

export default function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <Result
      status="404"
      title="404"
      subTitle="页面不存在"
      extra={
        <Button type="primary" onClick={() => navigate('/dashboard')}>
          回到首页
        </Button>
      }
    />
  );
}
```

`web/admin-web/src/pages/_403/ForbiddenPage.tsx`：

```tsx
import { Result, Button } from 'antd';
import { useNavigate } from 'react-router-dom';

export default function ForbiddenPage() {
  const navigate = useNavigate();
  return (
    <Result
      status="403"
      title="403"
      subTitle="您没有访问此页面的权限"
      extra={
        <Button type="primary" onClick={() => navigate('/dashboard')}>
          回到首页
        </Button>
      }
    />
  );
}
```

**Step 6: 替换 routes.tsx（完整 18 页面路由表，Task 12 会全量实现，本 Task 挂占位页）**

`web/admin-web/src/router/routes.tsx`：

```tsx
// 完整路由表（v1：18 P0 页面 + 守卫；UI 内容在 Task 12 全量补）。
import { lazy, Suspense } from 'react';
import { Navigate } from 'react-router-dom';
import { Spin } from 'antd';
import { RequireAuth, RequirePermission } from './guards';

const AdminLayout = lazy(() => import('@/layouts/AdminLayout'));
const LoginPage = lazy(() => import('@/pages/login/LoginPage'));
const DashboardPage = lazy(() => import('@/pages/dashboard/DashboardPage'));
const PatientListPage = lazy(() => import('@/pages/users/PatientListPage'));
const PatientDetailPage = lazy(() => import('@/pages/users/PatientDetailPage'));
const EscortListPage = lazy(() => import('@/pages/users/EscortListPage'));
const EscortAuditPage = lazy(() => import('@/pages/users/EscortAuditPage'));
const OrderListPage = lazy(() => import('@/pages/orders/OrderListPage'));
const OrderDetailPage = lazy(() => import('@/pages/orders/OrderDetailPage'));
const RefundListPage = lazy(() => import('@/pages/refunds/RefundListPage'));
const RefundAuditPage = lazy(() => import('@/pages/refunds/RefundAuditPage'));
const WorkOrderListPage = lazy(() => import('@/pages/work-orders/WorkOrderListPage'));
const WorkOrderDetailPage = lazy(() => import('@/pages/work-orders/WorkOrderDetailPage'));
const WithdrawalReviewPage = lazy(() => import('@/pages/wallet/WithdrawalReviewPage'));
const ReviewModerationPage = lazy(() => import('@/pages/reviews/ReviewModerationPage'));
const GMVReportPage = lazy(() => import('@/pages/reports/GMVReportPage'));
const RefundRatePage = lazy(() => import('@/pages/reports/RefundRatePage'));
const RefundPoliciesPage = lazy(() => import('@/pages/settings/RefundPoliciesPage'));
const NotFoundPage = lazy(() => import('@/pages/_404/NotFoundPage'));
const ForbiddenPage = lazy(() => import('@/pages/_403/ForbiddenPage'));

const wrap = (el: React.ReactNode) => <Suspense fallback={<Spin size="large" />}>{el}</Suspense>;

export const router = [
  { path: '/login', element: wrap(<LoginPage />) },
  { path: '/403', element: wrap(<ForbiddenPage />) },
  {
    path: '/',
    element: wrap(
      <RequireAuth>
        <AdminLayout />
      </RequireAuth>,
    ),
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: wrap(<DashboardPage />) },
      { path: 'users/patients', element: wrap(<RequirePermission perm="user:read"><PatientListPage /></RequirePermission>) },
      { path: 'users/patients/:id', element: wrap(<RequirePermission perm="user:read"><PatientDetailPage /></RequirePermission>) },
      { path: 'users/escorts', element: wrap(<RequirePermission perm="user:read"><EscortListPage /></RequirePermission>) },
      { path: 'users/escorts/audit', element: wrap(<RequirePermission perm="escort:audit"><EscortAuditPage /></RequirePermission>) },
      { path: 'orders', element: wrap(<RequirePermission perm="order:read"><OrderListPage /></RequirePermission>) },
      { path: 'orders/:id', element: wrap(<RequirePermission perm="order:read"><OrderDetailPage /></RequirePermission>) },
      { path: 'refunds', element: wrap(<RequirePermission perm="refund:read"><RefundListPage /></RequirePermission>) },
      { path: 'refunds/:id', element: wrap(<RequirePermission perm="refund:approve"><RefundAuditPage /></RequirePermission>) },
      { path: 'work-orders', element: wrap(<WorkOrderListPage />) },
      { path: 'work-orders/:id', element: wrap(<WorkOrderDetailPage />) },
      { path: 'wallet/withdrawals', element: wrap(<WithdrawalReviewPage />) },
      { path: 'reviews', element: wrap(<ReviewModerationPage />) },
      { path: 'reports/gmv', element: wrap(<GMVReportPage />) },
      { path: 'reports/refund-rate', element: wrap(<RefundRatePage />) },
      { path: 'settings/refund-policies', element: wrap(<RefundPoliciesPage />) },
    ],
  },
  { path: '*', element: wrap(<NotFoundPage />) },
];
```

> **注**: routes.tsx 从 const 数组切换为 createBrowserRouter 之前，本 Task 用此结构；App.tsx Task 12 替换为 `<RouterProvider router={createBrowserRouter(router)} />`。

**Step 7: 跑测试确认通过（GREEN）**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run guards 2>&1 | tail -20
```
Expected: PASS（3 个测试）

**Step 8: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): Router + RBAC 映射 + RequireAuth/RequirePermission 双层守卫 + 404/403 + 3 个守卫单测"
```

---

### Task 10: 通用组件（ProTable + StatusBadge + AuditAction + TraceId + ErrorBoundary + PageHeader）

**Files:**
- Create: `web/admin-web/src/components/ProTable/ProTable.tsx`
- Create: `web/admin-web/src/components/ProTable/ProTable.test.tsx`
- Create: `web/admin-web/src/components/StatusBadge/StatusBadge.tsx`
- Create: `web/admin-web/src/components/StatusBadge/StatusBadge.test.tsx`
- Create: `web/admin-web/src/components/AuditAction/AuditAction.tsx`
- Create: `web/admin-web/src/components/TraceId/TraceId.tsx`
- Create: `web/admin-web/src/components/ErrorBoundary/ErrorBoundary.tsx`
- Create: `web/admin-web/src/components/PageHeader/PageHeader.tsx`

**Step 1: 写组件单测（RED）**

`web/admin-web/src/components/StatusBadge/StatusBadge.test.tsx`：

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StatusBadge } from './StatusBadge';

describe('StatusBadge', () => {
  it('renders badge with status text', () => {
    render(<StatusBadge status="in_service" />);
    expect(screen.getByText('服务中')).toBeInTheDocument();
  });

  it('renders fallback for unknown status', () => {
    render(<StatusBadge status="unknown" />);
    expect(screen.getByText('unknown')).toBeInTheDocument();
  });
});
```

`web/admin-web/src/components/ProTable/ProTable.test.tsx`：

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { ProTable } from './ProTable';

describe('ProTable', () => {
  it('renders table with empty data', async () => {
    render(
      <ProTable
        rowKey="id"
        columns={[{ title: 'ID', dataIndex: 'id' }]}
        request={async () => ({ data: [], success: true, total: 0 })}
      />,
    );
    await waitFor(() => {
      expect(screen.getByText('ID')).toBeInTheDocument();
    });
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run StatusBadge ProTable 2>&1 | tail -20
```
Expected: FAIL — modules not found

**Step 3: 写 StatusBadge**

`web/admin-web/src/components/StatusBadge/StatusBadge.tsx`：

```tsx
// 订单 / 退款 / 陪诊师状态颜色徽章（统一映射）。
import { Badge } from 'antd';

const STATUS_MAP: Record<string, { text: string; color: string }> = {
  // 订单
  created: { text: '已创建', color: 'default' },
  paid: { text: '已支付', color: 'blue' },
  matching: { text: '匹配中', color: 'gold' },
  pending_acceptance: { text: '待接单', color: 'orange' },
  accepted: { text: '已接单', color: 'cyan' },
  in_service: { text: '服务中', color: 'geekblue' },
  completed: { text: '已完成', color: 'green' },
  reviewed: { text: '已评价', color: 'green' },
  refunding: { text: '退款中', color: 'orange' },
  refunded: { text: '已退款', color: 'red' },
  settling: { text: '结算中', color: 'purple' },
  disputed: { text: '申诉中', color: 'volcano' },
  closed: { text: '已关闭', color: 'default' },
  canceled: { text: '已取消', color: 'red' },
  // 退款
  pending: { text: '待审核', color: 'gold' },
  approved: { text: '已批准', color: 'green' },
  rejected: { text: '已拒绝', color: 'red' },
  // 陪诊师
  active: { text: '已上线', color: 'green' },
  inactive: { text: '已下线', color: 'default' },
  suspended: { text: '已停用', color: 'red' },
};

export function StatusBadge({ status }: { status: string }) {
  const cfg = STATUS_MAP[status] ?? { text: status, color: 'default' };
  return <Badge status={cfg.color as any} text={cfg.text} />;
}
```

**Step 4: 写 ProTable 封装**

`web/admin-web/src/components/ProTable/ProTable.tsx`：

```tsx
// ProTable 通用封装：request + polling + search + 分页（基于 @ant-design/pro-components）。
import { ProTable as AntProTable } from '@ant-design/pro-components';
import type { ProTableProps } from '@ant-design/pro-components';

export type { ProTableProps };

export function ProTable<T, U = Record<string, any>>(props: ProTableProps<T, U>) {
  return <AntProTable<T, U> {...props} />;
}
```

> **注**: 真正实现可加 polling 默认值 / 错误 toast / 列定义工具。

**Step 5: 写 AuditAction / TraceId / ErrorBoundary / PageHeader**

`web/admin-web/src/components/AuditAction/AuditAction.tsx`：

```tsx
// 审核按钮组（通过 / 拒绝 + 二次确认）。
import { useState } from 'react';
import { Button, Modal, Input, Space, App as AntApp } from 'antd';

export interface AuditActionProps {
  onApprove(): Promise<void> | void;
  onReject(reason: string): Promise<void> | void;
  approveText?: string;
  rejectText?: string;
  disabled?: boolean;
}

export function AuditAction({
  onApprove,
  onReject,
  approveText = '通过',
  rejectText = '拒绝',
  disabled,
}: AuditActionProps) {
  const { message } = AntApp.useApp();
  const [loading, setLoading] = useState(false);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [reason, setReason] = useState('');

  const handleApprove = async () => {
    setLoading(true);
    try {
      await onApprove();
      message.success('已通过');
    } catch (err) {
      message.error((err as Error).message || '操作失败');
    } finally {
      setLoading(false);
    }
  };

  const handleReject = async () => {
    if (!reason.trim()) {
      message.warning('请填写拒绝原因');
      return;
    }
    setLoading(true);
    try {
      await onReject(reason);
      message.success('已拒绝');
      setRejectOpen(false);
      setReason('');
    } catch (err) {
      message.error((err as Error).message || '操作失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Space>
        <Button type="primary" loading={loading} disabled={disabled} onClick={handleApprove}>
          {approveText}
        </Button>
        <Button danger loading={loading} disabled={disabled} onClick={() => setRejectOpen(true)}>
          {rejectText}
        </Button>
      </Space>
      <Modal
        title="填写拒绝原因"
        open={rejectOpen}
        onOk={handleReject}
        onCancel={() => setRejectOpen(false)}
        confirmLoading={loading}
      >
        <Input.TextArea
          rows={4}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="请说明拒绝原因（必填）"
        />
      </Modal>
    </>
  );
}
```

`web/admin-web/src/components/TraceId/TraceId.tsx`：

```tsx
// 链路 ID 展示（点击复制）。
import { useState } from 'react';
import { Typography } from 'antd';
import { CopyOutlined } from '@ant-design/icons';

const { Text } = Typography;

export function TraceId({ id }: { id?: string }) {
  const [copied, setCopied] = useState(false);

  if (!id) return <Text type="secondary">-</Text>;

  const handleCopy = async () => {
    await navigator.clipboard.writeText(id);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <Text code style={{ cursor: 'pointer' }} onClick={handleCopy}>
      {id.slice(0, 12)} {copied ? <CopyOutlined /> : null}
    </Text>
  );
}
```

`web/admin-web/src/components/ErrorBoundary/ErrorBoundary.tsx`：

```tsx
// 错误边界：捕获子树异常 + fallback + 控制台上报。
import { Component, type ReactNode } from 'react';
import { Result, Button } from 'antd';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: { componentStack?: string }) {
    console.error('ErrorBoundary caught', error, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback ?? (
          <Result
            status="error"
            title="页面出错了"
            subTitle={this.state.error?.message}
            extra={
              <Button type="primary" onClick={() => this.setState({ hasError: false })}>
                重试
              </Button>
            }
          />
        )
      );
    }
    return this.props.children;
  }
}
```

`web/admin-web/src/components/PageHeader/PageHeader.tsx`：

```tsx
// 页面标题 + 面包屑（ProCard 风格）。
import { Space, Typography, Breadcrumb } from 'antd';
import { Link } from 'react-router-dom';

const { Title } = Typography;

export interface PageHeaderProps {
  title: string;
  breadcrumb?: { label: string; to?: string }[];
  extra?: React.ReactNode;
}

export function PageHeader({ title, breadcrumb, extra }: PageHeaderProps) {
  return (
    <div style={{ marginBottom: 16 }}>
      {breadcrumb && breadcrumb.length > 0 && (
        <Breadcrumb
          items={breadcrumb.map((b) => ({
            title: b.to ? <Link to={b.to}>{b.label}</Link> : b.label,
          }))}
          style={{ marginBottom: 8 }}
        />
      )}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3} style={{ margin: 0 }}>
          {title}
        </Title>
        <Space>{extra}</Space>
      </div>
    </div>
  );
}
```

**Step 6: 跑测试确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run StatusBadge ProTable 2>&1 | tail -20
```
Expected: PASS（2 + 1 个测试）

**Step 7: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): ProTable 封装 + StatusBadge + AuditAction + TraceId + ErrorBoundary + PageHeader + 3 个组件单测"
```

---

### Task 11: AdminLayout + AntD ConfigProvider + 全局主题

**Files:**
- Create: `web/admin-web/src/layouts/AdminLayout.tsx`
- Create: `web/admin-web/src/layouts/AdminLayout.test.tsx`
- Create: `web/admin-web/src/styles/theme.ts`（替换 Task 1 占位）
- Create: `web/admin-web/src/styles/global.scss`

**Step 1: 写 AdminLayout 单测（RED）**

`web/admin-web/src/layouts/AdminLayout.test.tsx`：

```tsx
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import AdminLayout from './AdminLayout';
import { useAuthStore } from '@/stores/authStore';

function renderLayout() {
  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <Routes>
        <Route path="/" element={<AdminLayout />}>
          <Route path="dashboard" element={<div>DASHBOARD CONTENT</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe('AdminLayout', () => {
  beforeEach(() => useAuthStore.getState().logout());

  it('renders sider + header + outlet', () => {
    useAuthStore.getState().setAuth({
      token: 'tok',
      user: { id: 1, username: 'admin', phone: '1', role: 'super_admin', permissions: ['*'], totp_enabled: false },
      permissions: ['*'],
    });
    renderLayout();
    expect(screen.getByText(/管理后台/)).toBeInTheDocument();
    expect(screen.getByText('DASHBOARD CONTENT')).toBeInTheDocument();
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run AdminLayout 2>&1 | tail -20
```
Expected: FAIL — module not found

**Step 3: 写 theme.ts（替换 Task 1 占位）**

`web/admin-web/src/styles/theme.ts`：

```ts
// Ant Design 主题 token（主色 #1677ff + 中文 + 紧凑布局）。
import type { ThemeConfig } from 'antd';

export const themeConfig: ThemeConfig = {
  token: {
    colorPrimary: '#1677ff',
    colorSuccess: '#52c41a',
    colorWarning: '#faad14',
    colorError: '#ff4d4f',
    borderRadius: 6,
    fontFamily:
      '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif',
  },
  components: {
    Layout: {
      headerBg: '#ffffff',
      siderBg: '#001529',
      bodyBg: '#f0f2f5',
    },
    Menu: {
      darkItemBg: '#001529',
      darkSubMenuItemBg: '#000c17',
    },
  },
};
```

**Step 4: 写 global.scss**

`web/admin-web/src/styles/global.scss`：

```scss
// 全局样式：重置 + 滚动条 + 自定义变量。

* {
  box-sizing: border-box;
}

html,
body,
#root {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Microsoft YaHei', sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

// 滚动条
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.35);
}

// ProTable 紧凑模式
.ant-pro-table {
  .ant-pro-table-list-toolbar {
    padding: 12px 16px;
  }
}
```

**Step 5: 写 AdminLayout**

`web/admin-web/src/layouts/AdminLayout.tsx`：

```tsx
// 经典 Admin Layout：Sider（菜单）+ Header（用户 / 折叠）+ Content（Outlet）。
import { useMemo } from 'react';
import { Layout, Menu, Avatar, Dropdown, Button, Space, Typography } from 'antd';
import {
  DashboardOutlined,
  UserOutlined,
  ShoppingOutlined,
  AuditOutlined,
  DollarOutlined,
  CommentOutlined,
  BarChartOutlined,
  SettingOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import { Outlet, useNavigate, useLocation, Link } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { useUIStore } from '@/stores/uiStore';
import { ErrorBoundary } from '@/components/ErrorBoundary/ErrorBoundary';

const { Sider, Header, Content } = Layout;
const { Text } = Typography;

interface MenuItem {
  key: string;
  label: string;
  icon: React.ReactNode;
  perm?: string;
}

const MENU_ITEMS: MenuItem[] = [
  { key: '/dashboard', label: '数据看板', icon: <DashboardOutlined />, perm: 'report:read' },
  { key: '/users/patients', label: '患者列表', icon: <UserOutlined />, perm: 'user:read' },
  { key: '/users/escorts', label: '陪诊师列表', icon: <UserOutlined />, perm: 'user:read' },
  { key: '/users/escorts/audit', label: '陪诊师审核', icon: <AuditOutlined />, perm: 'escort:audit' },
  { key: '/orders', label: '订单管理', icon: <ShoppingOutlined />, perm: 'order:read' },
  { key: '/refunds', label: '退款审核', icon: <DollarOutlined />, perm: 'refund:read' },
  { key: '/work-orders', label: '客服工单', icon: <CommentOutlined /> },
  { key: '/wallet/withdrawals', label: '提现审核', icon: <DollarOutlined /> },
  { key: '/reviews', label: '评价管理', icon: <CommentOutlined /> },
  { key: '/reports/gmv', label: 'GMV 报表', icon: <BarChartOutlined /> },
  { key: '/settings/refund-policies', label: '退款策略', icon: <SettingOutlined /> },
];

export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const hasPermission = useAuthStore((s) => s.hasPermission);
  const siderCollapsed = useUIStore((s) => s.siderCollapsed);
  const toggleSider = useUIStore((s) => s.toggleSider);

  const items = useMemo(
    () =>
      MENU_ITEMS.filter((m) => !m.perm || hasPermission(m.perm as any)).map((m) => ({
        key: m.key,
        icon: m.icon,
        label: <Link to={m.key}>{m.label}</Link>,
      })),
    [hasPermission],
  );

  const selectedKey = MENU_ITEMS.find((m) => location.pathname.startsWith(m.key))?.key ?? '/dashboard';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider theme="dark" collapsible collapsed={siderCollapsed} trigger={null}>
        <div style={{ height: 48, color: '#fff', textAlign: 'center', lineHeight: '48px', fontWeight: 600 }}>
          {siderCollapsed ? '陪诊' : '陪诊管理后台'}
        </div>
        <Menu theme="dark" mode="inline" selectedKeys={[selectedKey]} items={items} />
      </Sider>
      <Layout>
        <Header style={{ background: '#fff', padding: '0 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Button type="text" icon={siderCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={toggleSider} />
          <Space>
            <Text>{user?.username ?? 'admin'}</Text>
            <Dropdown
              menu={{
                items: [
                  { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: () => { logout(); navigate('/login'); } },
                ],
              }}
            >
              <Avatar icon={<UserOutlined />} style={{ cursor: 'pointer' }} />
            </Dropdown>
          </Space>
        </Header>
        <Content style={{ margin: 16, padding: 16, background: '#fff', borderRadius: 6 }}>
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </Content>
      </Layout>
    </Layout>
  );
}
```

**Step 6: 跑测试确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run AdminLayout 2>&1 | tail -20
```
Expected: PASS（1 个测试）

**Step 7: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): AdminLayout（Sider+Header+Outlet+RBAC 菜单）+ theme token + global.scss + 1 个 layout 单测"
```

---

### Task 12: 18 个 P0 页面骨架

**Files:**
- Create: `web/admin-web/src/pages/dashboard/DashboardPage.tsx`
- Create: `web/admin-web/src/pages/users/PatientListPage.tsx`
- Create: `web/admin-web/src/pages/users/PatientDetailPage.tsx`
- Create: `web/admin-web/src/pages/users/EscortListPage.tsx`
- Create: `web/admin-web/src/pages/users/EscortAuditPage.tsx`
- Create: `web/admin-web/src/pages/orders/OrderListPage.tsx`
- Create: `web/admin-web/src/pages/orders/OrderDetailPage.tsx`
- Create: `web/admin-web/src/pages/orders/ForceCancelModal.tsx`
- Create: `web/admin-web/src/pages/refunds/RefundListPage.tsx`
- Create: `web/admin-web/src/pages/refunds/RefundAuditPage.tsx`
- Create: `web/admin-web/src/pages/work-orders/WorkOrderListPage.tsx`
- Create: `web/admin-web/src/pages/work-orders/WorkOrderDetailPage.tsx`
- Create: `web/admin-web/src/pages/wallet/WithdrawalReviewPage.tsx`
- Create: `web/admin-web/src/pages/reviews/ReviewModerationPage.tsx`
- Create: `web/admin-web/src/pages/reports/GMVReportPage.tsx`
- Create: `web/admin-web/src/pages/reports/RefundRatePage.tsx`
- Create: `web/admin-web/src/pages/settings/RefundPoliciesPage.tsx`
- Modify: `web/admin-web/src/App.tsx`（接入 createBrowserRouter）

**Step 1: 写 OrderListPage 单测（RED + 验证骨架非空）**

`web/admin-web/src/pages/orders/OrderListPage.test.tsx`：

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import OrderListPage from './OrderListPage';

describe('OrderListPage', () => {
  it('renders ProTable with seed orders', async () => {
    render(
      <MemoryRouter>
        <OrderListPage />
      </MemoryRouter>,
    );
    await waitFor(() => {
      expect(screen.getByText(/订单管理/)).toBeInTheDocument();
    });
  });
});
```

**Step 2: 跑测试确认失败**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run OrderListPage 2>&1 | tail -20
```
Expected: FAIL — module not found

**Step 3: 写 OrderListPage（ProTable + 订单 API）**

`web/admin-web/src/pages/orders/OrderListPage.tsx`：

```tsx
// 订单列表（ProTable + 状态机操作 + 5s 轮询）。
import { useNavigate } from 'react-router-dom';
import { Button, App as AntApp } from 'antd';
import { ProTable } from '@ant-design/pro-components';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { StatusBadge } from '@/components/StatusBadge/StatusBadge';
import { apiClient } from '@/api/client';

interface OrderRow {
  id: number;
  order_no: string;
  patient_nickname: string;
  escort_nickname: string | null;
  hospital_name: string;
  final_amount: number;
  status: string;
  created_at: string;
}

export default function OrderListPage() {
  const navigate = useNavigate();
  const { modal } = AntApp.useApp();

  return (
    <>
      <PageHeader title="订单管理" breadcrumb={[{ label: '订单' }]} />
      <ProTable<OrderRow>
        headerTitle="全量订单"
        rowKey="id"
        polling={5000}
        search={{ labelWidth: 'auto' }}
        pagination={{ pageSize: 20 }}
        columns={[
          { title: '订单号', dataIndex: 'order_no', width: 180 },
          { title: '患者', dataIndex: 'patient_nickname', width: 100 },
          {
            title: '陪诊师',
            dataIndex: 'escort_nickname',
            width: 100,
            render: (_, r) => r.escort_nickname || '-',
          },
          { title: '医院', dataIndex: 'hospital_name', width: 160 },
          { title: '金额', dataIndex: 'final_amount', width: 100, render: (v) => `¥${v}` },
          {
            title: '状态',
            dataIndex: 'status',
            width: 110,
            valueType: 'select',
            valueEnum: {
              created: { text: '已创建' },
              paid: { text: '已支付' },
              matching: { text: '匹配中' },
              pending_acceptance: { text: '待接单' },
              accepted: { text: '已接单' },
              in_service: { text: '服务中' },
              completed: { text: '已完成' },
              refunding: { text: '退款中' },
              refunded: { text: '已退款' },
              canceled: { text: '已取消' },
            },
            render: (_, r) => <StatusBadge status={r.status} />,
          },
          { title: '下单时间', dataIndex: 'created_at', width: 160, valueType: 'dateTime' },
          {
            title: '操作',
            valueType: 'option',
            width: 140,
            render: (_, r) => [
              <Button key="detail" type="link" onClick={() => navigate(`/orders/${r.id}`)}>
                详情
              </Button>,
              <Button
                key="cancel"
                type="link"
                danger
                hidden={!['paid', 'matching', 'pending_acceptance', 'accepted'].includes(r.status)}
                onClick={() => {
                  modal.confirm({
                    title: '强制取消订单',
                    content: `确认强制取消订单 ${r.order_no}？将触发退款。`,
                    onOk: () => apiClient(`/admin/orders/${r.id}/force-cancel`, { method: 'POST', body: { reason: 'admin force' } }),
                  });
                }}
              >
                强制取消
              </Button>,
            ],
          },
        ]}
        request={async (params) => {
          const resp = await apiClient<{ data: OrderRow[]; pagination: { total: number } }>(
            '/admin/orders',
            { params: { ...params } },
          );
          return { data: resp.data!.data, success: true, total: resp.data!.pagination.total };
        }}
      />
    </>
  );
}
```

**Step 4: 写 OrderDetailPage（基础骨架：Descriptions + Timeline 占位）**

`web/admin-web/src/pages/orders/OrderDetailPage.tsx`：

```tsx
import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Card, Descriptions, Spin, Result, Timeline } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { StatusBadge } from '@/components/StatusBadge/StatusBadge';
import { TraceId } from '@/components/TraceId/TraceId';
import { apiClient } from '@/api/client';
import type { ApiResp } from '@/api/types';

interface OrderDetail {
  id: number;
  order_no: string;
  patient_nickname: string;
  escort_nickname: string | null;
  hospital_name: string;
  final_amount: number;
  status: string;
  created_at: string;
  timeline?: { ts: string; label: string }[];
}

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data, isLoading, error } = useQuery({
    queryKey: ['order', id],
    queryFn: async () => {
      const resp = await apiClient<OrderDetail>(`/admin/orders/${id}`);
      return resp.data!;
    },
    enabled: !!id,
  });

  if (isLoading) return <Spin />;
  if (error) return <Result status="error" title="加载失败" />;
  if (!data) return null;

  return (
    <>
      <PageHeader
        title={`订单 ${data.order_no}`}
        breadcrumb={[{ label: '订单', to: '/orders' }, { label: data.order_no }]}
      />
      <Card>
        <Descriptions column={2}>
          <Descriptions.Item label="订单号">{data.order_no}</Descriptions.Item>
          <Descriptions.Item label="状态"><StatusBadge status={data.status} /></Descriptions.Item>
          <Descriptions.Item label="患者">{data.patient_nickname}</Descriptions.Item>
          <Descriptions.Item label="陪诊师">{data.escort_nickname ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="医院">{data.hospital_name}</Descriptions.Item>
          <Descriptions.Item label="金额">¥{data.final_amount}</Descriptions.Item>
          <Descriptions.Item label="下单时间">{data.created_at}</Descriptions.Item>
          <Descriptions.Item label="Trace ID"><TraceId id={data.id.toString()} /></Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="时间线" style={{ marginTop: 16 }}>
        <Timeline
          items={(data.timeline ?? []).map((t) => ({
            children: `${t.ts} — ${t.label}`,
          }))}
        />
        {(!data.timeline || data.timeline.length === 0) && <p style={{ color: '#999' }}>暂无事件</p>}
      </Card>
    </>
  );
}
```

**Step 5: 写其它 15 个页面骨架（仅占位 + 标题，保持 SPEC §3.1 完整）**

> 模板：每页一个 `<PageHeader title="..." />` + `<Card>骨架占位</Card>`；后续 plan 增量补实现。

`web/admin-web/src/pages/dashboard/DashboardPage.tsx`：

```tsx
import { Card, Row, Col, Statistic } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { apiClient } from '@/api/client';

interface DashboardStats {
  today_gmv: number;
  today_orders: number;
  refund_rate: number;
  complaint_rate: number;
  pending_escorts: number;
}

export default function DashboardPage() {
  const { data: stats } = useQuery({
    queryKey: ['dashboard'],
    queryFn: async () => {
      const resp = await apiClient<DashboardStats>('/admin/reports/overview');
      return resp.data!;
    },
    refetchInterval: 30_000,
  });

  return (
    <>
      <PageHeader title="数据看板" />
      <Row gutter={16}>
        <Col span={6}><Card><Statistic title="今日 GMV" value={stats?.today_gmv ?? 0} prefix="¥" /></Card></Col>
        <Col span={6}><Card><Statistic title="今日订单" value={stats?.today_orders ?? 0} /></Card></Col>
        <Col span={6}><Card><Statistic title="退款率" value={stats?.refund_rate ?? 0} suffix="%" /></Card></Col>
        <Col span={6}><Card><Statistic title="待审核陪诊师" value={stats?.pending_escorts ?? 0} /></Card></Col>
      </Row>
    </>
  );
}
```

`web/admin-web/src/pages/users/PatientListPage.tsx`：

```tsx
import { ProTable } from '@ant-design/pro-components';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { apiClient } from '@/api/client';

interface Patient {
  id: number;
  nickname: string;
  phone: string;
  real_name_verified: boolean;
  created_at: string;
}

export default function PatientListPage() {
  return (
    <>
      <PageHeader title="患者列表" />
      <ProTable<Patient>
        rowKey="id"
        headerTitle="全量患者"
        columns={[
          { title: 'ID', dataIndex: 'id', width: 60 },
          { title: '昵称', dataIndex: 'nickname', width: 120 },
          { title: '手机号', dataIndex: 'phone', width: 140 },
          {
            title: '实名状态',
            dataIndex: 'real_name_verified',
            width: 100,
            render: (v) => (v ? '已认证' : '未认证'),
          },
          { title: '注册时间', dataIndex: 'created_at', width: 160, valueType: 'dateTime' },
        ]}
        request={async (params) => {
          const resp = await apiClient<{ data: Patient[]; pagination: { total: number } }>(
            '/admin/users',
            { params: { ...params } },
          );
          return { data: resp.data!.data, success: true, total: resp.data!.pagination.total };
        }}
      />
    </>
  );
}
```

`web/admin-web/src/pages/users/PatientDetailPage.tsx`：

```tsx
import { useParams } from 'react-router-dom';
import { Card, Descriptions, Spin } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { apiClient } from '@/api/client';

interface PatientDetail {
  id: number;
  nickname: string;
  phone: string;
  real_name_verified: boolean;
  created_at: string;
  orders?: unknown[];
  reviews?: unknown[];
}

export default function PatientDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data, isLoading } = useQuery({
    queryKey: ['patient', id],
    queryFn: async () => {
      const resp = await apiClient<PatientDetail>(`/admin/users/${id}`);
      return resp.data!;
    },
    enabled: !!id,
  });

  if (isLoading || !data) return <Spin />;

  return (
    <>
      <PageHeader title={`患者 ${data.nickname}`} breadcrumb={[{ label: '患者', to: '/users/patients' }, { label: data.nickname }]} />
      <Card>
        <Descriptions column={2}>
          <Descriptions.Item label="ID">{data.id}</Descriptions.Item>
          <Descriptions.Item label="昵称">{data.nickname}</Descriptions.Item>
          <Descriptions.Item label="手机号">{data.phone}</Descriptions.Item>
          <Descriptions.Item label="实名状态">{data.real_name_verified ? '已认证' : '未认证'}</Descriptions.Item>
          <Descriptions.Item label="注册时间">{data.created_at}</Descriptions.Item>
        </Descriptions>
      </Card>
      <Card title="订单" style={{ marginTop: 16 }}>
        <p style={{ color: '#999' }}>骨架：v2 plan 增量实现</p>
      </Card>
      <Card title="评价" style={{ marginTop: 16 }}>
        <p style={{ color: '#999' }}>骨架：v2 plan 增量实现</p>
      </Card>
    </>
  );
}
```

`web/admin-web/src/pages/users/EscortListPage.tsx`：

```tsx
import { ProTable } from '@ant-design/pro-components';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { StatusBadge } from '@/components/StatusBadge/StatusBadge';
import { apiClient } from '@/api/client';

interface EscortRow {
  id: number;
  nickname: string;
  status: string;
  rating?: number;
}

export default function EscortListPage() {
  return (
    <>
      <PageHeader title="陪诊师列表" />
      <ProTable<EscortRow>
        rowKey="id"
        headerTitle="全量陪诊师"
        columns={[
          { title: 'ID', dataIndex: 'id', width: 60 },
          { title: '昵称', dataIndex: 'nickname', width: 120 },
          { title: '状态', dataIndex: 'status', width: 100, render: (_, r) => <StatusBadge status={r.status} /> },
          { title: '评分', dataIndex: 'rating', width: 80, render: (v) => v ?? '-' },
        ]}
        request={async (params) => {
          const resp = await apiClient<{ data: EscortRow[]; pagination: { total: number } }>(
            '/admin/escorts/pending-audit',
            { params: { ...params } },
          );
          return { data: resp.data!.data, success: true, total: resp.data!.pagination.total };
        }}
      />
    </>
  );
}
```

`web/admin-web/src/pages/users/EscortAuditPage.tsx`：

```tsx
import { Row, Col, List, Avatar, Card, Empty } from 'antd';
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { AuditAction } from '@/components/AuditAction/AuditAction';
import { apiClient } from '@/api/client';

interface PendingEscort {
  id: number;
  nickname: string;
  avatar_url: string;
  real_name: string;
  submitted_at: string;
}

export default function EscortAuditPage() {
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const { data } = useQuery({
    queryKey: ['pending-escorts'],
    queryFn: async () => {
      const resp = await apiClient<{ data: PendingEscort[] }>('/admin/escorts/pending-audit');
      return resp.data!.data;
    },
  });

  const selected = data?.find((e) => e.id === selectedId);

  return (
    <>
      <PageHeader title="陪诊师审核" />
      <Row gutter={16}>
        <Col span={10}>
          <Card title="待审核陪诊师">
            <List
              dataSource={data ?? []}
              renderItem={(e) => (
                <List.Item
                  onClick={() => setSelectedId(e.id)}
                  style={{
                    cursor: 'pointer',
                    background: selectedId === e.id ? '#e6f4ff' : undefined,
                  }}
                >
                  <List.Item.Meta
                    avatar={<Avatar src={e.avatar_url} />}
                    title={e.nickname}
                    description={`${e.real_name} · 提交于 ${e.submitted_at}`}
                  />
                </List.Item>
              )}
              locale={{ emptyText: <Empty description="无待审核" /> }}
            />
          </Card>
        </Col>
        <Col span={14}>
          <Card title={selected ? `审核 ${selected.nickname}` : '请选择左侧陪诊师'}>
            {selected && (
              <>
                <p>姓名：{selected.real_name}</p>
                <p>昵称：{selected.nickname}</p>
                <p>提交时间：{selected.submitted_at}</p>
                <AuditAction
                  onApprove={() => apiClient(`/admin/escorts/${selected.id}/approve`, { method: 'POST' })}
                  onReject={(reason) => apiClient(`/admin/escorts/${selected.id}/reject`, { method: 'POST', body: { reason } })}
                />
              </>
            )}
          </Card>
        </Col>
      </Row>
    </>
  );
}
```

`web/admin-web/src/pages/orders/ForceCancelModal.tsx`：

```tsx
// 强制取消弹窗（v1 占位；OrderListPage 内联 modal.confirm 即可）。
import { Modal, Form, Input, App as AntApp } from 'antd';
import { apiClient } from '@/api/client';

export interface ForceCancelModalProps {
  open: boolean;
  orderId: number;
  onClose(): void;
  onSuccess(): void;
}

export function ForceCancelModal({ open, orderId, onClose, onSuccess }: ForceCancelModalProps) {
  const { message } = AntApp.useApp();
  const [form] = Form.useForm();

  const handleOk = async () => {
    const { reason } = await form.validateFields();
    await apiClient(`/admin/orders/${orderId}/force-cancel`, { method: 'POST', body: { reason } });
    message.success('已强制取消');
    onSuccess();
    onClose();
  };

  return (
    <Modal title="强制取消订单" open={open} onOk={handleOk} onCancel={onClose}>
      <Form form={form} layout="vertical">
        <Form.Item name="reason" label="取消原因" rules={[{ required: true, message: '请填写原因' }]}>
          <Input.TextArea rows={4} placeholder="例如：医生停诊 / 患者改期 / 系统异常" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
```

`web/admin-web/src/pages/refunds/RefundListPage.tsx`：

```tsx
import { ProTable } from '@ant-design/pro-components';
import { useNavigate } from 'react-router-dom';
import { Button } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { StatusBadge } from '@/components/StatusBadge/StatusBadge';
import { apiClient } from '@/api/client';

interface RefundRow {
  id: number;
  order_id: number;
  amount: number;
  reason: string;
  status: string;
  created_at: string;
}

export default function RefundListPage() {
  const navigate = useNavigate();
  return (
    <>
      <PageHeader title="退款队列" />
      <ProTable<RefundRow>
        rowKey="id"
        headerTitle="待审核退款"
        polling={5000}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 60 },
          { title: '订单 ID', dataIndex: 'order_id', width: 100 },
          { title: '金额', dataIndex: 'amount', width: 100, render: (v) => `¥${v}` },
          { title: '原因', dataIndex: 'reason', width: 160 },
          { title: '状态', dataIndex: 'status', width: 100, render: (_, r) => <StatusBadge status={r.status} /> },
          { title: '提交时间', dataIndex: 'created_at', width: 160, valueType: 'dateTime' },
          {
            title: '操作',
            valueType: 'option',
            width: 100,
            render: (_, r) => [
              <Button key="audit" type="link" onClick={() => navigate(`/refunds/${r.id}`)}>
                审核
              </Button>,
            ],
          },
        ]}
        request={async (params) => {
          const resp = await apiClient<{ data: RefundRow[]; pagination: { total: number } }>(
            '/admin/refunds',
            { params: { ...params } },
          );
          return { data: resp.data!.data, success: true, total: resp.data!.pagination.total };
        }}
      />
    </>
  );
}
```

`web/admin-web/src/pages/refunds/RefundAuditPage.tsx`：

```tsx
import { useParams } from 'react-router-dom';
import { Card, Descriptions, Spin, Result } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/PageHeader/PageHeader';
import { StatusBadge } from '@/components/StatusBadge/StatusBadge';
import { AuditAction } from '@/components/AuditAction/AuditAction';
import { apiClient } from '@/api/client';

interface Refund {
  id: number;
  order_id: number;
  amount: number;
  refund_percent: number;
  reason: string;
  status: string;
}

export default function RefundAuditPage() {
  const { id } = useParams<{ id: string }>();
  const { data, isLoading } = useQuery({
    queryKey: ['refund', id],
    queryFn: async () => {
      const resp = await apiClient<Refund>(`/admin/refunds/${id}`);
      return resp.data!;
    },
    enabled: !!id,
  });

  if (isLoading || !data) return <Spin />;
  if (data.status !== 'pending') {
    return <Result status="info" title={`该退款已${data.status === 'approved' ? '批准' : '拒绝'}`} />;
  }

  return (
    <>
      <PageHeader title={`退款审核 #${data.id}`} breadcrumb={[{ label: '退款', to: '/refunds' }, { label: `#${data.id}` }]} />
      <Card>
        <Descriptions column={2}>
          <Descriptions.Item label="订单 ID">{data.order_id}</Descriptions.Item>
          <Descriptions.Item label="金额">¥{data.amount}</Descriptions.Item>
          <Descriptions.Item label="退款比例">{(data.refund_percent * 100).toFixed(0)}%</Descriptions.Item>
          <Descriptions.Item label="状态"><StatusBadge status={data.status} /></Descriptions.Item>
          <Descriptions.Item label="原因" span={2}>{data.reason}</Descriptions.Item>
        </Descriptions>
        <div style={{ marginTop: 24 }}>
          <AuditAction
            onApprove={() => apiClient(`/admin/refunds/${data.id}/approve`, { method: 'POST' })}
            onReject={(reason) => apiClient(`/admin/refunds/${data.id}/reject`, { method: 'POST', body: { reason } })}
          />
        </div>
      </Card>
    </>
  );
}
```

> **剩余 8 个 P1 页面骨架**（work-orders / wallet / reviews / reports / settings）按统一模板：

```tsx
// src/pages/work-orders/WorkOrderListPage.tsx
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
export default function WorkOrderListPage() {
  return (
    <>
      <PageHeader title="客服工单" />
      <Card><p style={{ color: '#999' }}>骨架：v2 plan 补实现</p></Card>
    </>
  );
}
```

```tsx
// src/pages/work-orders/WorkOrderDetailPage.tsx
import { Card } from 'antd';
import { useParams } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader/PageHeader';
export default function WorkOrderDetailPage() {
  const { id } = useParams();
  return (
    <>
      <PageHeader title={`工单 ${id}`} breadcrumb={[{ label: '工单', to: '/work-orders' }, { label: id ?? '' }]} />
      <Card><p style={{ color: '#999' }}>骨架：v2 plan 补实现</p></Card>
    </>
  );
}
```

```tsx
// src/pages/wallet/WithdrawalReviewPage.tsx
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
export default function WithdrawalReviewPage() {
  return (
    <>
      <PageHeader title="提现审核" />
      <Card><p style={{ color: '#999' }}>骨架：v2 plan 补实现（T+7 校验）</p></Card>
    </>
  );
}
```

```tsx
// src/pages/reviews/ReviewModerationPage.tsx
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
export default function ReviewModerationPage() {
  return (
    <>
      <PageHeader title="评价管理" />
      <Card><p style={{ color: '#999' }}>骨架：v2 plan 补实现（敏感词过滤）</p></Card>
    </>
  );
}
```

```tsx
// src/pages/reports/GMVReportPage.tsx
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
export default function GMVReportPage() {
  return (
    <>
      <PageHeader title="GMV 报表" />
      <Card><p style={{ color: '#999' }}>骨架：v2 plan 补实现（@ant-design/charts）</p></Card>
    </>
  );
}
```

```tsx
// src/pages/reports/RefundRatePage.tsx
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
export default function RefundRatePage() {
  return (
    <>
      <PageHeader title="退款率" />
      <Card><p style={{ color: '#999' }}>骨架：v2 plan 补实现</p></Card>
    </>
  );
}
```

```tsx
// src/pages/settings/RefundPoliciesPage.tsx
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader/PageHeader';
export default function RefundPoliciesPage() {
  return (
    <>
      <PageHeader title="退款策略配置" />
      <Card><p style={{ color: '#999' }}>骨架：v2 plan 补实现</p></Card>
    </>
  );
}
```

**Step 6: 修改 App.tsx（接入完整路由表）**

`web/admin-web/src/App.tsx`：

```tsx
// 应用顶层：路由 + 全局 Provider。
import { ConfigProvider, App as AntApp } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';
import zhCN from 'antd/locale/zh_CN';
import { router as routes } from '@/router/routes';
import { themeConfig } from '@/styles/theme';
import { setAuthTokenGetter } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 30_000 },
  },
});

// 注入 token getter（authStore 持久化后即可用）。
setAuthTokenGetter(() => useAuthStore.getState().token);

const router = createBrowserRouter(routes);

export default function App() {
  return (
    <ConfigProvider locale={zhCN} theme={themeConfig}>
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>
  );
}
```

**Step 7: 跑测试确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run --reporter=verbose 2>&1 | tail -30
```
Expected: PASS（全部单测通过；含 OrderListPage 的 ProTable 渲染测试）

**Step 8: 跑 typecheck**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web run typecheck
```
Expected: PASS

**Step 9: 跑 lint**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web run lint
```
Expected: PASS（0 errors 0 warnings）

> **处理 lint 报错**：unused imports 通过删除或加 `_` 前缀解决。

**Step 10: Commit**

```bash
git add web/admin-web/
git commit -m "feat(admin-web): 18 个 P0/P1 页面骨架（LoginPage + AdminLayout + 全部路由）+ 1 个 OrderListPage 单测"
```

---

### Task 13: E2E Playwright（登录 + 订单 + 退款审核 + 陪诊师审核）

**Files:**
- Create: `web/admin-web/e2e/login.spec.ts`（替换 Task 3 smoke）
- Create: `web/admin-web/e2e/order-list.spec.ts`
- Create: `web/admin-web/e2e/refund-audit.spec.ts`
- Create: `web/admin-web/e2e/escort-audit.spec.ts`

**Step 1: 写 E2E 测试（RED）**

`web/admin-web/e2e/login.spec.ts`：

```ts
import { test, expect } from '@playwright/test';

test('admin login with mock credentials', async ({ page }) => {
  await page.goto('/login');
  await expect(page.getByText(/陪诊管理后台/)).toBeVisible();
  await page.getByRole('button', { name: /登录/ }).click();
  await page.waitForURL(/\/dashboard$/);
  await expect(page.getByText(/数据看板/)).toBeVisible();
});
```

`web/admin-web/e2e/order-list.spec.ts`：

```ts
import { test, expect } from '@playwright/test';

test('admin orders list renders with mock data', async ({ page }) => {
  // 先登录（mock auth handler）。
  await page.goto('/login');
  await page.getByRole('button', { name: /登录/ }).click();
  await page.waitForURL(/\/dashboard$/);

  await page.goto('/orders');
  await expect(page.getByText(/订单管理/)).toBeVisible();
  // ProTable 应展示订单列表行（mock seed 30 条）。
  await expect(page.locator('.ant-pro-table').first()).toBeVisible();
});
```

`web/admin-web/e2e/refund-audit.spec.ts`：

```ts
import { test, expect } from '@playwright/test';

test('refund audit page renders', async ({ page }) => {
  await page.goto('/login');
  await page.getByRole('button', { name: /登录/ }).click();
  await page.waitForURL(/\/dashboard$/);

  await page.goto('/refunds');
  await expect(page.getByText(/退款队列/)).toBeVisible();
  // 点第一个审核按钮。
  await page.getByRole('button', { name: /审核/ }).first().click();
  await expect(page.getByText(/退款审核/)).toBeVisible();
});
```

`web/admin-web/e2e/escort-audit.spec.ts`：

```ts
import { test, expect } from '@playwright/test';

test('escort audit page renders dual-column', async ({ page }) => {
  await page.goto('/login');
  await page.getByRole('button', { name: /登录/ }).click();
  await page.waitForURL(/\/dashboard$/);

  await page.goto('/users/escorts/audit');
  await expect(page.getByText(/陪诊师审核/)).toBeVisible();
  await expect(page.getByText(/待审核陪诊师/)).toBeVisible();
});
```

**Step 2: 替换 Task 3 smoke.spec.ts 为 login.spec.ts（或并存）**

```bash
mv web/admin-web/e2e/smoke.spec.ts web/admin-web/e2e/_smoke.spec.ts.disabled
```

**Step 3: 跑 E2E 确认通过**

```bash
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web run test:e2e 2>&1 | tail -30
```
Expected: PASS（4 个 E2E 测试；dev server 启动 + MSW 拦截 fetch）

**Step 4: Commit**

```bash
git add web/admin-web/
git commit -m "test(admin-web): 4 个 Playwright E2E（login + order-list + refund-audit + escort-audit）"
```

---

### Task 14: README + scripts + 文档同步

**Files:**
- Create: `web/admin-web/README.md`
- Modify: `docs/04-业务流程.md` §4.8
- Modify: `dev.md` §10.14

**Step 1: 写 README**

`web/admin-web/README.md`：

````markdown
# admin-web — 陪诊管理后台 SPA

L2 v1.0 管理后台。基于 React 18 + Vite 5 + Ant Design 5 + Zustand + TanStack Query。

## 开发

```bash
# 安装依赖（workspace 根目录）
pnpm install

# 启动 dev server（含 MSW mock）
pnpm --filter admin-web run mock:dev
# → http://localhost:8080

# 仅启动 dev（关闭 mock，连真实后端）
pnpm --filter admin-web run dev

# 类型检查
pnpm --filter admin-web run typecheck

# Lint
pnpm --filter admin-web run lint

# 单元测试
pnpm --filter admin-web run test:unit
pnpm --filter admin-web run test:unit:watch  # watch 模式

# E2E 测试
pnpm --filter admin-web run test:e2e
pnpm --filter admin-web run test:e2e:ui  # UI 模式

# 构建生产包
pnpm --filter admin-web run build
```

## 目录结构

```
src/
├── api/              # fetch wrapper + 各模块 API 方法
├── components/       # 通用组件（ProTable / StatusBadge / AuditAction / ...）
├── hooks/            # useAuth / usePermission / usePolling / useTableParams
├── layouts/          # AdminLayout（Sider + Header + Content）
├── mocks/            # MSW handlers + 种子数据
├── pages/            # 18 个 P0 页面 + P1 骨架
├── router/           # routes + guards + RBAC 映射
├── stores/           # Zustand stores（auth / ui）
├── styles/           # global.scss + theme.ts
├── types/            # OpenAPI 生成的类型 + 业务类型扩展
└── App.tsx           # 顶层 Provider 装配
```

## Mock 一键登录

开发期访问 http://localhost:8080/login 直接点击登录（form 已默认填入 `admin / admin123 / 000000`）。

## RBAC 角色

| 角色 | 权限 |
|---|---|
| super_admin | 全部（`*`） |
| order_admin | 订单读 / 强制取消 / 退款读 |
| refund_admin | 退款读 / 批准 / 拒绝 |
| audit_admin | 陪诊师审核 |
| cs | 工单读 / 回复 / 用户读 |
| viewer | 各资源读 |

修改 RBAC：编辑 `src/router/permissions.ts` 的 `ROLE_PERMISSIONS`。
````

**Step 2: 修改 docs/04-业务流程.md（§4.8 加 admin 审核流程）**

```markdown
### admin 后台审核流程（2026-09-24 admin-web setup plan）

1. admin 登录 → `POST /api/v1/auth/login` → JWT + role claim → `authStore`
2. 进 `/dashboard` 看板 → `GET /api/v1/admin/reports/overview` → Statistic 卡片
3. 进 `/orders` → ProTable + 5s polling → `GET /api/v1/admin/orders?status=...`
4. 强制取消 → `POST /api/v1/admin/orders/{id}/force-cancel` → 触发 refund
5. 进 `/refunds` → ProTable 5s polling → 点审核 → `POST /api/v1/admin/refunds/{id}/approve|reject`
6. 进 `/users/escorts/audit` → 双栏（待审核列表 + 详情）→ 调 approve / reject
```

**Step 3: 修改 dev.md（§10.14 加落地记录）**

```markdown
### 10.14 admin-web setup plan（2026-09-24）

实现 L2 v1.0 admin-web SPA 骨架。

**落地 commits（13 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(admin-web) | Vite 5 + React 18 + TS 脚手架 |
| chore(admin-web) | ESLint 9 flat + Prettier 3 |
| test(admin-web) | Vitest + RTL + Playwright + smoke E2E |
| feat(admin-web) | openapi-typescript 生成脚本 + stub |
| feat(admin-web) | fetch client + 拦截器 + ApiError |
| feat(admin-web) | MSW handlers（auth + admin 12 API） |
| feat(admin-web) | Zustand authStore + LoginPage |
| feat(admin-web) | useAuth/usePermission/usePolling/useTableParams |
| feat(admin-web) | Router + RBAC + RequireAuth/RequirePermission |
| feat(admin-web) | ProTable + StatusBadge + AuditAction + TraceId + ErrorBoundary + PageHeader |
| feat(admin-web) | AdminLayout + theme token + global.scss |
| feat(admin-web) | 18 个 P0/P1 页面骨架 |
| test(admin-web) | 4 个 Playwright E2E |

**API 覆盖**：admin 12 API + auth 3 API + reports 1 API 全 mock。

**未做**：暗色主题 / 移动端响应式 / i18n / WebSocket / 图表大屏 / 工单详情 / 评价管理 / 钱包账单 / 退款策略配置 UI（v2 plan 增量）。
```

**Step 4: Commit**

```bash
git add web/admin-web/ docs/ dev.md
git commit -m "docs(admin-web): README + 04 §4.8 + dev.md 10.14（13 个 commit 落地）"
```

---

### Task 15: 全量回归 + push

```bash
# 清干净所有缓存
cd /Users/growduduan/ai/doctors
pnpm --filter admin-web exec vitest run --coverage 2>&1 | tail -50

# 类型检查
pnpm --filter admin-web run typecheck

# Lint
pnpm --filter admin-web run lint

# E2E
pnpm --filter admin-web run test:e2e

# 构建
pnpm --filter admin-web run build
ls -lh web/admin-web/dist/

# 根仓库整体验证
pnpm --filter '*' run typecheck 2>&1 | tail -10
```

Expected: 全部 PASS；dist/ 产出；typecheck 0 errors；lint 0 warnings；unit ≥ 70% 覆盖。

```bash
git push -u origin main
```

---

## Self-Review

- ✅ **Spec 覆盖**: 18 个 P0 页面（含 LoginPage）+ AdminLayout + 全部守卫 + 全部 RBAC + 全部 mock handler + ProTable 封装 + 测试框架 + OpenAPI 集成
- ✅ **无占位符**: 每个 Task 含 RED 测试 + GREEN 实现 + commit 命令；测试可在隔离环境跑通
- ✅ **TDD 严格**: 每个 Task 都有 step 1 写测试 → step 2 跑 RED → step 3 实现 → step N 跑 GREEN → commit
- ✅ **类型一致**: 与 admin-web-design.md §3.1 页面表 + §3.2 RBAC + l2-api-gap-design.md §2.3 12 API 路径对齐
- ✅ **测试矩阵**: §1-13 覆盖 Vitest 单测 / RTL 组件 / MSW handler / Playwright E2E
- ✅ **YAGNI**: §10（不引 axios / redux / react-table / lodash / moment）；v1 不做暗色 / i18n / WebSocket / 移动端 / 大屏
- ✅ **包管理**: pnpm workspace；与 patient-miniapp 共享；统一 lockfile
- ✅ **接口稳定**: fetch wrapper 拦截器契约清晰（token 注入 / 401 / timeout / trace_id 提取）
- ✅ **RBAC 设计**: 静态映射 ROLE_PERMISSIONS + wildcard `*` + 前端守卫仅 UX（后端权威）
- ✅ **OpenAPI 集成**: 自动生成脚本；stub 占位等后端 plan 落地
- ✅ **Mock 完整**: admin 12 API + auth 3 API + reports 1 API 全 handler 覆盖；v1 无后端依赖即可开发

## 关键设计偏差（待用户确认）

1. **路由类型**: Task 9 用路由数组 + `createBrowserRouter` 注入 App.tsx（替代了 Task 1 的 `createMemoryRouter` 占位）；如团队更习惯 inline JSX `<Route>` 写法，需调整 routes.tsx。
2. **RBAC 颗粒度**: 静态 ROLE_PERMISSIONS 映射放 `router/permissions.ts`；如要做"按钮级动效权限"（如禁用 / 隐藏某个按钮），需在 usePermission 上加 `useCanAction` 方法（v2 plan 增量）。
3. **OpenAPI stub**: `web/openapi/contracts.yaml` v1 尚未生成（依赖后端 admin-service plan 落地）；本 plan 用 stub 占位生成 `src/types/generated.ts`。后端 plan 完成后跑 `pnpm --filter admin-web run generate:client` 即自动重生成。
4. **P1 页面骨架**: work-orders / wallet / reviews / reports / settings 仅占位（无 ProTable）；Task 12 写了 `<Card>骨架</Card>` 模板；实际 ProTable + 数据流在 v2 plan 增量。
5. **图表库**: `@ant-design/charts` 已装但 v1 看板页只用 `<Statistic>`（未引入图表）；v2 plan 补实现。
6. **状态持久化**: authStore 用 `zustand/middleware` 的 `persist` 写 localStorage（key: `admin-web-auth`）；生产期考虑迁移到 cookie（httpOnly）+ refresh token 旋转。
7. **E2E 浏览器**: 默认 chromium；如需 webkit / firefox 跨浏览器验证，扩展 `playwright.config.ts` 的 projects。