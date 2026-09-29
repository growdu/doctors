/**
 * admin/messages/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染标题 + 提示说明
 *   - 渲染 2 个入口卡片（broadcast + patient-view）
 *   - 点击 patient-view 跳 patient 域
 *   - 点击 broadcast 弹 toast 不调 navigate
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AdminMessages from './index.vue';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/messages/index · 渲染', () => {
  it('渲染标题 + 提示说明', () => {
    const w = mount(AdminMessages);
    expect(w.text()).toContain('站内信管理');
    expect(w.find('[data-testid="admin-messages-note"]').exists()).toBe(true);
    expect(w.text()).toContain('群发 / 模板管理');
  });

  it('渲染 2 个入口（broadcast + patient-view）', () => {
    const w = mount(AdminMessages);
    expect(w.find('[data-testid="admin-messages-entry-broadcast"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-messages-entry-patient-view"]').exists()).toBe(true);
    expect(w.text()).toContain('群发消息');
    expect(w.text()).toContain('用户视角消息');
  });
});

describe('admin/messages/index · 跳转', () => {
  it('点击「用户视角消息」跳 patient 域', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.showToast = () => {};

    const w = mount(AdminMessages);
    await w.find('[data-testid="admin-messages-entry-patient-view"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/patient/message/list' });
  });

  it('点击「群发消息」占位弹 toast，不调 navigate', async () => {
    const navSpy = vi.fn();
    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(AdminMessages);
    await w.find('[data-testid="admin-messages-entry-broadcast"]').trigger('click');

    expect(navSpy).not.toHaveBeenCalled();
    expect(toastSpy).toHaveBeenCalledWith({ title: '功能开发中', icon: 'none' });
  });
});