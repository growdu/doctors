/**
 * admin/escorts/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染陪诊师管理首页标题
 *   - 渲染「待审核队列」入口卡片
 *   - 点击入口跳 /pages/admin/escorts/pending-audit
 *
 * 简单页：4 cases。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import EscortsIndex from './index.vue';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/escorts/index · 渲染', () => {
  it('渲染标题与副标题', () => {
    const w = mount(EscortsIndex);
    expect(w.text()).toContain('陪诊师管理');
    expect(w.text()).toContain('审核');
    expect(w.find('[data-testid="admin-escorts-index"]').exists()).toBe(true);
  });

  it('渲染「待审核队列」入口', () => {
    const w = mount(EscortsIndex);
    expect(w.find('[data-testid="admin-escorts-entry-pending-audit"]').exists()).toBe(true);
    expect(w.text()).toContain('待审核队列');
    expect(w.text()).toContain('查看待审陪诊师并执行通过/拒绝');
  });
});

describe('admin/escorts/index · 跳转', () => {
  it('点击入口跳 pending-audit', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(EscortsIndex);
    await w.find('[data-testid="admin-escorts-entry-pending-audit"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/escorts/pending-audit' });
  });
});