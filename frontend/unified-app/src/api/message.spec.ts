/**
 * api/message 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { sendMessage, listMessages, broadcast } from './message';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({ items: [] });
});

describe('api/message', () => {
  it('sendMessage POSTs body', async () => {
    await sendMessage({ to_user_id: 100, type: 'order', title: '订单状态更新', content: '已派单' });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/messages',
      method: 'POST',
      data: { to_user_id: 100, type: 'order', title: '订单状态更新', content: '已派单' },
      baseURL: 'http://127.0.0.1:8084',
    });
  });

  it('listMessages with read=false filter', async () => {
    await listMessages({ read: false });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('read=false');
  });

  it('broadcast POSTs to /messages/broadcast', async () => {
    await broadcast({ audience: 'all', title: '系统公告', content: '10月1日维护' });
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/messages/broadcast');
  });
});