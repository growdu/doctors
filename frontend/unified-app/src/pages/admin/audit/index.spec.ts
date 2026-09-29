/**
 * admin/audit/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染标题 + 提示说明
 *   - 渲染 3 个聚合入口卡片
 *   - 点击入口跳对应页面
 *
 * 简单页：4 cases。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AuditPage from './index.vue';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/audit/index · 渲染', () => {
  it('渲染标题 + 提示说明', () => {
    const w = mount(AuditPage);
    expect(w.text()).toContain('审计管理');
    expect(w.text()).toContain('审核 / 审批 / 处理历史');
    expect(w.find('[data-testid="admin-audit-note"]').exists()).toBe(true);
    expect(w.text()).toContain('timeline');
  });

  it('渲染 3 个聚合入口', () => {
    const w = mount(AuditPage);
    expect(w.find('[data-testid="admin-audit-entry-escorts"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-audit-entry-refunds"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-audit-entry-work-orders"]').exists()).toBe(true);

    expect(w.text()).toContain('陪诊师审核');
    expect(w.text()).toContain('退款审批');
    expect(w.text()).toContain('客服工单');
  });
});

describe('admin/audit/index · 跳转', () => {
  it('点击 escorts 入口跳 pending-audit', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(AuditPage);
    await w.find('[data-testid="admin-audit-entry-escorts"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/escorts/pending-audit' });
  });

  it('点击 refunds 入口跳 list', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(AuditPage);
    await w.find('[data-testid="admin-audit-entry-refunds"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/refunds/list' });
  });

  it('点击 work-orders 入口跳 index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(AuditPage);
    await w.find('[data-testid="admin-audit-entry-work-orders"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/work-orders/index' });
  });
});