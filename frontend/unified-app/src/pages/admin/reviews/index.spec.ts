/**
 * admin/reviews/index.vue 组件单测（vitest + Vue Test Utils）。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AdminReviews from './index.vue';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/reviews/index · 渲染', () => {
  it('渲染标题 + 提示说明', () => {
    const w = mount(AdminReviews);
    expect(w.text()).toContain('评价管理');
    expect(w.find('[data-testid="admin-reviews-note"]').exists()).toBe(true);
  });

  it('渲染 2 个入口（moderate + patient-view）', () => {
    const w = mount(AdminReviews);
    expect(w.find('[data-testid="admin-reviews-entry-moderate"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-reviews-entry-patient-view"]').exists()).toBe(true);
  });
});

describe('admin/reviews/index · 跳转', () => {
  it('点击 patient-view 跳 patient 域', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.showToast = () => {};

    const w = mount(AdminReviews);
    await w.find('[data-testid="admin-reviews-entry-patient-view"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/patient/reviews/index' });
  });

  it('点击 moderate 弹 toast 不调 navigate', async () => {
    const navSpy = vi.fn();
    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(AdminReviews);
    await w.find('[data-testid="admin-reviews-entry-moderate"]').trigger('click');

    expect(navSpy).not.toHaveBeenCalled();
    expect(toastSpy).toHaveBeenCalledWith({ title: '功能开发中', icon: 'none' });
  });
});