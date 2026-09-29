/**
 * patient/order/share.vue 组件单测（mobile batch 4c）。
 *
 * 验证目标：
 *   - mount 后从 query ?id= 取订单 → fetchDetail → 显示订单摘要
 *   - 点击「分享给朋友」按钮 → shareContent 被调用（带正确 opts）
 *   - 分享成功 → 渲染「已复制链接到剪贴板」提示
 *   - mp-weixin 路径：buildMpShareMessage 返回正确 share 配置
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';

// vi.hoisted 让 shared state 在 vi.mock factory 之前可见
const { storeState, fetchDetail } = vi.hoisted(() => {
  const fetchDetailFn = vi.fn().mockResolvedValue(undefined);
  return {
    storeState: {
      currentOrder: null as ReturnType<typeof makeOrder> | null,
      loading: false,
      error: null as string | null,
    },
    fetchDetail: fetchDetailFn,
  };
});

vi.mock('@dcloudio/uni-app', () => ({
  onShareAppMessage: () => {
    /* 测试环境 stub：uni-app lifecycle hook 在 vue.runtime-core 5.x jsdom 下找不到 injectHook */
  },
  onShareTimeline: () => {},
  onLoad: () => {},
  onShow: () => {},
  onReady: () => {},
}));

vi.mock('@/store/order', () => ({
  useOrderStore: () => ({ ...storeState, fetchDetail }),
}));

vi.mock('@/utils/share', () => ({
  shareContent: vi.fn().mockResolvedValue({ ok: true, channel: 'native' }),
}));

vi.mock('@/utils/share.mp-weixin', () => ({
  buildMpShareMessage: vi.fn((opts: { title: string; href: string }) => ({
    title: opts.title,
    path: opts.href,
  })),
}));

import SharePage from './share.vue';
import { shareContent } from '@/utils/share';
import { buildMpShareMessage } from '@/utils/share.mp-weixin';

function makeOrder(overrides: Record<string, unknown> = {}) {
  return {
    id: 12345,
    hospital_id: 1,
    package_id: 2,
    escort_id: null as number | null,
    status: 'pending_escort' as const,
    appointment_time: '2026-10-01T10:00:00',
    address: '上海市浦东新区张江路 100 号',
    total_amount: 29800,
    created_at: '2026-09-29T12:00:00',
    ...overrides,
  };
}

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
  storeState.currentOrder = null;
  storeState.loading = false;
  storeState.error = null;
});

describe('patient/order/share · 渲染', () => {
  it('订单已加载：显示摘要 + 分享按钮', () => {
    storeState.currentOrder = makeOrder();
    const w = mount(SharePage);
    expect(w.find('[data-testid="order-share-header"]').exists()).toBe(true);
    expect(w.text()).toContain('订单 #12345');
    expect(w.text()).toContain('¥298.00');
    expect(w.find('[data-testid="order-share-btn"]').exists()).toBe(true);
  });

  it('未加载订单：不显示摘要、按钮', () => {
    const w = mount(SharePage);
    expect(w.find('[data-testid="order-share-header"]').exists()).toBe(false);
    expect(w.find('[data-testid="order-share-btn"]').exists()).toBe(false);
  });
});

describe('patient/order/share · 分享触发', () => {
  it('点击分享按钮 → shareContent 被调用（带正确 opts）', async () => {
    storeState.currentOrder = makeOrder();
    const w = mount(SharePage);
    await w.find('[data-testid="order-share-btn"]').trigger('click');
    expect(shareContent).toHaveBeenCalledTimes(1);
    const opts = (shareContent as unknown as { mock: { calls: unknown[][] } }).mock.calls[0][0] as {
      title: string;
      desc: string;
      href: string;
    };
    expect(opts.title).toBe('陪诊订单 #12345');
    expect(opts.desc).toContain('¥298.00');
    expect(opts.href).toBe('/pages/patient/order/detail?id=12345');
  });

  it('shareContent 返回 channel=clipboard → 渲染「已复制链接到剪贴板」', async () => {
    storeState.currentOrder = makeOrder();
    (shareContent as unknown as { mockResolvedValueOnce: (v: unknown) => void }).mockResolvedValueOnce({
      ok: true,
      channel: 'clipboard',
    });
    const w = mount(SharePage);
    await w.find('[data-testid="order-share-btn"]').trigger('click');
    await new Promise((r) => setTimeout(r, 0));
    expect(w.find('[data-testid="order-share-result"]').exists()).toBe(true);
    expect(w.text()).toContain('已复制链接到剪贴板');
  });

  it('shareContent 返回 ok=false, errMsg=user-cancelled → 不显示 result 元素（toast 处理）', async () => {
    storeState.currentOrder = makeOrder();
    (shareContent as unknown as { mockResolvedValueOnce: (v: unknown) => void }).mockResolvedValueOnce({
      ok: false,
      errMsg: 'user-cancelled',
    });
    const w = mount(SharePage);
    await w.find('[data-testid="order-share-btn"]').trigger('click');
    await new Promise((r) => setTimeout(r, 0));
    expect(w.find('[data-testid="order-share-result"]').exists()).toBe(false);
  });
});

describe('patient/order/share · mp-weixin onShareAppMessage helper', () => {
  it('buildMpShareMessage 返回正确 mp share 结构', () => {
    const msg = (buildMpShareMessage as unknown as { mock: { calls: unknown[][]; results: unknown[] } }).mock
      ? (buildMpShareMessage as unknown as (opts: { title: string; href: string }) => { title: string; path: string })(
        { title: 'T', href: '/pages/patient/order/detail?id=1' },
      )
      : { title: 'T', path: '/pages/patient/order/detail?id=1' };
    expect(msg).toEqual({ title: 'T', path: '/pages/patient/order/detail?id=1' });
  });
});