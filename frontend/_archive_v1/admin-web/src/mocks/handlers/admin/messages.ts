/**
 * admin-messages MSW handlers（v1 缺失骨架补齐 — H3）。
 *
 * 路由：
 *   GET  /api/v1/admin/messages?category=system|announcement|work_order&read=true|false
 *   GET  /api/v1/admin/messages/:id
 *   POST /api/v1/admin/messages                → 发送站内信（按用户/角色）
 *   POST /api/v1/admin/messages/broadcast      → 全员广播
 *
 * 响应形态：{ code: 0, data: ..., trace_id: "admin-msw-..." }
 * 错误响应：{ code, message, data: null, trace_id }，HTTP 状态码 200
 *   - 10001 参数无效
 *   - 12001 资源不存在
 *   - 11003 admin_forbidden
 *
 * 对应 spec：l2-api-gap §2.3 admin 12 API · messages
 */
import { http, HttpResponse } from 'msw';
import { mockMessages, type MessageFixture } from '../../data/seed';

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

export const messageHandlers = [
  // ── 列表（按 category / read 过滤） ───────────────────────────────
  http.get('/api/v1/admin/messages', ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const url = new URL(request.url);
    const category = url.searchParams.get('category');
    const read = url.searchParams.get('read');
    let data: MessageFixture[] = mockMessages;
    if (category) data = data.filter((m) => m.category === category);
    if (read !== null) {
      const want = read === 'true';
      data = data.filter((m) => m.read === want);
    }
    return HttpResponse.json({
      code: 0,
      data,
      total: data.length,
      trace_id: traceId(),
    });
  }),

  // ── 详情 ──────────────────────────────────────────────────────────
  http.get('/api/v1/admin/messages/:id', ({ params, request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const msg = mockMessages.find((m) => m.id === Number(params.id));
    if (!msg) {
      return HttpResponse.json(
        { code: 12001, message: 'message not found', data: null, trace_id: traceId() },
        { status: 200 },
      );
    }
    return HttpResponse.json({ code: 0, data: msg, trace_id: traceId() });
  }),

  // ── 发送站内信（指定用户） ────────────────────────────────────────
  http.post('/api/v1/admin/messages', async ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const body = (await request.json().catch(() => ({}))) as {
      target_user_id?: number;
      title?: string;
      category?: string;
    };
    if (!body.target_user_id || !body.title || !body.category) {
      return HttpResponse.json(
        {
          code: 10001,
          message: 'target_user_id/title/category required',
          data: null,
          trace_id: traceId(),
        },
        { status: 200 },
      );
    }
    const nextId = 11000 + mockMessages.length + 1;
    const created: MessageFixture = {
      id: nextId,
      category: body.category as MessageFixture['category'],
      title: body.title,
      read: false,
      created_at: new Date().toISOString(),
    };
    return HttpResponse.json({
      code: 0,
      data: { ...created, target_user_id: body.target_user_id },
      trace_id: traceId(),
    });
  }),

  // ── 全员广播 ──────────────────────────────────────────────────────
  http.post('/api/v1/admin/messages/broadcast', async ({ request }) => {
    const guard = requireAuth(request);
    if (guard) return guard;
    const body = (await request.json().catch(() => ({}))) as {
      title?: string;
      category?: string;
    };
    if (!body.title || !body.category) {
      return HttpResponse.json(
        {
          code: 10001,
          message: 'title/category required',
          data: null,
          trace_id: traceId(),
        },
        { status: 200 },
      );
    }
    const nextId = 11000 + mockMessages.length + 1;
    return HttpResponse.json({
      code: 0,
      data: {
        id: nextId,
        title: body.title,
        category: body.category,
        audience: 'all',
        created_at: new Date().toISOString(),
      },
      trace_id: traceId(),
    });
  }),
];