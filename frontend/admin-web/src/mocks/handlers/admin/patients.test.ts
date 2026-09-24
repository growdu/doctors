/**
 * MSW patients handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表按 keyword 过滤；
 *   - 详情存在 / 不存在；
 *   - 封禁 ban 含 reason；
 *   - 封禁缺 reason → code=10001；
 *   - 解封 unban；
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

describe('MSW patients handler v1', () => {
  it('列表按 keyword=张 模糊匹配', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/patients?keyword=张', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ name: string }>;
    };
    expect(json.code).toBe(0);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const p of json.data) expect(p.name).toContain('张');
  });

  it('详情存在的 patient', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/patients/7001', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; order_count: number } };
    expect(json.data.id).toBe(7001);
    expect(json.data.order_count).toBeGreaterThanOrEqual(1);
  });

  it('详情不存在的 id → code=12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/patients/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(12001);
  });

  it('封禁缺 reason → code=10001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/patients/7001/ban', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({}),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(10001);
  });

  it('封禁含 reason → status=banned', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/patients/7001/ban', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: '恶意退款' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { status: string; ban_reason: string } };
    expect(json.data.status).toBe('banned');
    expect(json.data.ban_reason).toBe('恶意退款');
  });

  it('解封 unban → status=active', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/patients/7001/unban', {
      method: 'POST',
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { status: string; ban_reason: null } };
    expect(json.data.status).toBe('active');
    expect(json.data.ban_reason).toBeNull();
  });

  it('缺 Authorization → code=11003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/patients');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });
});