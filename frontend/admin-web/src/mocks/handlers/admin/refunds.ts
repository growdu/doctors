/**
 * admin-refunds MSW handlers（v1 缺失骨架）。
 *
 * 路由：
 *   GET  /api/v1/admin/refunds?status=pending|approved|rejected
 *   GET  /api/v1/admin/refunds/:id
 *   POST /api/v1/admin/refunds/:id/approve
 *   POST /api/v1/admin/refunds/:id/reject
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 13
 */
import { http, HttpResponse } from 'msw';
import { mockRefunds } from '../data/seed';

const traceId = () => `mock-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

export const refundHandlers = [
  // ── 列表 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/refunds', ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const filtered = status ? mockRefunds.filter((r) => r.status === status) : mockRefunds;
    return HttpResponse.json({
      code: 0,
      data: filtered,
      total: filtered.length,
      trace_id: traceId(),
    });
  }),

  // ── 详情 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/refunds/:id', ({ params }) => {
    const refund = mockRefunds.find((r) => r.id === Number(params.id));
    if (!refund) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    return HttpResponse.json({ code: 0, data: refund, trace_id: traceId() });
  }),

  // ── 审批通过（mock） ──────────────────────────────────────────────
  http.post('/api/v1/admin/refunds/:id/approve', async ({ params, request }) => {
    const refund = mockRefunds.find((r) => r.id === Number(params.id));
    if (!refund) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { note?: string };
    return HttpResponse.json({
      code: 0,
      data: { ...refund, status: 'approved', refund_note: body.note ?? null },
      trace_id: traceId(),
    });
  }),

  // ── 审批驳回（mock） ──────────────────────────────────────────────
  http.post('/api/v1/admin/refunds/:id/reject', async ({ params, request }) => {
    const refund = mockRefunds.find((r) => r.id === Number(params.id));
    if (!refund) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { reason?: string };
    return HttpResponse.json({
      code: 0,
      data: { ...refund, status: 'rejected', refund_note: body.reason ?? null },
      trace_id: traceId(),
    });
  }),
];