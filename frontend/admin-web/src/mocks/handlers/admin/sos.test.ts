/**
 * MSW sos handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表过滤 status=open；
 *   - 详情存在 / 不存在；
 *   - 处置 resolve → status=closed；
 *   - 升级 escalate → 含 escalate_level；
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

describe('MSW sos handler v1', () => {
  it('列表过滤 status=open 仅返回 12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/sos?status=open', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ id: number; status: string }>;
    };
    expect(json.code).toBe(0);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const s of json.data) expect(s.status).toBe('open');
  });

  it('详情存在的 sos', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/sos/12001', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; location: string } };
    expect(json.data.id).toBe(12001);
    expect(json.data.location).toBeTruthy();
  });

  it('详情不存在的 id → code=12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/sos/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(12001);
  });

  it('处置 resolve → status=closed + resolution_note', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/sos/12001/resolve', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ note: '已联系陪诊师' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: { status: string; resolution_note: string };
    };
    expect(json.data.status).toBe('closed');
    expect(json.data.resolution_note).toBe('已联系陪诊师');
  });

  it('升级 escalate 缺 level → code=10001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/sos/12001/escalate', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: '紧急' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(10001);
  });

  it('升级 escalate 含 level → 200 + escalate_level', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/sos/12001/escalate', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ level: 'P1', reason: '紧急' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { escalate_level: string } };
    expect(json.data.escalate_level).toBe('P1');
  });

  it('缺 Authorization → code=11003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/sos');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });
});