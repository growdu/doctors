/**
 * admin-sos MSW handlers（v1 缺失骨架补齐 — H4）。
 *
 * 路由：
 *   GET  /api/v1/admin/sos?status=open|closed
 *   GET  /api/v1/admin/sos/:id
 *   POST /api/v1/admin/sos/:id/resolve   → 处置完成
 *   POST /api/v1/admin/sos/:id/escalate  → 上报并升级
 *
 * 响应形态：{ code: 0, data: ..., trace_id: "admin-msw-..." }
 * 错误响应：{ code, message, data: null, trace_id }，HTTP 状态码 200
 *   - 10001 参数无效
 *   - 12001 资源不存在
 *   - 11003 admin_forbidden
 *
 * 对应 spec：l2-api-gap §2.3 admin 12 API · sos
 */
import { http, HttpResponse } from 'msw';
import { mockSosAlerts, type SosFixture } from '../../data/seed';

const traceId = () =>
  `admin-msw-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const requireAuth = (request: Request) => {
  if (!request.headers.get('authorization')) {
    return HttpResponse.json(
      { code: 11003, message: 'admin_forbidden', data: null, trace_id: traceId() },
      { status: 200 },
    );
  }
  return null;
};

export const sosHandlers = [
  // ── 列表（按 status 过滤） ────────────────────────────────────────
  http.get('/api/v1/admin/sos', ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const data: SosFixture[] = status
      ? mockSosAlerts.filter((s) => s.status === status)
      : mockSosAlerts;
    return HttpResponse.json({
      code: 0,
      data,
      total: data.length,
      trace_id: traceId(),
    });
  }),

  // ── 详情 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/sos/:id', ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const sos = mockSosAlerts.find((s) => s.id === Number(params.id));
    if (!sos) {
      return HttpResponse.json(
        { code: 12001, message: 'sos not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({ code: 0, data: sos, trace_id: traceId() });
  }),

  // ── 处置完成（resolve） ───────────────────────────────────────────
  http.post('/api/v1/admin/sos/:id/resolve', async ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const sos = mockSosAlerts.find((s) => s.id === Number(params.id));
    if (!sos) {
      return HttpResponse.json(
        { code: 12001, message: 'sos not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { note?: string };
    return HttpResponse.json({
      code: 0,
      data: { ...sos, status: 'closed', resolution_note: body.note ?? null },
      trace_id: traceId(),
    });
  }),

  // ── 升级（escalate） ──────────────────────────────────────────────
  http.post('/api/v1/admin/sos/:id/escalate', async ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const sos = mockSosAlerts.find((s) => s.id === Number(params.id));
    if (!sos) {
      return HttpResponse.json(
        { code: 12001, message: 'sos not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { level?: string; reason?: string };
    if (!body.level) {
      return HttpResponse.json(
        { code: 10001, message: 'level required', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({
      code: 0,
      data: {
        ...sos,
        escalate_level: body.level,
        escalate_reason: body.reason ?? null,
        escalated_at: new Date().toISOString(),
      },
      trace_id: traceId(),
    });
  }),
];