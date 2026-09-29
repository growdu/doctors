/**
 * escort/message/chat.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - parseQuery 提取 id 并调 getMessage
 *   - 渲染消息详情（类型 / 标题 / 内容 / 关联资源）
 *   - 点击「关联资源：order #xxx」跳 escort/order-detail?id=
 *   - 其他 ref_type 不调 navigate，弹 toast
 *   - 未传 id / 找不到 / API 异常 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import MessageChat from './chat.vue';

vi.mock('@/api/message', () => ({
  getMessage: vi.fn(),
}));

import * as apiMessage from '@/api/message';
import type { Message } from '@/api/message';

const makeMessage = (id: number, overrides: Partial<Message> = {}): Message => ({
  id,
  to_user_id: 2001,
  type: 'order',
  title: `订单通知 ${id}`,
  content: `订单已被接单（消息 #${id}）`,
  ref_id: 100 + id,
  ref_type: 'order',
  read: false,
  created_at: '2026-09-29T10:00:00Z',
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
  (globalThis as unknown as {
    uni: {
      getCurrentPages: () => Array<{ options?: Record<string, string> }>;
      navigateTo: (opts: unknown) => void;
      showToast: (opts: unknown) => void;
    };
  }).uni.getCurrentPages = () => [{ options: { id: '101' } }];
});

describe('escort/message/chat · 加载', () => {
  it('parseQuery 提取 id 并调 getMessage(101)', async () => {
    vi.mocked(apiMessage.getMessage).mockResolvedValue(makeMessage(101));
    mount(MessageChat);
    await flushPromises();
    expect(apiMessage.getMessage).toHaveBeenCalledWith(101);
  });

  it('渲染消息详情（类型 / 标题 / 内容 / 关联资源）', async () => {
    vi.mocked(apiMessage.getMessage).mockResolvedValue(makeMessage(101));
    const w = mount(MessageChat);
    await flushPromises();

    expect(w.find('[data-testid="escort-message-chat-title"]').text()).toContain('订单通知 101');
    expect(w.find('[data-testid="escort-message-chat-content"]').text()).toContain('订单已被接单');
    expect(w.find('[data-testid="escort-message-chat-ref-order"]').exists()).toBe(true);
    expect(w.text()).toContain('order #201');
  });

  it('未传 id 显示错误', async () => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: {} }];
    const w = mount(MessageChat);
    await flushPromises();

    expect(w.find('[data-testid="escort-message-chat-error"]').exists()).toBe(true);
    expect(w.text()).toContain('未指定消息 ID');
  });

  it('API 异常显示错误', async () => {
    vi.mocked(apiMessage.getMessage).mockRejectedValue(new Error('网络超时'));
    const w = mount(MessageChat);
    await flushPromises();

    expect(w.find('[data-testid="escort-message-chat-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});

describe('escort/message/chat · 关联资源跳转', () => {
  it('点击「order #xxx」跳 escort/order-detail?id=', async () => {
    vi.mocked(apiMessage.getMessage).mockResolvedValue(makeMessage(101));
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(MessageChat);
    await flushPromises();

    await w.find('[data-testid="escort-message-chat-ref-order"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/order-detail?id=201' });
  });

  it('非 order 关联资源弹 toast 不调 navigateTo', async () => {
    vi.mocked(apiMessage.getMessage).mockResolvedValue(
      makeMessage(101, { ref_type: 'payment', ref_id: 999 }),
    );
    const navSpy = vi.fn();
    const toastSpy = vi.fn();
    (globalThis as unknown as {
      uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy };
    }).uni.navigateTo = navSpy;
    (globalThis as unknown as {
      uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy };
    }).uni.showToast = toastSpy;

    const w = mount(MessageChat);
    await flushPromises();

    await w.find('[data-testid="escort-message-chat-ref-payment"]').trigger('click');

    expect(navSpy).not.toHaveBeenCalled();
    expect(toastSpy).toHaveBeenCalledWith({ title: '相关资源跳转暂未实现', icon: 'none' });
  });
});