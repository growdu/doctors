import { Routes, Route, Navigate } from 'react-router-dom';
import AdminLayout from '@/layouts/AdminLayout';
import DashboardPage from '@/pages/dashboard/DashboardPage';
import EscortsPage from '@/pages/escorts/EscortsPage';
import OrdersPage from '@/pages/orders/OrdersPage';

/**
 * 占位路由（Task 3 雏形）：
 * - 仅装配 AdminLayout + 3 个 P0 占位页面
 * - 后续 Task（4+）按 spec §3.1 加入 18 个页面 + AuthGuard/RequirePermission
 */
export function AppRouter() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route element={<AdminLayout />}>
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/escorts" element={<EscortsPage />} />
        <Route path="/orders" element={<OrdersPage />} />
        {/* 占位：未匹配路由跳回 dashboard */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}