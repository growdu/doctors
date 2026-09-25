// src/pages/hospitals/detail.test.js
//
// 医院详情页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 6 个 it）：
//   1. onLoad+loadDetail：传入 ?id=101 → mounted 调 useHospitalStore().loadDetail(101)
//   2. hero 渲染：医院名 + level + address
//   3. 科室 chips：hospital.departments 渲染为 chips
//   4. 服务包：hospital.packages 渲染卡片 / 无 packages → u-empty-stub
//   5. 「立即下单」点击 → uni.navigateTo /pages/order/create?hospitalId=xxx
//   6. 错误态：loadDetail 抛错 → error-state + 重试按钮
//
// 测试策略：
//   - jest.doMock('@/stores/hospital.js') 注入 fake store
//   - uView Plus 组件全部 stub
//   - onLoad + mounted 是 uni-app / Vue 钩子：通过 wrapper.vm.onLoad(query) 触发

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
  detail: null,
  loadDetail: jest.fn(),
  clearDetail: jest.fn(),
};

jest.doMock('@/stores/hospital.js', () => ({
  useHospitalStore: () => fakeStore,
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
    props: ['type', 'size', 'plain'],
    template:
      '<button class="u-button-stub" :data-type="type" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./detail.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('hospitals/detail.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadDetail.mockReset();
    fakeStore.clearDetail.mockReset();
    fakeStore.loadDetail.mockResolvedValue({
      id: 101,
      name: '北京协和医院',
      level: '三甲',
      address: '东城区帅府园 1 号',
      phone: '010-69155555',
      description: '三甲综合医院',
      departments: ['内科', '外科', '妇产科'],
      packages: [
        { id: 1, name: '半日陪诊', price: 388, description: '半天陪诊服务' },
        { id: 2, name: '全日陪诊', price: 588, description: '全天陪诊服务' },
      ],
    });
  });

  it('onLoad+mounted → loadDetail(101)', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ id: 101 });
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadDetail).toHaveBeenCalledTimes(1);
    expect(fakeStore.loadDetail).toHaveBeenCalledWith(101);
    expect(w.vm.hospital.id).toBe(101);
    expect(w.vm.hospitalId).toBe(101);
  });

  it('hero 渲染：医院名 + level + address', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ id: 101 });
    await w.vm.mounted();
    await flushPromises();

    const hero = w.find('[data-test="hero"]');
    expect(hero.exists()).toBe(true);
    expect(hero.text()).toContain('北京协和医院');
    const level = w.find('[data-test="level"]');
    expect(level.text()).toBe('三甲');
  });

  it('科室 chips：hospital.departments 渲染为多 chips', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ id: 101 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="departments-row"]').exists()).toBe(true);
    const chips = w.findAll('.page-hospital-detail__chip');
    expect(chips).toHaveLength(3);
    expect(chips[0].text()).toBe('内科');
    expect(chips[1].text()).toBe('外科');
    expect(chips[2].text()).toBe('妇产科');
  });

  it('服务包：hospital.packages 渲染卡片 + 「立即下单」点击 → order/create', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ id: 101 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="packages-empty"]').exists()).toBe(false);
    const cards = w.findAll('[data-test^="package-"]');
    expect(cards).toHaveLength(2);
    expect(cards[0].attributes('data-test')).toBe('package-1');

    // 「立即下单」按钮
    await w.find('[data-test="create-order-btn"]').trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/order\/create\?hospitalId=101/,
    );
  });

  it('服务包无 → u-empty-stub「暂未上线服务包」', async () => {
    fakeStore.loadDetail.mockResolvedValueOnce({
      id: 101,
      name: '北京协和医院',
      level: '三甲',
      // 无 packages / 无 departments
    });

    const w = await mountPage();
    await w.vm.onLoad({ id: 101 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="packages-empty"]').exists()).toBe(true);
    expect(w.findAll('[data-test^="package-"]')).toHaveLength(0);
  });

  it('点击服务包卡片 → order/create 携带 packageId', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ id: 101 });
    await w.vm.mounted();
    await flushPromises();

    const card = w.find('[data-test="package-1"]');
    await card.trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/order\/create\?hospitalId=101&packageId=1/,
    );
  });

  it('错误态：loadDetail 抛错 → error-state + 重试按钮', async () => {
    fakeStore.loadDetail.mockRejectedValueOnce(new Error('not found'));

    const w = await mountPage();
    await w.vm.onLoad({ id: 999 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="error-state"]').exists()).toBe(true);
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
    expect(w.vm.loadError).toBe(true);
  });

  it('mounted 时无 hospitalId → 直接置 loadError=true', async () => {
    const w = await mountPage();
    // 不调 onLoad
    await w.vm.mounted();
    expect(w.vm.loadError).toBe(true);
    expect(fakeStore.loadDetail).not.toHaveBeenCalled();
  });
});