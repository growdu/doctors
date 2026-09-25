// src/pages/profile/index.test.js
//
// 个人中心页单测 —— @vue/test-utils mount + mock stores。
//
// 测试点（brief 要求 5 个 it）：
//   1. mounted → 调 authStore.fetchMe() + orderStore.loadList({})
//   2. hero 渲染：昵称 + 手机号脱敏
//   3. 4 个订单状态 tile 渲染 + 点击 → uni.navigateTo /pages/order/index?status=xxx
//   4. 设置 menu 渲染（5 项）+ 点击 → uni.navigateTo
//   5. 订单数 badge：loadList 返回订单 → tile 上显示数字

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

// ---- fake auth store
const fakeAuthStore = {
  user: null,
  isLoggedIn: false,
  fetchMe: jest.fn(),
};

jest.doMock('@/stores/auth.js', () => ({
  useAuthStore: () => fakeAuthStore,
}));

// ---- fake order store
const fakeOrderStore = {
  list: [],
  loadList: jest.fn(),
};

jest.doMock('@/stores/order.js', () => ({
  useOrderStore: () => fakeOrderStore,
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
    props: ['rows', 'title', 'avatar'],
    template: '<div class="u-skeleton-stub"></div>',
  },
  'u-empty': {
    props: ['text', 'mode'],
    template: '<div class="u-empty-stub" :data-text="text"></div>',
  },
  'u-button': {
    props: ['type', 'size', 'plain'],
    template:
      '<button class="u-button-stub" :data-type="type" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('profile/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAuthStore.user = null;
    fakeAuthStore.isLoggedIn = false;
    fakeAuthStore.fetchMe.mockReset();
    fakeAuthStore.fetchMe.mockResolvedValue({
      id: 7,
      nickname: '张三',
      phone: '13800138000',
      avatar_url: '',
    });

    fakeOrderStore.loadList.mockReset();
    fakeOrderStore.loadList.mockResolvedValue([]);
  });

  it('mounted → 调 authStore.fetchMe + orderStore.loadList({})', async () => {
    fakeAuthStore.isLoggedIn = true;
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeAuthStore.fetchMe).toHaveBeenCalledTimes(1);
    expect(fakeOrderStore.loadList).toHaveBeenCalledTimes(1);
    expect(fakeOrderStore.loadList).toHaveBeenCalledWith({});
  });

  it('未登录态 → 不调 fetchMe；仍调 loadList', async () => {
    fakeAuthStore.isLoggedIn = false;
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeAuthStore.fetchMe).not.toHaveBeenCalled();
    expect(fakeOrderStore.loadList).toHaveBeenCalled();
  });

  it('hero 渲染：昵称 + 手机号脱敏', async () => {
    fakeAuthStore.user = {
      id: 7,
      nickname: '张三',
      phone: '13800138000',
    };
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const nickname = w.find('[data-test="nickname"]');
    expect(nickname.text()).toBe('张三');
    const phone = w.find('[data-test="phone"]');
    // 13800138000 → 138****8000
    expect(phone.text()).toBe('138****8000');
  });

  it('hero 未登录 → displayName 兜底「未登录」+ phone 空', async () => {
    fakeAuthStore.user = null;
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const nickname = w.find('[data-test="nickname"]');
    expect(nickname.text()).toBe('未登录');
    expect(w.find('[data-test="phone"]').text()).toBe('');
  });

  it('4 个订单状态 tile 渲染 + 点击 → order/index?status=xxx', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    // 4 个 tile
    expect(w.find('[data-test="order-tile-pendingPayment"]').exists()).toBe(true);
    expect(w.find('[data-test="order-tile-selectingEscort"]').exists()).toBe(true);
    expect(w.find('[data-test="order-tile-inService"]').exists()).toBe(true);
    expect(w.find('[data-test="order-tile-completed"]').exists()).toBe(true);

    // 点击「待选陪诊」tile
    await w.find('[data-test="order-tile-selectingEscort"]').trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe(
      '/pages/order/index?status=selectingEscort',
    );
  });

  it('「全部订单」点击 → /pages/order/index', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="all-orders"]').trigger('click');
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe('/pages/order/index');
  });

  it('设置 menu 5 项渲染 + 点击 → 对应 url', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="menu-wallet"]').exists()).toBe(true);
    expect(w.find('[data-test="menu-coupons"]').exists()).toBe(true);
    expect(w.find('[data-test="menu-address"]').exists()).toBe(true);
    expect(w.find('[data-test="menu-settings"]').exists()).toBe(true);
    expect(w.find('[data-test="menu-support"]').exists()).toBe(true);

    // 点击「优惠券」menu
    await w.find('[data-test="menu-coupons"]').trigger('click');
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe('/pages/coupons/index');
  });

  it('订单数 badge：loadList 返回各状态订单 → tile badge 显示数字', async () => {
    fakeOrderStore.loadList.mockResolvedValue([
      { id: 1, status: 'selectingEscort' },
      { id: 2, status: 'selectingEscort' },
      { id: 3, status: 'completed' },
      { id: 4, status: 'completed' },
      { id: 5, status: 'completed' },
      { id: 6, status: 'inService' },
    ]);

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="order-tile-badge-selectingEscort"]').exists()).toBe(true);
    expect(w.find('[data-test="order-tile-badge-selectingEscort"]').text()).toBe('2');
    expect(w.find('[data-test="order-tile-badge-completed"]').text()).toBe('3');
    expect(w.find('[data-test="order-tile-badge-inService"]').text()).toBe('1');
    // pendingPayment 0 条 → 无 badge
    expect(w.find('[data-test="order-tile-badge-pendingPayment"]').exists()).toBe(false);
  });

  it('错误态：loadList 抛错 → u-empty-stub + 重试按钮', async () => {
    fakeOrderStore.loadList.mockRejectedValueOnce(new Error('boom'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('加载失败');
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
  });

  it('重试按钮 → 重新调 fetchProfile', async () => {
    fakeOrderStore.loadList.mockRejectedValueOnce(new Error('boom'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeOrderStore.loadList).toHaveBeenCalledTimes(1);
    await w.vm.onRetry();
    await flushPromises();
    expect(fakeOrderStore.loadList).toHaveBeenCalledTimes(2);
  });
});