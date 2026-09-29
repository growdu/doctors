/**
 * admin-patients MSW handlers（v1 缺失骨架补齐 — H5）。
 *
 * 路由：
 *   GET  /api/v1/admin/patients?keyword=...
 *   GET  /api/v1/admin/patients/:id
 *   POST /api/v1/admin/patients/:id/ban    → 封禁
 *   POST /api/v1/admin/patients/:id/unban  → 解封
 *
 * 响应形态：{ code: 0, data: ..., trace_id: "admin-msw-..." }
 * 错误响应：{ code, message, data: null, trace_id }，HTTP 状态码 200
 *   - 10001 参数无效
 *   - 12001 资源不存在
 *   - 11003 admin_forbidden
 *
 * 对应 spec：l2-api-gap §2.3 admin 12 API · patients
 */
import { http, HttpResponse } from 'msw';
import { mockPatients, type PatientFixture } from '../../data/seed';

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

export const patientHandlers = [
  // ── 列表（按 name / phone 模糊） ──────────────────────────────────
  http.get('/api/v1/admin/patients', ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const url = new URL(request.url);
    const keyword = url.searchParams.get('keyword');
    let data: PatientFixture[] = mockPatients;
    if (keyword) {
      const k = keyword.toLowerCase();
      data = data.filter(
        (p) => p.name.toLowerCase().includes(k) || p.phone.includes(k),
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
  http.get('/api/v1/admin/patients/:id', ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const patient = mockPatients.find((p) => p.id === Number(params.id));
    if (!patient) {
      return HttpResponse.json(
        { code: 12001, message: 'patient not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({ code: 0, data: patient, trace_id: traceId() });
  }),

  // ── 封禁 ──────────────────────────────────────────────────────────
  http.post('/api/v1/admin/patients/:id/ban', async ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const patient = mockPatients.find((p) => p.id === Number(params.id));
    if (!patient) {
      return HttpResponse.json(
        { code: 12001, message: 'patient not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { reason?: string };
    if (!body.reason) {
      return HttpResponse.json(
        { code: 10001, message: 'reason required', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({
      code: 0,
      data: { ...patient, status: 'banned', ban_reason: body.reason },
      trace_id: traceId(),
    });
  }),

  // ── 解封 ──────────────────────────────────────────────────────────
  http.post('/api/v1/admin/patients/:id/unban', ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const patient = mockPatients.find((p) => p.id === Number(params.id));
    if (!patient) {
      return HttpResponse.json(
        { code: 12001, message: 'patient not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({
      code: 0,
      data: { ...patient, status: 'active', ban_reason: null },
      trace_id: traceId(),
    });
  }),
];