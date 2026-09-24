/**
 * admin-packages MSW handlers（v1 缺失骨架补齐 — H7）。
 *
 * 路由：
 *   GET    /api/v1/admin/packages?status=on|off
 *   GET    /api/v1/admin/packages/:id
 *   POST   /api/v1/admin/packages            → 创建套餐
 *   PATCH  /api/v1/admin/packages/:id        → 更新套餐（含上下架）
 *
 * 响应形态：{ code: 0, data: ..., trace_id: "admin-msw-..." }
 * 错误响应：{ code, message, data: null, trace_id }，HTTP 状态码 200
 *   - 10001 参数无效
 *   - 12001 资源不存在
 *   - 11003 admin_forbidden
 *
 * 对应 spec：l2-api-gap §2.3 admin 12 API · packages
 */
import { http, HttpResponse } from 'msw';
import { mockPackages, type PackageFixture } from '../../data/seed';

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

export const packageHandlers = [
  // ── 列表（按 status 过滤） ────────────────────────────────────────
  http.get('/api/v1/admin/packages', ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const data: PackageFixture[] = status
      ? mockPackages.filter((p) => p.status === status)
      : mockPackages;
    return HttpResponse.json({
      code: 0,
      data,
      total: data.length,
      trace_id: traceId(),
    });
  }),

  // ── 详情 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/packages/:id', ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const pkg = mockPackages.find((p) => p.id === Number(params.id));
    if (!pkg) {
      return HttpResponse.json(
        { code: 12001, message: 'package not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({ code: 0, data: pkg, trace_id: traceId() });
  }),

  // ── 创建 ──────────────────────────────────────────────────────────
  http.post('/api/v1/admin/packages', async ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const body = (await request.json().catch(() => ({}))) as {
      name?: string;
      price?: number;
      duration?: string;
    };
    if (!body.name || body.price === undefined || !body.duration) {
      return HttpResponse.json(
        {
          code: 10001,
          message: 'name/price/duration required',
          data: null,
          trace_id: traceId(),
        },
        { status: 200 },
      );
    }
    if (body.duration !== 'half_day' && body.duration !== 'full_day') {
      return HttpResponse.json(
        {
          code: 10001,
          message: 'duration must be half_day|full_day',
          data: null,
          trace_id: traceId(),
        },
        { status: 200 },
      );
    }
    const nextId = 14000 + mockPackages.length + 1;
    const created: PackageFixture = {
      id: nextId,
      name: body.name,
      price: body.price,
      duration: body.duration,
      status: 'on',
    };
    return HttpResponse.json({ code: 0, data: created, trace_id: traceId() });
  }),

  // ── 更新（含 status 上下架） ──────────────────────────────────────
  http.patch('/api/v1/admin/packages/:id', async ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const pkg = mockPackages.find((p) => p.id === Number(params.id));
    if (!pkg) {
      return HttpResponse.json(
        { code: 12001, message: 'package not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as {
      name?: string;
      price?: number;
      duration?: string;
      status?: string;
    };
    if (body.status && body.status !== 'on' && body.status !== 'off') {
      return HttpResponse.json(
        { code: 10001, message: 'status must be on|off', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    const updated: PackageFixture = {
      ...pkg,
      ...(body.name && { name: body.name }),
      ...(body.price !== undefined && { price: body.price }),
      ...(body.duration && { duration: body.duration as PackageFixture['duration'] }),
      ...(body.status && { status: body.status as PackageFixture['status'] }),
    };
    return HttpResponse.json({ code: 0, data: updated, trace_id: traceId() });
  }),
];