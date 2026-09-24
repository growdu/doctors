/**
 * MSW orders handler v2 测试：
 *   - 列表过滤 selecting_escort 返回含 selected_escort_id=null 的订单；
 *   - 详情 9002 返回 escort_pending_acceptance 含 escort_pending_expire_at 字段。
 *
 * 注：本测试文件为「契约 + 验收脚本」式样，当前骨架未装 vitest + msw/node
 * 完整工具链，文件就绪待用户在本地 `npm install` 后跑 `vitest run`。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 5
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('MSW orders handler v2', () => {
  it('列表过滤 selecting_escort 返回含 selected_escort_id=null 的订单', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/orders?status=selecting_escort');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{
        id: number;
        status: string;
        selected_escort_id: number | null;
        escort_pending_expire_at: string | null;
      }>;
      total: number;
    };
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    // fixture id=9001 / 9003 都是 selecting_escort
    const ids = json.data.map((o) => o.id).sort();
    expect(ids).toEqual(expect.arrayContaining([9001, 9003]));
    for (const o of json.data) {
      expect(o.status).toBe('selecting_escort');
      expect(o).toHaveProperty('selected_escort_id');
      expect(o).toHaveProperty('escort_pending_expire_at');
    }
  });

  it('列表过滤 escort_pending_acceptance 仅返回 id=9002', async () => {
    const resp = await fetch(
      'http://localhost/api/v1/admin/orders?status=escort_pending_acceptance',
    );
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{ id: number; status: string }>;
      total: number;
    };
    expect(json.data.length).toBe(1);
    expect(json.data[0].id).toBe(9002);
    expect(json.data[0].status).toBe('escort_pending_acceptance');
  });

  it('详情 9002 返回 escort_pending_acceptance + selected_escort_id=42 + 截止时间字段', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/orders/9002');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: {
        id: number;
        status: string;
        selected_escort_id: number | null;
        escort_pending_expire_at: string | null;
        escort_reject_reason: string | null;
      };
    };
    expect(json.data.id).toBe(9002);
    expect(json.data.status).toBe('escort_pending_acceptance');
    expect(json.data.selected_escort_id).toBe(42);
    expect(json.data.escort_pending_expire_at).toBeTruthy();
    expect(json.data.escort_reject_reason).toBeNull();
  });

  it('详情 9003 拒接回退：status=selecting_escort + escort_reject_reason=escort_declined', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/orders/9003');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: {
        id: number;
        status: string;
        escort_reject_reason: string | null;
      };
    };
    expect(json.data.id).toBe(9003);
    expect(json.data.status).toBe('selecting_escort');
    expect(json.data.escort_reject_reason).toBe('escort_declined');
  });

  it('详情不存在的 id 返回 404', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/orders/999999');
    expect(resp.status).toBe(404);
  });
});