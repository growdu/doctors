/**
 * admin/messages/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染标题
 *   - 加载成功：5 类型聚合 + 消息列表
 *   - 加载失败：空态 + 重试按钮
 *   - 点击重试重新调 onLoad
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import * as msgApi from '@/api/message';
import type { Message } from '@/api/message';
import AdminMessages from './index.vue';

const msg = (id: number, type: Message['type'], read = false): Message => ({
  id,
  to_user_id: 1,
  type,
  title: `消息 #${id}`,
  content: `内容 ${id}`,
  ref_id: null,
  ref_type: null,
  read,
  created_at: '2026-10-01T10:00:00Z',
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('admin/messages/index · 渲染', () => {
  it('渲染标题', () => {
    vi.spyOn(msgApi, 'listMessages').mockResolvedValue({ items: [], total: 0, unread: 0 });

    const w = mount(AdminMessages);
    expect(w.text()).toContain('站内信管理');
  });

  it('加载成功渲染 5 类型 + 总数 + 未读', async () => {
    vi.spyOn(msgApi, 'listMessages').mockResolvedValue({
      items: [
        msg(1, 'system', true),
        msg(2, 'order', false),
        msg(3, 'payment', false),
      ],
      total: 3,
      unread: 2,
    });

    const w = mount(AdminMessages);
    await flushPromises();

    expect(w.find('[data-testid="admin-messages-stat-total"]').text()).toContain('3');
    expect(w.find('[data-testid="admin-messages-stat-unread"]').text()).toContain('2');
    expect(w.find('[data-testid="admin-messages-type-system"]').text()).toContain('1');
    expect(w.find('[data-testid="admin-messages-type-order"]').text()).toContain('1');
    expect(w.find('[data-testid="admin-messages-type-payment"]').text()).toContain('1');
  });

  it('渲染消息列表（截前 20 条）', async () => {
    vi.spyOn(msgApi, 'listMessages').mockResolvedValue({
      items: [msg(1, 'system'), msg(2, 'order')],
      total: 2,
      unread: 2,
    });

    const w = mount(AdminMessages);
    await flushPromises();

    expect(w.find('[data-testid="admin-message-1"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-message-2"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-message-unread-1"]').exists()).toBe(true);
  });
});

describe('admin/messages/index · 失败回退', () => {
  it('加载失败渲染空态 + 重试按钮', async () => {
    vi.spyOn(msgApi, 'listMessages').mockRejectedValue(new Error('network'));

    const w = mount(AdminMessages);
    await flushPromises();

    expect(w.find('[data-testid="admin-messages-error"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-messages-retry"]').exists()).toBe(true);
  });

  it('点击重试重新调 onLoad', async () => {
    const listSpy = vi.spyOn(msgApi, 'listMessages')
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce({ items: [], total: 0, unread: 0 });

    const w = mount(AdminMessages);
    await flushPromises();

    await w.find('[data-testid="admin-messages-retry"]').trigger('click');
    await flushPromises();

    expect(listSpy).toHaveBeenCalledTimes(2);
  });
});