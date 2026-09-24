/**
 * admin-orders MSW handlers（v2 增量适配）。
 *
 * 路由：
 *   GET  /api/v1/admin/orders?status=...&page=...&page_size=...
 *   GET  /api/v1/admin/orders/:id
 *   POST /api/v1/admin/orders/:id/force-cancel   （v1 既有）
 *
 * v2 变更：
 *   - 列表过滤支持 selecting_escort / escort_pending_acceptance 2 状态；
 *   - 详情返回 3 个新字段：selected_escort_id / escort_pending_expire_at
 *     / escort_reject_reason（透传自 seed.ts）。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 5
 */
import { http, HttpResponse } from 'msw';
import { mockOrders } from '../data/seed';

export const orderHandlers = [
  // ── 列表 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/orders', ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const filtered = status ? mockOrders.filter((o) => o.status === status) : mockOrders;
    return HttpResponse.json({
      data: filtered,
      total: filtered.length,
    });
  }),

  // ── 详情（v2 透传 3 个新字段） ───────────────────────────────────
  http.get('/api/v1/admin/orders/:id', ({ params }) => {
    const order = mockOrders.find((o) => o.id === Number(params.id));
    if (!order) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 });
    }
    return HttpResponse.json({
      data: {
        ...order,
        // 显式列字段，确保 TS 编译期与 generated.ts 对齐
        selected_escort_id: order.selected_escort_id,
        escort_pending_expire_at: order.escort_pending_expire_at,
        escort_reject_reason: order.escort_reject_reason,
      },
    });
  }),

  // ── 强制取消（v1 既有） ──────────────────────────────────────────
  http.post('/api/v1/admin/orders/:id/force-cancel', async ({ params, request }) => {
    const body = (await request.json()) as { reason: string };
    return HttpResponse.json({
      data: { id: Number(params.id), status: 'canceled', reason: body.reason },
    });
  }),
];