/**
 * escort/message/list.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listMessages
 *   - 2 tab（全部 / 未读）+ 默认 active 是 all
 *   - 点击 unread tab 重新拉数据并带 ?read=false
 *   - 渲染消息卡（类型 / 标题 / 内容 / 时间 / 未读小红点）
 *   - 点击消息卡跳 chat?id=
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import MessageList from './list.vue';

vi.mock('@/api/message', () => ({
  listMessages: vi.fn(),
}));

import * as apiMessage from '@/api/message';
import type { Message } from '@/api/message';

const makeMessage = (id: number, overrides: Partial<Message> = {}): Message => ({
  id,
  to_user_id: 2001,
  type: 'order',
  title: `订单消息 ${id}`,
  content: `内容 ${id}`,
  ref_id: 100 + id,
  ref_type: 'order',
  read: id % 2 === 0,
  created_at: '2026-09-29T10:00:00Z',
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('escort/message/list · 加载与 tab', () => {
  it('mount 时调 listMessages 不带 read（默认 all）', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({ items: [], total: 0, unread: 0 });
    mount(MessageList);
    await flushPromises();
    expect(apiMessage.listMessages).toHaveBeenCalledWith({});
  });

  it('渲染 2 tab（all / unread）', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({ items: [], total: 0, unread: 0 });
    const w = mount(MessageList);
    await flushPromises();

    expect(w.find('[data-testid="escort-message-list-tab-all"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-message-list-tab-unread"]').exists()).toBe(true);
  });

  it('默认 active tab 是 all', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({ items: [], total: 0, unread: 0 });
    const w = mount(MessageList);
    await flushPromises();
    expect(w.find('[data-testid="escort-message-list-tab-all"]').classes()).toContain(
      'escort-message-list__tab--active',
    );
  });

  it('点击 unread tab 重新拉数据并带 ?read=false', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({ items: [], total: 0, unread: 0 });
    const w = mount(MessageList);
    await flushPromises();
    expect(apiMessage.listMessages).toHaveBeenCalledTimes(1);

    await w.find('[data-testid="escort-message-list-tab-unread"]').trigger('click');
    await flushPromises();
    expect(apiMessage.listMessages).toHaveBeenCalledTimes(2);
    expect(apiMessage.listMessages).toHaveBeenLastCalledWith({ read: false });
  });
});

describe('escort/message/list · 渲染', () => {
  it('渲染消息卡（类型 / 标题 / 内容 / 时间 / 未读小红点）', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({
      items: [makeMessage(101, { read: false }), makeMessage(102, { read: true })],
      total: 2,
      unread: 1,
    });
    const w = mount(MessageList);
    await flushPromises();

    expect(w.find('[data-testid="escort-message-list-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-message-list-card-102"]').exists()).toBe(true);
    expect(w.text()).toContain('订单消息 101');
    expect(w.text()).toContain('内容 101');
    expect(w.text()).toContain('2026-09-29 10:00');

    // 未读小红点
    expect(w.find('[data-testid="escort-message-list-dot-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-message-list-dot-102"]').exists()).toBe(false);
  });

  it('未读数显示在 tab 标题', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({ items: [], total: 0, unread: 5 });
    const w = mount(MessageList);
    await flushPromises();
    expect(w.text()).toContain('未读 (5)');
  });

  it('点击消息卡跳 chat?id=', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({
      items: [makeMessage(101)],
      total: 1,
      unread: 0,
    });
    const w = mount(MessageList);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="escort-message-list-card-101"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/message/chat?id=101' });
  });
});

describe('escort/message/list · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiMessage.listMessages).mockReturnValue(new Promise(() => {}));
    const w = mount(MessageList);
    await flushPromises();
    expect(w.find('[data-testid="escort-message-list-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiMessage.listMessages).mockResolvedValue({ items: [], total: 0, unread: 0 });
    const w = mount(MessageList);
    await flushPromises();
    expect(w.find('[data-testid="escort-message-list-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无消息');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiMessage.listMessages).mockRejectedValue(new Error('网络异常'));
    const w = mount(MessageList);
    await flushPromises();
    expect(w.find('[data-testid="escort-message-list-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });
});