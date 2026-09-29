/**
 * MSW escorts handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表按 audit_status 过滤；
 *   - 详情存在 / 不存在；
 *   - 通过 / 拒绝返回 audit_status 更新。
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('MSW escorts handler v1', () => {
  it('列表过滤 audit_status=pending', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/escorts?status=pending');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{ id: number; audit_status: string }>;
      total: number;
    };
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const e of json.data) expect(e.audit_status).toBe('pending');
  });

  it('详情存在的 escort', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/escorts/1001');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; audit_status: string } };
    expect(json.data.id).toBe(1001);
  });

  it('详情不存在的 id 返回 404', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/escorts/999999');
    expect(resp.status).toBe(404);
  });

  it('审核通过 → audit_status=approved', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/escorts/1001/approve', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ note: 'OK' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { audit_status: string; audit_note: string } };
    expect(json.data.audit_status).toBe('approved');
    expect(json.data.audit_note).toBe('OK');
  });

  it('审核拒绝 → audit_status=rejected', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/escorts/1002/reject', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: '资料不全' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { audit_status: string; audit_note: string } };
    expect(json.data.audit_status).toBe('rejected');
    expect(json.data.audit_note).toBe('资料不全');
  });
});