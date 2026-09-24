/**
 * Router 守卫：AuthGuard + RequireRole RBAC。
 *
 * AuthGuard
 *   - 包裹 <Outlet/>；
 *   - 未登录（!isAuthed）→ <Navigate to="/login" replace />；
 *   - 已登录 → render children。
 *
 * RequireRole
 *   - 子路由级 RBAC：传入 roles: AdminRole[]；
 *   - 角色不在列表 → 渲染 403 页面（Result 403）；
 *   - 未登录 → 与 AuthGuard 同等行为（跳 /login）。
 *
 * 设计要点：
 *   - 守卫读 useAuthStore（Zustand）；
 *   - 测试用 vi.mock('@/stores/authStore') 替换 store；
 *   - 守卫组件不依赖 react-router v6 data router，纯 <Outlet/> 模式。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.3 (RBAC)
 */
import type { ReactNode } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { Button, Result } from 'antd';
import { useAuthStore, type AdminRole } from '@/stores/authStore';

// ── AuthGuard ───────────────────────────────────────────────────────
export interface AuthGuardProps {
  children?: ReactNode;
}

/**
 * 包裹 admin 子路由：未登录跳 /login，已登录 render <Outlet/>。
 *
 * 用法：
 *   <Route element={<AuthGuard />}>
 *     <Route path="/dashboard" element={<DashboardPage />} />
 *     ...
 *   </Route>
 */
export function AuthGuard({ children }: AuthGuardProps) {
  const isAuthed = useAuthStore((s) => s.isAuthed);
  const location = useLocation();
  if (!isAuthed) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  return <>{children ?? <Outlet />}</>;
}

// ── RequireRole ─────────────────────────────────────────────────────
export interface RequireRoleProps {
  roles: AdminRole[];
  children?: ReactNode;
}

/**
 * RBAC 子路由守卫：当前 role 不在 roles 列表 → 渲染 403 页面。
 *
 * 用法：
 *   <Route element={<RequireRole roles={['super_admin','order_admin']} />}>
 *     <Route path="/orders/:id/force-cancel" element={<ForceCancelPage />} />
 *   </Route>
 */
export function RequireRole({ roles, children }: RequireRoleProps) {
  const isAuthed = useAuthStore((s) => s.isAuthed);
  const role = useAuthStore((s) => s.role);
  const location = useLocation();

  // 未登录也兜底跳 /login（即便外层 AuthGuard 漏挂）
  if (!isAuthed) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  const allowed = role != null && roles.includes(role);
  if (!allowed) {
    return (
      <Result
        status="403"
        title="403"
        subTitle={`抱歉，当前角色（${role ?? '未知'}）无权访问该页面。需要：${roles.join(' / ')}`}
        extra={
          <Button type="primary" onClick={() => window.history.back()}>
            返回上一页
          </Button>
        }
        data-testid="rbac-403"
      />
    );
  }

  return <>{children ?? <Outlet />}</>;
}

export default AuthGuard;