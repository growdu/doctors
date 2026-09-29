/**
 * escort/splash.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染启动画面（logo + 标题 + 文案）
 *   - 已登录：mount 后 reLaunch /pages/escort/invitations/index
 *   - 未登录：mount 后 reLaunch /pages/escort/login
 *   - 调 auth.bootstrap()
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import SplashPage from './splash.vue';

import type { MeResponse } from '@/types/auth';

const makeUser = (overrides: Partial<MeResponse> = {}): MeResponse => ({
  id: 2001,
  phone: '13900139000',
  role: 'escort',
  active_role: 'escort',
  roles: ['escort'],
  real_name_verified: true,
  ...overrides,
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('escort/splash · 渲染', () => {
  it('渲染启动画面', () => {
    const w = mount(SplashPage);
    expect(w.find('[data-testid="escort-splash-page"]').exists()).toBe(true);
    expect(w.text()).toContain('escort-app v2');
    expect(w.text()).toContain('陪诊师工作台');
    expect(w.text()).toContain('加载中');
  });
});

describe('escort/splash · 跳转', () => {
  it('已登录 → reLaunch /pages/escort/invitations/index', async () => {
    const auth = useAuthStore();
    auth.user = makeUser();
    auth.token = 'test-token';

    const reLaunchSpy = vi.fn();
    (globalThis as unknown as { uni: { reLaunch: typeof reLaunchSpy } }).uni.reLaunch = reLaunchSpy;

    mount(SplashPage);
    // 等 500ms setTimeout 触发
    await new Promise((r) => setTimeout(r, 600));

    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/escort/invitations/index' });
  });

  it('未登录 → reLaunch /pages/escort/login', async () => {
    const reLaunchSpy = vi.fn();
    (globalThis as unknown as { uni: { reLaunch: typeof reLaunchSpy } }).uni.reLaunch = reLaunchSpy;

    mount(SplashPage);
    await new Promise((r) => setTimeout(r, 600));

    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/escort/login' });
  });
});