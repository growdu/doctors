import { Routes, Route, Navigate } from 'react-router-dom';
import AdminLayout from '@/layouts/AdminLayout';
import DashboardPage from '@/pages/dashboard/DashboardPage';
import EscortsPage from '@/pages/escorts/EscortsPage';
import OrderListPage from '@/pages/orders/OrderListPage';
import OrderDetailPage from '@/pages/orders/OrderDetailPage';

/**
 * 路由表（v2 增量）：
 *   /              → 重定向 /dashboard
 *   /dashboard     → DashboardPage（v2 加 2 个待确认 Statistic）
 *   /escorts       → EscortsPage（占位）
 *   /orders        → OrderListPage（v2 加 2 状态筛选 + 2 列）
 *   /orders/:id    → OrderDetailPage（v2 加 2 状态分支 + 30s 倒计时 + 拒接回退卡）
 *
 * v1 占位 OrdersPage 仍保留在 src/pages/orders/OrdersPage.tsx 但不再路由挂载，
 * 留作未来 v1 任务（订单批量操作页）的占位。
 */
export function AppRouter() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route element={<AdminLayout />}>
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/escorts" element={<EscortsPage />} />
        <Route path="/orders" element={<OrderListPage />} />
        <Route path="/orders/:id" element={<OrderDetailPage />} />
        {/* 占位：未匹配路由跳回 dashboard */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}