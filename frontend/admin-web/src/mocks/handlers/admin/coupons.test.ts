/**
 * MSW coupons handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表过滤 type=amount_off；
 *   - 详情存在 / 不存在；
 *   - 创建卡券 type 非法 → code=10001；
 *   - 创建卡券 → 成功；
 *   - 停用卡券；
 *   - 缺 Authorization → code=11003。
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

const AUTH = { Authorization: 'Bearer test' };

describe('MSW coupons handler v1', () => {
  it('列表过滤 type=amount_off', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/coupons?type=amount_off', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ type: string }>;
    };
    expect(json.code).toBe(0);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const c of json.data) expect(c.type).toBe('amount_off');
  });

  it('详情存在的 coupon', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/coupons/15001', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; type: string; value: number } };
    expect(json.data.id).toBe(15001);
    expect(json.data.type).toBe('amount_off');
    expect(json.data.value).toBe(50);
  });

  it('详情不存在的 id → code=12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/coupons/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(12001);
  });

  it('创建卡券 type 非法 → code=10001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/coupons', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: '非法券',
        type: 'free',
        value: 0,
        valid_until: '2026-12-31T23:59:59Z',
      }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(10001);
  });

  it('创建卡券 → 成功', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/coupons', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: '满 100 减 20',
        type: 'amount_off',
        value: 20,
        valid_until: '2026-12-31T23:59:59Z',
      }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: { name: string; type: string; value: number };
    };
    expect(json.code).toBe(0);
    expect(json.data.name).toBe('满 100 减 20');
    expect(json.data.type).toBe('amount_off');
    expect(json.data.value).toBe(20);
  });

  it('停用卡券 → disabled=true', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/coupons/15001/disable', {
      method: 'POST',
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; disabled: boolean } };
    expect(json.data.id).toBe(15001);
    expect(json.data.disabled).toBe(true);
  });

  it('缺 Authorization → code=11003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/coupons');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });
});