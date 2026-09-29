/**
 * MSW hospitals handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表按 city 过滤；
 *   - 列表按 status=active 过滤；
 *   - 详情存在 / 不存在；
 *   - 创建医院；
 *   - 更新医院（PATCH）；
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

describe('MSW hospitals handler v1', () => {
  it('列表按 city=北京 过滤', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/hospitals?city=北京', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ city: string }>;
    };
    expect(json.code).toBe(0);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const h of json.data) expect(h.city).toBe('北京');
  });

  it('列表按 status=active 仅返回启用', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/hospitals?status=active', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: Array<{ status: string }> };
    for (const h of json.data) expect(h.status).toBe('active');
  });

  it('详情不存在的 id → code=12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/hospitals/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(12001);
  });

  it('创建医院', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/hospitals', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: '测试医院', city: '深圳', level: '三甲' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: { name: string; city: string; status: string };
    };
    expect(json.code).toBe(0);
    expect(json.data.name).toBe('测试医院');
    expect(json.data.city).toBe('深圳');
    expect(json.data.status).toBe('active');
  });

  it('创建医院缺参 → code=10001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/hospitals', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: '缺 city/level' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(10001);
  });

  it('更新医院 PATCH → 字段更新', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/hospitals/13003', {
      method: 'PATCH',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: 'active' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number; data: { id: number; status: string } };
    expect(json.code).toBe(0);
    expect(json.data.id).toBe(13003);
    expect(json.data.status).toBe('active');
  });

  it('缺 Authorization → code=11003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/hospitals');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });
});