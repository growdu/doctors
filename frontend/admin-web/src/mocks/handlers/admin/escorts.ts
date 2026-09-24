/**
 * admin-escorts MSW handlers（v1 缺失骨架）。
 *
 * 路由：
 *   GET  /api/v1/admin/escorts?status=pending|approved|rejected
 *   GET  /api/v1/admin/escorts/:id
 *   POST /api/v1/admin/escorts/:id/approve    （mock 通过）
 *   POST /api/v1/admin/escorts/:id/reject     （mock 拒绝）
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 12
 */
import { http, HttpResponse } from 'msw';
import { mockEscorts } from '../data/seed';

const traceId = () => `mock-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

export const escortHandlers = [
  // ── 列表 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/escorts', ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const filtered = status
      ? mockEscorts.filter((e) => e.audit_status === status)
      : mockEscorts;
    return HttpResponse.json({
      code: 0,
      data: filtered,
      total: filtered.length,
      trace_id: traceId(),
    });
  }),

  // ── 详情 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/escorts/:id', ({ params }) => {
    const escort = mockEscorts.find((e) => e.id === Number(params.id));
    if (!escort) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    return HttpResponse.json({ code: 0, data: escort, trace_id: traceId() });
  }),

  // ── 审核通过（mock） ──────────────────────────────────────────────
  http.post('/api/v1/admin/escorts/:id/approve', async ({ params, request }) => {
    const escort = mockEscorts.find((e) => e.id === Number(params.id));
    if (!escort) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { note?: string };
    return HttpResponse.json({
      code: 0,
      data: { ...escort, audit_status: 'approved', audit_note: body.note ?? null },
      trace_id: traceId(),
    });
  }),

  // ── 审核拒绝（mock） ──────────────────────────────────────────────
  http.post('/api/v1/admin/escorts/:id/reject', async ({ params, request }) => {
    const escort = mockEscorts.find((e) => e.id === Number(params.id));
    if (!escort) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { reason?: string };
    return HttpResponse.json({
      code: 0,
      data: { ...escort, audit_status: 'rejected', audit_note: body.reason ?? null },
      trace_id: traceId(),
    });
  }),
];