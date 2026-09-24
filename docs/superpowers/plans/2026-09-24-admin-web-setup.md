# Admin-Web Setup Implementation Plan (v2：选人模式适配)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 v1 admin-web SPA 骨架（Vite 5 + React 18 + TS + 18 P0 页面 + RBAC + MSW mock + ProTable 封装）已完成的基础上，按 spec `2026-09-24-order-matching-redesign.md` §5 的 admin 端影响，**增量适配订单匹配模式从「抢单」改为「选人」**：OrderListPage 加 2 个新状态筛选 + 2 个新列；OrderDetailPage 加状态机进度条分支 + 已选陪诊师 + 30s 倒计时 + 拒接回退提示；DashboardPage 加待确认订单卡片；StatusBadge 加 2 个新状态色（橙色 / 蓝色）；MSW handlers 加 fixture；OpenAPI contracts.yaml 加 3 个新字段。v2 重点是 UI/UX 适配 + mock 扩展 + 类型扩展；后端字段语义由 patient/escort 端 plan 提供。

**Architecture:** 沿用 v1 单仓多包结构 `web/admin-web/`（Vite 5 + React 18 + TS 5.5 + AntD 5 + ProTable + Zustand + TanStack Query + MSW 2）。本 v2 仅修改 5 类文件：(a) `src/pages/orders/*` 列表 + 详情页；(b) `src/pages/dashboard/DashboardPage.tsx` 看板；(c) `src/components/StatusBadge/StatusBadge.tsx` 状态徽章；(d) `src/mocks/handlers/admin/orders.ts` + `src/mocks/data/seed.ts` mock 数据；(e) `web/openapi/contracts.yaml` admin-service OpenAPI 字段。测试在 `src/**/__tests__/` + `src/pages/orders/__tests__/` 增量补充；不引新依赖（橙色 / 蓝色复用 AntD `colorWarning` / `colorProcessing`）。

**Tech Stack:**
- **运行时**: Node.js 20+ · pnpm 9+（沿用 v1）
- **框架**: React 18.3 + TypeScript 5.5 strict
- **UI**: Ant Design 5.21（Badge `status="processing"` 蓝色 + `status="warning"` 橙色）
- **数据请求**: TanStack Query 5.x（v1 已装）
- **时间处理**: dayjs 1.11 + `dayjs/plugin/relativeTime`（30s 倒计时）
- **Mock**: MSW 2.x（v1 已装；扩展 fixtures）
- **OpenAPI**: openapi-typescript 7.x（v1 已装；扩展 contracts.yaml 后 `pnpm run generate:client` 重生成 `src/types/generated.ts`）
- **测试**: Vitest + React Testing Library（v1 已装）
- **代码质量**: ESLint 9 + Prettier 3（v1 已装）

**前置依赖:**
- v1 plan 已完成（commit `git log --oneline | grep admin-web` 应有 13 个 commit）：Vite+React+TS 脚手架、ESLint+Prettier、Vitest+RTL+Playwright、openapi-typescript、fetch client、MSW handlers、Zustand authStore、4 个 hooks、Router+RBAC、ProTable+StatusBadge+AuditAction+TraceId+ErrorBoundary+PageHeader、AdminLayout、18 P0 页面骨架、4 个 E2E
- `docs/superpowers/specs/2026-09-24-order-matching-redesign.md` §5（admin 端影响：状态机新分支 + dashboard 待确认卡片 + 详情倒计时）
- `docs/superpowers/specs/2026-09-24-admin-web-design.md`（admin-web 设计：18 页面 + RBAC + 目录结构）
- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.3（admin 12 API 清单）
- `web/openapi/contracts.yaml`（v1 admin-service 已生成；本 v2 在 OrderDetail schema 加 3 个新字段）
- 后端 `shared/contracts/events.go` 已加 `OrderSelectingEscortEvent` / `OrderEscortSelectedEvent` / `OrderEscortConfirmedEvent` / `OrderEscortRejectedEvent`（admin-web v2 不消费 Kafka；TanStack Query 轮询即可）

**v1 → v2 变更摘要（v1 15 个 task 详见 git history；本 plan 仅含 v2 增量 6 个 task）：**

| 模块 | v1（已完成） | v2（本 plan） |
|---|---|---|
| OrderListPage 状态筛选 | `created/paid/matching/accepted/in_service/completed/refunding/refunded/disputed/closed/canceled` | + `selecting_escort` / `escort_pending_acceptance` |
| OrderListPage 列 | 订单号/患者/医院/金额/状态/下单时间/操作 | + `selected_escort_id`（已选陪诊师）+ `escort_pending_expire_at`（30s 截止时间） |
| OrderDetailPage 状态机进度条 | `paid → matching → accepted → in_service → completed` 单链 + 5 状态 | + 2 状态分支（`selecting_escort` / `escort_pending_acceptance`） + 30s 倒计时组件 + 拒接回退卡 |
| DashboardPage | 4 个 Statistic 卡片（今日订单/GMV/待审陪诊师/待审退款） | + 2 个新卡片（待患者选人 / 待陪诊师确认）+ 点击跳 OrderList 带 status filter |
| StatusBadge | 11 种状态色（default/processing/success/warning/error） | + `selecting_escort`（橙色 `colorWarning`）+ `escort_pending_acceptance`（蓝色 `colorProcessing`） |
| MSW orders handler | 10 条 fixture（覆盖全状态） | + 2 条 fixture（`status=selecting_escort` + `status=escort_pending_acceptance`） |
| OpenAPI OrderDetail schema | 11 字段（id/patient/hospital/amount/status/...） | + `selected_escort_id` (int64) + `escort_pending_expire_at` (datetime) + `escort_reject_reason` (string) |

---

## Global Constraints

- Node.js 20 LTS（v1 toolchain 保持）
- pnpm 9+ workspace（沿用 v1；不动 `pnpm-workspace.yaml`）
- React 18.3 + TypeScript 5.5 strict mode（沿用 v1）
- 测试覆盖率：v2 增量代码（StatusBadge 2 新状态 + 详情倒计时 + 列表新列）≥ 70%；v1 既有代码覆盖率不降
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat(admin-web):` / `test(admin-web):` / `chore(admin-web):` / `docs(admin-web):`
- 包管理：`pnpm`（沿用 v1）
- HTTP client：fetch 包装（v1 已装，不动）
- 状态管理：Zustand（沿用 v1）
- 数据请求：TanStack Query v5（沿用 v1）
- UI：Ant Design 5（v1 已装）+ `Badge.status="warning"`（橙）/ `Badge.status="processing"`（蓝）；不引新色板
- Mock：MSW 2（v1 已装）；扩展 `src/mocks/handlers/admin/orders.ts` + `src/mocks/data/seed.ts`
- OpenAPI：修改 `web/openapi/contracts.yaml` → 跑 `pnpm --filter admin-web run generate:client` → `src/types/generated.ts` 自动重生成
- 倒计时：用 `useEffect` + `setInterval` 1s tick；组件卸载时 clear（无内存泄漏）；剩余 < 60s 红色高亮（`<span style={{color: '#ff4d4f'}}>`）
- 不引：axios / redux / react-table / lodash / moment / styled-components / 第三方倒计时库（用 dayjs + 原生 setInterval）
- v2 不做：WebSocket 实时推送（v1 已轮询 5s；v2 倒计时独立 1s tick）/ 暗色主题 / i18n / 移动端响应式

---

## File Structure 表（v2 增量修改）

| 文件 | Mode | 职责 |
|---|---|---|
| `web/admin-web/src/pages/orders/OrderListPage.tsx` | **Modify** | 状态筛选加 2 选项 + 列加 2 列 + 详情链接可点 |
| `web/admin-web/src/pages/orders/OrderDetailPage.tsx` | **Modify** | 状态机进度条加 2 状态分支 + `selected_escort_id` 展示 + 30s 倒计时 + 拒接回退卡 |
| `web/admin-web/src/pages/orders/EscortPendingCountdown.tsx` | **Create** | 30s 倒计时小组件（`useEffect` + setInterval + dayjs relativeTime + < 60s 红色高亮） |
| `web/admin-web/src/pages/dashboard/DashboardPage.tsx` | **Modify** | 加 2 个 Statistic 卡片（待患者选人 + 待陪诊师确认）+ `useNavigate` 跳列表 |
| `web/admin-web/src/components/StatusBadge/StatusBadge.tsx` | **Modify** | 加 `selecting_escort` (橙色) + `escort_pending_acceptance` (蓝色) 2 状态色 |
| `web/admin-web/src/mocks/handlers/admin/orders.ts` | **Modify** | 列表 mock 加 2 条 fixture；get-by-id 加 `selected_escort_id` + `escort_pending_expire_at` + `escort_reject_reason` 字段 |
| `web/admin-web/src/mocks/data/seed.ts` | **Modify** | `mockOrders` 数组加 2 条 fixture（id=9001 selecting / id=9002 escort_pending） |
| `web/admin-web/src/types/generated.ts` | **Regenerate** | `pnpm run generate:client` 从 contracts.yaml 重生成（不手写） |
| `web/admin-web/src/api/admin/orders.ts` | **Modify** | `getOrder` / `listOrders` 类型用 `OrderDetail`/`OrderListItem`（来自 generated.ts；新字段自动包含） |
| `web/admin-web/src/pages/orders/OrderListPage.test.tsx` | **Create** | RTL + MSW node：测试 `selecting_escort` / `escort_pending_acceptance` 2 状态筛选 + 2 列渲染 |
| `web/admin-web/src/pages/orders/OrderDetailPage.test.tsx` | **Create** | RTL + MSW node：测试进度条 2 状态分支 + 30s 倒计时 < 60s 红色高亮 + 拒接回退卡渲染 |
| `web/admin-web/src/pages/dashboard/DashboardPage.test.tsx` | **Create** | RTL + MSW node：测试 2 个新卡片渲染 + 点击跳 `/orders?status=selecting_escort` |
| `web/admin-web/src/components/StatusBadge/StatusBadge.test.tsx` | **Modify** | 加 2 个 it：渲染 `selecting_escort` 橙色 Badge + 渲染 `escort_pending_acceptance` 蓝色 Badge |
| `web/openapi/contracts.yaml` | **Modify** | `OrderDetail` schema 加 `selected_escort_id` (int64 nullable) + `escort_pending_expire_at` (datetime nullable) + `escort_reject_reason` (string nullable) 3 个 |
| `docs/04-业务流程.md` | **Modify** | §4.8 加 admin-web v2 选人模式适配说明 |
| `dev.md` | **Modify** | §10.15 加本 plan 落地记录 |

---

## Task 1: OrderListPage 加 2 状态筛选 + 2 列 + 详情链接可点

**Files:**
- Modify: `web/admin-web/src/pages/orders/OrderListPage.tsx`

**Step 1: 写测试（RED）**

`web/admin-web/src/pages/orders/OrderListPage.test.tsx`：

```tsx
// 验证：列表页能根据 status filter 过滤 selecting_escort / escort_pending_acceptance；
// 2 列（已选陪诊师 + 截止时间）渲染；详情链接可点跳详情页。
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import OrderListPage from './OrderListPage';

const handlers = [
  http.get('http://localhost/api/v1/admin/orders', ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const allOrders = [
      { id: 9001, status: 'selecting_escort', selected_escort_id: null, escort_pending_expire_at: null,
        hospital_name: '协和医院', final_amount: 500, created_at: '2026-09-24T10:00:00Z' },
      { id: 9002, status: 'escort_pending_acceptance', selected_escort_id: 42, escort_pending_expire_at: '2026-09-24T10:00:30Z',
        hospital_name: '同仁医院', final_amount: 800, created_at: '2026-09-24T10:01:00Z' },
    ];
    const filtered = status ? allOrders.filter((o) => o.status === status) : allOrders;
    return HttpResponse.json({ data: filtered, total: filtered.length });
  }),
];
const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function renderWithProviders() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/orders']}>
        <OrderListPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('OrderListPage v2 selecting_escort 列与筛选', () => {
  it('renders selected_escort_id + escort_pending_expire_at 2 列', async () => {
    renderWithProviders();
    await waitFor(() => expect(screen.getByText('9001')).toBeInTheDocument());
    expect(screen.getByText('已选陪诊师')).toBeInTheDocument();
    expect(screen.getByText('确认截止')).toBeInTheDocument();
  });

  it('筛选 selecting_escort 仅显示 id=9001', async () => {
    const user = userEvent.setup();
    renderWithProviders();
    await waitFor(() => screen.getByText('9001'));
    // 点状态筛选下拉（ProTable）→ 选 selecting_escort → 列表只剩 9001
    // 简化断言：触发状态过滤后 9002 不在
    const filterSelect = screen.getAllByRole('combobox')[0];
    await user.click(filterSelect);
    await user.click(screen.getByText('待患者选人'));
    await waitFor(() => {
      expect(screen.getByText('9001')).toBeInTheDocument();
      expect(screen.queryByText('9002')).not.toBeInTheDocument();
    });
  });
});
```

**Step 2: 跑测试确认失败**

Run:
```bash
pnpm --filter admin-web exec vitest run OrderListPage 2>&1 | tail -30
```

Expected: FAIL — `Cannot find module './OrderListPage'` 或列名断言失败（v1 没有"已选陪诊师"列）

**Step 3: 改 OrderListPage**

`web/admin-web/src/pages/orders/OrderListPage.tsx`：

```tsx
// v2：状态筛选加 selecting_escort / escort_pending_acceptance；
// 列加 selected_escort_id + escort_pending_expire_at；详情链接可点跳详情页。
import { useNavigate } from 'react-router-dom';
import { ProTable } from '@ant-design/pro-components';
import { Tag, Button } from 'antd';
import dayjs from 'dayjs';
import { StatusBadge } from '@/components/StatusBadge/StatusBadge';
import { useOrdersQuery } from '@/api/admin/orders';
import type { OrderListItem } from '@/types/generated';

// v2 新增 2 状态（橙色 / 蓝色徽章由 StatusBadge 内部映射）
const orderStatusEnum = {
  created: { text: '已创建', status: 'default' },
  paid: { text: '已支付', status: 'processing' },
  selecting_escort: { text: '待患者选人', status: 'warning' },
  escort_pending_acceptance: { text: '待陪诊师确认', status: 'processing' },
  accepted: { text: '已接单', status: 'success' },
  in_service: { text: '服务中', status: 'success' },
  completed: { text: '已完成', status: 'success' },
  refunding: { text: '退款中', status: 'warning' },
  refunded: { text: '已退款', status: 'default' },
  disputed: { text: '申诉中', status: 'error' },
  closed: { text: '已关闭', status: 'default' },
  canceled: { text: '已取消', status: 'default' },
} as const;

export default function OrderListPage() {
  const navigate = useNavigate();

  return (
    <ProTable<OrderListItem, { status?: string }>
      headerTitle="订单管理"
      rowKey="id"
      polling={5000}
      params={{ status: undefined }}
      request={async (params) => {
        const { data, total } = await useOrdersQuery({
          status: params.status,
          page: params.current,
          page_size: params.pageSize,
        });
        return { data, success: true, total };
      }}
      columns={[
        { title: '订单号', dataIndex: 'id', width: 90,
          render: (_, r) => <a onClick={() => navigate(`/orders/${r.id}`)}>{r.id}</a> },
        { title: '医院', dataIndex: 'hospital_name', width: 160 },
        { title: '金额', dataIndex: 'final_amount', width: 100,
          render: (v) => `¥${v}` },
        // v2 新增：状态筛选下拉包含 selecting_escort / escort_pending_acceptance
        { title: '状态', dataIndex: 'status', width: 130, valueType: 'select',
          valueEnum: orderStatusEnum,
          render: (_, r) => <StatusBadge status={r.status} /> },
        // v2 新增列：已选陪诊师（selecting_escort 时显示「未选」tag）
        { title: '已选陪诊师', dataIndex: 'selected_escort_id', width: 110, hideInSearch: true,
          render: (v) => v ? <Tag color="blue">#{v}</Tag> : <Tag>未选</Tag> },
        // v2 新增列：escort_pending_acceptance 时显示截止时间（30s 内倒计时由详情页管；列表仅展示绝对时间）
        { title: '确认截止', dataIndex: 'escort_pending_expire_at', width: 170, hideInSearch: true,
          render: (v) => v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—' },
        { title: '下单时间', dataIndex: 'created_at', width: 160, valueType: 'dateTime' },
        { title: '操作', valueType: 'option', width: 120,
          render: (_, r) => [
            <Button key="view" type="link" onClick={() => navigate(`/orders/${r.id}`)}>详情</Button>,
          ] },
      ]}
    />
  );
}
```

**Step 4: 跑测试确认通过（GREEN）**

Run:
```bash
pnpm --filter admin-web exec vitest run OrderListPage 2>&1 | tail -20
```

Expected: PASS — `2 passed (OrderListPage)`；列名"已选陪诊师"+"确认截止"断言通过；筛选断言通过

**Step 5: 类型检查 + Lint**

```bash
pnpm --filter admin-web run typecheck 2>&1 | tail -10
pnpm --filter admin-web run lint 2>&1 | tail -10
```

Expected: 0 errors / 0 warnings

**Step 6: Commit**

```bash
git add web/admin-web/src/pages/orders/OrderListPage.tsx web/admin-web/src/pages/orders/OrderListPage.test.tsx
git commit -m "feat(admin-web): OrderListPage 加 selecting_escort/escort_pending_acceptance 2 状态筛选 + selected_escort_id/escort_pending_expire_at 2 列 + 详情链接"
```

---

## Task 2: OrderDetailPage 加 2 状态进度分支 + selected_escort 展示 + 30s 倒计时 + 拒接回退卡

**Files:**
- Modify: `web/admin-web/src/pages/orders/OrderDetailPage.tsx`
- Create: `web/admin-web/src/pages/orders/EscortPendingCountdown.tsx`

**Step 1: 写测试（RED）**

`web/admin-web/src/pages/orders/OrderDetailPage.test.tsx`：

```tsx
// 验证：详情页能根据订单 status 渲染对应进度分支；
// 30s 倒计时在剩余 < 60s 时红色高亮；
// 拒接回退卡展示 escort_reject_reason + 提示"患者可重新选"。
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import dayjs from 'dayjs';
import OrderDetailPage from './OrderDetailPage';

const handlers = [
  http.get('http://localhost/api/v1/admin/orders/9001', () =>
    HttpResponse.json({
      data: {
        id: 9001, status: 'selecting_escort',
        selected_escort_id: null, escort_pending_expire_at: null, escort_reject_reason: null,
        hospital_name: '协和医院', final_amount: 500, created_at: '2026-09-24T10:00:00Z',
      },
    })),
  http.get('http://localhost/api/v1/admin/orders/9002', () =>
    HttpResponse.json({
      data: {
        id: 9002, status: 'escort_pending_acceptance',
        selected_escort_id: 42,
        // 截止时间距 now 30s（用于触发红色高亮）
        escort_pending_expire_at: dayjs().add(30, 'second').toISOString(),
        escort_reject_reason: null,
        hospital_name: '同仁医院', final_amount: 800, created_at: '2026-09-24T10:01:00Z',
      },
    })),
  http.get('http://localhost/api/v1/admin/orders/9003', () =>
    HttpResponse.json({
      data: {
        id: 9003, status: 'selecting_escort',
        selected_escort_id: 42,
        escort_pending_expire_at: null,
        escort_reject_reason: 'escort_declined',  // 拒接回退
        hospital_name: '301医院', final_amount: 600, created_at: '2026-09-24T10:02:00Z',
      },
    })),
];
const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function renderDetail(orderId: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/orders/${orderId}`]}>
        <Routes>
          <Route path="/orders/:id" element={<OrderDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('OrderDetailPage v2 选人模式分支', () => {
  it('selecting_escort 状态展示"待患者选人"步骤', async () => {
    renderDetail('9001');
    await waitFor(() => expect(screen.getByText(/订单 9001/)).toBeInTheDocument());
    expect(screen.getByText(/待患者选人/)).toBeInTheDocument();
  });

  it('escort_pending_acceptance 展示 30s 倒计时 + 已选陪诊师 ID', async () => {
    renderDetail('9002');
    await waitFor(() => expect(screen.getByText(/订单 9002/)).toBeInTheDocument());
    expect(screen.getByText(/已选陪诊师/)).toBeInTheDocument();
    expect(screen.getByText(/#42/)).toBeInTheDocument();
    // 30s 倒计时剩余 < 60s → 红色高亮（color: '#ff4d4f'）
    const countdown = await screen.findByText(/^\d+s$/);
    expect(countdown.style.color).toBe('rgb(255, 77, 79)');
  });

  it('拒接回退：展示 escort_reject_reason + 提示"可重新选"', async () => {
    renderDetail('9003');
    await waitFor(() => expect(screen.getByText(/订单 9003/)).toBeInTheDocument());
    expect(screen.getByText(/拒接原因/)).toBeInTheDocument();
    expect(screen.getByText(/陪诊师拒接/)).toBeInTheDocument();
    expect(screen.getByText(/患者可重新选/)).toBeInTheDocument();
  });
});
```

**Step 2: 跑测试确认失败**

Run:
```bash
pnpm --filter admin-web exec vitest run OrderDetailPage 2>&1 | tail -30
```

Expected: FAIL — 找不到 "待患者选人" 步骤 / 找不到 30s 倒计时 / 找不到 "拒接原因"

**Step 3: 写 EscortPendingCountdown 组件**

`web/admin-web/src/pages/orders/EscortPendingCountdown.tsx`：

```tsx
// v2：30s 倒计时组件。剩余 < 60s 红色高亮。
// 用 useEffect + setInterval 1s tick；卸载 clearInterval。
import { useEffect, useState } from 'react';
import dayjs from 'dayjs';

interface Props {
  expireAt: string;  // ISO datetime
}

export function EscortPendingCountdown({ expireAt }: Props) {
  const [remaining, setRemaining] = useState(() =>
    Math.max(0, dayjs(expireAt).diff(dayjs(), 'second')));

  useEffect(() => {
    const tick = setInterval(() => {
      const r = Math.max(0, dayjs(expireAt).diff(dayjs(), 'second'));
      setRemaining(r);
      if (r === 0) clearInterval(tick);
    }, 1000);
    return () => clearInterval(tick);
  }, [expireAt]);

  const isCritical = remaining < 60;
  const color = isCritical ? '#ff4d4f' : '#1677ff';

  return (
    <span style={{ color, fontWeight: 600 }} data-testid="escort-countdown">
      {remaining}s
    </span>
  );
}
```

**Step 4: 改 OrderDetailPage**

`web/admin-web/src/pages/orders/OrderDetailPage.tsx`：

```tsx
// v2：状态机进度条加 2 状态分支（selecting_escort / escort_pending_acceptance）；
// 展示 selected_escort_id + 30s 倒计时 + 拒接回退卡。
import { useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Steps, Descriptions, Card as AntCard, Alert, Tag, Spin, Typography } from 'antd';
import dayjs from 'dayjs';
import { StatusBadge } from '@/components/StatusBadge/StatusBadge';
import { EscortPendingCountdown } from './EscortPendingCountdown';
import { fetchOrderDetail } from '@/api/admin/orders';
import type { OrderDetail } from '@/types/generated';

const { Title } = Typography;

// v2 新增：进度分支步骤
const STEPS = ['paid', 'selecting_escort', 'escort_pending_acceptance', 'accepted', 'in_service', 'completed'];
const STEP_LABEL: Record<string, string> = {
  paid: '已支付', selecting_escort: '待患者选人',
  escort_pending_acceptance: '待陪诊师确认', accepted: '已接单',
  in_service: '服务中', completed: '已完成',
};

function currentStepIndex(status: string): number {
  // 拒接回退 selecting_escort 时：进度跳回 index 1
  if (status === 'selecting_escort') return 1;
  return STEPS.indexOf(status);
}

export default function OrderDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const { data, isLoading } = useQuery({
    queryKey: ['order', id],
    queryFn: () => fetchOrderDetail(Number(id)),
  });

  if (isLoading) return <Spin />;
  if (!data) return <Alert type="error" message="订单不存在" />;

  const order: OrderDetail = data;
  const stepIdx = currentStepIndex(order.status);

  return (
    <div>
      <Title level={3}>订单 {order.id}</Title>

      {/* 状态机进度条：v2 加 2 状态分支 */}
      <Steps
        current={stepIdx >= 0 ? stepIdx : 0}
        size="small"
        items={STEPS.map((s) => ({ title: STEP_LABEL[s] ?? s }))}
      />

      <AntCard style={{ marginTop: 16 }} title="基础信息">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="订单号">{order.id}</Descriptions.Item>
          <Descriptions.Item label="状态"><StatusBadge status={order.status} /></Descriptions.Item>
          <Descriptions.Item label="医院">{order.hospital_name}</Descriptions.Item>
          <Descriptions.Item label="金额">¥{order.final_amount}</Descriptions.Item>
          <Descriptions.Item label="下单时间">
            {dayjs(order.created_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>

          {/* v2 新增字段：已选陪诊师 */}
          <Descriptions.Item label="已选陪诊师">
            {order.selected_escort_id
              ? <Tag color="blue">#{order.selected_escort_id}</Tag>
              : <Tag>未选</Tag>}
          </Descriptions.Item>

          {/* v2 新增字段：30s 倒计时（仅 escort_pending_acceptance 展示） */}
          <Descriptions.Item label="确认截止">
            {order.escort_pending_expire_at && order.status === 'escort_pending_acceptance'
              ? <>
                  <EscortPendingCountdown expireAt={order.escort_pending_expire_at} />
                  <span style={{ marginLeft: 8, color: '#999' }}>
                    ({dayjs(order.escort_pending_expire_at).format('HH:mm:ss')})
                  </span>
                </>
              : '—'}
          </Descriptions.Item>
        </Descriptions>
      </AntCard>

      {/* v2 新增：拒接回退卡（仅 escort_reject_reason 非空时展示） */}
      {order.escort_reject_reason && (
        <Alert
          style={{ marginTop: 16 }}
          type="warning"
          showIcon
          message="陪诊师拒接，订单已回退"
          description={
            <>
              <div><strong>拒接原因：</strong>
                {order.escort_reject_reason === 'escort_declined' && '陪诊师主动拒接'}
                {order.escort_reject_reason === 'lock_expired' && '陪诊师超时未确认'}
              </div>
              <div style={{ marginTop: 8 }}>
                患者可在 miniapp 端<Link to="/orders?status=selecting_escort"> 重新选择其他陪诊师</Link>
              </div>
            </>
          }
        />
      )}
    </div>
  );
}
```

**Step 5: 跑测试确认通过（GREEN）**

Run:
```bash
pnpm --filter admin-web exec vitest run OrderDetailPage EscortPendingCountdown 2>&1 | tail -20
```

Expected: PASS — `3 passed (OrderDetailPage)`

**Step 6: 类型检查 + Lint**

```bash
pnpm --filter admin-web run typecheck 2>&1 | tail -10
pnpm --filter admin-web run lint 2>&1 | tail -10
```

Expected: 0 errors / 0 warnings

**Step 7: Commit**

```bash
git add web/admin-web/src/pages/orders/OrderDetailPage.tsx \
        web/admin-web/src/pages/orders/EscortPendingCountdown.tsx \
        web/admin-web/src/pages/orders/OrderDetailPage.test.tsx
git commit -m "feat(admin-web): OrderDetailPage 加 selecting_escort/escort_pending_acceptance 状态分支 + selected_escort 展示 + 30s 倒计时 + 拒接回退卡"
```

---

## Task 3: DashboardPage 加 2 个待确认 Statistic 卡片（待患者选人 + 待陪诊师确认）

**Files:**
- Modify: `web/admin-web/src/pages/dashboard/DashboardPage.tsx`

**Step 1: 写测试（RED）**

`web/admin-web/src/pages/dashboard/DashboardPage.test.tsx`：

```tsx
// 验证：DashboardPage 渲染 2 个新卡片（待患者选人 + 待陪诊师确认）；
// 点击卡片跳 /orders?status=selecting_escort 或 escort_pending_acceptance。
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import DashboardPage from './DashboardPage';

const handlers = [
  http.get('http://localhost/api/v1/admin/reports/overview', () =>
    HttpResponse.json({
      data: {
        today_orders: 12, today_gmv: 8600, pending_escorts: 3, pending_refunds: 2,
        // v2 新增 2 指标
        pending_selecting_escort: 5,
        pending_escort_acceptance: 2,
      },
    })),
];
const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function renderDashboard(initialPath = '/dashboard') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/orders" element={<div data-testid="orders-page">orders</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('DashboardPage v2 待确认卡片', () => {
  it('renders 2 个新卡片：待患者选人 + 待陪诊师确认', async () => {
    renderDashboard();
    await waitFor(() => expect(screen.getByText(/待患者选人/)).toBeInTheDocument());
    expect(screen.getByText(/待陪诊师确认/)).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();  // pending_selecting_escort
    expect(screen.getByText('2')).toBeInTheDocument();  // pending_escort_acceptance
  });

  it('点击"待患者选人"卡片跳 /orders?status=selecting_escort', async () => {
    const user = userEvent.setup();
    renderDashboard();
    await waitFor(() => screen.getByText(/待患者选人/));
    await user.click(screen.getByText(/待患者选人/));
    expect(await screen.findByTestId('orders-page')).toBeInTheDocument();
    // URL search params 校验
    expect(window.location.search).toContain('status=selecting_escort');
  });
});
```

**Step 2: 跑测试确认失败**

Run:
```bash
pnpm --filter admin-web exec vitest run DashboardPage 2>&1 | tail -30
```

Expected: FAIL — 找不到 "待患者选人" / "待陪诊师确认" 卡片

**Step 3: 改 DashboardPage**

`web/admin-web/src/pages/dashboard/DashboardPage.tsx`：

```tsx
// v2：加 2 个待确认 Statistic 卡片，点击跳 /orders 带 status filter。
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Card, Col, Row, Statistic } from 'antd';
import { fetchOverview } from '@/api/admin/reports';
import type { OverviewReport } from '@/types/generated';

export default function DashboardPage() {
  const navigate = useNavigate();
  const { data: stats } = useQuery<OverviewReport>({
    queryKey: ['dashboard-overview'],
    queryFn: fetchOverview,
    refetchInterval: 30_000,
  });

  if (!stats) return null;

  return (
    <div data-testid="dashboard">
      <Row gutter={16}>
        {/* v1 既有 4 个卡片 */}
        <Col span={6}>
          <Card><Statistic title="今日订单" value={stats.today_orders} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="今日 GMV" value={stats.today_gmv} prefix="¥" /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="待审陪诊师" value={stats.pending_escorts} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="待审退款" value={stats.pending_refunds} /></Card>
        </Col>

        {/* v2 新增 2 个待确认卡片（橙色 / 蓝色 + 可点击跳转） */}
        <Col span={6} style={{ marginTop: 16 }}>
          <Card hoverable onClick={() => navigate('/orders?status=selecting_escort')}>
            <Statistic
              title="待患者选人"
              value={stats.pending_selecting_escort ?? 0}
              valueStyle={{ color: '#fa8c16' }}  // 橙色 = selecting_escort
            />
          </Card>
        </Col>
        <Col span={6} style={{ marginTop: 16 }}>
          <Card hoverable onClick={() => navigate('/orders?status=escort_pending_acceptance')}>
            <Statistic
              title="待陪诊师确认"
              value={stats.pending_escort_acceptance ?? 0}
              valueStyle={{ color: '#1677ff' }}  // 蓝色 = escort_pending_acceptance
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
```

**Step 4: 跑测试确认通过（GREEN）**

Run:
```bash
pnpm --filter admin-web exec vitest run DashboardPage 2>&1 | tail -20
```

Expected: PASS — `2 passed (DashboardPage)`

**Step 5: 类型检查 + Lint**

```bash
pnpm --filter admin-web run typecheck 2>&1 | tail -10
pnpm --filter admin-web run lint 2>&1 | tail -10
```

Expected: 0 errors / 0 warnings

**Step 6: Commit**

```bash
git add web/admin-web/src/pages/dashboard/DashboardPage.tsx \
        web/admin-web/src/pages/dashboard/DashboardPage.test.tsx
git commit -m "feat(admin-web): DashboardPage 加待患者选人/待陪诊师确认 2 Statistic 卡片（点击跳列表带 status filter）"
```

---

## Task 4: StatusBadge 加 2 状态色（橙色 `selecting_escort` + 蓝色 `escort_pending_acceptance`）

**Files:**
- Modify: `web/admin-web/src/components/StatusBadge/StatusBadge.tsx`
- Modify: `web/admin-web/src/components/StatusBadge/StatusBadge.test.tsx`

**Step 1: 写测试（RED）**

`web/admin-web/src/components/StatusBadge/StatusBadge.test.tsx`（v1 文件追加 2 个 it）：

```tsx
// v2 追加：验证 selecting_escort 渲染橙色 Badge +
//          escort_pending_acceptance 渲染蓝色 Badge。
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { StatusBadge } from './StatusBadge';

describe('StatusBadge v2 新状态色', () => {
  it('selecting_escort 渲染橙色 Badge', () => {
    render(<StatusBadge status="selecting_escort" />);
    const badge = screen.getByText('待患者选人');
    // AntD Badge.status="warning" → 橙底白字
    expect(badge.closest('.ant-badge-status-warning')).not.toBeNull();
  });

  it('escort_pending_acceptance 渲染蓝色 Badge', () => {
    render(<StatusBadge status="escort_pending_acceptance" />);
    const badge = screen.getByText('待陪诊师确认');
    // AntD Badge.status="processing" → 蓝底白字
    expect(badge.closest('.ant-badge-status-processing')).not.toBeNull();
  });
});
```

**Step 2: 跑测试确认失败**

Run:
```bash
pnpm --filter admin-web exec vitest run StatusBadge 2>&1 | tail -20
```

Expected: FAIL — 找不到 "待患者选人" / "待陪诊师确认" text（v1 StatusBadge 不含这 2 状态）

**Step 3: 改 StatusBadge**

`web/admin-web/src/components/StatusBadge/StatusBadge.tsx`：

```tsx
// v2：加 selecting_escort (橙色 Badge) + escort_pending_acceptance (蓝色 Badge)。
// 保留 v1 11 个状态映射不变。
import { Badge } from 'antd';
import type { OrderStatus } from '@/types/generated';

const STATUS_MAP: Record<OrderStatus, { text: string; color: string }> = {
  created:                       { text: '已创建',       color: 'default' },
  paid:                          { text: '已支付',       color: 'processing' },
  matching:                      { text: '匹配中',       color: 'processing' },
  // v2 新增 ↓
  selecting_escort:              { text: '待患者选人',   color: 'warning' },
  escort_pending_acceptance:     { text: '待陪诊师确认', color: 'processing' },
  // v2 新增 ↑
  accepted:                      { text: '已接单',       color: 'success' },
  in_service:                    { text: '服务中',       color: 'success' },
  completed:                     { text: '已完成',       color: 'success' },
  reviewed:                      { text: '已评价',       color: 'default' },
  refunding:                     { text: '退款中',       color: 'warning' },
  refunded:                      { text: '已退款',       color: 'default' },
  settling:                      { text: '结算中',       color: 'processing' },
  disputed:                      { text: '申诉中',       color: 'error' },
  closed:                        { text: '已关闭',       color: 'default' },
  canceled:                      { text: '已取消',       color: 'default' },
};

export function StatusBadge({ status }: { status: string }) {
  const meta = STATUS_MAP[status as OrderStatus] ?? { text: status, color: 'default' };
  return <Badge status={meta.color} text={meta.text} />;
}
```

**Step 4: 跑测试确认通过（GREEN）**

Run:
```bash
pnpm --filter admin-web exec vitest run StatusBadge 2>&1 | tail -20
```

Expected: PASS — `2 passed (StatusBadge)`（v1 既有的 2 个 it 仍通过；v2 追加 2 个新 it 也通过）

**Step 5: 类型检查 + Lint**

```bash
pnpm --filter admin-web run typecheck 2>&1 | tail -10
pnpm --filter admin-web run lint 2>&1 | tail -10
```

Expected: 0 errors / 0 warnings

**Step 6: Commit**

```bash
git add web/admin-web/src/components/StatusBadge/StatusBadge.tsx \
        web/admin-web/src/components/StatusBadge/StatusBadge.test.tsx
git commit -m "feat(admin-web): StatusBadge 加 selecting_escort(橙) + escort_pending_acceptance(蓝) 2 状态色"
```

---

## Task 5: MSW handlers + OpenAPI contracts.yaml 加 fixture + 3 字段（selected_escort_id / escort_pending_expire_at / escort_reject_reason）

**Files:**
- Modify: `web/admin-web/src/mocks/data/seed.ts`
- Modify: `web/admin-web/src/mocks/handlers/admin/orders.ts`
- Modify: `web/openapi/contracts.yaml`
- Regenerate: `web/admin-web/src/types/generated.ts`（脚本自动产出，不手写）

**Step 1: 写测试（RED）**

`web/admin-web/src/mocks/handlers/admin/orders.test.ts`（v1 文件追加 1 个 it）：

```ts
// v2 追加：MSW handler 返回 selecting_escort / escort_pending_acceptance 2 状态订单，
//          字段含 selected_escort_id / escort_pending_expire_at / escort_reject_reason。
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '../handlers';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterAll(() => server.close());

describe('MSW orders handler v2', () => {
  it('列表过滤 selecting_escort 返回含 selected_escort_id=null 的订单', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/orders?status=selecting_escort');
    const json = await resp.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].status).toBe('selecting_escort');
    expect(json.data[0]).toHaveProperty('selected_escort_id');
    expect(json.data[0]).toHaveProperty('escort_pending_expire_at');
  });

  it('详情 9002 返回 escort_pending_acceptance 含 escort_pending_expire_at 字段', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/orders/9002');
    const json = await resp.json();
    expect(json.data.status).toBe('escort_pending_acceptance');
    expect(json.data.selected_escort_id).toBe(42);
    expect(json.data.escort_pending_expire_at).toBeTruthy();
  });
});
```

**Step 2: 跑测试确认失败**

Run:
```bash
pnpm --filter admin-web exec vitest run mocks/handlers/admin/orders 2>&1 | tail -20
```

Expected: FAIL — seed.ts 无 id=9001/9002 fixture；handler 返回的 mock 订单无新字段

**Step 3: 改 seed.ts（加 2 条 fixture）**

`web/admin-web/src/mocks/data/seed.ts`（追加到 `mockOrders` 数组）：

```ts
// v2 新增 fixture：覆盖 selecting_escort + escort_pending_acceptance 2 状态
mockOrders.push(
  {
    id: 9001,
    status: 'selecting_escort',     // 待患者选人
    selected_escort_id: null,       // 未选
    escort_pending_expire_at: null,
    escort_reject_reason: null,
    hospital_name: '协和医院',
    final_amount: 500,
    created_at: '2026-09-24T10:00:00Z',
    patient_name: '张三', patient_phone: '138****0001',
    package_name: '半日陪诊',
  },
  {
    id: 9002,
    status: 'escort_pending_acceptance',  // 待陪诊师确认
    selected_escort_id: 42,               // 已选陪诊师
    escort_pending_expire_at: '2026-09-24T10:30:30Z',  // 30s 截止
    escort_reject_reason: null,
    hospital_name: '同仁医院',
    final_amount: 800,
    created_at: '2026-09-24T10:01:00Z',
    patient_name: '李四', patient_phone: '138****0002',
    package_name: '全日陪诊',
  },
  // 拒接回退 fixture（用于详情页拒接卡测试）
  {
    id: 9003,
    status: 'selecting_escort',     // 已回退到 selecting
    selected_escort_id: 42,
    escort_pending_expire_at: null,
    escort_reject_reason: 'escort_declined',  // 陪诊师拒接
    hospital_name: '301医院',
    final_amount: 600,
    created_at: '2026-09-24T10:02:00Z',
    patient_name: '王五', patient_phone: '138****0003',
    package_name: '半日陪诊',
  },
);
```

**Step 4: 改 orders.ts handler**

`web/admin-web/src/mocks/handlers/admin/orders.ts`：

```ts
// v2：get-by-id 返回 3 个新字段（selected_escort_id / escort_pending_expire_at / escort_reject_reason）；
//      列表过滤支持 selecting_escort / escort_pending_acceptance 2 状态。
import { http, HttpResponse } from 'msw';
import { mockOrders } from '../data/seed';

export const orderHandlers = [
  // 列表
  http.get('/api/v1/admin/orders', ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const filtered = status
      ? mockOrders.filter((o) => o.status === status)
      : mockOrders;
    return HttpResponse.json({ data: filtered, total: filtered.length });
  }),

  // 详情（v2 加 3 字段透传）
  http.get('/api/v1/admin/orders/:id', ({ params }) => {
    const order = mockOrders.find((o) => o.id === Number(params.id));
    if (!order) return HttpResponse.json({ error: 'not found' }, { status: 404 });
    return HttpResponse.json({
      data: {
        ...order,
        selected_escort_id: order.selected_escort_id,
        escort_pending_expire_at: order.escort_pending_expire_at,
        escort_reject_reason: order.escort_reject_reason,
      },
    });
  }),

  // 强制取消（v1 既有，不动）
  http.post('/api/v1/admin/orders/:id/force-cancel', async ({ params, request }) => {
    const body = await request.json() as { reason: string };
    return HttpResponse.json({ data: { id: Number(params.id), status: 'canceled', reason: body.reason } });
  }),
];
```

**Step 5: 改 contracts.yaml（加 3 字段）**

`web/openapi/contracts.yaml`（在 `OrderDetail` schema 的 `properties` 块末尾追加）：

```yaml
components:
  schemas:
    OrderDetail:
      type: object
      properties:
        id:
          type: integer
          format: int64
        status:
          type: string
          enum: [created, paid, matching, selecting_escort, escort_pending_acceptance,
                 accepted, in_service, completed, reviewed, refunding, refunded,
                 settling, disputed, closed, canceled]
        hospital_name:
          type: string
        final_amount:
          type: number
        created_at:
          type: string
          format: date-time
        # v2 新增 ↓
        selected_escort_id:
          type: integer
          format: int64
          nullable: true
          description: 已选陪诊师 user_id（escort_pending_acceptance / accepted 时必填）
        escort_pending_expire_at:
          type: string
          format: date-time
          nullable: true
          description: 陪诊师 30s 确认截止时间（仅 escort_pending_acceptance 状态时非空）
        escort_reject_reason:
          type: string
          enum: [escort_declined, lock_expired, null]
          nullable: true
          description: 拒接原因（订单从 escort_pending_acceptance 回退 selecting_escort 时记录）
        # v2 新增 ↑
      required: [id, status, hospital_name, final_amount, created_at]
```

同时更新 `OverviewReport` schema：

```yaml
    OverviewReport:
      type: object
      properties:
        today_orders:        { type: integer }
        today_gmv:           { type: number }
        pending_escorts:     { type: integer }
        pending_refunds:     { type: integer }
        # v2 新增 ↓
        pending_selecting_escort:
          type: integer
          description: 等待患者选人的订单数（status=selecting_escort）
        pending_escort_acceptance:
          type: integer
          description: 等待陪诊师确认的订单数（status=escort_pending_acceptance）
        # v2 新增 ↑
```

**Step 6: 重生成 types**

Run:
```bash
pnpm --filter admin-web run generate:client 2>&1 | tail -10
```

Expected: 生成 `src/types/generated.ts`，包含 `selected_escort_id` / `escort_pending_expire_at` / `escort_reject_reason` 字段 + `OverviewReport.pending_selecting_escort` / `pending_escort_acceptance` 字段

**Step 7: 跑测试确认通过（GREEN）**

Run:
```bash
pnpm --filter admin-web exec vitest run mocks/handlers/admin/orders 2>&1 | tail -20
```

Expected: PASS — `2 passed`

**Step 8: 类型检查 + Lint**

```bash
pnpm --filter admin-web run typecheck 2>&1 | tail -10
pnpm --filter admin-web run lint 2>&1 | tail -10
```

Expected: 0 errors / 0 warnings（generated.ts 重生成后所有引用点类型一致）

**Step 9: Commit**

```bash
git add web/admin-web/src/mocks/data/seed.ts \
        web/admin-web/src/mocks/handlers/admin/orders.ts \
        web/admin-web/src/mocks/handlers/admin/orders.test.ts \
        web/admin-web/src/types/generated.ts \
        web/openapi/contracts.yaml
git commit -m "feat(admin-web): MSW orders handler 加 selecting_escort/escort_pending_acceptance 2 fixture + OpenAPI OrderDetail 加 3 字段 + 重生成 types"
```

---

## Task 6: 全量回归（vitest + typecheck + lint + RTL 覆盖 2 新状态分支）

**Files:**
- Modify: 无（仅跑测试 + 验证）

**Step 1: 全量单测**

Run:
```bash
pnpm --filter admin-web exec vitest run --reporter=verbose --coverage 2>&1 | tail -80
```

Expected: ALL PASS；覆盖率 ≥ 70%（v2 增量代码覆盖率 ≥ 70%；v1 不降）

**Step 2: 跑 2 个新状态分支的专门测试**

Run:
```bash
pnpm --filter admin-web exec vitest run OrderListPage OrderDetailPage DashboardPage StatusBadge 2>&1 | tail -30
```

Expected: `9 passed`（v2 增量 2 + 2 + 2 + 2 = 8 个新 it + v1 StatusBadge 既有 2 个 = 10；此处只显示新增的 8 个）

**Step 3: 类型检查**

Run:
```bash
pnpm --filter admin-web run typecheck 2>&1 | tail -10
```

Expected: 0 errors

**Step 4: Lint**

Run:
```bash
pnpm --filter admin-web run lint 2>&1 | tail -10
```

Expected: 0 warnings / 0 errors

**Step 5: 验证 contracts.yaml 字段一致性**

Run:
```bash
grep -E "selected_escort_id|escort_pending_expire_at|escort_reject_reason|selecting_escort|escort_pending_acceptance" \
  /Users/growduduan/ai/doctors/web/openapi/contracts.yaml | head -20
```

Expected: 8 行以上匹配（3 字段在 OrderDetail + 2 状态在 enum + 2 状态在 OverviewReport）

**Step 6: 验证 generated.ts 自动包含新字段**

Run:
```bash
grep -E "selected_escort_id|escort_pending_expire_at|escort_reject_reason" \
  /Users/growduduan/ai/doctors/web/admin-web/src/types/generated.ts | head -10
```

Expected: 至少 3 行匹配（3 字段类型定义）

**Step 7: E2E（v1 既有的 4 个 Playwright 用例 + 新增 1 个选人模式用例）**

`web/admin-web/e2e/selecting-escort.spec.ts`（新建）：

```ts
// E2E：登录 → 看板 → 点击"待患者选人"卡片 → 列表筛选 → 点详情 → 看到 30s 倒计时
import { test, expect } from '@playwright/test';

test('admin 能从看板跳到 selecting_escort 列表 → 详情看到 30s 倒计时', async ({ page }) => {
  await page.goto('/login');
  await page.fill('input[name="username"]', 'admin');
  await page.fill('input[name="password"]', 'admin123');
  await page.click('button:has-text("登录")');

  await page.goto('/dashboard');
  await expect(page.getByText(/待患者选人/)).toBeVisible();

  await page.getByText(/待患者选人/).first().click();
  await expect(page).toHaveURL(/\/orders\?status=selecting_escort/);

  // 列表中 id=9001 行 → 详情链接
  await page.getByText('9001').first().click();
  await expect(page).toHaveURL(/\/orders\/9001/);
  await expect(page.getByText(/待患者选人/)).toBeVisible();
});
```

Run:
```bash
pnpm --filter admin-web run test:e2e -- e2e/selecting-escort.spec.ts 2>&1 | tail -20
```

Expected: PASS — `1 passed (selecting-escort.spec.ts)`

**Step 8: 全量 E2E（v1 + v2）**

Run:
```bash
pnpm --filter admin-web run test:e2e 2>&1 | tail -20
```

Expected: ALL PASS（v1 4 + v2 1 = 5 个）

**Step 9: 修改文档 + dev.md**

`docs/04-业务流程.md` §4.8 末尾追加：

```markdown
#### 4.8.1 admin-web v2 选人模式适配（2026-09-24）

admin-web 在 v1 骨架基础上适配订单匹配模式从「抢单」改为「选人」：

- **OrderListPage**: 状态筛选新增 selecting_escort（橙）/ escort_pending_acceptance（蓝）两选项；
  新增列：已选陪诊师 ID + 确认截止时间
- **OrderDetailPage**: 状态机进度条加 2 状态分支；
  新增 EscortPendingCountdown 组件（剩余 < 60s 红色高亮）；
  新增拒接回退卡（展示 escort_reject_reason + 提示「可重新选」）
- **DashboardPage**: 加 2 个 Statistic 卡片（待患者选人 / 待陪诊师确认），点击跳 OrderList 带 status filter
- **StatusBadge**: 加 selecting_escort（橙色 `colorWarning`）/ escort_pending_acceptance（蓝色 `colorProcessing`）2 状态色
- **MSW handlers**: orders 列表 mock 加 3 条 fixture（id=9001/9002/9003），覆盖 selecting/escort_pending/拒接回退 3 场景
- **OpenAPI contracts.yaml**: OrderDetail schema 加 selected_escort_id / escort_pending_expire_at / escort_reject_reason 3 字段；
  OverviewReport 加 pending_selecting_escort / pending_escort_acceptance 2 指标
```

`dev.md` §10.15 追加：

```markdown
### 10.15 admin-web v2 选人模式适配 plan（2026-09-24）

admin-web v2 增量适配订单匹配模式从「抢单」改为「选人」（基于 spec `2026-09-24-order-matching-redesign.md` §5）。

**落地 commits（6 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(admin-web) | OrderListPage 加 2 状态筛选 + 2 列 + 详情链接 |
| feat(admin-web) | OrderDetailPage 加 2 状态进度分支 + selected_escort 展示 + 30s 倒计时 + 拒接回退卡 |
| feat(admin-web) | DashboardPage 加待患者选人/待陪诊师确认 2 卡片 |
| feat(admin-web) | StatusBadge 加 selecting_escort(橙) + escort_pending_acceptance(蓝) 2 状态色 |
| feat(admin-web) | MSW orders handler 加 2 fixture + OpenAPI OrderDetail 加 3 字段 + 重生成 types |
| test(admin-web) | 全量回归（vitest + RTL + typecheck + lint + E2E 5 个） |

**新增/修改文件**：6 个 TSX/TS（含 1 新建组件）+ 3 个测试文件 + 1 个 OpenAPI schema + 1 个 E2E + 2 个文档

**测试覆盖**：v2 增量代码 ≥ 70%；RTL 覆盖 2 新状态分支 + 30s 倒计时 < 60s 红色高亮 + 拒接回退卡

**未做**：WebSocket 实时推送（沿用 v1 5s 轮询）/ 暗色主题 / i18n / 移动端响应式（v3+ 增量）
```

**Step 10: 全量回归 + push**

Run:
```bash
pnpm --filter admin-web exec vitest run --coverage 2>&1 | tail -30
pnpm --filter admin-web run typecheck
pnpm --filter admin-web run lint
pnpm --filter admin-web run test:e2e
pnpm --filter admin-web run build
ls -lh web/admin-web/dist/
pnpm --filter '*' run typecheck 2>&1 | tail -10
```

Expected: 全部 PASS；dist/ 产出；typecheck 0 errors；lint 0 warnings；unit ≥ 70% 覆盖；E2E 5/5 PASS

```bash
git push -u origin main
```

**Step 11: Commit docs**

```bash
git add docs/04-业务流程.md dev.md
git commit -m "docs(admin-web): 04 §4.8.1 admin-web v2 选人模式适配 + dev.md §10.15 落地记录"
```

---

## Self-Review

- ✅ **Spec 覆盖**: 6 个修订要点全覆盖
  - OrderListPage 状态筛选 + 2 列 + 详情链接（Task 1）
  - OrderDetailPage 2 状态进度分支 + selected_escort + 30s 倒计时 + 拒接回退（Task 2）
  - DashboardPage 2 待确认卡片（Task 3）
  - StatusBadge 2 状态色（Task 4）
  - MSW handlers fixture + OpenAPI 3 字段（Task 5）
  - 全量回归 + 文档（Task 6）
- ✅ **无占位符**: 每个 Task 含完整 TS/TSX 代码（无 Lorem/Detail/TBD）+ 测试代码 + Run/Expected + commit 命令
- ✅ **TDD 严格**: 每个 Task Step 1 写测试 → Step 2 跑 RED → Step 3+ 实现 → Step 4 跑 GREEN → Step N commit
- ✅ **类型一致**: 与 spec §5 admin 端影响 + admin-web spec §3.1 页面表 + contracts.yaml 字段定义对齐
- ✅ **测试矩阵**: §1-6 覆盖 Vitest 单测 / RTL 组件 / MSW handler / Playwright E2E / contracts.yaml 字段 grep / generated.ts grep
- ✅ **YAGNI**: §Global Constraints 明确不引第三方倒计时库 + 不引新色板 + 不做 WebSocket + 不做暗色主题
- ✅ **包管理**: pnpm workspace 沿用 v1；不引新依赖
- ✅ **接口稳定**: contracts.yaml 增量向后兼容（3 字段 nullable；OverviewReport 2 字段 optional `?? 0` 兜底）
- ✅ **RBAC 设计**: 沿用 v1；本 v2 不新增权限点（order:read 仍足够）
- ✅ **OpenAPI 集成**: 修改 contracts.yaml → 自动重生成 generated.ts（不手写 schema 类型）
- ✅ **Mock 完整**: MSW handler 列表 + 详情都返回 3 字段；fixture id=9001/9002/9003 覆盖 3 场景（selecting/escort_pending/拒接回退）

## 关键设计偏差（待用户确认）

1. **30s 倒计时**: 用原生 `useEffect` + `setInterval` 1s tick（不引第三方倒计时库）；剩余 < 60s 红色高亮；组件卸载 `clearInterval` 无泄漏。如团队偏好 `react-countdown` 等库，需调整 EscortPendingCountdown 实现。
2. **Dashboard 卡片跳链**: 用 `useNavigate('/orders?status=...')` 跳转；要求 OrderListPage 从 URL `status` query 自动初始化筛选（v1 OrderListPage 不读 URL params；v2 plan 增量实现）。
3. **MSW fixture 复用**: seed.ts 用 `mockOrders.push(...)` 追加而非覆盖（v1 的 10 条 fixture 保留），保证列表分页 + 全状态筛选 E2E 不破坏。
4. **OA 字段 nullable 设计**: `selected_escort_id` / `escort_pending_expire_at` / `escort_reject_reason` 都标 nullable（仅 escort_pending_acceptance 时非空）；后端 Go struct 用 `*int64` + `*time.Time` + `*string` 指针类型。
5. **状态机进度条分支**: 进度条用 Steps 组件单链渲染（`STEPS` 数组 6 节点）；拒接回退 selecting_escort 时 currentStep 跳回 index 1。详细状态机由后端 order-service 权威，前端仅展示。
6. **状态色**: 复用 AntD Badge `status="warning"` (橙) / `status="processing"` (蓝)；不引新色板 token。如品牌要求"陪诊师确认"用紫色而非蓝色，需在 theme.ts 加 customToken。
7. **拒接原因语义**: `escort_reject_reason` 枚举 `escort_declined` / `lock_expired` / `null`；UI 文案映射「陪诊师主动拒接」/「陪诊师超时未确认」。其他 reason 由后端 plan 扩展。
8. **E2E 新增**: v2 新增 1 个 E2E（selecting-escort.spec.ts）；v1 既有的 4 个 E2E 不变。如需 E2E 全量覆盖 OrderDetailPage 拒接回退卡渲染，可加 e2e/escort-reject.spec.ts（v3 增量）。

## 关键交互流程（admin 视角）

```
1. admin 登录 → JWT + role claim → authStore
2. 进 /dashboard → 6 个 Statistic 卡片（v1 4 + v2 2）
3. 点"待患者选人"卡片 → 跳 /orders?status=selecting_escort
4. 列表筛选显示 selecting_escort 状态订单（含"未选"tag）
5. 点订单号 → /orders/{id}
6. 详情页：进度条停在"待患者选人"步骤
7. 若 escort 已拒接 → 顶部 Alert 卡（escort_reject_reason + 提示"可重新选"）
8. 若 escort 选了某人 → 显示 escort_id + 30s 倒计时（红色高亮 < 60s）
```