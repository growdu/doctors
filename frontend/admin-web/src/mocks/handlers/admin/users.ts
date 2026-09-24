/**
 * admin-users MSW handlers（v1 缺失骨架）。
 *
 * 路由：
 *   GET  /api/v1/admin/users                  → 管理员列表（支持角色筛选）
 *   GET  /api/v1/admin/users/:id              → 管理员详情
 *   POST /api/v1/admin/users/:id/disable      → 停用管理员（mock）
 *
 * 响应形态（与 v2 一致）：{ data: ..., total?: ..., trace_id: "..." }
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.3 (RBAC) + §5.2
 */
import { http, HttpResponse } from 'msw';
import { mockUsers } from '../data/seed';

const traceId = () => `mock-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

export const userHandlers = [
  // ── 列表 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/users', ({ request }) => {
    const url = new URL(request.url);
    const role = url.searchParams.get('role');
    const keyword = url.searchParams.get('keyword');
    let data = mockUsers;
    if (role) data = data.filter((u) => u.role === role);
    if (keyword) {
      const k = keyword.toLowerCase();
      data = data.filter(
        (u) =>
          u.username.toLowerCase().includes(k) ||
          (u.display_name ?? '').toLowerCase().includes(k),
      );
    }
    return HttpResponse.json({
      code: 0,
      data,
      total: data.length,
      trace_id: traceId(),
    });
  }),

  // ── 详情 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/users/:id', ({ params }) => {
    const user = mockUsers.find((u) => u.id === Number(params.id));
    if (!user) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    return HttpResponse.json({ code: 0, data: user, trace_id: traceId() });
  }),

  // ── 停用（mock） ──────────────────────────────────────────────────
  http.post('/api/v1/admin/users/:id/disable', ({ params }) => {
    const user = mockUsers.find((u) => u.id === Number(params.id));
    if (!user) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    return HttpResponse.json({
      code: 0,
      data: { ...user, status: 'disabled' },
      trace_id: traceId(),
    });
  }),
];