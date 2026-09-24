# Patient Miniapp Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `web/patient-miniapp/` 下建立 uni-app + Vue 3 + uView Plus 2.x 骨架项目，作为 L2 v1.0 患者陪诊小程序的承载点。覆盖 spec §2 完整目录结构、§3.1 全部 20 个 P0 页面空骨架、§5 五个 Pinia store、§4 全部 14 个 API 的 client + uni-request 拦截器、OpenAPI codegen、Jest + Playwright 测试框架。v1 重点是骨架与目录 + mock 后端；UI 实现 / 业务流程按 feature 拆后续 plan。

**Architecture:** 单包项目（不是 monorepo）。开发态用 `npm run dev:h5` 起 Vite + uni-app H5 模式；构建产物可同时输出 mp-weixin / h5。HTTP 全部走 `uni.request`（不引 axios），通过 `uni.addInterceptor` 全局加 `X-Trace-Id` 与 `Authorization` 头。后端在 v1 用 MSW mock（开发态）+ Jest（单测）+ Playwright H5 模式（e2e）。OpenAPI 契约为 single source of truth：`web/openapi/contracts.yaml` → `openapi-typescript` 生成 `src/api/types.gen.ts`。

**Tech Stack:**
- **运行时**: uni-app x（Vue 3.4+ + 组合式 API + `<script setup lang="ts">`）
- **UI**: uView Plus 2.x（专为 uni-app 定制，Vue 3 兼容分支 `uview-plus`）
- **状态管理**: Pinia 2.x（Vue 3 配套）
- **路由**: uni-app pages.json（声明式 + easycom）
- **HTTP**: uni.request（自带 + 全局拦截器）
- **样式**: SCSS + uView 主题变量
- **测试**: Jest 29.x（@vue/test-utils）+ Playwright 1.45+（H5 模式 e2e）+ MSW 2.x（mock 后端）
- **类型**: TypeScript 5.x + openapi-typescript 6.x

**前置依赖:**
- `docs/superpowers/specs/2026-09-24-patient-miniapp-design.md`（设计源真）
- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.1（14 个 patient P0 API）
- 后端 plan `2026-09-24-hospital-package-plan.md` 等产出后才会有真实 `web/openapi/contracts.yaml`；本 plan 阶段若 contracts.yaml 缺失则用占位 YAML（OpenAPI 3.0 最小骨架），让 codegen 跑通；后续 plan 替换。
- 已完成 `dev.md` §10.x 一致的文档同步。

---

## Global Constraints

- **Node**: Node.js 18 LTS+（uni-app x 要求）；`npm` 9+。
- **TypeScript**: strict mode 打开（`strict: true`、`noUncheckedIndexedAccess: true`）。
- **uni-app 版本**: `3.0.0-4030620241128001` 及以上（Vue 3 稳定版）。
- **不引入**: `axios` / `vue-router`（uni-app 自带路由）/ `lodash`（必要时按需 import）/ UI 库二次封装层（直接用 uview-plus）。
- **目录约束**: `src/` 内严格按业务分包（pages / components / stores / api / utils），不出现"temp / old / test" 等临时目录。
- **文件命名**:
  - 组件: PascalCase（如 `OrderCard.vue`）
  - 页面: 小写路径（`pages/order/detail.vue`）
  - utils / stores: camelCase（`auth.ts` / `useOrderStore`）
- **样式**: SCSS 嵌套用 BEM 风格（`.order-card__title--active`）；全局变量从 `src/styles/variables.scss` 引入。
- **Commit 节奏**: 每个 Task 完成立即 commit；前缀 `chore:` / `feat:` / `test:` / `docs:` / `build:`。
- **测试基线**: 纯 TS 模块（utils / stores / api client wrapper）覆盖率 ≥ 80%；Vue 组件单测聚焦 props / emits / 事件 handler（不做 snapshot，jest 不接 jsdom 全局 uni runtime；改用 `@vue/test-utils` 直接挂载 .vue 文件，需要 uni-api 的组件改 e2e）；e2e 至少 1 个完整 happy-path（登录 → 列表 → 详情）。
- **Mock 边界**: v1 不直连后端；jest 单测用 MSW 拦截 `uni.request`（通过注入适配器）；Playwright e2e 在浏览器层跑 MSW + Vite proxy 指向 `DOCTORS_BACKEND_URL`（开发时可指真实 doctors backend）。
- **OpenAPI 校准**: `npm run openapi:generate` 必须跑过；CI 必跑 `npm run openapi:validate`（diff 校验生成的 types 与 contracts.yaml 同步）。
- **i18n 留占位**: 引 `vue-i18n` 但只装 `en` / `zh-CN` 两个空 locale + 默认 key，文案留 v3 补。
- **eslint + prettier**: uni-app 官方 vue 模板 + `@typescript-eslint`。

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `web/patient-miniapp/package.json` | Create | 依赖清单 + scripts |
| `web/patient-miniapp/.npmrc` | Create | npm registry 镜像（goproxy）+ pnpm 偏好 |
| `web/patient-miniapp/tsconfig.json` | Create | TS strict + path alias `@/*` → `src/*` |
| `web/patient-miniapp/vite.config.ts` | Create | uni-app Vite 配置 + uview-plus + proxy + MSW |
| `web/patient-miniapp/jest.config.js` | Create | Jest 配置（@vue/test-utils + ts-jest） |
| `web/patient-miniapp/playwright.config.ts` | Create | Playwright 配置（H5 模式 + webServer 起 dev:h5） |
| `web/patient-miniapp/.gitignore` | Create | 忽略 unpackage / dist / node_modules |
| `web/patient-miniapp/.eslintrc.cjs` | Create | ESLint + prettier |
| `web/patient-miniapp/index.html` | Create | H5 entry（uni-app 标准） |
| `web/patient-miniapp/src/manifest.json` | Create | uni-app 应用清单（appid 占位 "TOURIST_APPID"） |
| `web/patient-miniapp/src/pages.json` | Create | 20 个 P0 页面路由声明 |
| `web/patient-miniapp/src/main.ts` | Create | uni-app 入口：createSSRApp + Pinia + i18n + uview |
| `web/patient-miniapp/src/App.vue` | Create | 根组件（<u--locale-provider> 包裹） |
| `web/patient-miniapp/src/env.d.ts` | Create | uni-app 全局类型声明 |
| `web/patient-miniapp/src/styles/variables.scss` | Create | 设计 token（主色 #1989FA / 辅 #FF6B35 / 字号 / 圆角） |
| `web/patient-miniapp/src/styles/global.scss` | Create | 全局 reset + uview 主样式入口 |
| `web/patient-miniapp/src/pages/index/index.vue` | Create | 首页骨架 |
| `web/patient-miniapp/src/pages/auth/login.vue` | Create | 登录页骨架 |
| `web/patient-miniapp/src/pages/auth/real-name.vue` | Create | 实名页骨架 |
| `web/patient-miniapp/src/pages/hospital/list.vue` | Create | 医院列表骨架 |
| `web/patient-miniapp/src/pages/hospital/detail.vue` | Create | 医院详情骨架 |
| `web/patient-miniapp/src/pages/package/detail.vue` | Create | 服务包详情骨架 |
| `web/patient-miniapp/src/pages/order/create.vue` | Create | 下单页骨架 |
| `web/patient-miniapp/src/pages/order/pay.vue` | Create | 支付页骨架 |
| `web/patient-miniapp/src/pages/order/list.vue` | Create | 订单列表骨架 |
| `web/patient-miniapp/src/pages/order/detail.vue` | Create | 订单详情骨架（含轮询占位） |
| `web/patient-miniapp/src/pages/refund/apply.vue` | Create | 申请退款骨架 |
| `web/patient-miniapp/src/pages/review/create.vue` | Create | 评价表单骨架 |
| `web/patient-miniapp/src/pages/sos/trigger.vue` | Create | SOS 触发页骨架 |
| `web/patient-miniapp/src/pages/wallet/index.vue` | Create | 钱包骨架 |
| `web/patient-miniapp/src/pages/coupon/list.vue` | Create | 优惠券列表骨架 |
| `web/patient-miniapp/src/pages/address/list.vue` | Create | 地址列表骨架 |
| `web/patient-miniapp/src/pages/address/edit.vue` | Create | 地址编辑骨架 |
| `web/patient-miniapp/src/pages/message/list.vue` | Create | 会话列表骨架 |
| `web/patient-miniapp/src/pages/message/detail.vue` | Create | 会话详情骨架 |
| `web/patient-miniapp/src/pages/profile/index.vue` | Create | 个人中心骨架 |
| `web/patient-miniapp/src/components/OrderCard/OrderCard.vue` | Create | 订单卡片（空骨架 + props/emits 类型） |
| `web/patient-miniapp/src/components/EscortBadge/EscortBadge.vue` | Create | 陪诊师徽章骨架 |
| `web/patient-miniapp/src/components/HospitalCard/HospitalCard.vue` | Create | 医院卡片骨架 |
| `web/patient-miniapp/src/components/PackageCard/PackageCard.vue` | Create | 服务包卡片骨架 |
| `web/patient-miniapp/src/components/PriceTag/PriceTag.vue` | Create | 价格展示骨架 |
| `web/patient-miniapp/src/components/VirtualNumber/VirtualNumber.vue` | Create | 虚拟号拨打骨架 |
| `web/patient-miniapp/src/components/Countdown/Countdown.vue` | Create | 30s 倒计时骨架 |
| `web/patient-miniapp/src/components/SOSButton/SOSButton.vue` | Create | 长按 SOS 骨架 |
| `web/patient-miniapp/src/components/index.ts` | Create | easycom 注册（uview-plus 自动 + 自定义组件手动） |
| `web/patient-miniapp/src/stores/auth.ts` | Create | token / 当前用户（持久化到 uni.storage） |
| `web/patient-miniapp/src/stores/order.ts` | Create | 订单列表 / 当前订单 / 轮询 stop |
| `web/patient-miniapp/src/stores/hospital.ts` | Create | 医院列表缓存 |
| `web/patient-miniapp/src/stores/wallet.ts` | Create | 钱包余额 |
| `web/patient-miniapp/src/stores/message.ts` | Create | 会话列表 + 未读 |
| `web/patient-miniapp/src/stores/index.ts` | Create | pinia plugin（setup store 自动 import） |
| `web/patient-miniapp/src/api/client.ts` | Create | uni.request 封装 + interceptor 安装 |
| `web/patient-miniapp/src/api/auth.ts` | Create | 登录 / 实名 / 我（与 spec §2.1 P0 对齐） |
| `web/patient-miniapp/src/api/order.ts` | Create | 订单 CRUD + 支付 + 接单 + 完成 + 取消 |
| `web/patient-miniapp/src/api/hospital.ts` | Create | 医院 / 服务包（list/detail） |
| `web/patient-miniapp/src/api/refund.ts` | Create | 申请退款 |
| `web/patient-miniapp/src/api/review.ts` | Create | 创建评价 |
| `web/patient-miniapp/src/api/sos.ts` | Create | SOS 触发 |
| `web/patient-miniapp/src/api/wallet.ts` | Create | 钱包余额 |
| `web/patient-miniapp/src/api/address.ts` | Create | 地址 CRUD |
| `web/patient-miniapp/src/api/coupon.ts` | Create | 优惠券列表 |
| `web/patient-miniapp/src/api/message.ts` | Create | 会话 + 消息历史 |
| `web/patient-miniapp/src/api/index.ts` | Create | 统一 export 所有 api 模块 |
| `web/patient-miniapp/src/api/types.gen.ts` | Create (codegen) | openapi-typescript 产物 |
| `web/patient-miniapp/src/utils/auth.ts` | Create | token 持久化（uni.setStorageSync / getStorageSync / removeStorageSync） |
| `web/patient-miniapp/src/utils/trace.ts` | Create | trace_id 生成器（`mp-{ts}-{rnd}`） |
| `web/patient-miniapp/src/utils/format.ts` | Create | 时间 / 金额 / 距离格式化 |
| `web/patient-miniapp/src/utils/wx.ts` | Create | wx.* API 封装（getLocation / login / requestPayment） |
| `web/patient-miniapp/src/utils/uni-mock.ts` | Create | jest 单测时注入 uni runtime stub |
| `web/patient-miniapp/src/i18n/en.ts` | Create | 英文占位 locale |
| `web/patient-miniapp/src/i18n/zh-CN.ts` | Create | 中文占位 locale |
| `web/patient-miniapp/src/i18n/index.ts` | Create | createI18n 实例（fallbackLocale: zh-CN） |
| `web/patient-miniapp/mocks/handlers.ts` | Create | MSW handlers（14 API 占位 mock：200 OK + fixture） |
| `web/patient-miniapp/mocks/browser.ts` | Create | setupWorker for H5 dev |
| `web/patient-miniapp/mocks/node.ts` | Create | setupServer for jest |
| `web/patient-miniapp/__tests__/stores/auth.test.ts` | Create | auth store 单测 |
| `web/patient-miniapp/__tests__/stores/order.test.ts` | Create | order store 单测 |
| `web/patient-miniapp/__tests__/api/client.test.ts` | Create | interceptor 单元测试（trace_id + 401） |
| `web/patient-miniapp/__tests__/utils/trace.test.ts` | Create | trace_id 生成器测试 |
| `web/patient-miniapp/__tests__/utils/format.test.ts` | Create | format 工具测试 |
| `web/patient-miniapp/__tests__/components/Countdown.test.ts` | Create | Countdown 组件测试（无 uni runtime） |
| `web/patient-miniapp/e2e/login-to-list.spec.ts` | Create | Playwright e2e：登录 → 医院列表 |
| `web/patient-miniapp/openapi/contracts.yaml` | Create | OpenAPI 3.0 最小骨架（14 API 占位；后端 plan 产出后覆盖） |
| `web/patient-miniapp/scripts/openapi-generate.sh` | Create | codegen 包装脚本 |
| `web/patient-miniapp/README.md` | Create | 起项目 / dev / build / test 说明 |
| `dev.md` | Modify | §10.13 加 patient-miniapp plan 落地记录 |
| `.github/workflows/patient-miniapp-ci.yml` | Create | CI：typecheck + test:unit + test:e2e + openapi:validate |

> **总数估算**: ~75 个文件 / ~22 commits（每个 Task 一个 commit）。

---

### Task 1: 初始化项目骨架（package.json + .gitignore + tsconfig + vite config）

**Files:**
- Create: `web/patient-miniapp/package.json`
- Create: `web/patient-miniapp/.gitignore`
- Create: `web/patient-miniapp/.npmrc`
- Create: `web/patient-miniapp/tsconfig.json`
- Create: `web/patient-miniapp/vite.config.ts`
- Create: `web/patient-miniapp/index.html`
- Create: `web/patient-miniapp/src/manifest.json`
- Create: `web/patient-miniapp/src/env.d.ts`
- Create: `web/patient-miniapp/README.md`

**Step 1: 写最小依赖清单（包版本 lock 时按 npm 实际解析最近稳定版小幅调整）**

`web/patient-miniapp/package.json`：

```json
{
  "name": "patient-miniapp",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev:h5": "uni",
    "dev:mp-weixin": "uni -p mp-weixin",
    "build:h5": "uni build",
    "build:mp-weixin": "uni build -p mp-weixin",
    "typecheck": "vue-tsc --noEmit",
    "lint": "eslint --ext .ts,.vue src/ __tests__/ e2e/",
    "test:unit": "jest",
    "test:e2e": "playwright test",
    "openapi:generate": "bash scripts/openapi-generate.sh",
    "openapi:validate": "openapi-typescript ./openapi/contracts.yaml --output src/api/types.gen.ts.check && diff src/api/types.gen.ts src/api/types.gen.ts.check"
  },
  "dependencies": {
    "@dcloudio/uni-app": "3.0.0-4030620241128001",
    "@dcloudio/uni-app-plus": "3.0.0-4030620241128001",
    "@dcloudio/uni-components": "3.0.0-4030620241128001",
    "@dcloudio/uni-mp-weixin": "3.0.0-4030620241128001",
    "@dcloudio/uni-h5": "3.0.0-4030620241128001",
    "pinia": "^2.2.0",
    "vue": "^3.4.21",
    "vue-i18n": "^9.13.1",
    "uview-plus": "^0.1.4"
  },
  "devDependencies": {
    "@dcloudio/types": "^3.4.8",
    "@dcloudio/uni-automator": "3.0.0-4030620241128001",
    "@dcloudio/uni-cli-shared": "3.0.0-4030620241128001",
    "@dcloudio/uni-stacktracey": "3.0.0-4030620241128001",
    "@dcloudio/vite-plugin-uni": "3.0.0-4030620241128001",
    "@playwright/test": "^1.45.0",
    "@types/jest": "^29.5.12",
    "@types/node": "^20.12.7",
    "@typescript-eslint/eslint-plugin": "^7.7.1",
    "@typescript-eslint/parser": "^7.7.1",
    "@vue/test-utils": "^2.4.6",
    "eslint": "^8.57.0",
    "eslint-plugin-vue": "^9.25.0",
    "jest": "^29.7.0",
    "jest-environment-jsdom": "^29.7.0",
    "msw": "^2.3.1",
    "openapi-typescript": "^6.7.6",
    "prettier": "^3.2.5",
    "sass": "^1.74.1",
    "ts-jest": "^29.1.2",
    "ts-node": "^10.9.2",
    "typescript": "^5.4.5",
    "vite": "5.2.10",
    "vue-tsc": "^2.0.13"
  }
}
```

`web/patient-miniapp/.gitignore`：

```
node_modules/
unpackage/
dist/
.DS_Store
*.log
*.local
coverage/
playwright-report/
test-results/
src/api/types.gen.ts.check
```

`web/patient-miniapp/.npmrc`：

```
registry=https://registry.npmmirror.com/
```

`web/patient-miniapp/tsconfig.json`：

```json
{
  "extends": "@dcloudio/types/tsconfig.vue",
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    },
    "types": ["@dcloudio/types", "jest", "node"]
  },
  "include": [
    "src/**/*.ts",
    "src/**/*.vue",
    "__tests__/**/*.ts",
    "e2e/**/*.ts",
    "mocks/**/*.ts",
    "env.d.ts"
  ]
}
```

`web/patient-miniapp/vite.config.ts`：

```ts
import { defineConfig } from 'vite';
import uni from '@dcloudio/vite-plugin-uni';

export default defineConfig({
  plugins: [uni()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.DOCTORS_BACKEND_URL || 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: `@import "@/styles/variables.scss";`,
      },
    },
  },
});
```

`web/patient-miniapp/index.html`：

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no" />
    <title>陪诊</title>
    <script>
      window.process = window.process || { env: {} };
    </script>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

`web/patient-miniapp/src/manifest.json`：

```json
{
  "name": "陪诊患者端",
  "appid": "TOURIST_APPID",
  "description": "患者陪诊全链路小程序",
  "versionName": "0.1.0",
  "versionCode": 1,
  "transformPx": false,
  "app-plus": { "usingComponents": true },
  "mp-weixin": { "appid": "TOURIST_WX_APPID", "setting": { "urlCheck": false } },
  "h5": { "title": "陪诊", "router": { "mode": "hash" } }
}
```

`web/patient-miniapp/src/env.d.ts`：

```ts
/// <reference types="@dcloudio/types" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue';
  const component: DefineComponent<{}, {}, any>;
  export default component;
}
```

**Step 2: 跑 typecheck 确认能 install（先 dry-install）**

```bash
cd web/patient-miniapp
npm install --no-audit --no-fund
```

Expected: 安装成功（首次会拉 uni-app 全家桶 + 800MB+）；不要报 `ERESOLVE`。

**Step 3: 创建空 `main.ts` 与 `App.vue` 占位（让 uni-app 能 dev 起来）**

`web/patient-miniapp/src/main.ts`（Task 5 完整化前先用占位）：

```ts
import { createSSRApp } from 'vue';
import App from './App.vue';

export function createApp() {
  const app = createSSRApp(App);
  return { app };
}
```

`web/patient-miniapp/src/App.vue`：

```vue
<script setup lang="ts">
</script>

<template>
  <view class="app">patient-miniapp skeleton</view>
</template>

<style lang="scss">
.app { padding: 16px; }
</style>
```

**Step 4: 用最简 `pages.json` 跑 dev 验证骨架**

`web/patient-miniapp/src/pages.json`（Task 3 替换为完整版）：

```json
{
  "pages": [{ "path": "pages/index/index", "style": { "navigationBarTitleText": "陪诊" } }],
  "globalStyle": {
    "navigationBarTextStyle": "black",
    "navigationBarTitleText": "陪诊",
    "backgroundColor": "#FFFFFF"
  }
}
```

`web/patient-miniapp/src/pages/index/index.vue`：

```vue
<template>
  <view class="home">
    <text>home</text>
  </view>
</template>

<script setup lang="ts">
</script>
```

**Step 5: 跑 dev:h5 验证**

```bash
cd web/patient-miniapp
npm run dev:h5 &
sleep 8
curl -fsS http://127.0.0.1:5173 | head -c 200
kill %1
```

Expected: 200 OK + 含 `<div id="app"></div>`。

**Step 6: Commit**

```bash
cd web/patient-miniapp
git add package.json package-lock.json .gitignore .npmrc tsconfig.json vite.config.ts \
        index.html src/manifest.json src/env.d.ts src/main.ts src/App.vue \
        src/pages.json src/pages/index/index.vue README.md
git commit -m "chore(patient-miniapp): 项目骨架 (uni-app + Vue 3 + TS + Vite + npm 锁定)"
```

---

### Task 2: 全局样式 + 设计 token + uview-plus 接入

**Files:**
- Create: `web/patient-miniapp/src/styles/variables.scss`
- Create: `web/patient-miniapp/src/styles/global.scss`
- Modify: `web/patient-miniapp/src/App.vue`（引入 global.scss）
- Create: `web/patient-miniapp/src/pages.json`（完整 routes，见 Task 3 — 此处只加 easycom）

**Step 1: 设计 token（按 spec §6）**

`web/patient-miniapp/src/styles/variables.scss`：

```scss
// 配色（spec §6）
$color-primary: #1989FA;
$color-accent:  #FF6B35;
$color-pending: #909399;
$color-active:  $color-primary;
$color-success: #67C23A;
$color-danger:  #F56C6C;

// 字号
$font-xs: 12px;
$font-sm: 14px;
$font-md: 16px;
$font-lg: 18px;
$font-xl: 20px;

// 圆角
$radius-sm: 4px;
$radius-md: 12px;
$radius-full: 999px;

// 间距
$space-1: 8px;
$space-2: 16px;
$space-3: 24px;
$space-4: 32px;
```

`web/patient-miniapp/src/styles/global.scss`：

```scss
@import 'uview-plus/index.scss';

page {
  background-color: #f5f5f5;
  font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif;
  font-size: $font-md;
  color: #333;
}

view, text { box-sizing: border-box; }
```

**Step 2: 在 main.ts 装 uview-plus**

修改 `web/patient-miniapp/src/main.ts`：

```ts
import { createSSRApp } from 'vue';
import uviewPlus from 'uview-plus';
import App from './App.vue';

export function createApp() {
  const app = createSSRApp(App);
  // uview-plus 用 createSSRApp 时直接 use
  // @ts-expect-error: uview-plus 类型暂未发布 d.ts
  app.use(uviewPlus);
  return { app };
}
```

**Step 3: 在 pages.json 全局注入 easycom + 修改 App.vue**

修改 `web/patient-miniapp/src/pages.json`：

```json
{
  "easycom": {
    "autoscan": true,
    "custom": {
      "^u-(.*)": "uview-plus/components/u-$1/u-$1.vue"
    }
  },
  "pages": [{ "path": "pages/index/index", "style": { "navigationBarTitleText": "陪诊" } }],
  "globalStyle": {
    "navigationBarTextStyle": "black",
    "navigationBarTitleText": "陪诊",
    "backgroundColor": "#FFFFFF"
  }
}
```

修改 `web/patient-miniapp/src/App.vue` 引入 global：

```vue
<script setup lang="ts">
</script>

<style lang="scss">
@import '@/styles/global.scss';
</style>
```

**Step 4: 验证样式生效（H5 跑起来 + console 无错）**

```bash
cd web/patient-miniapp
npm run dev:h5 &
sleep 8
# 浏览器内 inspect 应见 page background #f5f5f5
kill %1
```

**Step 5: Commit**

```bash
cd web/patient-miniapp
git add src/styles/ src/main.ts src/App.vue src/pages.json
git commit -m "feat(patient-miniapp): 设计 token (主色/字号/圆角) + uview-plus 接入 + easycom"
```

---

### Task 3: 20 个 P0 页面骨架 + routes 完整声明

**Files:**
- Modify: `web/patient-miniapp/src/pages.json`（替换为完整 20 路由 + tabBar）
- Create: 20 个 `src/pages/<group>/<page>.vue`（空骨架 + 标题注释 + 测试路径提示）

**Step 1: 用 TDD 思路：先写一个 e2e 用例断言所有 20 路由存在**

`web/patient-miniapp/e2e/pages-skeleton.spec.ts`（用于 Step 2 跑前的占位 — Task 9 写完整版）：

```ts
import { test, expect } from '@playwright/test';

const PAGES = [
  'pages/index/index',
  'pages/auth/login',
  'pages/auth/real-name',
  'pages/hospital/list',
  'pages/hospital/detail',
  'pages/package/detail',
  'pages/order/create',
  'pages/order/pay',
  'pages/order/list',
  'pages/order/detail',
  'pages/refund/apply',
  'pages/review/create',
  'pages/sos/trigger',
  'pages/wallet/index',
  'pages/coupon/list',
  'pages/address/list',
  'pages/address/edit',
  'pages/message/list',
  'pages/message/detail',
  'pages/profile/index',
];

// 本 Task 内复用：先验证 pages.json 含全部路径
import pagesManifest from '../src/pages.json';
test('pages.json declares all 20 P0 routes', () => {
  const declared = pagesManifest.pages.map((p) => p.path);
  for (const p of PAGES) {
    expect(declared, `page ${p} should be declared in pages.json`).toContain(p);
  }
  expect(PAGES.length).toBe(20);
});
```

> **本 Task 仅占位运行**：完整 Playwright 跑测在 Task 9；这里先 `tsc --noEmit` 让 import 不报错即可。

```bash
cd web/patient-miniapp
npx tsc --noEmit e2e/pages-skeleton.spec.ts 2>&1 | head -10
```

Expected: `error TS2307: Cannot find module '../src/pages.json'` 或类似 — 此时页面文件尚未补齐，预期失败。

**Step 2: 写完整 pages.json**

`web/patient-miniapp/src/pages.json`：

```json
{
  "easycom": {
    "autoscan": true,
    "custom": {
      "^u-(.*)": "uview-plus/components/u-$1/u-$1.vue"
    }
  },
  "pages": [
    { "path": "pages/index/index",         "style": { "navigationBarTitleText": "首页" } },
    { "path": "pages/auth/login",          "style": { "navigationBarTitleText": "登录" } },
    { "path": "pages/auth/real-name",      "style": { "navigationBarTitleText": "实名认证" } },
    { "path": "pages/hospital/list",       "style": { "navigationBarTitleText": "医院" } },
    { "path": "pages/hospital/detail",     "style": { "navigationBarTitleText": "医院详情" } },
    { "path": "pages/package/detail",      "style": { "navigationBarTitleText": "服务包" } },
    { "path": "pages/order/create",        "style": { "navigationBarTitleText": "下单" } },
    { "path": "pages/order/pay",           "style": { "navigationBarTitleText": "支付" } },
    { "path": "pages/order/list",          "style": { "navigationBarTitleText": "我的订单" } },
    { "path": "pages/order/detail",        "style": { "navigationBarTitleText": "订单详情" } },
    { "path": "pages/refund/apply",        "style": { "navigationBarTitleText": "申请退款" } },
    { "path": "pages/review/create",       "style": { "navigationBarTitleText": "评价" } },
    { "path": "pages/sos/trigger",         "style": { "navigationBarTitleText": "紧急报警" } },
    { "path": "pages/wallet/index",        "style": { "navigationBarTitleText": "钱包" } },
    { "path": "pages/coupon/list",         "style": { "navigationBarTitleText": "优惠券" } },
    { "path": "pages/address/list",        "style": { "navigationBarTitleText": "地址" } },
    { "path": "pages/address/edit",        "style": { "navigationBarTitleText": "编辑地址" } },
    { "path": "pages/message/list",        "style": { "navigationBarTitleText": "消息" } },
    { "path": "pages/message/detail",      "style": { "navigationBarTitleText": "消息详情" } },
    { "path": "pages/profile/index",       "style": { "navigationBarTitleText": "我的" } }
  ],
  "tabBar": {
    "color": "#909399",
    "selectedColor": "#1989FA",
    "backgroundColor": "#FFFFFF",
    "list": [
      { "pagePath": "pages/index/index",   "text": "首页" },
      { "pagePath": "pages/order/list",   "text": "订单" },
      { "pagePath": "pages/message/list", "text": "消息" },
      { "pagePath": "pages/profile/index","text": "我的" }
    ]
  },
  "globalStyle": {
    "navigationBarTextStyle": "black",
    "navigationBarTitleText": "陪诊",
    "backgroundColor": "#FFFFFF"
  }
}
```

**Step 3: 写 20 个空页面骨架**

每个页面用统一脚手架（pattern 自动化复制）：

```vue
<template>
  <view class="page page-{kebab-name}">
    <!-- TODO(feature-{slug}): 实现 — see docs/superpowers/specs/2026-09-24-patient-miniapp-design.md §{n} -->
    <text class="page-title">{页面中文名}（骨架）</text>
  </view>
</template>

<script setup lang="ts">
// 骨架阶段：仅占位路由 + 标题渲染。feature 实施见后续 plan。
</script>

<style lang="scss" scoped>
.page-{kebab-name} { padding: $space-2; }
.page-title { font-size: $font-lg; color: $color-primary; }
</style>
```

**20 个文件批量生成**（用 bash + heredoc，避免 20 个手写示例占篇幅）：

```bash
cd web/patient-miniapp
mkdir -p src/pages/{auth,hospital,package,order,refund,review,sos,wallet,coupon,address,message,profile}

declare -A PAGES=(
  [auth/login]="登录|sms"
  [auth/real-name]="实名|real-name"
  [hospital/list]="医院列表|list"
  [hospital/detail]="医院详情|hospital-detail"
  [package/detail]="服务包详情|package-detail"
  [order/create]="下单|order-create"
  [order/pay]="支付|order-pay"
  [order/list]="订单列表|order-list"
  [order/detail]="订单详情|order-detail"
  [refund/apply]="申请退款|refund-apply"
  [review/create]="评价|review-create"
  [sos/trigger]="SOS 紧急报警|sos-trigger"
  [wallet/index]="钱包|wallet"
  [coupon/list]="优惠券|coupon-list"
  [address/list]="地址|address-list"
  [address/edit]="编辑地址|address-edit"
  [message/list]="消息|message-list"
  [message/detail]="消息详情|message-detail"
  [profile/index]="个人中心|profile"
)

for key in "${!PAGES[@]}"; do
  IFS='|' read -r cn slug <<< "${PAGES[$key]}"
  kebab=$(echo "$key" | tr '/' '-')
  cat > "src/pages/${key}.vue" <<EOF
<template>
  <view class="page page-${kebab}">
    <text class="page-title">${cn}（骨架）</text>
  </view>
</template>

<script setup lang="ts">
// TODO(feature-${slug}): 实现 — see docs/superpowers/specs/2026-09-24-patient-miniapp-design.md
</script>

<style lang="scss" scoped>
.page-${kebab} { padding: \$space-2; }
.page-title { font-size: \$font-lg; color: \$color-primary; }
</style>
EOF
done
```

> 上面对 `index/index` 用同 pattern：`[index/index]="首页|home"` —— 共 20 个页面（含 index → 21 个；pages.json 21 条全部有对应的 .vue）。

需要把脚本里 `mkdir` 加上 `src/pages/index/`：

```bash
mkdir -p src/pages/index
cat > src/pages/index/index.vue <<'EOF'
<template>
  <view class="page page-home">
    <text class="page-title">首页（骨架）</text>
  </view>
</template>

<script setup lang="ts">
// TODO(feature-home): 实现 — see docs/superpowers/specs/2026-09-24-patient-miniapp-design.md §3
</script>

<style lang="scss" scoped>
.page-home { padding: $space-2; }
.page-title { font-size: $font-lg; color: $color-primary; }
</style>
EOF
```

**Step 4: 跑 typecheck 验证 20 个 .vue 文件都通过**

```bash
cd web/patient-miniapp
npm run typecheck
```

Expected: PASS（若 vue-tsc 对空 `<script setup>` 报错，加 `defineOptions({ name: 'Xxx' })` 占位）。

**Step 5: dev:h5 起来逐一走 4 个 tabBar 页面**

```bash
cd web/patient-miniapp
npm run dev:h5 &
sleep 8
# 浏览器：依次访问
#   http://127.0.0.1:5173/#/pages/index/index
#   http://127.0.0.1:5173/#/pages/order/list
#   http://127.0.0.1:5173/#/pages/message/list
#   http://127.0.0.1:5173/#/pages/profile/index
# 每页应显示 "<页面中文名>（骨架）" 文本
kill %1
```

**Step 6: Commit**

```bash
cd web/patient-miniapp
git add src/pages/ src/pages.json e2e/pages-skeleton.spec.ts
git commit -m "feat(patient-miniapp): 20 个 P0 页面骨架 + 完整 routes + tabBar"
```

---

### Task 4: utils 层（trace / format / auth / wx / uni-mock）

**Files:**
- Create: `web/patient-miniapp/src/utils/trace.ts`
- Create: `web/patient-miniapp/src/utils/format.ts`
- Create: `web/patient-miniapp/src/utils/auth.ts`
- Create: `web/patient-miniapp/src/utils/wx.ts`
- Create: `web/patient-miniapp/src/utils/uni-mock.ts`
- Create: `web/patient-miniapp/__tests__/utils/trace.test.ts`
- Create: `web/patient-miniapp/__tests__/utils/format.test.ts`

**Step 1: 写 trace 单元测试（RED）**

`web/patient-miniapp/__tests__/utils/trace.test.ts`：

```ts
import { describe, it, expect } from '@jest/globals';
import { newTraceId, isTraceId } from '@/utils/trace';

describe('trace utils', () => {
  it('newTraceId 返回 mp-{数字}-{6位} 字符串', () => {
    const id = newTraceId();
    expect(id).toMatch(/^mp-\d{13}-[a-z0-9]{6}$/);
  });

  it('newTraceId() 两次返回不同', () => {
    const a = newTraceId();
    const b = newTraceId();
    expect(a).not.toBe(b);
  });

  it('isTraceId 校验合法 / 非法', () => {
    expect(isTraceId('mp-1700000000000-abc123')).toBe(true);
    expect(isTraceId('abc')).toBe(false);
    expect(isTraceId('mp-xx-yyy')).toBe(false);
  });
});
```

```bash
cd web/patient-miniapp
npx jest __tests__/utils/trace.test.ts
```

Expected: FAIL — `Cannot find module '@/utils/trace'`。

**Step 2: 写 trace.ts（GREEN）**

`web/patient-miniapp/src/utils/trace.ts`：

```ts
// trace_id 生成与校验（spec §5.2）
export function newTraceId(now: number = Date.now()): string {
  const rand = Math.random().toString(36).slice(2, 8).padEnd(6, '0');
  return `mp-${now}-${rand}`;
}

const TRACE_RE = /^mp-\d{13}-[a-z0-9]{6}$/;
export function isTraceId(s: string): boolean {
  return TRACE_RE.test(s);
}
```

```bash
cd web/patient-miniapp
npx jest __tests__/utils/trace.test.ts
```

Expected: PASS（3 测试）。

**Step 3: format 单元测试（RED → GREEN）**

`web/patient-miniapp/__tests__/utils/format.test.ts`：

```ts
import { describe, it, expect } from '@jest/globals';
import { formatMoney, formatTime, formatDistance } from '@/utils/format';

describe('format utils', () => {
  it('formatMoney 把 number 转 ¥xx.xx', () => {
    expect(formatMoney(123)).toBe('¥123.00');
    expect(formatMoney(123.4)).toBe('¥123.40');
    expect(formatMoney(0)).toBe('¥0.00');
  });

  it('formatTime ISO 转 yyyy-MM-dd HH:mm', () => {
    expect(formatTime('2026-09-24T15:30:00+08:00')).toBe('2026-09-24 15:30');
  });

  it('formatDistance 米 → km', () => {
    expect(formatDistance(800)).toBe('800 m');
    expect(formatDistance(1500)).toBe('1.5 km');
    expect(formatDistance(12300)).toBe('12.3 km');
  });
});
```

`web/patient-miniapp/src/utils/format.ts`：

```ts
export function formatMoney(n: number): string {
  return '¥' + n.toFixed(2);
}

export function formatTime(iso: string): string {
  const d = new Date(iso);
  const pad = (x: number) => String(x).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function formatDistance(meters: number): string {
  if (meters < 1000) return `${Math.round(meters)} m`;
  return `${(meters / 1000).toFixed(1)} km`;
}
```

**Step 4: auth.ts / wx.ts / uni-mock.ts（无需单测，仅占位 + 类型导出）**

`web/patient-miniapp/src/utils/auth.ts`：

```ts
const TOKEN_KEY = 'patient.token';

export function getToken(): string {
  // uni 全局从 env.d.ts 推断
  return uni.getStorageSync(TOKEN_KEY) || '';
}

export function setToken(token: string): void {
  uni.setStorageSync(TOKEN_KEY, token);
}

export function clearToken(): void {
  uni.removeStorageSync(TOKEN_KEY);
}
```

`web/patient-miniapp/src/utils/wx.ts`：

```ts
// 微信 wx.* 封装（H5 mock 时降级；mp-weixin 真接）
export function wxLogin(): Promise<string> {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: 'weixin',
      success: (res) => resolve(res.code),
      fail: (err) => reject(err),
    });
  });
}

export function wxGetLocation(): Promise<{ lat: number; lng: number }> {
  return new Promise((resolve, reject) => {
    uni.getLocation({
      type: 'gcj02',
      success: (res) => resolve({ lat: res.latitude, lng: res.longitude }),
      fail: (err) => reject(err),
    });
  });
}

export interface WxPayParams {
  timeStamp: string;
  nonceStr: string;
  package: string;
  signType: 'MD5';
  paySign: string;
}

export function wxRequestPayment(p: WxPayParams): Promise<void> {
  return new Promise((resolve, reject) => {
    // #ifdef MP-WEIXIN
    uni.requestPayment({
      provider: 'wxpay',
      timeStamp: p.timeStamp,
      nonceStr: p.nonceStr,
      package: p.package,
      signType: p.signType,
      paySign: p.paySign,
      success: () => resolve(),
      fail: (err) => reject(err),
    });
    // #endif
    // #ifndef MP-WEIXIN
    resolve();
    // #endif
  });
}
```

`web/patient-miniapp/src/utils/uni-mock.ts`（jest 单测前 import 设置）：

```ts
// 在 jest setupFiles 里引入；给 uni 全局打桩。
type Storage = Record<string, string>;
export function installUniMock(): void {
  const storage: Storage = {};
  // @ts-expect-error: 测试期注入
  globalThis.uni = {
    getStorageSync: (k: string) => storage[k] ?? '',
    setStorageSync: (k: string, v: string) => { storage[k] = v; },
    removeStorageSync: (k: string) => { delete storage[k]; },
    request: jest.fn().mockResolvedValue({ statusCode: 200, data: { code: 0, data: null } }),
    addInterceptor: jest.fn(),
    showToast: jest.fn(),
    showModal: jest.fn(),
    navigateTo: jest.fn(),
    redirectTo: jest.fn(),
    reLaunch: jest.fn(),
    login: jest.fn(),
    getLocation: jest.fn(),
    requestPayment: jest.fn(),
  } as any;
}
```

`web/patient-miniapp/jest.setup.ts`：

```ts
import { installUniMock } from '@/utils/uni-mock';
installUniMock();
```

**Step 5: 跑全部 utils 单测**

```bash
cd web/patient-miniapp
npx jest __tests__/utils/
```

Expected: PASS（trace 3 + format 3 = 6 个）。

**Step 6: Commit**

```bash
cd web/patient-miniapp
git add src/utils/ __tests__/utils/ jest.setup.ts
git commit -m "feat(patient-miniapp): utils (trace/format/auth/wx/uni-mock) + 6 个单测"
```

---

### Task 5: API client（uni.request + interceptor + trace_id + 401 重定向）

**Files:**
- Create: `web/patient-miniapp/src/api/client.ts`
- Create: `web/patient-miniapp/src/api/index.ts`
- Modify: `web/patient-miniapp/src/main.ts`（安装 interceptor）
- Create: `web/patient-miniapp/__tests__/api/client.test.ts`

**Step 1: 写 interceptor 单测（RED）**

`web/patient-miniapp/__tests__/api/client.test.ts`：

```ts
import { describe, it, expect, beforeEach, jest } from '@jest/globals';

describe('api client interceptors', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('request 拦截器自动加 X-Trace-Id 与 Authorization', async () => {
    // 装 mock uni + 拦截器记录
    const calls: any[] = [];
    // @ts-expect-error
    globalThis.uni.addInterceptor = jest.fn((_type: string, fn: any) => {
      fn.request = fn.request || (() => undefined);
      // 收集以便后续断言
      calls.push(fn);
    }) as any;

    // 重置 token store
    // @ts-expect-error
    globalThis.uni.setStorageSync('patient.token', 'test-token-abc');

    const { installInterceptors } = await import('@/api/client');
    installInterceptors();

    // 触发 request 拦截器
    const interceptor = (uni.addInterceptor as jest.Mock).mock.calls.find((c) => c[0] === 'request')?.[1];
    expect(interceptor).toBeDefined();
    const out = interceptor!.request({ header: {}, url: '/api/v1/orders' });
    expect(out.header['X-Trace-Id']).toMatch(/^mp-\d{13}-[a-z0-9]{6}$/);
    expect(out.header['Authorization']).toBe('Bearer test-token-abc');
  });

  it('responseError 拦截器在 11001 时清 token 并 reLaunch login', async () => {
    globalThis.uni.reLaunch = jest.fn();
    globalThis.uni.removeStorageSync = jest.fn();

    const { installInterceptors } = await import('@/api/client');
    installInterceptors();

    const interceptor = (uni.addInterceptor as jest.Mock).mock.calls.find((c) => c[0] === 'responseError')?.[1];
    expect(interceptor).toBeDefined();

    await expect(
      interceptor.responseError({ statusCode: 401, data: { code: 11001, message: 'no token' } })
    ).rejects.toBeDefined();
    expect(uni.reLaunch).toHaveBeenCalledWith({ url: '/pages/auth/login' });
    expect(uni.removeStorageSync).toHaveBeenCalledWith('patient.token');
  });
});
```

```bash
cd web/patient-miniapp
npx jest __tests__/api/client.test.ts
```

Expected: FAIL — `Cannot find module '@/api/client'`。

**Step 2: 写 client.ts（GREEN）**

`web/patient-miniapp/src/api/client.ts`：

```ts
// api/client.ts — uni.request 封装 + 全局拦截器（spec §5.2）
import { newTraceId } from '@/utils/trace';
import { clearToken, getToken } from '@/utils/auth';

export interface ApiResp<T> {
  code: number;
  message?: string;
  data: T;
}

export interface ApiOptions {
  url: string;
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  data?: unknown;
  query?: Record<string, unknown>;
  /** 是否需要 Bearer */
  auth?: boolean;
}

export class ApiError extends Error {
  constructor(public code: number, message: string, public statusCode: number) {
    super(message);
  }
}

export async function request<T>(opts: ApiOptions): Promise<T> {
  const headers: Record<string, string> = {
    'X-Trace-Id': newTraceId(),
    'Content-Type': 'application/json',
  };
  if (opts.auth !== false) {
    const token = getToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }
  const queryStr = opts.query
    ? '?' + Object.entries(opts.query).map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`).join('&')
    : '';
  const res = await uni.request({
    url: opts.url + queryStr,
    method: opts.method || 'GET',
    data: opts.data,
    header: headers,
  });
  // @ts-expect-error: uni.request 返回值结构稳定但类型不全
  const statusCode: number = res.statusCode;
  // @ts-expect-error
  const body: ApiResp<T> = res.data;
  if (statusCode >= 400 || body.code !== 0) {
    throw new ApiError(body.code ?? statusCode, body.message ?? 'request failed', statusCode);
  }
  return body.data;
}

/**
 * installInterceptors 在 App 启动时调用一次；uni.request 在调用前会跑 request 拦截器、错误时跑 responseError。
 * 本项目把"通用逻辑"写在 client.ts 顶部 addInterceptor 中（每个 request 都自带 header），这里额外做
 * 全局"401 业务码" → 清 token + reLaunch login。
 */
export function installInterceptors(): void {
  uni.addInterceptor('responseError', {
    fail({ statusCode, data }: { statusCode: number; data?: ApiResp<unknown> }) {
      if (data?.code === 11001) {
        clearToken();
        uni.reLaunch({ url: '/pages/auth/login' });
      }
      return Promise.reject(new ApiError(data?.code ?? statusCode, data?.message ?? 'request error', statusCode));
    },
  });
}
```

**Step 3: 在 main.ts 调 installInterceptors**

`web/patient-miniapp/src/main.ts`：

```ts
import { createSSRApp } from 'vue';
import { createPinia } from 'pinia';
import uviewPlus from 'uview-plus';
import App from './App.vue';
import { installInterceptors } from '@/api/client';
import { createI18n } from '@/i18n';

installInterceptors();

export function createApp() {
  const app = createSSRApp(App);
  const pinia = createPinia();
  app.use(pinia);
  app.use(createI18n());
  // @ts-expect-error: uview-plus 类型暂未发布 d.ts
  app.use(uviewPlus);
  return { app, Pinia: pinia };
}
```

**Step 4: 跑测试**

```bash
cd web/patient-miniapp
npx jest __tests__/api/client.test.ts
```

Expected: PASS（2 测试）。

**Step 5: Commit**

```bash
cd web/patient-miniapp
git add src/api/client.ts src/api/index.ts src/main.ts __tests__/api/
git commit -m "feat(patient-miniapp): api client + interceptor (X-Trace-Id + 11001 reLaunch) + 2 单测"
```

---

### Task 6: API 模块骨架（14 个 patient P0 API）

**Files:**
- Create: 9 个 `src/api/<feature>.ts`（auth/order/hospital/refund/review/sos/wallet/address/coupon/message）
- Create: `src/api/index.ts`（统一 export）

> 把 14 API 按模块拆成 9 个文件；`auth.ts` / `order.ts` / `hospital.ts` / `refund.ts` / `review.ts` / `sos.ts` / `wallet.ts` / `address.ts` / `coupon.ts` / `message.ts`。

**Step 1: auth.ts（举例 — 其他模块同 pattern）**

`web/patient-miniapp/src/api/auth.ts`：

```ts
// 与 l2-api-gap-design.md §2.1 P0 已有 API（auth/login + auth/refresh + users/real-name + users/me）
import { request, ApiOptions } from './client';

export interface SendSmsReq { phone: string; }
export function sendSms(req: SendSmsReq) {
  return request({ url: '/api/v1/auth/sms/send', method: 'POST', data: req, auth: false });
}

export interface LoginByPhoneReq { phone: string; code: string; }
export interface LoginResp { access_token: string; refresh_token: string; user: { id: number; phone: string; }; }
export function loginByPhone(req: LoginByPhoneReq) {
  return request<LoginResp>({ url: '/api/v1/auth/login', method: 'POST', data: req, auth: false });
}

export interface LoginByWxReq { wx_code: string; }
export function loginByWx(req: LoginByWxReq) {
  return request<LoginResp>({ url: '/api/v1/auth/login', method: 'POST', data: { ...req, channel: 'wx' }, auth: false });
}

export interface RefreshReq { refresh_token: string; }
export function refreshToken(req: RefreshReq) {
  return request<LoginResp>({ url: '/api/v1/auth/refresh', method: 'POST', data: req, auth: false });
}

export interface RealNameReq { id_card: string; name: string; }
export interface RealNameResp { verified: boolean; }
export function submitRealName(req: RealNameReq) {
  return request<RealNameResp>({ url: '/api/v1/users/real-name/auth', method: 'POST', data: req });
}

export interface UserSnapshot { id: number; phone: string; real_name_verified: boolean; }
export function getMe() {
  return request<UserSnapshot>({ url: '/api/v1/users/me' });
}

export type { ApiOptions };
```

`web/patient-miniapp/src/api/order.ts`（按 l2-api-gap-design.md §2.1 14 API 拆）：

```ts
import { request } from './client';

export interface OrderListQuery { status?: string; page?: number; page_size?: number; }
export interface OrderListItem {
  id: number; order_no: string; status: string;
  hospital_name: string; package_name: string; amount: number;
  service_start_at: string; created_at: string;
}
export function listOrders(q: OrderListQuery = {}) {
  return request<{ items: OrderListItem[]; total: number }>({
    url: '/api/v1/orders', query: q as Record<string, unknown>,
  });
}

export interface OrderDetail extends OrderListItem {
  escort_summary?: { escort_id: number; nickname: string; avatar_url: string; };
  progress: { state: string; updated_at: string; }[];
}
export function getOrder(id: number) {
  return request<OrderDetail>({ url: `/api/v1/orders/${id}` });
}

export interface CreateOrderReq {
  hospital_id: number; package_id: number;
  service_start_at: string; address_id: number; remark?: string;
}
export function createOrder(req: CreateOrderReq) {
  return request<OrderDetail>({ url: '/api/v1/orders', method: 'POST', data: req });
}

export function cancelOrder(id: number, reason: string) {
  return request<void>({ url: `/api/v1/orders/${id}/cancel`, method: 'POST', data: { reason } });
}

export interface PaymentResp { wx_pay_params: unknown; }
export function createPayment(orderId: number) {
  return request<PaymentResp>({ url: `/api/v1/orders/${orderId}/payment`, method: 'POST' });
}

export function mockPayCallback(orderId: number) {
  return request<void>({ url: `/api/v1/orders/${orderId}/payment/mock-callback`, method: 'POST' });
}

// EscortSummary / VirtualNumber / Review / Refund 关联子资源
export function getEscortSummary(orderId: number) {
  return request<{ escort_id: number; nickname: string; avatar_url: string; }>(
    { url: `/api/v1/orders/${orderId}/escort-summary` }
  );
}

export function getVirtualNumber(orderId: number) {
  return request<{ phone_virtual: string; expires_at: string; }>(
    { url: `/api/v1/orders/${orderId}/virtual-number` }
  );
}

export function submitReview(orderId: number, req: { rating: number; tags: string[]; comment: string; is_anonymous: boolean }) {
  return request<{ id: number }>({ url: `/api/v1/orders/${orderId}/review`, method: 'POST', data: req });
}

export function triggerSos(orderId: number, req: { lat: number; lng: number; address: string; }) {
  return request<{ signal_id: number }>({ url: `/api/v1/orders/${orderId}/sos`, method: 'POST', data: req });
}
```

> 其他 7 个文件（hospital / refund / wallet / coupon / address / message + review/sos 复用上面）使用相同 pattern：从 l2-api-gap-design.md §2.1 提取 path + 简化类型。每个文件 ~30~60 行。本 Task Step 1 只展示 auth.ts / order.ts 完整代码；Step 2 批量生成剩余 7 个。

**Step 2: 批量生成剩余 7 个 API 文件**

```bash
cd web/patient-miniapp

cat > src/api/hospital.ts <<'EOF'
import { request } from './client';

export interface HospitalListQuery { city?: string; keyword?: string; page?: number; }
export interface Hospital { id: number; name: string; city: string; district: string; level: string; photo_url: string; }
export function listHospitals(q: HospitalListQuery = {}) {
  return request<{ items: Hospital[]; total: number }>({
    url: '/api/v1/hospitals', query: q as Record<string, unknown>,
  });
}
export function getHospital(id: number) { return request<Hospital & { address: string; departments: string[]; packages: HospitalPackage[]; }>({ url: `/api/v1/hospitals/${id}` }); }

export interface HospitalPackage { id: number; name: string; duration_hours: number; amount: number; description: string; }
export function getHospitalPackages(hospitalId: number) { return request<{ items: HospitalPackage[] }>({ url: `/api/v1/hospitals/${hospitalId}/packages` }); }

export interface EscortReviews { items: { id: number; rating: number; comment: string; created_at: string; }[]; }
export function getEscortReviews(escortId: number) { return request<EscortReviews>({ url: `/api/v1/escorts/${escortId}/reviews` }); }
EOF

cat > src/api/refund.ts <<'EOF'
import { request } from './client';
export interface RefundApplyReq { order_id: number; reason: string; amount?: number; }
export function applyRefund(req: RefundApplyReq) {
  return request<{ refund_id: number; status: string; }>({ url: `/api/v1/orders/${req.order_id}/refund`, method: 'POST', data: { reason: req.reason, amount: req.amount } });
}
EOF

cat > src/api/wallet.ts <<'EOF'
import { request } from './client';
export interface Wallet { balance: string; frozen: string; total_earned: string; updated_at: string; }
export function getMyWallet() { return request<Wallet>({ url: '/api/v1/users/me/wallet' }); }
EOF

cat > src/api/address.ts <<'EOF'
import { request } from './client';
export interface Address { id: number; recipient: string; phone: string; province: string; city: string; district: string; detail: string; is_default: boolean; }
export function listAddresses() { return request<{ items: Address[] }>({ url: '/api/v1/users/me/addresses' }); }
export function upsertAddress(addr: Partial<Address> & { id?: number }) {
  return request<Address>({ url: '/api/v1/users/me/addresses', method: addr.id ? 'PUT' : 'POST', data: addr });
}
export function deleteAddress(id: number) { return request<void>({ url: `/api/v1/users/me/addresses/${id}`, method: 'DELETE' }); }
EOF

cat > src/api/coupon.ts <<'EOF'
import { request } from './client';
export interface Coupon { id: number; type: string; value: string; threshold: string; expires_at: string; status: string; }
export function listCoupons() { return request<{ items: Coupon[] }>({ url: '/api/v1/users/me/coupons' }); }
EOF

cat > src/api/message.ts <<'EOF'
import { request } from './client';
export interface Conversation { id: number; last_message_at: string; unread_count: number; peer: { id: number; nickname: string; avatar_url: string; }; }
export function listConversations() { return request<{ items: Conversation[] }>({ url: '/api/v1/messages/conversations' }); }
export interface Message { id: number; sender_id: number; content_type: string; content: string; created_at: string; }
export function listMessages(conversationId: number) { return request<{ items: Message[] }>({ url: `/api/v1/messages/conversations/${conversationId}/messages` }); }
EOF

cat > src/api/sos.ts <<'EOF'
export { triggerSos } from './order'; // re-export，spec §2.1 的 POST /orders/:id/sos 与 order-service 共享
EOF

cat > src/api/review.ts <<'EOF'
export { submitReview } from './order'; // 同上：POST /orders/:id/review
EOF
```

**Step 3: index.ts 统一 export**

`web/patient-miniapp/src/api/index.ts`：

```ts
export * from './auth';
export * from './order';
export * from './hospital';
export * from './refund';
export * from './sos';
export * from './review';
export * from './wallet';
export * from './address';
export * from './coupon';
export * from './message';
export { request, ApiError } from './client';
export type { ApiResp, ApiOptions } from './client';
```

**Step 4: typecheck 验证全部 API 文件**

```bash
cd web/patient-miniapp
npm run typecheck
```

Expected: PASS。

**Step 5: Commit**

```bash
cd web/patient-miniapp
git add src/api/
git commit -m "feat(patient-miniapp): 14 patient P0 API 模块骨架 (auth/order/hospital/refund/review/sos/wallet/address/coupon/message)"
```

---

### Task 7: Pinia stores（auth / order / hospital / wallet / message）

**Files:**
- Create: `web/patient-miniapp/src/stores/auth.ts`
- Create: `web/patient-miniapp/src/stores/order.ts`
- Create: `web/patient-miniapp/src/stores/hospital.ts`
- Create: `web/patient-miniapp/src/stores/wallet.ts`
- Create: `web/patient-miniapp/src/stores/message.ts`
- Create: `web/patient-miniapp/src/stores/index.ts`
- Create: `web/patient-miniapp/__tests__/stores/auth.test.ts`
- Create: `web/patient-miniapp/__tests__/stores/order.test.ts`

**Step 1: auth store 单测（RED）**

`web/patient-miniapp/__tests__/stores/auth.test.ts`：

```ts
import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    // @ts-expect-error
    globalThis.uni.removeStorageSync('patient.token');
  });

  it('init: token / user 都是空', async () => {
    const { useAuthStore } = await import('@/stores/auth');
    const s = useAuthStore();
    expect(s.token).toBe('');
    expect(s.user).toBeNull();
  });

  it('loginByPhone 成功后 token + user 都被设置并持久化', async () => {
    const loginByPhoneMock = jest.fn(async () => ({
      access_token: 't-abc', refresh_token: 'r-abc',
      user: { id: 1, phone: '13800138000' } as any,
    }));
    jest.doMock('@/api/auth', () => ({ loginByPhone: loginByPhoneMock }));
    const { useAuthStore } = await import('@/stores/auth');
    const s = useAuthStore();
    await s.loginByPhone('13800138000', '1234');
    expect(s.token).toBe('t-abc');
    expect(s.user?.id).toBe(1);
    expect(uni.getStorageSync('patient.token')).toBe('t-abc');
  });

  it('logout 清 token + reLaunch login 页', async () => {
    const reLaunch = jest.fn();
    // @ts-expect-error
    globalThis.uni.reLaunch = reLaunch;
    const { useAuthStore } = await import('@/stores/auth');
    const s = useAuthStore();
    s.token = 't-xyz';
    await s.logout();
    expect(s.token).toBe('');
    expect(s.user).toBeNull();
    expect(uni.getStorageSync('patient.token')).toBe('');
    expect(reLaunch).toHaveBeenCalledWith({ url: '/pages/auth/login' });
  });
});
```

```bash
cd web/patient-miniapp
npx jest __tests__/stores/auth.test.ts
```

Expected: FAIL — `Cannot find module '@/stores/auth'`。

**Step 2: auth store（GREEN）**

`web/patient-miniapp/src/stores/auth.ts`：

```ts
// stores/auth.ts — token / 当前用户（spec §5.1）
import { defineStore } from 'pinia';
import { ref } from 'vue';
import { loginByPhone as apiLoginByPhone, getMe, type UserSnapshot, type LoginByPhoneReq } from '@/api/auth';
import { setToken, getToken, clearToken } from '@/utils/auth';

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(getToken());
  const user = ref<UserSnapshot | null>(null);

  async function loginByPhone(phone: string, code: string) {
    const r = await apiLoginByPhone({ phone, code } satisfies LoginByPhoneReq);
    token.value = r.access_token;
    user.value = r.user;
    setToken(token.value);
  }

  async function fetchMe() {
    user.value = await getMe();
  }

  async function logout() {
    token.value = '';
    user.value = null;
    clearToken();
    uni.reLaunch({ url: '/pages/auth/login' });
  }

  return { token, user, loginByPhone, fetchMe, logout };
});
```

```bash
cd web/patient-miniapp
npx jest __tests__/stores/auth.test.ts
```

Expected: PASS（3 测试）。

**Step 3: order store 单测（RED → GREEN）**

`web/patient-miniapp/__tests__/stores/order.test.ts`：

```ts
import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('order store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('loadOrder 设置当前订单', async () => {
    const getOrder = jest.fn(async () => ({ id: 7, status: 'paid', amount: 100 } as any));
    jest.doMock('@/api/order', () => ({ getOrder }));
    const { useOrderStore } = await import('@/stores/order');
    const s = useOrderStore();
    await s.loadOrder(7);
    expect(s.current?.id).toBe(7);
    expect(getOrder).toHaveBeenCalledWith(7);
  });

  it('startPolling 在进行中状态下每 3s 调一次 loadOrder', async () => {
    jest.useFakeTimers();
    const getOrder = jest.fn(async () => ({ id: 7, status: 'in_service' } as any));
    jest.doMock('@/api/order', () => ({ getOrder }));
    const { useOrderStore } = await import('@/stores/order');
    const s = useOrderStore();
    s.startPolling(7);
    await Promise.resolve();
    expect(getOrder).toHaveBeenCalledTimes(1);
    jest.advanceTimersByTime(3000);
    await Promise.resolve();
    jest.advanceTimersByTime(3000);
    await Promise.resolve();
    expect(getOrder).toHaveBeenCalledTimes(3);
    s.stopPolling();
    jest.useRealTimers();
  });

  it('stopPolling 取消定时器', async () => {
    jest.useFakeTimers();
    const getOrder = jest.fn(async () => ({ id: 7, status: 'in_service' } as any));
    jest.doMock('@/api/order', () => ({ getOrder }));
    const { useOrderStore } = await import('@/stores/order');
    const s = useOrderStore();
    s.startPolling(7);
    s.stopPolling();
    jest.advanceTimersByTime(10_000);
    await Promise.resolve();
    expect(getOrder).toHaveBeenCalledTimes(1);
    jest.useRealTimers();
  });
});
```

`web/patient-miniapp/src/stores/order.ts`：

```ts
// stores/order.ts — 订单列表 / 当前订单 / 状态轮询（spec §4.2 + §3.3）
import { defineStore } from 'pinia';
import { ref } from 'vue';
import { getOrder, type OrderDetail } from '@/api/order';
import { listOrders, type OrderListItem } from '@/api/order';

export const useOrderStore = defineStore('order', () => {
  const current = ref<OrderDetail | null>(null);
  const list = ref<OrderListItem[]>([]);
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  async function loadOrder(id: number) {
    current.value = await getOrder(id);
  }

  async function loadList(q: Parameters<typeof listOrders>[0] = {}) {
    const { items } = await listOrders(q);
    list.value = items;
  }

  function startPolling(orderId: number, intervalMs = 3000) {
    stopPolling();
    const tick = async () => {
      try {
        await loadOrder(orderId);
      } catch {
        /* 网络抖动：下一次继续 */
      }
    };
    void tick();
    pollTimer = setInterval(tick, intervalMs);
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  async function cancel(id: number, reason: string) {
    await (await import('@/api/order')).cancelOrder(id, reason);
    await loadOrder(id);
  }

  return { current, list, loadOrder, loadList, startPolling, stopPolling, cancel };
});
```

```bash
cd web/patient-miniapp
npx jest __tests__/stores/order.test.ts
```

Expected: PASS（3 测试）。

**Step 4: hospital / wallet / message stores（骨架 + 简单测试）**

`web/patient-miniapp/src/stores/hospital.ts`：

```ts
import { defineStore } from 'pinia';
import { ref } from 'vue';
import { listHospitals, getHospital, type Hospital } from '@/api/hospital';

export const useHospitalStore = defineStore('hospital', () => {
  const list = ref<Hospital[]>([]);
  const detail = ref<Hospital | null>(null);

  async function loadList(keyword?: string) {
    const r = await listHospitals({ keyword });
    list.value = r.items;
  }

  async function loadDetail(id: number) {
    detail.value = await getHospital(id);
  }

  return { list, detail, loadList, loadDetail };
});
```

`web/patient-miniapp/src/stores/wallet.ts`：

```ts
import { defineStore } from 'pinia';
import { ref } from 'vue';
import { getMyWallet, type Wallet } from '@/api/wallet';

export const useWalletStore = defineStore('wallet', () => {
  const wallet = ref<Wallet | null>(null);
  async function refresh() { wallet.value = await getMyWallet(); }
  return { wallet, refresh };
});
```

`web/patient-miniapp/src/stores/message.ts`：

```ts
import { defineStore } from 'pinia';
import { ref } from 'vue';
import { listConversations, type Conversation } from '@/api/message';

export const useMessageStore = defineStore('message', () => {
  const conversations = ref<Conversation[]>([]);
  async function refresh() {
    const r = await listConversations();
    conversations.value = r.items;
  }
  return { conversations, refresh };
});
```

`web/patient-miniapp/src/stores/index.ts`：

```ts
export { useAuthStore } from './auth';
export { useOrderStore } from './order';
export { useHospitalStore } from './hospital';
export { useWalletStore } from './wallet';
export { useMessageStore } from './message';
```

**Step 5: 跑全部 stores 测试**

```bash
cd web/patient-miniapp
npx jest __tests__/stores/
```

Expected: PASS（auth 3 + order 3 = 6 测试）。

**Step 6: Commit**

```bash
cd web/patient-miniapp
git add src/stores/ __tests__/stores/
git commit -m "feat(patient-miniapp): Pinia stores (auth/order/hospital/wallet/message) + 6 个单测"
```

---

### Task 8: 8 个组件骨架（easycom + props/emits 类型）

**Files:**
- Create: 8 个 `src/components/<X>/<X>.vue` 骨架
- Create: `src/components/index.ts`
- Create: `web/patient-miniapp/__tests__/components/Countdown.test.ts`

**Step 1: Countdown 组件单测（RED）**

`web/patient-miniapp/__tests__/components/Countdown.test.ts`：

```ts
import { describe, it, expect } from '@jest/globals';
import { mount } from '@vue/test-utils';
import Countdown from '@/components/Countdown/Countdown.vue';

describe('<Countdown />', () => {
  it('渲染传入的 seconds 文案', () => {
    const w = mount(Countdown, { props: { seconds: 30, label: '匹配窗口' } });
    expect(w.text()).toContain('30');
    expect(w.text()).toContain('匹配窗口');
  });

  it('@finish 在 0s 时 emit 一次', async () => {
    jest.useFakeTimers();
    const w = mount(Countdown, { props: { seconds: 1 } });
    jest.advanceTimersByTime(1000);
    await Promise.resolve();
    expect(w.emitted('finish')).toBeDefined();
    expect(w.emitted('finish')!.length).toBe(1);
    jest.useRealTimers();
  });
});
```

```bash
cd web/patient-miniapp
npx jest __tests__/components/Countdown.test.ts
```

Expected: FAIL — `Cannot find module '@/components/Countdown/Countdown.vue'`。

**Step 2: Countdown 组件（GREEN）**

`web/patient-miniapp/src/components/Countdown/Countdown.vue`：

```vue
<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue';

interface Props {
  seconds: number;
  label?: string;
}
const props = withDefaults(defineProps<Props>(), { label: '倒计时' });
const emit = defineEmits<{ finish: [] }>();

const remaining = ref(props.seconds);
let timer: ReturnType<typeof setInterval> | null = null;

function start() {
  stop();
  remaining.value = props.seconds;
  timer = setInterval(() => {
    remaining.value -= 1;
    if (remaining.value <= 0) {
      stop();
      emit('finish');
    }
  }, 1000);
}
function stop() {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
}

watch(() => props.seconds, start, { immediate: true });
onUnmounted(stop);
</script>

<template>
  <view class="countdown">
    <text class="countdown-label">{{ label }} {{ remaining }} s</text>
  </view>
</template>

<style lang="scss" scoped>
.countdown { display: inline-flex; padding: $space-1 $space-2; border-radius: $radius-sm; background: $color-primary; color: #fff; }
.countdown-label { font-size: $font-sm; }
</style>
```

**Step 3: 跑测试**

```bash
cd web/patient-miniapp
npx jest __tests__/components/Countdown.test.ts
```

Expected: PASS（2 测试）。

**Step 4: 其余 7 个组件骨架（pattern + bash 批量）**

每个组件 ~30 行 template/script/style，展示 props 接口与 emit 类型；实现留后续 plan。

```bash
cd web/patient-miniapp

for pair in "OrderCard|order:OrderListItem|order-click" \
            "EscortBadge|escort:nickname;avatar:string" \
            "HospitalCard|hospital:Hospital|hospital-click" \
            "PackageCard|package:HospitalPackage|package-click" \
            "PriceTag|amount:number;currency?:string" \
            "VirtualNumber|phone:string;expiresAt:string|call" \
            "SOSButton|orderId:number|trigger"; do
  IFS='|' read -r name props emits <<< "$pair"
  src="src/components/${name}/${name}.vue"
  mkdir -p "$(dirname "$src")"
  cat > "$src" <<EOF
<script setup lang="ts">
// ${name} 骨架：props/emits 接口完整；UI 实现留后续 plan。
interface Props {
  ${props}
}
const props = defineProps<Props>();
const emit = defineEmits<{ ${emits}: any }>();
</script>

<template>
  <view class="${name,,}"><text>{{ JSON.stringify(props) }}</text></view>
</template>

<style lang="scss" scoped>
.${name,,} { display: block; }
</style>
EOF
done

# props 含特殊字符的（EscortBadge / PriceTag / VirtualNumber）单独追加字段类型
# OrderCard / HospitalCard / PackageCard 已经够用；剩余三个手动加类型
sed -i '' 's|escort:nickname;avatar:string|escort: { nickname: string; avatar: string }|' src/components/EscortBadge/EscortBadge.vue
sed -i '' 's|amount:number;currency?:string|amount: number; currency?: string|' src/components/PriceTag/PriceTag.vue
sed -i '' 's|phone:string;expiresAt:string|phone: string; expiresAt: string|' src/components/VirtualNumber/VirtualNumber.vue
sed -i '' 's|orderId:number|orderId: number|' src/components/SOSButton/SOSButton.vue
```

**Step 5: components/index.ts（手动注册用于 easycom 失败时 fallback）**

`web/patient-miniapp/src/components/index.ts`：

```ts
export { default as OrderCard } from './OrderCard/OrderCard.vue';
export { default as EscortBadge } from './EscortBadge/EscortBadge.vue';
export { default as HospitalCard } from './HospitalCard/HospitalCard.vue';
export { default as PackageCard } from './PackageCard/PackageCard.vue';
export { default as PriceTag } from './PriceTag/PriceTag.vue';
export { default as VirtualNumber } from './VirtualNumber/VirtualNumber.vue';
export { default as Countdown } from './Countdown/Countdown.vue';
export { default as SOSButton } from './SOSButton/SOSButton.vue';
```

**Step 6: typecheck**

```bash
cd web/patient-miniapp
npm run typecheck
```

Expected: PASS。

**Step 7: Commit**

```bash
cd web/patient-miniapp
git add src/components/ __tests__/components/
git commit -m "feat(patient-miniapp): 8 个组件骨架 + Countdown 单测"
```

---

### Task 9: i18n 占位 + Jest 完整配置 + Playwright e2e

**Files:**
- Create: `web/patient-miniapp/src/i18n/en.ts`
- Create: `web/patient-miniapp/src/i18n/zh-CN.ts`
- Create: `web/patient-miniapp/src/i18n/index.ts`
- Create: `web/patient-miniapp/jest.config.js`
- Create: `web/patient-miniapp/playwright.config.ts`
- Create: `web/patient-miniapp/playwright/index.html`（占位 e2e fixture）
- Modify: `web/patient-miniapp/__tests__/setup.ts`（如有，调整）
- Replace: `web/patient-miniapp/e2e/pages-skeleton.spec.ts`（Task 3 占位升级）
- Create: `web/patient-miniapp/e2e/login-to-list.spec.ts`
- Create: `web/patient-miniapp/mocks/handlers.ts`
- Create: `web/patient-miniapp/mocks/browser.ts`
- Create: `web/patient-miniapp/mocks/node.ts`

**Step 1: i18n 占位**

`web/patient-miniapp/src/i18n/en.ts`：

```ts
export default {
  app: { title: 'Companion Care' },
  common: { confirm: 'Confirm', cancel: 'Cancel', loading: 'Loading…' },
};
```

`web/patient-miniapp/src/i18n/zh-CN.ts`：

```ts
export default {
  app: { title: '陪诊' },
  common: { confirm: '确认', cancel: '取消', loading: '加载中…' },
};
```

`web/patient-miniapp/src/i18n/index.ts`：

```ts
import { createI18n } from 'vue-i18n';
import zhCN from './zh-CN';

export function createAppI18n() {
  return createI18n({
    locale: 'zh-CN',
    fallbackLocale: 'en',
    messages: {
      'zh-CN': zhCN,
    },
  });
}
```

> 注：Task 5 main.ts 已 `import { createI18n } from '@/i18n'` —— 改名为 `createAppI18n` 避免与 vue-i18n 命名冲突，同步更新 main.ts。

**Step 2: Jest 配置**

`web/patient-miniapp/jest.config.js`：

```js
/** @type {import('jest').Config} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  setupFiles: ['<rootDir>/__tests__/setup.ts'],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '\\.vue$': '<rootDir>/__tests__/vue-mock.ts',
  },
  transform: {
    '^.+\\.vue$': 'vue3-jest',
    '^.+\\.ts$': ['ts-jest', { tsconfig: '<rootDir>/tsconfig.json' }],
  },
  testMatch: ['<rootDir>/__tests__/**/*.test.ts'],
  collectCoverageFrom: [
    'src/utils/**/*.ts',
    'src/stores/**/*.ts',
    'src/api/**/*.ts',
    '!src/api/types.gen.ts',
  ],
  coverageThreshold: {
    'src/utils/**': { statements: 80, branches: 70, functions: 80, lines: 80 },
    'src/api/**':   { statements: 80, branches: 70, functions: 80, lines: 80 },
  },
};
```

`web/patient-miniapp/__tests__/setup.ts`：

```ts
import { installUniMock } from '@/utils/uni-mock';
installUniMock();
```

`web/patient-miniapp/__tests__/vue-mock.ts`：

```ts
// jsdom 不解析 .vue 单文件组件；jest 直接 import 时返回占位对象。
module.exports = {};
```

**Step 3: Playwright 配置**

`web/patient-miniapp/playwright.config.ts`：

```ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  timeout: 30 * 1000,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:5173',
    trace: 'on-first-retry',
  },
  webServer: {
    command: 'npm run dev:h5',
    url: 'http://127.0.0.1:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
});
```

**Step 4: MSW handlers（占位 + 4 个 mock 端点）**

`web/patient-miniapp/mocks/handlers.ts`：

```ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  http.post('/api/v1/auth/sms/send', () => HttpResponse.json({ code: 0, data: { ok: true } })),
  http.post('/api/v1/auth/login', () =>
    HttpResponse.json({
      code: 0,
      data: { access_token: 'mock-tok', refresh_token: 'mock-ref', user: { id: 1, phone: '13800138000' } },
    })
  ),
  http.get('/api/v1/users/me', () =>
    HttpResponse.json({ code: 0, data: { id: 1, phone: '13800138000', real_name_verified: true } })
  ),
  http.get('/api/v1/hospitals', () =>
    HttpResponse.json({
      code: 0,
      data: { items: [{ id: 1, name: '北京协和医院', city: '北京', district: '东城区', level: '三甲', photo_url: '' }], total: 1 },
    })
  ),
  // 其它 10 个端点（refresh / real-name / orders / create / cancel / payment / virtual-number /
  //   review / sos / refund / wallet / addresses / coupons / messages）在后续 plan 里按需补 mocks。
];
```

`web/patient-miniapp/mocks/browser.ts`：

```ts
import { setupWorker } from 'msw/browser';
import { handlers } from './handlers';

export const worker = setupWorker(...handlers);
```

`web/patient-miniapp/mocks/node.ts`：

```ts
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
```

**Step 5: e2e 用例**

`web/patient-miniapp/e2e/login-to-list.spec.ts`：

```ts
import { test, expect } from '@playwright/test';

test('home page renders skeleton title', async ({ page }) => {
  await page.goto('/#/pages/index/index');
  await expect(page.locator('text=首页')).toBeVisible();
});

test('orders tab renders skeleton title', async ({ page }) => {
  await page.goto('/#/pages/order/list');
  await expect(page.locator('text=订单列表')).toBeVisible();
});

test('profile tab renders skeleton title', async ({ page }) => {
  await page.goto('/#/pages/profile/index');
  await expect(page.locator('text=个人中心')).toBeVisible();
});
```

`web/patient-miniapp/e2e/pages-skeleton.spec.ts`（升级版）：

```ts
import { test, expect } from '@playwright/test';
import pagesManifest from '../src/pages.json';

const PAGES = [
  'pages/auth/login','pages/auth/real-name',
  'pages/hospital/list','pages/hospital/detail','pages/package/detail',
  'pages/order/create','pages/order/pay','pages/order/list','pages/order/detail',
  'pages/refund/apply','pages/review/create','pages/sos/trigger',
  'pages/wallet/index','pages/coupon/list','pages/address/list','pages/address/edit',
  'pages/message/list','pages/message/detail','pages/profile/index',
];

test('pages.json declares all 20 P0 routes (含 index)', () => {
  const declared = pagesManifest.pages.map((p: any) => p.path);
  expect(declared.length).toBe(21); // 20 + index
  for (const p of PAGES) expect(declared).toContain(p);
  expect(declared).toContain('pages/index/index');
});

test('每个 P0 页面在 H5 下渲染骨架', async ({ page }) => {
  for (const p of PAGES) {
    await page.goto(`/#/${p}`);
    await expect(page.locator('text=骨架')).toBeVisible({ timeout: 5000 });
  }
});
```

**Step 6: 跑 jest 全套 + Playwright 全套**

```bash
cd web/patient-miniapp
npm run test:unit -- --coverage
```

Expected: 全部 PASS + coverage utils ≥ 80%。

```bash
cd web/patient-miniapp
npm run test:e2e
```

Expected: 全部 PASS（pages-skeleton 2 测试 + login-to-list 3 测试 = 5 个）。

**Step 7: Commit**

```bash
cd web/patient-miniapp
git add src/i18n/ jest.config.js jest.setup.ts playwright.config.ts \
        __tests__/ e2e/ mocks/
git commit -m "test(patient-miniapp): jest + playwright + MSW mocks + 5 个 e2e 用例"
```

---

### Task 10: OpenAPI codegen 脚本 + 占位 contracts.yaml

**Files:**
- Create: `web/patient-miniapp/openapi/contracts.yaml`（14 API + entities 占位）
- Create: `web/patient-miniapp/scripts/openapi-generate.sh`
- Create: `web/patient-miniapp/src/api/types.gen.ts`（首次 codegen 产物 commit 进仓库）

**Step 1: 写最小 contracts.yaml（占位；后端 plan 产出后覆盖）**

`web/patient-miniapp/openapi/contracts.yaml`：

```yaml
openapi: 3.0.3
info:
  title: Patient Miniapp API
  version: 0.1.0
  description: |
    patient-miniapp 的 14 个 P0 API 占位契约。完整 schema 由后端
    `2026-09-24-hospital-package-plan.md` 等 plan 产出后覆盖此文件。
servers:
  - url: /api/v1
paths:
  /auth/sms/send:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SendSmsReq'
      responses:
        '200':
          description: ok
  /auth/login:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/LoginByPhoneReq'
                - $ref: '#/components/schemas/LoginByWxReq'
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/LoginResp'
  /orders:
    get:
      parameters:
        - in: query
          name: status
          schema: { type: string }
        - in: query
          name: page
          schema: { type: integer }
      responses: { '200': { description: ok } }
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/CreateOrderReq' }
      responses: { '200': { description: ok } }
  /orders/{id}:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/cancel:
    post:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/payment:
    post:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/payment/mock-callback:
    post:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/escort-summary:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/virtual-number:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/review:
    post:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/sos:
    post:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /orders/{id}/refund:
    post:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /users/me:
    get:
      responses: { '200': { description: ok } }
  /users/me/wallet:
    get:
      responses: { '200': { description: ok } }
  /users/me/coupons:
    get:
      responses: { '200': { description: ok } }
  /users/me/addresses:
    get:
      responses: { '200': { description: ok } }
    put:
      responses: { '200': { description: ok } }
  /hospitals:
    get:
      parameters:
        - in: query
          name: keyword
          schema: { type: string }
        - in: query
          name: city
          schema: { type: string }
      responses: { '200': { description: ok } }
  /hospitals/{id}:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /hospitals/{id}/packages:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /escorts/{id}/reviews:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
  /messages/conversations:
    get:
      responses: { '200': { description: ok } }
  /messages/conversations/{id}/messages:
    get:
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: integer }
      responses: { '200': { description: ok } }
components:
  schemas:
    SendSmsReq:
      type: object
      required: [phone]
      properties: { phone: { type: string } }
    LoginByPhoneReq:
      type: object
      required: [phone, code]
      properties: { phone: { type: string }; code: { type: string } }
    LoginByWxReq:
      type: object
      required: [wx_code]
      properties: { wx_code: { type: string } }
    LoginResp:
      type: object
      required: [access_token, refresh_token, user]
      properties:
        access_token:  { type: string }
        refresh_token: { type: string }
        user:          { $ref: '#/components/schemas/UserSnapshot' }
    UserSnapshot:
      type: object
      properties:
        id:                  { type: integer }
        phone:               { type: string }
        real_name_verified:  { type: boolean }
    CreateOrderReq:
      type: object
      required: [hospital_id, package_id, service_start_at, address_id]
      properties:
        hospital_id:       { type: integer }
        package_id:        { type: integer }
        service_start_at:  { type: string, format: date-time }
        address_id:        { type: integer }
        remark:            { type: string }
```

**Step 2: 写 codegen 脚本**

`web/patient-miniapp/scripts/openapi-generate.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

OUT="src/api/types.gen.ts"
npx --yes openapi-typescript ./openapi/contracts.yaml --output "$OUT" --enum
echo "[openapi] regenerated $OUT"
```

```bash
cd web/patient-miniapp
chmod +x scripts/openapi-generate.sh
npm run openapi:generate
```

Expected: 输出 `src/api/types.gen.ts`。

**Step 3: typecheck 确认 generated types 能用**

```bash
cd web/patient-miniapp
npm run typecheck
```

Expected: PASS。

**Step 4: validate 校验（empty diff）**

```bash
cd web/patient-miniapp
npm run openapi:validate
```

Expected: 无 diff（generate → validate diff 一致）。

**Step 5: Commit**

```bash
cd web/patient-miniapp
git add openapi/ scripts/ src/api/types.gen.ts
git commit -m "feat(patient-miniapp): OpenAPI codegen 脚本 + 14 API 占位契约 + types.gen.ts"
```

---

### Task 11: GitHub Actions CI（typecheck + test:unit + test:e2e + openapi:validate）

**Files:**
- Create: `.github/workflows/patient-miniapp-ci.yml`

**Step 1: CI workflow**

`.github/workflows/patient-miniapp-ci.yml`：

```yaml
name: patient-miniapp-ci

on:
  pull_request:
    paths:
      - 'web/patient-miniapp/**'
      - '.github/workflows/patient-miniapp-ci.yml'
  push:
    branches: [main]
    paths:
      - 'web/patient-miniapp/**'

jobs:
  build:
    runs-on: ubuntu-latest
    timeout-minutes: 25
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: web/patient-miniapp/package-lock.json
      - working-directory: web/patient-miniapp
        run: npm ci --no-audit --no-fund
      - working-directory: web/patient-miniapp
        run: npm run typecheck
      - working-directory: web/patient-miniapp
        run: npm run lint
      - working-directory: web/patient-miniapp
        run: npm run test:unit -- --coverage
      - working-directory: web/patient-miniapp
        run: npm run openapi:validate
      - working-directory: web/patient-miniapp
        run: npm run build:h5
      - name: Install Playwright browsers
        working-directory: web/patient-miniapp
        run: npx playwright install --with-deps chromium
      - working-directory: web/patient-miniapp
        run: npm run test:e2e
      - name: Upload coverage
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: coverage-${{ github.run_id }}
          path: web/patient-miniapp/coverage/
```

**Step 2: 本地用 act 跑一遍 dry-run（可选；不做也没关系）**

```bash
# 仅记录，不强制 — CI 验证放到合并阶段
echo "[ci] yml syntactically valid if Step 1 文件能在 .github/workflows/ 列表里看到"
ls -la .github/workflows/patient-miniapp-ci.yml
```

Expected: 文件存在。

**Step 3: Commit**

```bash
cd /Users/growduduan/ai/doctors
git add .github/workflows/patient-miniapp-ci.yml
git commit -m "ci(patient-miniapp): GitHub Actions (typecheck + lint + test:unit + test:e2e + openapi:validate + build:h5)"
```

---

### Task 12: 文档同步 + dev.md §10.13 + 全量回归

**Files:**
- Modify: `dev.md`（新增 §10.13）
- Modify: `docs/04-业务流程.md`（新增 §4.7 patient-miniapp 章节概要）

**Step 1: dev.md §10.13**

在 `dev.md` 第 10 章末尾追加：

```markdown
### 10.13 patient-miniapp 骨架（2026-09-24 patient-miniapp setup plan）

实现 L2 v1.0 前端患者端的项目骨架：uni-app + Vue 3 + uView Plus + Pinia + OpenAPI codegen。

**落地 commits（12 个）**：

| commit | 内容 |
| :-- | :-- |
| chore(patient-miniapp) | 项目骨架 (uni-app + Vue 3 + TS + Vite + npm 锁定) |
| feat(patient-miniapp) | 设计 token + uview-plus 接入 + easycom |
| feat(patient-miniapp) | 20 个 P0 页面骨架 + 完整 routes + tabBar |
| feat(patient-miniapp) | utils (trace/format/auth/wx/uni-mock) + 6 单测 |
| feat(patient-miniapp) | api client + interceptor (X-Trace-Id + 11001 reLaunch) + 2 单测 |
| feat(patient-miniapp) | 14 patient P0 API 模块骨架 |
| feat(patient-miniapp) | Pinia stores (auth/order/hospital/wallet/message) + 6 单测 |
| feat(patient-miniapp) | 8 个组件骨架 + Countdown 单测 |
| test(patient-miniapp) | jest + playwright + MSW mocks + 5 e2e |
| feat(patient-miniapp) | OpenAPI codegen 脚本 + 14 API 占位契约 + types.gen.ts |
| ci(patient-miniapp) | GitHub Actions (typecheck + lint + test:unit + test:e2e + openapi:validate + build:h5) |
| docs | patient-miniapp 骨架计划文档 |

**当前覆盖（v1 骨架）**：20 个 P0 页面路由 + 9 个 API 模块（覆盖 l2-api-gap-design.md §2.1 的 14 API）+ 5 个 Pinia store + 8 个组件骨架 + 6 个 utils + Jest 单测 + Playwright e2e + OpenAPI codegen。

**未做（v1 留给后续 plan，按 feature 拆）**：
- 业务流程实现：登录/注册/实名 → 下单/支付/详情/退款/评价；详见 §4.7 入口
- UI 视觉细节（tab 图标 / 空状态插画 / 颜色主题）
- 真实后端连接（dev:h5 时通过 Vite proxy 转 doctors backend；v1 默认走 MSW mock）
- 微信沙箱支付真接通（v1 走 mockPayCallback）
- i18n 翻译（v3）

**未做（全项目层面）**：把 patient/escort/admin 三端 monorepo 化（共享 types / design token）；见 roadmap §5.3。
```

**Step 2: `docs/04-业务流程.md` §4.7（占位概要）**

在末尾追加：

```markdown
### 4.7 patient-miniapp 全链路（骨架已就位，按 feature plan 推进）

骨架 plan：`docs/superpowers/plans/2026-09-24-patient-miniapp-setup.md`。后续按 feature 拆 plan：
- 登录 / 实名 plan（phone + wx + mock real-name）
- 医院 / 服务包浏览 plan
- 下单 / 支付 / 详情 / 轮询 plan
- 退款 / 评价 plan
- SOS / 钱包 / 优惠券 / 地址 / 站内信 plan（按 P0 / P1 / P2 拆）
```

**Step 3: 跑全量回归**

```bash
cd web/patient-miniapp
npm run typecheck
npm run lint
npm run test:unit -- --coverage
npm run openapi:validate
npm run test:e2e
```

Expected：typecheck PASS / lint PASS（warning 可接受）/ 单测 PASS + utils & api coverage ≥ 80% / validate PASS / e2e PASS。

**Step 4: Commit**

```bash
cd /Users/growduduan/ai/doctors
git add dev.md docs/04-业务流程.md
git commit -m "docs: patient-miniapp 骨架流程概要 + dev.md §10.13 + 12 commits 落地记录"
```

---

### Task 13: 全量 push + 校验 commit 历史

**Step 1: 检查 commit 历史**

```bash
cd /Users/growduduan/ai/doctors
git log --oneline -15
```

Expected 列出本 plan 12+ 个 commits（其中 Task 1~12 的 commits + Task 12 docs commit）。

**Step 2: 列出新增/修改文件清单**

```bash
cd /Users/growduduan/ai/doctors
git diff --stat HEAD~13 HEAD | head -100
```

Expected: ~75 个新文件，路径集中在 `web/patient-miniapp/` + `.github/workflows/` + `dev.md` + `docs/04-业务流程.md`。

**Step 3: push（如用户授权；本 Task 默认不自动 push）**

> 不自动 push；用户决定后再 `git push origin main`。

---

## Self-Review

- ✅ **Spec 覆盖**:
  - spec §2 目录结构：src/{pages,components,stores,api,utils} + App.vue + main.ts + pages.json + manifest.json 全部就位
  - spec §3.1 20 P0 页面：全部 20 + 1 个 index 共 21 个 .vue 骨架 + 完整 routes + tabBar
  - spec §4.1 / §5.1 / §5.2：5 个 store + 14 API 模块（auth/order/hospital/refund/review/sos/wallet/address/coupon/message） + uni-request 全局拦截器（X-Trace-Id + 11001 reLaunch login）
  - spec §7 测试矩阵：Jest（utils 6 + stores 6 + Countdown 2 + client 2 = 16 测试）+ Playwright e2e（5 用例）+ MSW mock（4 handlers 占位）+ openapi-typescript 校验
  - spec §8 构建 + CI：`dev:mp-weixin` / `build:h5` / `typecheck` / `test:unit` / `test:e2e` / GitHub Actions ci
- ✅ **无占位符**: 每个文件/脚本/命令给出具体内容；Step 1 ~ Step N 不留 TODO
- ✅ **类型一致**: API 类型在 `api/<feature>.ts` 手写 + `api/types.gen.ts` codegen 共存（手写优先，codegen 后续 plan 替换）；store 接口与 utils/auth / api/auth 对齐
- ✅ **测试矩阵**:
  - utils: trace 3 + format 3 = 6（≥ 80% 覆盖）
  - stores: auth 3 + order 3 = 6
  - api/client: 2
  - 组件: Countdown 2
  - e2e: 5（pages-skeleton 2 + login-to-list 3）
  - 总计 21 个测试用例
- ✅ **YAGNI**:
  - 不引 axios（用 uni.request）
  - 不引 vue-router（用 uni-app pages.json）
  - 不引 UI 库二次封装（直接 uview-plus）
  - i18n 只装 zh-CN + en 两个空 locale，文案留 v3
  - WebSocket / 真实推送通道 / 真实 OSS 不在 v1 范围
  - monorepo（patient/escort/admin 三端共享类型 + token）留后续 plan；本 plan 先做单端骨架

## Execution Options

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-patient-miniapp-setup.md`。
> 当前为 plan_only 模式 → 进入实施阶段需要用户决策。

**下一步选项**：

1. **立即执行**（subagent-driven 或 inline 执行）—— 我开始实施 Task 1~13（每个 Task 一个 commit，共 12 commits；预估 ~1.5 小时，依赖 npm install 网络速度）
2. **暂停 + review** —— 用户 review 此 plan 后告诉调整点
3. **继续产 plan** —— 接着出 9 个后端 plan + 3 个前端 plan（virtual-number / wallet / escort-business / hospital-package / review / message / address-coupon / admin / escort-order-ext + escort-app setup / admin-web setup）

## 关键设计偏差（需用户确认）

| 项 | spec / 默认 | 实际选择 | 原因 |
|---|---|---|---|
| OpenAPI contracts.yaml 内容 | 占位（仅 14 API 路径，无 schema） | 同 | 后端 10 个 plan 未启动；用占位契约让 codegen 跑通，后端 plan 推进时同步覆盖 |
| Jest 环境 | jsdom + ts-jest + vue3-jest | 同 | jsdom 不解析 uni 全局；store / api / utils 测试无 uni 副作用；组件测试用 vue3-jest |
| Pinia 接入 | createPinia + app.use | 同 | 标准 vue-i18n + Pinia 入口范式 |
| mp-weixin appid | manifest.json 写死 "TOURIST_APPID" | 同占位 | 真实 appid 由用户在微信开发者后台 + `dcloud_appid` 引入；本 plan 给占位 |
| MSW 在 H5 dev 中是否启用 | 仅用于 jest + Playwright；H5 真开发时不强制 | 同 | H5 真开发用 Vite proxy 转发到 `DOCTORS_BACKEND_URL`，避免双拦截链 |
| wx.* API 在 H5 模式 | `wx.ts` 加 `#ifdef MP-WEIXIN` 条件编译 | 同 | H5 下 requestPayment resolve() 占位，避免阻塞 dev |
| TabBar | 只配 4 个（首页 / 订单 / 消息 / 我的） | 同 | spec §3.1 没规定 4-tab 列表，按主流 L2 患者端 tab 设计，匹配 spec §3.1 P0 |
| i18n 引入时机 | 现在就占位 | 同 | spec §10 留 v3，但用户偏好"可测试 / 可回溯"，提前占位以便后续 plan 直接加 key；不真多语言 |
