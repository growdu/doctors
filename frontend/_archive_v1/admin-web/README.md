# Doctors Admin Web

内部运营后台 SPA（L2 v1.0 admin-web）。

## 技术栈

- **React** 18.3 + **TypeScript** 5.5
- **Vite** 5.4（fast HMR）
- **Ant Design** 5.21（中文 + 主题定制）
- **状态管理** Zustand 4.5
- **数据请求** TanStack Query 5.x
- **HTTP** axios
- **路由** React Router 6.26
- **Mock** MSW 2.x（开发期拦截 fetch 模拟后端）

## 启动

```bash
cd frontend/admin-web
npm install     # 或 pnpm install
npm run dev     # http://127.0.0.1:5173
```

## 目录结构

```
admin-web/
├── src/
│   ├── main.tsx                  # React 入口
│   ├── App.tsx                   # ConfigProvider + QueryClientProvider + AntdApp + BrowserRouter
│   ├── layouts/
│   │   └── AdminLayout.tsx       # 侧边栏 + 顶栏 + 内容区 (Header/Sider/Content/Outlet)
│   ├── router/
│   │   └── index.tsx             # 占位路由：/dashboard /escorts /orders
│   ├── pages/
│   │   ├── dashboard/DashboardPage.tsx
│   │   ├── escorts/EscortsPage.tsx
│   │   └── orders/OrdersPage.tsx
│   └── vite-env.d.ts
├── index.html
├── vite.config.ts                # @ 别名 @/
├── tsconfig.json
└── package.json
```

## 当前状态

**v1 初始化（Task 1-3）**：Vite + React + TS 脚手架完成；依赖装好但未 `npm install`；
App 骨架就绪（ConfigProvider + QueryClientProvider + AntdApp + BrowserRouter + AdminLayout + 占位路由）。

**未做（Task 4+）**：18 个 P0 页面 / RBAC / Zustand authStore / MSW handlers / StatusBadge / ProTable 封装等。
详见 `docs/superpowers/specs/2026-09-24-admin-web-design.md` 与后续 plan。

## 路径别名

`@/` → `src/`（在 `vite.config.ts` 与 `tsconfig.json` 中均配置）。

## 主题

主色 `#1677ff`（蓝）/ 警告 `#faad14` / 成功 `#52c41a` / 错误 `#ff4d4f`，
与 spec §6 UI 设计对齐。

## 下一步

1. `npm install` 拉依赖
2. `npm run dev` 启动 dev server（应能打开 http://127.0.0.1:5173 看到 Dashboard 占位）
3. 进入 Task 4（继续 plan 中的 v1 实施步骤）