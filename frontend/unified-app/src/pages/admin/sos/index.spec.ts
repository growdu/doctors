/**
 * admin/sos/index.vue 组件单测（vitest + Vue Test Utils）。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AdminSos from './index.vue';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/sos/index · 渲染', () => {
  it('渲染标题 + 提示说明', () => {
    const w = mount(AdminSos);
    expect(w.text()).toContain('SOS 管理');
    expect(w.find('[data-testid="admin-sos-note"]').exists()).toBe(true);
    expect(w.text()).toContain('sos-service');
  });

  it('渲染 2 个入口（list + patient-view）', () => {
    const w = mount(AdminSos);
    expect(w.find('[data-testid="admin-sos-entry-list"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-sos-entry-patient-view"]').exists()).toBe(true);
  });
});

describe('admin/sos/index · 跳转', () => {
  it('点击 patient-view 跳 patient 域', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.showToast = () => {};

    const w = mount(AdminSos);
    await w.find('[data-testid="admin-sos-entry-patient-view"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/patient/sos/trigger' });
  });

  it('点击 list 弹 toast 不调 navigate', async () => {
    const navSpy = vi.fn();
    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(AdminSos);
    await w.find('[data-testid="admin-sos-entry-list"]').trigger('click');

    expect(navSpy).not.toHaveBeenCalled();
    expect(toastSpy).toHaveBeenCalledWith({ title: '功能开发中', icon: 'none' });
  });
});