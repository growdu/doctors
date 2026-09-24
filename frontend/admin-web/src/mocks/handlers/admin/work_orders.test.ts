/**
 * MSW work-orders handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表过滤 status=open 返回含 8001；
 *   - 详情存在的工单；
 *   - 不存在 id 返回 code=12001（HTTP 200）；
 *   - 创建/分配/关闭返回状态更新；
 *   - 缺 Authorization 头返 code=11003（HTTP 200）。
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

const AUTH = { Authorization: 'Bearer test' };

describe('MSW work-orders handler v1', () => {
  it('列表过滤 status=open 返回含 8001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/work-orders?status=open', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ id: number; status: string }>;
      trace_id: string;
    };
    expect(json.code).toBe(0);
    expect(json.trace_id.startsWith('admin-msw-')).toBe(true);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data.map((w) => w.id)).toContain(8001);
  });

  it('详情存在的工单', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/work-orders/8001', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number; data: { id: number; status: string } };
    expect(json.code).toBe(0);
    expect(json.data.id).toBe(8001);
  });

  it('详情不存在的 id 返回 code=12001（HTTP 200）', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/work-orders/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number; data: unknown };
    expect(json.code).toBe(12001);
    expect(json.data).toBeNull();
  });

  it('缺 Authorization 头返 code=11003（HTTP 200）', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/work-orders');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });

  it('关闭工单 → status=closed', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/work-orders/8001/close', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ resolution: '已联系用户' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { status: string; resolution: string } };
    expect(json.data.status).toBe('closed');
    expect(json.data.resolution).toBe('已联系用户');
  });
});