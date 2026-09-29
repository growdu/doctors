/**
 * MSW wallets handler v1 测试契约（待 vitest 启用后跑）：
 *   - 流水按 type=patient 过滤；
 *   - 流水按 tx_type=refund 过滤；
 *   - 钱包主体详情含 recent_transactions。
 */
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '..';

const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('MSW wallets handler v1', () => {
  it('流水按 type=patient 过滤', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/wallets?type=patient');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{ id: number; subject_type: string }>;
      total: number;
    };
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const t of json.data) expect(t.subject_type).toBe('patient');
  });

  it('流水按 tx_type=refund 过滤', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/wallets?tx_type=refund');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: Array<{ id: number; tx_type: string }>;
      total: number;
    };
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    for (const t of json.data) expect(t.tx_type).toBe('refund');
  });

  it('主体详情含 recent_transactions', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/wallets/5001');
    expect(resp.status).toBe(200);
    const json = (await resp.json()) as {
      data: { id: number; balance: number; recent_transactions: unknown[] };
    };
    expect(json.data.id).toBe(5001);
    expect(typeof json.data.balance).toBe('number');
    expect(Array.isArray(json.data.recent_transactions)).toBe(true);
  });

  it('不存在的 id 返回 404', async () => {
    const resp = await fetch('http://localhost/api/v1/admin/wallets/999999');
    expect(resp.status).toBe(404);
  });
});