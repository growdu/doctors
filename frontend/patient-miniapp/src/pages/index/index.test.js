// src/pages/index/index.test.js
//
// 首页单测 —— @vue/test-utils mount + 模拟 uni-app 生命周期 + mock store。
//
// 测试点（brief 要求 5 个 it）：
//   1. mounted → 调 useHospitalStore().loadList({ limit: 5, page: 1 }) 拉推荐
//   2. 4 个快捷入口渲染（createOrder / myOrders / coupons / address）
//   3. 点击快捷入口 → uni.navigateTo 对应 url
//   4. 推荐医院渲染：医院卡片（name + level） + 点击 → uni.navigateTo detail
//   5. 空状态：loadList 返回 [] → u-empty-stub 可见 + 不渲染医院卡片
//
// 测试策略：
//   - jest.doMock('@/stores/hospital.js') 注入 fake store
//   - uView Plus 组件全部 stub
//   - mounted 是 Vue 标准生命周期：通过 wrapper.vm.mounted() 手动触发

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

// ---- fake store
const fakeStore = {
  list: [],
  loadList: jest.fn(),
};

jest.doMock('@/stores/hospital.js', () => ({
  useHospitalStore: () => fakeStore,
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-skeleton': {
    props: ['rows', 'title', 'avatar', 'avatarShape', 'avatarSize'],
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

describe('home/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadList.mockReset();
    // 默认：loadList 返回 3 条医院
    fakeStore.loadList.mockResolvedValue({
      items: [
        { id: 101, name: '北京协和医院', level: '三甲', address: '东城区帅府园 1 号' },
        { id: 102, name: '华西医院', level: '三甲', address: '国学巷 37 号' },
        { id: 103, name: '上海瑞金医院', level: '三甲', address: '瑞金二路 197 号' },
      ],
      total: 3,
      page: 1,
      limit: 5,
    });
  });

  it('mounted → 调 hospital store loadList({ limit: 5, page: 1 })', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);
    expect(fakeStore.loadList).toHaveBeenCalledWith({ limit: 5, page: 1 });
    // recommended 已写入
    expect(w.vm.recommended).toHaveLength(3);
  });

  it('4 个快捷入口渲染 + 各自 data-test', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="quick-createOrder"]').exists()).toBe(true);
    expect(w.find('[data-test="quick-myOrders"]').exists()).toBe(true);
    expect(w.find('[data-test="quick-coupons"]').exists()).toBe(true);
    expect(w.find('[data-test="quick-address"]').exists()).toBe(true);
  });

  it('点击快捷入口 → uni.navigateTo 对应 url', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    // 点击「我的订单」快捷入口
    await w.find('[data-test="quick-myOrders"]').trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe('/pages/order/index');
  });

  it('点击「优惠券」快捷入口 → /pages/coupons/index', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="quick-coupons"]').trigger('click');
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe('/pages/coupons/index');
  });

  it('推荐医院：医院卡片渲染 + 点击 → uni.navigateTo detail', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const cards = w.findAll('[data-test^="hospital-card-"]');
    expect(cards).toHaveLength(3);
    expect(cards[0].attributes('data-hospital-id')).toBe('101');

    // 点击第一张医院卡片
    await cards[0].trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/hospitals\/detail\?id=101/,
    );
  });

  it('「查看更多」点击 → uni.navigateTo /pages/hospitals/list', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="view-more"]').trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe('/pages/hospitals/list');
  });

  it('空状态：loadList 返回 [] → u-empty-stub 可见 + 不渲染医院卡片', async () => {
    fakeStore.loadList.mockResolvedValueOnce({ items: [], total: 0, page: 1, limit: 5 });

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无可用医院');
    expect(w.findAll('[data-test^="hospital-card-"]')).toHaveLength(0);
    expect(w.vm.recommended).toEqual([]);
  });

  it('错误态：loadList 抛错 → u-empty-stub「加载失败」+ 重试按钮', async () => {
    fakeStore.loadList.mockRejectedValueOnce(new Error('network down'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('加载失败');
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
    expect(w.vm.loadError).toBe(true);
    expect(w.vm.loading).toBe(false);
  });

  it('重试按钮 → 重新调 loadList', async () => {
    fakeStore.loadList.mockRejectedValueOnce(new Error('boom'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);
    await w.vm.onRetry();
    await flushPromises();
    expect(fakeStore.loadList).toHaveBeenCalledTimes(2);
  });
});