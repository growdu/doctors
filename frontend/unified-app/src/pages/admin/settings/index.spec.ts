/**
 * admin/settings/index.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - 渲染系统信息卡（版本 / 环境 / 登录状态 / 当前角色）
 *   - 登录态映射：isAuthed=true 显示「已登录」
 *   - 「返回首页」按钮调 uni.reLaunch
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import SettingsPage from './index.vue';

import type { MeResponse } from '@/types/auth';

const makeUser = (overrides: Partial<MeResponse> = {}): MeResponse => ({
  id: 1001,
  phone: '13800138000',
  role: 'super_admin',
  active_role: 'super_admin',
  roles: ['super_admin'],
  real_name_verified: true,
  ...overrides,
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('admin/settings/index · 渲染', () => {
  it('渲染系统信息卡 + 提示说明', () => {
    const auth = useAuthStore();
    auth.user = makeUser();
    auth.token = 'test-token';

    const w = mount(SettingsPage);
    expect(w.find('[data-testid="admin-settings-system-card"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-settings-note"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-settings-version"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-settings-env"]').exists()).toBe(true);
  });

  it('已登录显示「已登录」+ 当前角色', () => {
    const auth = useAuthStore();
    auth.user = makeUser({ active_role: 'order_admin' });
    auth.token = 'test-token';

    const w = mount(SettingsPage);
    expect(w.find('[data-testid="admin-settings-online"]').exists()).toBe(true);
    expect(w.text()).toContain('order_admin');
  });

  it('未登录显示「未登录」', () => {
    const auth = useAuthStore();
    auth.user = null;
    auth.token = '';

    const w = mount(SettingsPage);
    expect(w.find('[data-testid="admin-settings-offline"]').exists()).toBe(true);
    expect(w.text()).toContain('未登录');
  });
});

describe('admin/settings/index · 操作', () => {
  it('点击「返回首页」调 uni.reLaunch', async () => {
    const auth = useAuthStore();
    auth.user = makeUser();
    auth.token = 'test-token';

    const reLaunchSpy = vi.fn();
    (globalThis as unknown as { uni: { reLaunch: typeof reLaunchSpy } }).uni.reLaunch = reLaunchSpy;

    const w = mount(SettingsPage);
    await w.find('[data-testid="admin-settings-relaunch"]').trigger('click');

    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/home/index' });
  });
});