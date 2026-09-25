// src/pages/coupons/index.test.js
//
// 优惠券中心单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 5 个 it）：
//   1. mounted → 调 coupon store loadTemplates + loadMine
//   2. 2 个顶部 tab 渲染（领券 / 我的券）+ 切换 tab
//   3. 领券 tab：模板列表渲染 + 「立即领取」点击 → 调 store.claim(id) + 切到「我的券」tab
//   4. 我的券 tab：4 个 status 子 tab 渲染 + filter
//   5. 空状态：loadTemplates 返回 [] → u-empty-stub

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { jest } from '@jest/globals';

// ---- 全局 uni mock
const mockUni = {
  showToast: jest.fn(),
  showModal: jest.fn(),
  navigateTo: jest.fn(),
  redirectTo: jest.fn(),
  reLaunch: jest.fn(),
  showLoading: jest.fn(),
  hideLoading: jest.fn(),
};

beforeAll(() => {
  global.uni = mockUni;
});

afterEach(() => {
  jest.clearAllMocks();
});

// ---- fake coupon store
const fakeCouponStore = {
  templates: [],
  mine: [],
  loading: false,
  loadingMine: false,
  error: null,
  lastClaimedId: null,
  loadTemplates: jest.fn(),
  loadMine: jest.fn(),
  claim: jest.fn(),
  useCoupon: jest.fn(),
  filterByStatus: jest.fn(),
  statusCount: jest.fn(() => ({ unused: 0, used: 0, expired: 0, total: 0 })),
};

jest.doMock('@/stores/coupon.js', () => ({
  useCouponStore: () => fakeCouponStore,
  COUPON_STATUS_UNUSED: 'unused',
  COUPON_STATUS_USED: 'used',
  COUPON_STATUS_EXPIRED: 'expired',
  COUPON_STATUS_LABEL: {
    unused: '未使用',
    used: '已使用',
    expired: '已过期',
  },
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': {
    props: ['title', 'autoBack'],
    template: '<div class="u-navbar-stub" :data-title="title"></div>',
  },
  'u-skeleton': {
    props: ['rows', 'title'],
    template: '<div class="u-skeleton-stub"></div>',
  },
  'u-empty': {
    props: ['text', 'mode'],
    template: '<div class="u-empty-stub" :data-text="text"></div>',
  },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled'],
    emits: ['click'],
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('coupons/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeCouponStore.loadTemplates.mockReset();
    fakeCouponStore.loadMine.mockReset();
    fakeCouponStore.claim.mockReset();

    // 默认：loadTemplates 返回 2 张券
    fakeCouponStore.loadTemplates.mockResolvedValue([
      {
        id: 1,
        name: '新人立减券',
        type: 'amount',
        value: 30,
        threshold: 100,
        stock: 100,
        valid_until: '2026-12-31T23:59:59+08:00',
      },
      {
        id: 2,
        name: '满减券',
        type: 'amount',
        value: 50,
        threshold: 200,
        stock: 50,
        valid_until: '2026-12-31T23:59:59+08:00',
      },
    ]);
    fakeCouponStore.loadMine.mockResolvedValue([]);
    fakeCouponStore.claim.mockResolvedValue({
      id: 100,
      user_id: 7,
      coupon_id: 1,
      status: 'unused',
      coupon: { id: 1, name: '新人立减券', value: 30 },
    });
  });

  it('mounted → 调 coupon store loadTemplates + loadMine', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeCouponStore.loadTemplates).toHaveBeenCalledTimes(1);
    expect(fakeCouponStore.loadTemplates).toHaveBeenCalledWith({});
    expect(fakeCouponStore.loadMine).toHaveBeenCalledTimes(1);
    expect(w.vm.templates).toHaveLength(2);
  });

  it('2 个顶部 tab 渲染（领券 / 我的券）+ 切换 tab', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="top-tab-claim"]').exists()).toBe(true);
    expect(w.find('[data-test="top-tab-mine"]').exists()).toBe(true);

    // 默认在领券 tab
    expect(w.vm.currentTab).toBe('claim');

    // 切到「我的券」tab
    await w.find('[data-test="top-tab-mine"]').trigger('click');
    expect(w.vm.currentTab).toBe('mine');
  });

  it('领券 tab：模板列表渲染 + 「立即领取」点击 → store.claim(id) + 切到「我的券」tab', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    // 模板列表渲染
    const cards = w.findAll('[data-test^="coupon-card-"]');
    expect(cards).toHaveLength(2);
    expect(w.find('[data-test="claim-btn-1"]').exists()).toBe(true);

    // 点击「立即领取」
    await w.find('[data-test="claim-btn-1"]').trigger('click');
    await flushPromises();
    expect(fakeCouponStore.claim).toHaveBeenCalledTimes(1);
    expect(fakeCouponStore.claim).toHaveBeenCalledWith(1);
    expect(w.vm.currentTab).toBe('mine'); // 领取后自动切到「我的券」
    // 切到我的券 tab 后会重新拉
    expect(fakeCouponStore.loadMine).toHaveBeenCalledTimes(2);
    // toast 触发
    expect(mockUni.showToast).toHaveBeenCalled();
  });

  it('我的券 tab：4 个 status 子 tab + filter', async () => {
    fakeCouponStore.loadMine.mockResolvedValue([
      { id: 100, status: 'unused', coupon: { id: 1, name: 'x', value: 30, threshold: 100 }, expires_at: '2026-12-31T23:59:59+08:00' },
      { id: 101, status: 'used',   coupon: { id: 1, name: 'x', value: 30, threshold: 100 }, expires_at: '2026-12-31T23:59:59+08:00' },
      { id: 102, status: 'expired',coupon: { id: 1, name: 'x', value: 30, threshold: 100 }, expires_at: '2026-09-01T23:59:59+08:00' },
    ]);

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    // 切到「我的券」tab
    await w.find('[data-test="top-tab-mine"]').trigger('click');
    await flushPromises();

    // 4 个 status 子 tab
    expect(w.find('[data-test="status-tab-all"]').exists()).toBe(true);
    expect(w.find('[data-test="status-tab-unused"]').exists()).toBe(true);
    expect(w.find('[data-test="status-tab-used"]').exists()).toBe(true);
    expect(w.find('[data-test="status-tab-expired"]').exists()).toBe(true);

    // 默认 all → 3 张卡
    expect(w.findAll('[data-test^="mine-card-"]')).toHaveLength(3);

    // 切到 unused
    await w.find('[data-test="status-tab-unused"]').trigger('click');
    expect(w.vm.currentStatus).toBe('unused');
    expect(w.findAll('[data-test^="mine-card-"]')).toHaveLength(1);
    expect(w.find('[data-test="mine-card-100"]').exists()).toBe(true);

    // 切到 expired
    await w.find('[data-test="status-tab-expired"]').trigger('click');
    expect(w.vm.currentStatus).toBe('expired');
    expect(w.findAll('[data-test^="mine-card-"]')).toHaveLength(1);
    expect(w.find('[data-test="mine-card-102"]').exists()).toBe(true);
  });

  it('空状态：loadTemplates 返回 [] → u-empty-stub「暂无可领取的券」', async () => {
    fakeCouponStore.loadTemplates.mockResolvedValueOnce([]);

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无可领取的券');
    expect(w.findAll('[data-test^="coupon-card-"]')).toHaveLength(0);
  });

  it('错误态：loadTemplates 抛错 → error-state + 重试按钮', async () => {
    fakeCouponStore.loadTemplates.mockRejectedValueOnce(new Error('boom'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('加载失败');
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
    expect(w.vm.loadError).toBe(true);
  });

  it('我的券 空状态：filter 后无结果 → u-empty-stub「暂无该状态的券」', async () => {
    fakeCouponStore.loadMine.mockResolvedValue([
      { id: 100, status: 'unused', coupon: { id: 1, name: 'x', value: 30, threshold: 100 }, expires_at: '2026-12-31T23:59:59+08:00' },
    ]);

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();
    await w.find('[data-test="top-tab-mine"]').trigger('click');
    await flushPromises();

    // 切到 expired（无此状态）→ 空态
    await w.find('[data-test="status-tab-expired"]').trigger('click');
    expect(w.vm.filteredMine).toEqual([]);
    const empty = w.find('[data-test="empty-mine"]');
    expect(empty.exists()).toBe(true);
  });
});