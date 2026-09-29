/**
 * MSW messages handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表过滤 category；
 *   - 列表过滤 read=false；
 *   - 详情存在 / 不存在；
 *   - 发送站内信创建记录；
 *   - 广播全员；
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

describe('MSW messages handler v1', () => {
  it('列表过滤 category=system', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/messages?category=system', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: Array<{ category: string }>;
    };
    expect(json.code).toBe(0);
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const m of json.data) expect(m.category).toBe('system');
  });

  it('列表过滤 read=false 仅返回未读', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/messages?read=false', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{ id: number; read: boolean }>;
    };
    for (const m of json.data) expect(m.read).toBe(false);
  });

  it('详情不存在的 id → code=12001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/messages/999999', {
      headers: AUTH,
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(12001);
  });

  it('发送站内信 → 创建成功', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/messages', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        target_user_id: 5,
        title: '测试消息',
        category: 'system',
      }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      code: number;
      data: { title: string; category: string; read: boolean };
    };
    expect(json.code).toBe(0);
    expect(json.data.title).toBe('测试消息');
    expect(json.data.category).toBe('system');
    expect(json.data.read).toBe(false);
  });

  it('发送站内信缺参 → code=10001', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/messages', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: '缺参' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(10001);
  });

  it('全员广播 → audience=all', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/messages/broadcast', {
      method: 'POST',
      headers: { ...AUTH, 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: '系统维护通知', category: 'announcement' }),
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { audience: string } };
    expect(json.data.audience).toBe('all');
  });

  it('缺 Authorization → code=11003', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/messages');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { code: number };
    expect(json.code).toBe(11003);
  });
});