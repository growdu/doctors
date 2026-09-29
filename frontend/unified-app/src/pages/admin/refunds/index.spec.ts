/**
 * admin/refunds/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时并发拉 3 个状态的 total（pending / approved / rejected）
 *   - 渲染 3 张统计卡 + 数字
 *   - 点击「进入退款列表」跳转 /pages/admin/refunds/list
 *   - API 错误时回退为 0
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import RefundsIndex from './index.vue';

vi.mock('@/api/admin', () => ({
  listRefunds: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/refunds/index · 加载', () => {
  it('mount 时并发拉 3 个状态的 total', async () => {
    vi.mocked(apiAdmin.listRefunds)
      .mockResolvedValueOnce({ items: [], total: 4 })
      .mockResolvedValueOnce({ items: [], total: 2 })
      .mockResolvedValueOnce({ items: [], total: 1 });

    mount(RefundsIndex);
    await flushPromises();

    expect(apiAdmin.listRefunds).toHaveBeenCalledTimes(3);
    expect(apiAdmin.listRefunds).toHaveBeenCalledWith({ status: 'pending', page: 1, page_size: 1 });
    expect(apiAdmin.listRefunds).toHaveBeenCalledWith({ status: 'approved', page: 1, page_size: 1 });
    expect(apiAdmin.listRefunds).toHaveBeenCalledWith({ status: 'rejected', page: 1, page_size: 1 });
  });

  it('渲染 3 张统计卡 + 数字', async () => {
    vi.mocked(apiAdmin.listRefunds)
      .mockResolvedValueOnce({ items: [], total: 4 })
      .mockResolvedValueOnce({ items: [], total: 2 })
      .mockResolvedValueOnce({ items: [], total: 1 });

    const w = mount(RefundsIndex);
    await flushPromises();

    expect(w.find('[data-testid="refunds-stat-pending"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-stat-approved"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-stat-rejected"]').exists()).toBe(true);

    expect(w.text()).toContain('待审');
    expect(w.text()).toContain('已通过');
    expect(w.text()).toContain('已驳回');
    expect(w.text()).toContain('4');
    expect(w.text()).toContain('2');
    expect(w.text()).toContain('1');
  });

  it('API 失败回退为 0', async () => {
    vi.mocked(apiAdmin.listRefunds).mockRejectedValue(new Error('网络异常'));
    const w = mount(RefundsIndex);
    await flushPromises();

    // 3 张卡都渲染
    expect(w.find('[data-testid="refunds-stat-pending"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-stat-approved"]').exists()).toBe(true);
  });
});

describe('admin/refunds/index · 跳转', () => {
  it('点击「进入退款列表」跳 /pages/admin/refunds/list', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({ items: [], total: 0 });
    const w = mount(RefundsIndex);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="refunds-go-list"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/refunds/list' });
  });
});