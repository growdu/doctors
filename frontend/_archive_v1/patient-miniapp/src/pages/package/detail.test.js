// src/pages/package/detail.test.js
//
// 服务包详情页单测 —— @vue/test-utils mount + jest.doMock 注入 fake hospital store。
//
// 测试点（4 个 it）：
//   1. onLoad(hospitalId, packageId) → 写入组件 data
//   2. mounted 触发 hospitalStore.loadDetail(hospitalId)
//   3. pkg 计算属性：按 packageId 从 packages 数组中匹配；缺 packageId → 取第一个
//   4. 「立即下单」点击 → navigateTo 订单创建页（带 hospitalId + packageId）

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { reactive } from 'vue';
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

// ---- fake hospital store（用 reactive 包裹让 computed 可追踪 + 模拟 Pinia 行为）
const fakeHospitalStore = reactive({
  detail: null,
  loadDetail: jest.fn(),
  clearDetail: jest.fn(),
});

jest.doMock('@/stores/hospital.js', () => ({
  useHospitalStore: () => fakeHospitalStore,
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-skeleton': { template: '<div class="u-skeleton-stub"></div>' },
  'u-empty': { template: '<div class="u-empty-stub"></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled', 'loading'],
    emits: ['click'],
        inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" :disabled="!!disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./detail.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('package/detail.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeHospitalStore.loadDetail.mockReset();
    fakeHospitalStore.clearDetail.mockReset();
    fakeHospitalStore.detail = null;

    fakeHospitalStore.loadDetail.mockImplementation(async () => {
      fakeHospitalStore.detail = {
        id: 101,
        name: '北京协和医院',
        level: '三甲',
        packages: [
          { id: 1, name: '半日陪诊', price: 388, description: '半天陪诊服务', durationHours: 4 },
          { id: 2, name: '全日陪诊', price: 588, description: '全天陪诊服务', durationHours: 8 },
        ],
      };
      return fakeHospitalStore.detail;
    });
  });

  it('onLoad(hospitalId, packageId) → 写入组件 data', async () => {
    const w = await mountPage();
    w.vm.onLoad({ hospitalId: 101, packageId: 2 });
    await w.vm.mounted();
    expect(w.vm.hospitalId).toBe(101);
    expect(w.vm.packageId).toBe(2);
  });

  it('mounted 触发 hospitalStore.loadDetail(hospitalId)', async () => {
    const w = await mountPage();
    w.vm.onLoad({ hospitalId: 101, packageId: 1 });
    await w.vm.mounted();
    await flushPromises();

    expect(fakeHospitalStore.loadDetail).toHaveBeenCalledWith(101);
    expect(w.vm.hospital.name).toBe('北京协和医院');
    expect(w.find('[data-test="hospital-banner"]').exists()).toBe(true);
  });

  it('pkg 计算属性：按 packageId 匹配；缺 packageId → 取第一个', async () => {
    const w = await mountPage();
    w.vm.onLoad({ hospitalId: 101 });
    await w.vm.mounted();
    await flushPromises();

    // 无 packageId → 取第一个（半日陪诊）
    expect(w.vm.pkg.id).toBe(1);
    expect(w.find('[data-test="package-name"]').text()).toBe('半日陪诊');
    expect(w.find('[data-test="package-price"]').text()).toBe('¥388');

    // 切换 packageId=2 → 全日陪诊
    await w.setData({ packageId: 2 });
    expect(w.vm.pkg.id).toBe(2);
    expect(w.find('[data-test="package-name"]').text()).toBe('全日陪诊');
    expect(w.find('[data-test="package-price"]').text()).toBe('¥588');

    // 不存在的 packageId → fallback 第一个
    await w.setData({ packageId: 999 });
    expect(w.vm.pkg.id).toBe(1);
  });

  it('「立即下单」点击 → navigateTo 订单创建页（带 hospitalId + packageId）', async () => {
    const w = await mountPage();
    w.vm.onLoad({ hospitalId: 101, packageId: 2 });
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="order-btn"]').trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe(
      '/pages/order/create?hospitalId=101&packageId=2',
    );
  });

  it('缺 hospitalId → mounted 不拉数据，置 loadError', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeHospitalStore.loadDetail).not.toHaveBeenCalled();
    expect(w.vm.loadError).toBe(true);
  });

  it('loadDetail 抛错 → error-state + 重试按钮', async () => {
    fakeHospitalStore.loadDetail.mockRejectedValueOnce(new Error('network down'));

    const w = await mountPage();
    w.vm.onLoad({ hospitalId: 999 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="error-state"]').exists()).toBe(true);
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
  });
});