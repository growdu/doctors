// src/pages/order/create.test.js
//
// 订单创建页单测 —— @vue/test-utils mount + mock stores。
//
// 测试点（brief 要求 5 个 it）：
//   1. onLoad+mounted → 拉 hospital / address / coupon 3 个 store
//   2. 5 个 section 渲染（医院 / 时间 / 联系人 / 地址 / 优惠券）
//   3. 服务包选择 + 金额合计 = pkg.price
//   4. 优惠券选择 + 折扣计算 = min(pkg.price, uc.value)
//   5. 缺必填项 → 提交按钮 disabled
//
// 测试策略：
//   - jest.doMock 注入 3 个 fake store（hospital / address / coupon）
//   - uView Plus 组件全部 stub
//   - onLoad + mounted 通过 wrapper.vm 手动触发

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

// ---- fake hospital store
const fakeHospitalStore = {
  detail: null,
  loadDetail: jest.fn(),
  clearDetail: jest.fn(),
};

jest.doMock('@/stores/hospital.js', () => ({
  useHospitalStore: () => fakeHospitalStore,
}));

// ---- fake address store
const fakeAddressStore = {
  list: [],
  defaultId: null,
  loadList: jest.fn(),
  defaultAddress: jest.fn(() => null),
};

jest.doMock('@/stores/address.js', () => ({
  useAddressStore: () => fakeAddressStore,
}));

// ---- fake coupon store
const fakeCouponStore = {
  mine: [],
  loadMine: jest.fn(),
};

jest.doMock('@/stores/coupon.js', () => ({
  useCouponStore: () => fakeCouponStore,
  COUPON_STATUS_UNUSED: 'unused',
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
  const mod = await import('./create.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('order/create.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeHospitalStore.loadDetail.mockReset();
    fakeAddressStore.loadList.mockReset();
    fakeCouponStore.loadMine.mockReset();

    fakeHospitalStore.loadDetail.mockResolvedValue({
      id: 101,
      name: '北京协和医院',
      packages: [
        { id: 1, name: '半日陪诊', price: 388 },
        { id: 2, name: '全日陪诊', price: 588 },
      ],
    });
    fakeAddressStore.loadList.mockResolvedValue([
      { id: 1, recipient: 'a', phone: '13800138000', detail: 'addr-1', is_default: true },
      { id: 2, recipient: 'b', phone: '13900139000', detail: 'addr-2', is_default: false },
    ]);
    fakeAddressStore.defaultId = 1;
    fakeCouponStore.loadMine.mockResolvedValue([
      {
        id: 100,
        status: 'unused',
        coupon: { id: 1, name: '新人立减券', type: 'amount', value: 30, threshold: 100 },
      },
      {
        id: 101,
        status: 'used', // 非 unused → 不进 usableCoupons
        coupon: { id: 1, name: '用过', type: 'amount', value: 50, threshold: 100 },
      },
    ]);
  });

  it('onLoad+mounted → 拉 hospital / address / coupon 3 个 store', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 101, packageId: 1 });
    await w.vm.mounted();
    await flushPromises();

    expect(fakeHospitalStore.loadDetail).toHaveBeenCalledWith(101);
    expect(fakeAddressStore.loadList).toHaveBeenCalledTimes(1);
    expect(fakeCouponStore.loadMine).toHaveBeenCalledTimes(1);
    expect(w.vm.hospital.name).toBe('北京协和医院');
    expect(w.vm.packages).toHaveLength(2);
    expect(w.vm.addresses).toHaveLength(2);
    expect(w.vm.usableCoupons).toHaveLength(1); // 1 used 被过滤
  });

  it('5 个 section 渲染（医院 / 时间 / 联系人 / 地址 / 优惠券）', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 101, packageId: 1 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="hospital-section"]').exists()).toBe(true);
    expect(w.find('[data-test="time-section"]').exists()).toBe(true);
    expect(w.find('[data-test="contact-section"]').exists()).toBe(true);
    expect(w.find('[data-test="address-section"]').exists()).toBe(true);
    expect(w.find('[data-test="coupon-section"]').exists()).toBe(true);

    // 医院名渲染
    expect(w.find('[data-test="hospital-name"]').text()).toBe('北京协和医院');
  });

  it('服务包选择 + 金额合计 = pkg.price', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 101, packageId: 1 });
    await w.vm.mounted();
    await flushPromises();

    // onLoad 传了 packageId=1 → 自动选中
    expect(w.vm.selectedPackageId).toBe(1);
    expect(w.find('[data-test="total-amount"]').text()).toBe('¥388.00');

    // 切换到全日陪诊
    await w.find('[data-test="package-2"]').trigger('click');
    expect(w.vm.selectedPackageId).toBe(2);
    expect(w.find('[data-test="total-amount"]').text()).toBe('¥588.00');
  });

  it('优惠券选择 + 折扣计算 = min(pkg.price, uc.value)', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 101, packageId: 1 });
    await w.vm.mounted();
    await flushPromises();

    // 不使用优惠券 → 总额 388
    expect(w.find('[data-test="total-amount"]').text()).toBe('¥388.00');
    expect(w.find('[data-test="discount-amount"]').exists()).toBe(false);

    // 选择 30 元券
    await w.find('[data-test="coupon-100"]').trigger('click');
    expect(w.vm.selectedUserCouponId).toBe(100);
    expect(w.find('[data-test="discount-amount"]').exists()).toBe(true);
    expect(w.find('[data-test="discount-amount"]').text()).toBe('已优惠 ¥30.00');
    expect(w.find('[data-test="total-amount"]').text()).toBe('¥358.00');

    // 切回「不使用」
    await w.find('[data-test="coupon-none"]').trigger('click');
    expect(w.vm.selectedUserCouponId).toBeNull();
    expect(w.find('[data-test="discount-amount"]').exists()).toBe(false);
    expect(w.find('[data-test="total-amount"]').text()).toBe('¥388.00');
  });

  it('缺必填项 → 提交按钮 disabled', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 101 });
    await w.vm.mounted();
    await flushPromises();

    // 此时：医院✓ / 地址✓（默认 1）/ 联系人✗ / 时间✗ / 包✗
    expect(w.vm.canSubmit).toBe(false);
    const btn = w.find('[data-test="submit-btn"]');
    expect(btn.attributes('disabled')).toBeDefined();

    // 选服务包
    await w.find('[data-test="package-1"]').trigger('click');
    // 填联系人
    w.vm.contactName = '张三';
    w.vm.contactPhone = '13800138000';
    w.vm.appointmentAt = '今天 09:00';

    expect(w.vm.canSubmit).toBe(true);
    expect(w.find('[data-test="submit-btn"]').attributes('disabled')).toBeUndefined();
  });

  it('「提交」点击 → toast + redirectTo 我的订单', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 101, packageId: 1 });
    await w.vm.mounted();
    await flushPromises();

    w.vm.contactName = '张三';
    w.vm.contactPhone = '13800138000';
    w.vm.appointmentAt = '今天 09:00';

    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '订单已提交' }),
    );
    expect(mockUni.redirectTo).toHaveBeenCalledTimes(1);
    expect(mockUni.redirectTo.mock.calls[0][0].url).toBe('/pages/order/index');
  });

  it('「管理」地址链接 → /pages/address/list', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 101, packageId: 1 });
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="manage-address"]').trigger('click');
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe('/pages/address/list');
  });

  it('无 hospitalId → mounted 直接置 loadError', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    expect(w.vm.loadError).toBe(true);
    expect(fakeHospitalStore.loadDetail).not.toHaveBeenCalled();
  });

  it('错误态：loadDetail 抛错 → error-state + 重试按钮', async () => {
    fakeHospitalStore.loadDetail.mockRejectedValueOnce(new Error('boom'));

    const w = await mountPage();
    await w.vm.onLoad({ hospitalId: 999 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="error-state"]').exists()).toBe(true);
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
  });
});