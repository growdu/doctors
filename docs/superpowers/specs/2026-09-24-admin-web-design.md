# L2 v1.0 — 管理后台 (admin-web) 设计

> **For spec reviewers:** 这是 L2 v1.0 前端的**管理后台**（Web SPA）详细设计。依赖 `l2-api-gap-design.md` 提供的后端 API 契约。

**Goal:** 内部运营工具：订单管理、陪诊师审核、退款审核、客服工单、数据报表。产出可在 Chrome / Edge 跑通的 SPA。

**Tech Stack:**
- **框架**: React 18.3 + TypeScript 5.5
- **构建**: Vite 5.4（fast HMR）
- **UI**: Ant Design 5.21（中文 + 主题定制）
- **状态管理**: Zustand 4.5（轻量 + 简单；不引 Redux）
- **数据请求**: TanStack Query 5.x（缓存 + 自动 retry + 后台刷新）
- **路由**: React Router 6.26（声明式 + 嵌套）
- **表单**: Ant Design Form（自带校验）
- **表格**: Ant Design Table + ProTable（复杂筛选 + 排序 + 分页）
- **图表**: @ant-design/charts（基于 G2Plot）
- **Mock**: MSW 2.x（开发期拦截 fetch 模拟后端）
- **测试**: Vitest（单元）+ React Testing Library（组件）+ Playwright（E2E）

**前置依赖:**
- 后端：`docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` 全部 P0 admin API
- 跨端类型：`web/openapi/contracts.yaml` → `openapi-typescript` 生成

---

## 1. 功能模块（来自 docs/03 / 09.2.3）

| # | 模块 | 状态 |
|---|---|---|
| 1 | 登录 / 权限（多角色 admin） | P0 |
| 2 | 用户管理（患者 + 陪诊师） | P0 |
| 3 | 订单管理（全量 + 状态机操作） | P0 |
| 4 | 陪诊师审核（通过 / 拒绝） | P0 |
| 5 | 退款审核 | P0 |
| 6 | 客服工单（首响 SLA 监控） | P1 |
| 7 | 钱包 / 账单（陪诊师 T+7） | P1 |
| 8 | 数据看板（GMV / 订单 / 退款率 / 投诉率） | P1 |
| 9 | 评价管理（敏感词过滤 + 隐藏） | P1 |
| 10 | 系统设置（退款策略 / 抢单池阈值） | P1 |

---

## 2. 目录结构

```
web/admin-web/
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   ├── pages/
│   │   ├── login/                       # 登录
│   │   ├── dashboard/                   # 数据看板
│   │   ├── users/
│   │   │   ├── patient_list/
│   │   │   ├── patient_detail/
│   │   │   ├── escort_list/
│   │   │   └── escort_audit/            # 待审核陪诊师
│   │   ├── orders/
│   │   │   ├── order_list/              # ProTable 筛选
│   │   │   ├── order_detail/
│   │   │   └── force_cancel_modal/
│   │   ├── refunds/
│   │   │   ├── refund_list/
│   │   │   └── refund_audit/
│   │   ├── work-orders/
│   │   │   ├── work_order_list/
│   │   │   └── work_order_detail/
│   │   ├── wallet/
│   │   │   ├── wallet_overview/
│   │   │   └── withdrawal_review/
│   │   ├── reviews/
│   │   │   └── review_moderation/
│   │   ├── reports/
│   │   │   ├── gmv_report/
│   │   │   ├── refund_rate/
│   │   │   └── complaint_rate/
│   │   └── settings/
│   │       ├── refund_policies/
│   │       └── pool_threshold/
│   ├── components/
│   │   ├── ProTable/                    # 通用 ProTable 封装
│   │   ├── StatusBadge/                 # 订单状态机颜色
│   │   ├── AuditAction/                 # 审核按钮（通过 / 拒绝）
│   │   ├── TraceId/                     # 链路 ID 展示
│   │   ├── ErrorBoundary/
│   │   └── PageHeader/
│   ├── stores/
│   │   ├── authStore.ts                 # 当前用户 + 角色 + 权限
│   │   └── uiStore.ts                   # 侧边栏 / 主题
│   ├── api/
│   │   ├── client.ts                    # fetch + interceptor
│   │   ├── auth.ts
│   │   ├── admin/
│   │   │   ├── users.ts
│   │   │   ├── orders.ts
│   │   │   ├── refunds.ts
│   │   │   ├── work-orders.ts
│   │   │   ├── escorts.ts
│   │   │   ├── wallet.ts
│   │   │   ├── reviews.ts
│   │   │   └── reports.ts
│   │   └── types.ts
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── usePermission.ts             # 权限 hook
│   │   ├── usePolling.ts                # 轮询（如退款队列）
│   │   └── useTableParams.ts
│   ├── router/
│   │   ├── routes.tsx                   # 路由表
│   │   ├── guards.tsx                   # AuthGuard + RoleGuard
│   │   └── permissions.ts               # 角色 → 权限映射
│   ├── utils/
│   │   ├── format.ts                    # 时间 / 金额 / 状态格式化
│   │   ├── trace.ts
│   │   └── download.ts                  # 导出 CSV
│   ├── mocks/
│   │   ├── browser.ts                   # MSW 启动
│   │   └── handlers/                    # mock handlers
│   ├── styles/
│   │   ├── global.scss
│   │   └── theme.ts                     # Ant Design 主题
│   └── config.ts
├── __tests__/
├── e2e/
├── public/
├── index.html
├── vite.config.ts
├── tsconfig.json
├── package.json
└── README.md
```

---

## 3. 核心页面

### 3.1 页面列表（v1.0 MVP）

| 路径 | 页面 | 说明 | 状态 |
|---|---|---|---|
| `/login` | 登录 | 账号 + 密码 + 2FA | P0 |
| `/dashboard` | 数据看板 | GMV / 订单 / 退款率 / 待审核数 | P1 |
| `/users/patients` | 患者列表 | ProTable 筛选 + 搜索 | P0 |
| `/users/patients/:id` | 患者详情 | 订单 / 钱包 / 评价 | P0 |
| `/users/escorts` | 陪诊师列表 | 状态 / 评分筛选 | P0 |
| `/users/escorts/:id` | 陪诊师详情 | 档案 / 订单 / 钱包 / 培训 | P0 |
| `/users/escorts/audit` | 陪诊师审核 | 待审核队列 | P0 |
| `/orders` | 订单列表 | 多维度筛选 + 状态机操作 | P0 |
| `/orders/:id` | 订单详情 | 状态机 + 时间线 + 事件流 | P0 |
| `/orders/:id/force-cancel` | 强制取消 | admin_cancel 触发 refund | P0 |
| `/refunds` | 退款队列 | 待审核（轮询 5s）| P0 |
| `/refunds/:id` | 退款审核 | 金额 + 策略计算预览 | P0 |
| `/work-orders` | 工单列表 | P1 | P1 |
| `/work-orders/:id` | 工单详情 | SLA 倒计时 | P1 |
| `/wallet/withdrawals` | 提现审核 | T+7 到期检查 | P1 |
| `/reviews` | 评价管理 | 敏感词过滤 + 隐藏 | P1 |
| `/reports/gmv` | GMV 报表 | 按天 / 按医院 / 按服务包 | P1 |
| `/reports/refund-rate` | 退款率 | P1 |
| `/settings/refund-policies` | 退款策略配置 | 与 refund_policies 表同步 | P1 |

### 3.2 RBAC 权限模型

```ts
// router/permissions.ts
type Role = 'super_admin' | 'order_admin' | 'refund_admin' | 'cs' | 'audit_admin' | 'viewer'

const PERMISSIONS: Record<Role, Permission[]> = {
  super_admin: ['*'],
  order_admin: ['order:read', 'order:force_cancel', 'refund:read'],
  refund_admin: ['refund:read', 'refund:approve', 'refund:reject'],
  audit_admin: ['escort:audit'],
  cs: ['work_order:read', 'work_order:reply', 'user:read'],
  viewer: ['*:read'],
}

// 后端 token 内含 role 字段；前端再细分 permission 集合。
```

### 3.3 路由守卫

```tsx
// router/guards.tsx
export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { token } = useAuth();
  const location = useLocation();
  if (!token) return <Navigate to="/login" state={{ from: location }} replace />;
  return <>{children}</>;
}

export function RequirePermission({ perm, children }: { perm: Permission; children: React.ReactNode }) {
  const { permissions } = useAuth();
  if (!hasPermission(perm, permissions)) return <Result status="403" title="无权限" />;
  return <>{children}</>;
}
```

---

## 4. 核心页面实现

### 4.1 订单列表（ProTable + 状态机操作）

```tsx
// pages/orders/order_list/index.tsx
export default function OrderListPage() {
  return (
    <ProTable<Order, { status?: string; hospital_id?: number }>
      headerTitle="订单管理"
      rowKey="id"
      columns={[
        { title: '订单号', dataIndex: 'order_no', fixed: 'left', width: 180 },
        { title: '患者', dataIndex: 'patient_nickname', width: 100 },
        { title: '陪诊师', dataIndex: 'escort_nickname', width: 100, render: (_, r) => r.escort_nickname || '-' },
        { title: '医院', dataIndex: 'hospital_name', width: 160 },
        { title: '金额', dataIndex: 'final_amount', width: 100, render: (v) => `¥${v}` },
        { title: '状态', dataIndex: 'status', width: 110, valueType: 'select',
          valueEnum: orderStatusEnum, render: (_, r) => <StatusBadge status={r.status} /> },
        { title: '下单时间', dataIndex: 'created_at', width: 160, valueType: 'dateTime' },
        { title: '操作', valueType: 'option', width: 120,
          render: (_, r) => [
            <Button key="detail" type="link" onClick={() => navigate(`/orders/${r.id}`)}>详情</Button>,
            <Button key="cancel" type="link" danger hidden={!canCancel(r.status)}
              onClick={() => openForceCancel(r)}>强制取消</Button>,
          ] },
      ]}
      request={async (params, sort, filter) => {
        return api.admin.orders.list({ ...params, ...sort, ...filter });
      }}
      search={{ labelWidth: 'auto' }}
      pagination={{ pageSize: 20 }}
      polling={5000}  // 5s 自动刷新
    />
  );
}
```

### 4.2 退款审核（含策略计算预览）

```tsx
// pages/refunds/refund_audit/index.tsx
export default function RefundAuditPage() {
  const { id } = useParams();
  const { data: refund, isLoading } = useQuery(['refund', id], () => api.admin.refunds.get(id));

  if (isLoading) return <Spin />;
  return (
    <Card>
      <Descriptions title="退款详情">
        <Descriptions.Item label="订单 ID">{refund.order_id}</Descriptions.Item>
        <Descriptions.Item label="金额">¥{refund.amount}</Descriptions.Item>
        <Descriptions.Item label="策略">{(refund.refund_percent * 100).toFixed(0)}%</Descriptions.Item>
        <Descriptions.Item label="原因">{refund.reason}</Descriptions.Item>
        <Descriptions.Item label="状态">{refund.status}</Descriptions.Item>
      </Descriptions>

      <Space>
        <Button type="primary" onClick={() => approve(refund.id)}>批准</Button>
        <Button danger onClick={() => reject(refund.id)}>拒绝</Button>
      </Space>
    </Card>
  );
}
```

### 4.3 陪诊师审核（双栏对照）

```tsx
// pages/users/escorts/audit/index.tsx
export default function EscortAuditPage() {
  return (
    <Row gutter={16}>
      <Col span={10}>
        <List
          header={<b>待审核陪诊师</b>}
          dataSource={pendingEscorts}
          renderItem={(e) => (
            <List.Item onClick={() => setSelected(e.id)} className={selected === e.id ? 'selected' : ''}>
              <List.Item.Meta
                avatar={<Avatar src={e.avatar_url} />}
                title={e.nickname}
                description={`${e.real_name} · 提交于 ${e.submitted_at}`}
              />
            </List.Item>
          )}
        />
      </Col>
      <Col span={14}>
        {selectedEscort && <EscortAuditDetail escort={selectedEscort} />}
      </Col>
    </Row>
  );
}
```

### 4.4 数据看板（图表）

```tsx
// pages/dashboard/index.tsx
export default function DashboardPage() {
  const { data: stats } = useQuery(['dashboard'], api.admin.reports.overview);

  return (
    <Row gutter={16}>
      <Col span={6}><Card><Statistic title="今日 GMV" value={stats?.today_gmv} prefix="¥" /></Card></Col>
      <Col span={6}><Card><Statistic title="今日订单" value={stats?.today_orders} /></Card></Col>
      <Col span={6}><Card><Statistic title="退款率" value={stats?.refund_rate} suffix="%" valueStyle={{ color: stats?.refund_rate > 5 ? 'red' : 'green' }} /></Card></Col>
      <Col span={6}><Card><Statistic title="待审核陪诊师" value={stats?.pending_escorts} /></Card></Col>

      <Col span={24}>
        <Card title="订单趋势（最近 30 天）">
          <Line {...orderTrendConfig(stats?.order_trend || [])} />
        </Card>
      </Col>

      <Col span={12}>
        <Card title="退款原因分布">
          <Pie {...refundReasonConfig(stats?.refund_reasons || [])} />
        </Card>
      </Col>
      <Col span={12}>
        <Card title="陪诊师活跃度">
          <Column {...escortActivityConfig(stats?.escort_activity || [])} />
        </Card>
      </Col>
    </Row>
  );
}
```

---

## 5. 状态管理

### 5.1 Auth Store（Zustand）

```ts
// stores/authStore.ts
interface AuthState {
  token: string | null
  user: AdminUser | null
  permissions: Permission[]
  login(phone: string, password: string, totp?: string): Promise<void>
  logout(): void
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem('admin-token'),
  user: null,
  permissions: [],

  async login(phone, password, totp) {
    const { data } = await api.admin.auth.login({ phone, password, totp });
    localStorage.setItem('admin-token', data.access_token);
    set({ token: data.access_token, user: data.user, permissions: data.permissions });
  },

  logout() {
    localStorage.removeItem('admin-token');
    set({ token: null, user: null, permissions: [] });
  },
}));
```

### 5.2 TanStack Query 集成

```ts
// hooks/useOrders.ts
export function useOrders(params: ListParams) {
  return useQuery({
    queryKey: ['orders', params],
    queryFn: () => api.admin.orders.list(params),
    refetchInterval: 5000, // 5s 自动刷新（订单队列）
    staleTime: 3000,
  });
}

export function useForceCancel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.admin.orders.forceCancel(id, 'admin force'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      message.success('已强制取消');
    },
  });
}
```

---

## 6. UI 设计

- **Ant Design 主题**：主色 #1677ff（蓝）/ 警告 #faad14 / 成功 #52c41a / 错误 #ff4d4f
- **布局**：经典 Admin Layout（左侧菜单 + 顶部 Header + 内容区）
- **表格**：统一用 ProTable（搜索栏 + 工具栏 + 数据 + 分页）
- **响应式**：≥ 1200px 三栏；< 1200px 折叠侧边栏
- **暗色主题**：v1.5 引入

---

## 7. 测试矩阵

| 类型 | 工具 | 覆盖 |
|---|---|---|
| 单元测试 | Vitest | utils / hooks / stores |
| 组件测试 | React Testing Library | ProTable / StatusBadge / AuditAction |
| E2E 测试 | Playwright | 登录 / 订单列表 / 退款审核 / 陪诊师审核 |
| 视觉回归 | Playwright screenshot | 关键页面 |
| Mock 后端 | MSW | 开发期无后端依赖 |

---

## 8. 构建 + CI

```bash
# 开发
npm run dev                   # http://localhost:8080
npm run mock:dev               # 同时启动 MSW

# 构建
npm run build                 # 输出 dist/

# 测试
npm run test:unit
npm run test:e2e

# 类型检查
npm run typecheck
npm run lint

# 静态导出（如需 SPA 部署到 OSS）
npm run build:static
```

CI 必跑：`typecheck` / `lint` / `test:unit` / `test:e2e` / `openapi-validate`。

---

## 9. 性能预算

| 指标 | 目标 |
|---|---|
| 首屏渲染 | < 1.5s |
| 路由切换 | < 200ms |
| 表格滚动（1000 行） | 60 FPS（虚拟滚动） |
| 包体积（gzipped） | < 500 KB |
| Lighthouse 评分 | ≥ 90 |

---

## 10. 不做 / 留 v2

| 不做 | 留给 |
|---|---|
| 暗色主题 | v1.5 |
| 移动端响应式（适配手机浏览器） | v1.5 |
| 国际化（i18n） | v3 |
| 权限可视化编辑（拖拽式） | v3 |
| 实时通知 WebSocket | v2 |
| 数据可视化大屏（独立 TV 端） | v2 |
| 自定义 SQL 查询 | v3 |

---

## Self-Review

- ✅ Spec 覆盖: 10 个功能模块 + 18 个页面 + 完整目录结构
- ✅ 无占位符: 每节都有具体代码示例 / 路径 / 数字
- ✅ 类型一致: 与 l2-api-gap-design.md 的 entity 对齐
- ✅ 测试矩阵: §7 给出 5 种测试方式
- ✅ YAGNI: §10 明确不做 / 留 v2

## 关联 spec

- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md`（已批）
- `docs/superpowers/specs/2026-09-24-patient-miniapp-design.md`（已批）
- `docs/superpowers/specs/2026-09-24-escort-app-design.md`（已批）

## L2 全部交付（4/4）

| Spec | commit | 状态 |
|---|---|---|
| l2-api-gap-design.md | `c438198` | ✅ |
| patient-miniapp-design.md | `b23f660` | ✅ |
| escort-app-design.md | `d7c7028` | ✅ |
| admin-web-design.md | （待 commit） | ✅ |

下一步：可进入 writing-plans 阶段出实施 plan（10 个后端 API plan + 3 个前端 setup plan）；或继续 L3 集成设计 / v2 路线图。