/**
 * admin/profile/index.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - authStore.user 存在时渲染基本信息 + 角色信息
 *   - 手机号脱敏（138****1234）
 *   - 实名 / 未实名状态显示
 *   - 当前角色 + 所有角色 chip 渲染（active 高亮）
 *   - 退出登录按钮调 auth.onUnauthorized
 *   - authStore.user 不存在时显示空态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import ProfilePage from './index.vue';

import type { MeResponse } from '@/types/auth';

const makeUser = (overrides: Partial<MeResponse> = {}): MeResponse => ({
  id: 1001,
  phone: '13800138000',
  role: 'super_admin',
  active_role: 'super_admin',
  roles: ['super_admin', 'audit_admin', 'order_admin'],
  real_name_verified: true,
  ...overrides,
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('admin/profile/index · 渲染', () => {
  it('authStore.user 存在时渲染基本信息 + 角色信息', () => {
    const auth = useAuthStore();
    auth.user = makeUser();

    const w = mount(ProfilePage);

    expect(w.find('[data-testid="admin-profile-content"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-profile-info-card"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-profile-roles-card"]').exists()).toBe(true);

    expect(w.text()).toContain('#1001');
    expect(w.text()).toContain('138****8000');
    expect(w.text()).toContain('已实名');
    expect(w.text()).toContain('超级管理员');
  });

  it('未实名显示「未实名」', () => {
    const auth = useAuthStore();
    auth.user = makeUser({ real_name_verified: false });

    const w = mount(ProfilePage);
    expect(w.find('[data-testid="admin-profile-unverified"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-profile-verified"]').exists()).toBe(false);
  });

  it('所有角色 chip 渲染 + 当前角色高亮', () => {
    const auth = useAuthStore();
    auth.user = makeUser({
      active_role: 'audit_admin',
      roles: ['super_admin', 'audit_admin', 'order_admin'],
    });

    const w = mount(ProfilePage);

    expect(w.find('[data-testid="admin-profile-role-super_admin"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-profile-role-audit_admin"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-profile-role-order_admin"]').exists()).toBe(true);

    // 当前角色显示在「当前角色」字段
    expect(w.find('[data-testid="admin-profile-active-role"]').text()).toContain('审核管理员');

    // chip --active
    expect(w.find('[data-testid="admin-profile-role-audit_admin"]').classes()).toContain(
      'admin-profile__role-chip--active',
    );
    expect(w.find('[data-testid="admin-profile-role-super_admin"]').classes()).not.toContain(
      'admin-profile__role-chip--active',
    );
  });

  it('手机号非 11 位不脱敏', () => {
    const auth = useAuthStore();
    auth.user = makeUser({ phone: '12345' });

    const w = mount(ProfilePage);
    expect(w.text()).toContain('12345');
  });
});

describe('admin/profile/index · 退出登录', () => {
  it('点击「退出登录」调 auth.onUnauthorized', async () => {
    const auth = useAuthStore();
    auth.user = makeUser();
    const onUnauthSpy = vi.spyOn(auth, 'onUnauthorized');

    const w = mount(ProfilePage);
    await w.find('[data-testid="admin-profile-logout"]').trigger('click');

    expect(onUnauthSpy).toHaveBeenCalled();
  });
});

describe('admin/profile/index · 空态', () => {
  it('authStore.user 不存在时显示空态', () => {
    const auth = useAuthStore();
    auth.user = null;

    const w = mount(ProfilePage);
    expect(w.find('[data-testid="admin-profile-empty"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-profile-content"]').exists()).toBe(false);
    expect(w.text()).toContain('请先登录');
  });
});