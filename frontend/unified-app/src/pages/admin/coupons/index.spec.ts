/**
 * admin/coupons/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染标题 + 提示说明
 *   - 渲染 2 个入口卡片（create + patient-view）
 *   - 点击 patient-view 跳 patient 域
 *   - 点击 create 占位弹 toast 不调 navigate
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AdminCoupons from './index.vue';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/coupons/index · 渲染', () => {
  it('渲染标题 + 提示说明', () => {
    const w = mount(AdminCoupons);
    expect(w.text()).toContain('优惠券管理');
    expect(w.find('[data-testid="admin-coupons-note"]').exists()).toBe(true);
    expect(w.text()).toContain('v2 后端暂未上线');
  });

  it('渲染 2 个入口（create + patient-view）', () => {
    const w = mount(AdminCoupons);
    expect(w.find('[data-testid="admin-coupons-entry-create"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-coupons-entry-patient-view"]').exists()).toBe(true);
    expect(w.text()).toContain('创建优惠券');
    expect(w.text()).toContain('用户视角优惠券');
  });
});

describe('admin/coupons/index · 跳转', () => {
  it('点击「用户视角优惠券」跳 patient 域', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: () => void } }).uni.showToast = () => {};

    const w = mount(AdminCoupons);
    await w.find('[data-testid="admin-coupons-entry-patient-view"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/patient/coupons/index' });
  });

  it('点击「创建优惠券」占位弹 toast，不调 navigate', async () => {
    const navSpy = vi.fn();
    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.navigateTo = navSpy;
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy; showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(AdminCoupons);
    await w.find('[data-testid="admin-coupons-entry-create"]').trigger('click');

    expect(navSpy).not.toHaveBeenCalled();
    expect(toastSpy).toHaveBeenCalledWith({ title: '功能开发中', icon: 'none' });
  });
});