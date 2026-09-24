/**
 * admin-reviews MSW handlers（v1 缺失骨架补齐 — H2）。
 *
 * 路由：
 *   GET  /api/v1/admin/reviews?escort_id=...&min_rating=...
 *   GET  /api/v1/admin/reviews/:id
 *   POST /api/v1/admin/reviews/:id/audit   → 审核通过/隐藏
 *   POST /api/v1/admin/reviews/:id/reply   → 客服回复
 *
 * 响应形态：{ code: 0, data: ..., trace_id: "admin-msw-..." }
 * 错误响应：{ code, message, data: null, trace_id }，HTTP 状态码 200
 *   - 10001 参数无效
 *   - 12001 资源不存在
 *   - 11003 admin_forbidden
 *
 * 对应 spec：l2-api-gap §2.3 admin 12 API · reviews
 */
import { http, HttpResponse } from 'msw';
import { mockReviews, type ReviewFixture } from '../../data/seed';

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

export const reviewHandlers = [
  // ── 列表（按 escort_id / min_rating 过滤） ────────────────────────
  http.get('/api/v1/admin/reviews', ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const url = new URL(request.url);
    const escortId = url.searchParams.get('escort_id');
    const minRating = url.searchParams.get('min_rating');
    let data: ReviewFixture[] = mockReviews;
    if (escortId) data = data.filter((r) => r.escort_id === Number(escortId));
    if (minRating) {
      const min = Number(minRating);
      data = data.filter((r) => r.rating >= min);
    }
    return HttpResponse.json({
      code: 0,
      data,
      total: data.length,
      trace_id: traceId(),
    });
  }),

  // ── 详情 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/reviews/:id', ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const review = mockReviews.find((r) => r.id === Number(params.id));
    if (!review) {
      return HttpResponse.json(
        { code: 12001, message: 'review not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({ code: 0, data: review, trace_id: traceId() });
  }),

  // ── 审核（pass/hide） ─────────────────────────────────────────────
  http.post('/api/v1/admin/reviews/:id/audit', async ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const review = mockReviews.find((r) => r.id === Number(params.id));
    if (!review) {
      return HttpResponse.json(
        { code: 12001, message: 'review not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as {
      result?: 'pass' | 'hide';
      reason?: string;
    };
    if (!body.result || (body.result !== 'pass' && body.result !== 'hide')) {
      return HttpResponse.json(
        { code: 10001, message: 'result must be pass|hide', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({
      code: 0,
      data: {
        ...review,
        audit_result: body.result,
        audit_reason: body.reason ?? null,
      },
      trace_id: traceId(),
    });
  }),

  // ── 客服回复 ──────────────────────────────────────────────────────
  http.post('/api/v1/admin/reviews/:id/reply', async ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const review = mockReviews.find((r) => r.id === Number(params.id));
    if (!review) {
      return HttpResponse.json(
        { code: 12001, message: 'review not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    const body = (await request.json().catch(() => ({}))) as { reply?: string };
    if (!body.reply) {
      return HttpResponse.json(
        { code: 10001, message: 'reply required', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({
      code: 0,
      data: { ...review, admin_reply: body.reply },
      trace_id: traceId(),
    });
  }),
];