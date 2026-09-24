/**
 * MSW reviews handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表过滤 escort_id；
 *   - 详情存在 / 不存在；
 *   - 审核 pass/hide；
 *   - 客服回复；
 *   - 缺 Authorization 头返 code=11003。
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

const AUTH = { Authorization: 'Bearer test' };

describe('MSW reviews handler v1', () => {
  it('列表过滤 escort_id=1003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/reviews?escort_id=1003', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ id: number; escort_id: number }>;
    };
    expect(json.code).toBe(0);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const r of json.data) expect(r.escort_id).toBe(1003);
  });

  it('详情存在的 review', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/reviews/9001', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number; data: { id: number; rating: number } };
    expect(json.code).toBe(0);
    expect(json.data.id).toBe(9001);
    expect(json.data.rating).toBe(5);
  });

  it('详情不存在的 id → code=12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/reviews/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(12001);
  });

  it('审核 pass → audit_result=pass', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/reviews/9001/audit', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ result: 'pass' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { audit_result: string } };
    expect(json.data.audit_result).toBe('pass');
  });

  it('缺 Authorization → code=11003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/reviews');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });
});