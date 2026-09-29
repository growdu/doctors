/**
 * escort/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染 3 个聚合入口（invitations / orders / profile）
 *   - 点击入口跳对应 page
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import EscortHome from './index.vue';

beforeEach(() => {
  setActivePinia(createPinia());
  const auth = useAuthStore();
  auth.user = {
    id: 1,
    phone: '13800138000',
    role: 'escort',
    active_role: 'escort',
    roles: ['escort'],
    real_name_verified: true,
  };
  vi.clearAllMocks();
});

describe('escort/index · 渲染', () => {
  it('渲染 3 个入口（invitations / orders / profile）', () => {
    const w = mount(EscortHome);
    expect(w.find('[data-testid="escort-home-entry-invitations"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-orders"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-profile"]').exists()).toBe(true);

    expect(w.text()).toContain('陪诊师工作台');
    expect(w.text()).toContain('抢单池');
    expect(w.text()).toContain('我的任务');
    expect(w.text()).toContain('个人中心');
  });
});

describe('escort/index · 跳转', () => {
  it('点击「抢单池」跳 invitations/index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(EscortHome);
    await w.find('[data-testid="escort-home-entry-invitations"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/invitations/index' });
  });

  it('点击「我的任务」跳 orders/index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(EscortHome);
    await w.find('[data-testid="escort-home-entry-orders"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/orders/index' });
  });

  it('点击「个人中心」跳 profile/index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    const w = mount(EscortHome);
    await w.find('[data-testid="escort-home-entry-profile"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/profile/index' });
  });
});