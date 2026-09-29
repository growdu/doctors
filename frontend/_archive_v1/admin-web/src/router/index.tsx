/**
 * Router 路由表（v1 完整版 + RBAC 联动）。
 *
 * v2 阶段：
 *   /              → 重定向 /dashboard
 *   /dashboard     → DashboardPage（v2 加 2 个待确认 Statistic）
 *   /escorts       → EscortsPage（v1 增量占位）
 *   /orders        → OrderListPage（v2 加 2 状态筛选 + 2 列）
 *   /orders/:id    → OrderDetailPage（v2 加 2 状态分支 + 30s 倒计时 + 拒接回退卡）
 *
 * v1 增量（本批次 Task A11）：
 *   /login                    → LoginPage           （AuthGuard 之外）
 *   /dashboard                → RequireRole(['super_admin','order_admin','refund_admin','audit_admin','cs','viewer'])  // 实际所有 admin 角色都可看
 *   /patients, /patients/:id
 *   /escorts, /escorts/:id, /escorts/audit
 *   /refunds, /refunds/:id
 *   /wallets, /wallets/:id
 *   /work-orders
 *   /reviews
 *   /messages
 *   /sos
 *   /settings                 → RequireRole(['super_admin'])
 *   /profile
 *   /audit                    → RequireRole(['super_admin','audit_admin'])
 *   /finance                  → RequireRole(['super_admin'])
 *   /reports
 *   /coupons                  → RequireRole(['super_admin','order_admin'])
 *   /hospitals                → RequireRole(['super_admin'])
 *   /packages                 → RequireRole(['super_admin','order_admin'])
 *   /dashboard-detail
 *   /orders（仍保留）
 *
 * 守卫：
 *   - <AuthGuard> 包裹 <Outlet/>：未登录跳 /login；
 *   - 部分 admin 子路由用 <RequireRole roles=[...]> 再做精细 RBAC。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.2 + §3.3
 */
import { Routes, Route, Navigate } from 'react-router-dom';
import AdminLayout from '@/layouts/AdminLayout';
import { AuthGuard, RequireRole } from './guards';

import DashboardPage from '@/pages/dashboard/DashboardPage';
import DashboardDetailPage from '@/pages/dashboard-detail/DashboardDetailPage';
import EscortsPage from '@/pages/escorts/EscortsPage';
import EscortDetailPage from '@/pages/escorts/EscortDetailPage';
import EscortAuditPage from '@/pages/escorts/EscortAuditPage';
import OrderListPage from '@/pages/orders/OrderListPage';
import OrderDetailPage from '@/pages/orders/OrderDetailPage';
import PatientsPage from '@/pages/patients/PatientsPage';
import PatientDetailPage from '@/pages/patients/PatientDetailPage';
import RefundsPage from '@/pages/refunds/RefundsPage';
import RefundDetailPage from '@/pages/refunds/RefundDetailPage';
import WalletsPage from '@/pages/wallets/WalletsPage';
import WalletDetailPage from '@/pages/wallets/WalletDetailPage';
import WorkOrdersPage from '@/pages/work-orders/WorkOrdersPage';
import ReviewsPage from '@/pages/reviews/ReviewsPage';
import MessagesPage from '@/pages/messages/MessagesPage';
import SosPage from '@/pages/sos/SosPage';
import SettingsPage from '@/pages/settings/SettingsPage';
import ProfilePage from '@/pages/profile/ProfilePage';
import LoginPage from '@/pages/login/LoginPage';
import AuditPage from '@/pages/audit/AuditPage';
import FinancePage from '@/pages/finance/FinancePage';
import ReportsPage from '@/pages/reports/ReportsPage';
import CouponsPage from '@/pages/coupons/CouponsPage';
import HospitalsPage from '@/pages/hospitals/HospitalsPage';
import PackagesPage from '@/pages/packages/PackagesPage';

const ALL_ROLES = ['super_admin', 'order_admin', 'refund_admin', 'audit_admin', 'cs', 'viewer'] as const;
const ADMIN_ONLY = ['super_admin'] as const;
const ADMIN_OR_ORDER = ['super_admin', 'order_admin'] as const;
const ADMIN_OR_AUDIT = ['super_admin', 'audit_admin'] as const;

export function AppRouter() {
  return (
    <Routes>
      {/* /login 不在 AuthGuard 内（登录页本身） */}
      <Route path="/login" element={<LoginPage />} />

      {/* 根路径 → /dashboard（AuthGuard 会先校验登录） */}
      <Route path="/" element={<Navigate to="/dashboard" replace />} />

      {/* 受保护 admin 区 */}
      <Route element={<AuthGuard />}>
        <Route element={<AdminLayout />}>
          {/* 看板：所有 admin 角色可看 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/dashboard-detail" element={<DashboardDetailPage />} />
          </Route>

          {/* 订单：所有 admin 角色可看 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/orders" element={<OrderListPage />} />
            <Route path="/orders/:id" element={<OrderDetailPage />} />
          </Route>

          {/* 患者 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/patients" element={<PatientsPage />} />
            <Route path="/patients/:id" element={<PatientDetailPage />} />
          </Route>

          {/* 陪诊师 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/escorts" element={<EscortsPage />} />
            <Route path="/escorts/:id" element={<EscortDetailPage />} />
          </Route>
          <Route element={<RequireRole roles={[...ADMIN_OR_AUDIT]} />}>
            <Route path="/escorts/audit" element={<EscortAuditPage />} />
          </Route>

          {/* 退款 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/refunds" element={<RefundsPage />} />
            <Route path="/refunds/:id" element={<RefundDetailPage />} />
          </Route>

          {/* 钱包 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/wallets" element={<WalletsPage />} />
            <Route path="/wallets/:id" element={<WalletDetailPage />} />
          </Route>

          {/* 工单 / 评价 / 消息 / SOS：所有 admin 角色可看 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/work-orders" element={<WorkOrdersPage />} />
            <Route path="/reviews" element={<ReviewsPage />} />
            <Route path="/messages" element={<MessagesPage />} />
            <Route path="/sos" element={<SosPage />} />
            <Route path="/reports" element={<ReportsPage />} />
          </Route>

          {/* 财务 / 设置 / 医院：仅 super_admin */}
          <Route element={<RequireRole roles={[...ADMIN_ONLY]} />}>
            <Route path="/finance" element={<FinancePage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/hospitals" element={<HospitalsPage />} />
          </Route>

          {/* 套餐 / 优惠券：super_admin + order_admin */}
          <Route element={<RequireRole roles={[...ADMIN_OR_ORDER]} />}>
            <Route path="/packages" element={<PackagesPage />} />
            <Route path="/coupons" element={<CouponsPage />} />
          </Route>

          {/* 审计日志：super_admin + audit_admin */}
          <Route element={<RequireRole roles={[...ADMIN_OR_AUDIT]} />}>
            <Route path="/audit" element={<AuditPage />} />
          </Route>

          {/* 个人资料：所有 admin 角色 */}
          <Route element={<RequireRole roles={[...ALL_ROLES]} />}>
            <Route path="/profile" element={<ProfilePage />} />
          </Route>

          {/* 占位：未匹配路由跳回 dashboard */}
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Route>
      </Route>
    </Routes>
  );
}