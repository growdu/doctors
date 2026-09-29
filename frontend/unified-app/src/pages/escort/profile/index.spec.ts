/**
 * escort/profile/index.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - authStore.user 存在时渲染基本信息 + 角色信息
 *   - 手机号脱敏
 *   - 实名 / 未实名状态显示
 *   - 当前角色 + 所有角色 chip 渲染
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
  id: 2001,
  phone: '13900139000',
  role: 'escort',
  active_role: 'escort',
  roles: ['escort', 'patient'],
  real_name_verified: true,
  ...overrides,
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('escort/profile · 基本渲染', () => {
  it('authStore.user 存在时渲染基本信息 + 角色信息', () => {
    const auth = useAuthStore();
    auth.user = makeUser();

    const w = mount(ProfilePage);
    expect(w.find('[data-testid="escort-profile-content"]').exists()).toBe(true);
    expect(w.text()).toContain('#2001');
    expect(w.text()).toContain('139****9000');
    expect(w.text()).toContain('已实名');
    expect(w.text()).toContain('escort');
  });

  it('未实名显示「未实名」', () => {
    const auth = useAuthStore();
    auth.user = makeUser({ real_name_verified: false });

    const w = mount(ProfilePage);
    expect(w.find('[data-testid="escort-profile-unverified"]').exists()).toBe(true);
  });

  it('所有角色 chip 渲染 + 当前角色高亮', () => {
    const auth = useAuthStore();
    auth.user = makeUser({
      active_role: 'patient',
      roles: ['escort', 'patient'],
    });

    const w = mount(ProfilePage);
    expect(w.find('[data-testid="escort-profile-role-escort"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-profile-role-patient"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-profile-role-patient"]').classes()).toContain(
      'escort-profile__role-chip--active',
    );
  });
});

describe('escort/profile · 退出登录', () => {
  it('点击「退出登录」调 auth.onUnauthorized', async () => {
    const auth = useAuthStore();
    auth.user = makeUser();
    const onUnauthSpy = vi.spyOn(auth, 'onUnauthorized');

    const w = mount(ProfilePage);
    await w.find('[data-testid="escort-profile-logout"]').trigger('click');

    expect(onUnauthSpy).toHaveBeenCalled();
  });
});

describe('escort/profile · 空态', () => {
  it('authStore.user 不存在时显示空态', () => {
    const auth = useAuthStore();
    auth.user = null;

    const w = mount(ProfilePage);
    expect(w.find('[data-testid="escort-profile-empty"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-profile-content"]').exists()).toBe(false);
    expect(w.text()).toContain('请先登录');
  });
});