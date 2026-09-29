/**
 * MSW users handler v1 测试契约（待 vitest 启用后跑）：
 *   - 列表按 role 过滤；
 *   - 详情 id=存在 / 不存在（404）；
 *   - 停用返回 status=disabled。
 *
 * 注：当前骨架未装 vitest / msw/node 完整工具链，本文件为契约样。
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('MSW users handler v1', () => {
  it('列表过滤 role=super_admin', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/users?role=super_admin');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{ id: number; role: string }>;
      total: number;
    };
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const u of json.data) expect(u.role).toBe('super_admin');
  });

  it('详情存在的 id 返回 user', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/users/1');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; role: string } };
    expect(json.data.id).toBe(1);
  });

  it('详情不存在的 id 返回 404', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/users/999999');
    expect(resp.status).toBe(404);
  });

  it('停用 → status=disabled', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/users/2/disable', {
      method: 'POST',
    });
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as { data: { id: number; status: string } };
    expect(json.data.status).toBe('disabled');
  });
});