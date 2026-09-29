/**
 * MSW packages handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表过滤 status=on；
 *   - 详情存在 / 不存在；
 *   - 创建套餐（duration 合法）；
 *   - 创建套餐 duration 非法 → code=10001；
 *   - 更新上下架 status=off；
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

describe('MSW packages handler v1', () => {
  it('列表过滤 status=on', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/packages?status=on', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ status: string }>;
    };
    expect(json.code).toBe(0);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const p of json.data) expect(p.status).toBe('on');
  });

  it('详情存在的 package', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/packages/14001', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; price: number } };
    expect(json.data.id).toBe(14001);
    expect(json.data.price).toBe(300);
  });

  it('详情不存在的 id → code=12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/packages/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(12001);
  });

  it('创建套餐 duration 非法 → code=10001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/packages', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: '非法', price: 100, duration: 'forever' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(10001);
  });

  it('创建套餐 → 成功', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/packages', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'VIP 套餐', price: 2000, duration: 'full_day' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: { name: string; price: number; duration: string; status: string };
    };
    expect(json.code).toBe(0);
    expect(json.data.price).toBe(2000);
    expect(json.data.duration).toBe('full_day');
    expect(json.data.status).toBe('on');
  });

  it('更新套餐 status=off（下架）', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/packages/14002', {
      method: 'PATCH',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: 'off' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; status: string } };
    expect(json.data.id).toBe(14002);
    expect(json.data.status).toBe('off');
  });

  it('缺 Authorization → code=11003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/packages');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });
});