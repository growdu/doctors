/**
 * escort/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染 6 个聚合入口（invitations / orders / wallet / availability / training / profile）
 *   - 点击入口跳对应 page
 *   - 钱包余额卡加载成功（mock getEscortWallet）
 *   - 钱包余额卡加载失败降级（不阻塞入口列表）
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import * as walletApi from '@/api/wallet';
import EscortHome from './index.vue';

beforeEach(() => {
  setActivePinia(createPinia());
  const auth = useAuthStore();
  auth.token = 'test-token';
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
  it('渲染 6 个入口（invitations / orders / wallet / availability / training / profile）', () => {
    vi.spyOn(walletApi, 'getEscortWallet').mockResolvedValue({
      user_id: 1,
      balance: 0,
      frozen: 0,
      total_earned: 0,
      total_withdrawn: 0,
      updated_at: '',
    });

    const w = mount(EscortHome);
    expect(w.find('[data-testid="escort-home-entry-invitations"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-orders"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-wallet"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-availability"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-training"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-profile"]').exists()).toBe(true);

    expect(w.text()).toContain('陪诊师工作台');
    expect(w.text()).toContain('抢单池');
    expect(w.text()).toContain('我的任务');
    expect(w.text()).toContain('我的钱包');
    expect(w.text()).toContain('可接时段');
    expect(w.text()).toContain('培训记录');
    expect(w.text()).toContain('个人中心');
  });
});

describe('escort/index · 跳转', () => {
  it('点击「抢单池」跳 invitations/index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;
    vi.spyOn(walletApi, 'getEscortWallet').mockResolvedValue({
      user_id: 1, balance: 0, frozen: 0, total_earned: 0, total_withdrawn: 0, updated_at: '',
    });

    const w = mount(EscortHome);
    await w.find('[data-testid="escort-home-entry-invitations"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/invitations/index' });
  });

  it('点击「我的任务」跳 orders/index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;
    vi.spyOn(walletApi, 'getEscortWallet').mockResolvedValue({
      user_id: 1, balance: 0, frozen: 0, total_earned: 0, total_withdrawn: 0, updated_at: '',
    });

    const w = mount(EscortHome);
    await w.find('[data-testid="escort-home-entry-orders"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/orders/index' });
  });

  it('点击「我的钱包」跳 wallet/index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;
    vi.spyOn(walletApi, 'getEscortWallet').mockResolvedValue({
      user_id: 1, balance: 0, frozen: 0, total_earned: 0, total_withdrawn: 0, updated_at: '',
    });

    const w = mount(EscortHome);
    await w.find('[data-testid="escort-home-entry-wallet"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/wallet/index' });
  });

  it('点击「个人中心」跳 profile/index', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;
    vi.spyOn(walletApi, 'getEscortWallet').mockResolvedValue({
      user_id: 1, balance: 0, frozen: 0, total_earned: 0, total_withdrawn: 0, updated_at: '',
    });

    const w = mount(EscortHome);
    await w.find('[data-testid="escort-home-entry-profile"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/profile/index' });
  });
});

describe('escort/index · 钱包', () => {
  it('加载成功显示可用余额（分 → 元）', async () => {
    vi.spyOn(walletApi, 'getEscortWallet').mockResolvedValue({
      user_id: 1,
      balance: 29900,
      frozen: 1000,
      total_earned: 100000,
      total_withdrawn: 50000,
      updated_at: '',
    });

    const w = mount(EscortHome);
    await flushPromises();

    expect(w.find('[data-testid="escort-home-wallet-balance"]').text()).toBe('¥299.00');
    expect(w.text()).toContain('¥10.00');
    expect(w.text()).toContain('¥1000.00');
  });

  it('加载失败降级为空态（不抛错 + 入口列表仍渲染）', async () => {
    vi.spyOn(walletApi, 'getEscortWallet').mockRejectedValue(new Error('network'));

    const w = mount(EscortHome);
    await flushPromises();

    // 入口列表仍渲染
    expect(w.find('[data-testid="escort-home-entry-invitations"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-home-entry-orders"]').exists()).toBe(true);
    // 余额空态
    expect(w.text()).toContain('暂未获取余额');
    expect(w.find('[data-testid="escort-home-wallet-fallback-link"]').exists()).toBe(true);
  });
});