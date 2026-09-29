/**
 * MSW refunds handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表按 status 过滤；
 *   - 详情存在 / 不存在；
 *   - 通过 / 驳回返回 status 更新。
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('MSW refunds handler v1', () => {
  it('列表过滤 status=pending', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/refunds?status=pending');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{ id: number; status: string }>;
      total: number;
    };
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const r of json.data) expect(r.status).toBe('pending');
  });

  it('详情存在的 refund', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/refunds/2001');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; status: string } };
    expect(json.data.id).toBe(2001);
  });

  it('详情不存在的 id 返回 404', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/refunds/999999');
    expect(resp.status).toBe(404);
  });

  it('审批通过 → status=approved', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/refunds/2001/approve', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ note: '同意退款' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { status: string; refund_note: string } };
    expect(json.data.status).toBe('approved');
    expect(json.data.refund_note).toBe('同意退款');
  });

  it('审批驳回 → status=rejected', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/refunds/2002/reject', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: '超出退款期限' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { status: string; refund_note: string } };
    expect(json.data.status).toBe('rejected');
    expect(json.data.refund_note).toBe('超出退款期限');
  });
});