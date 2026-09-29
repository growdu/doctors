// src/pages/order/pay.test.js
//
// 订单支付页单测 —— @vue/test-utils mount + jest.doMock 注入 fake api/pay 与 api/order。
//
// 测试点（5 个 it）：
//   1. onLoad(orderId, amount) → 写入组件 data
//   2. mounted 调 api.payOrder(orderId) → 写入 payUrl / expireAt / amount
//   3. 「立即支付」点击（未 expired） → toast「已生成支付 URL」
//   4. 「我已支付」点击 → getPayStatus 轮询 → paid → redirectTo 详情页
//   5. 超时态：expired=true → 「支付已超时」+ 「取消订单」按钮显示 + 「立即支付」disabled
//
// 测试策略：
//   - jest.doMock('@/api/pay.js', virtual) 注入 fake
//   - jest.doMock('@/api/order.js', virtual) 注入 fake（cancelOrder）
//   - CountdownBadge 通过 mount stubs 注入（不依赖 vue3-jest 编译 .vue）
//   - 全局 uni mock

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { jest } from '@jest/globals';

// ---- 全局 uni mock
const mockUni = {
  showToast: jest.fn(),
  showModal: jest.fn(),
  navigateTo: jest.fn(),
  navigateBack: jest.fn(),
  redirectTo: jest.fn(),
  reLaunch: jest.fn(),
};

beforeAll(() => {
  global.uni = mockUni;
});

afterEach(() => {
  jest.clearAllMocks();
});

// ---- fake api/pay
const fakePay = {
  payOrder: jest.fn(),
  getPayStatus: jest.fn(),
};

jest.doMock(
  '@/api/pay.js',
  () => fakePay,
  { virtual: true },
);

// ---- fake api/order（cancelOrder）
const fakeOrder = {
  cancelOrder: jest.fn(),
  getOrder: jest.fn(),
  selectEscort: jest.fn(),
  listOrders: jest.fn(),
  getCandidates: jest.fn(),
};

jest.doMock(
  '@/api/order.js',
  () => fakeOrder,
  { virtual: true },
);

// ---- CountdownBadge stub（避免 vue3-jest 处理子组件时的额外副作用）
const CountdownBadgeStub = {
  name: 'CountdownBadge',
  props: ['expireAt', 'label'],
  template:
    '<div class="countdown-badge-stub" :data-expire="expireAt" :data-label="label"></div>',
};

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled', 'loading'],
    emits: ['click'],
        inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" :disabled="!!disabled" :data-loading="!!loading" @click="$emit(\'click\')"><slot /></button>',
  },
  CountdownBadge: CountdownBadgeStub,
};

const mountPage = async () => {
  const mod = await import('./pay.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('order/pay.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakePay.payOrder.mockReset();
    fakePay.getPayStatus.mockReset();
    fakeOrder.cancelOrder.mockReset();

    fakePay.payOrder.mockResolvedValue({
      order_id: 7,
      channel: 'wechat',
      pay_url: 'https://sandbox.example.com/wechat/pay?token=mock',
      prepay_id: 'mock-prepay-123',
      nonce_str: 'mock-nonce',
      timestamp: '1700000000',
      sign: 'mock-sign',
      expire_at: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
      amount: 388,
      amountText: '388.00',
    });
    fakePay.getPayStatus.mockResolvedValue({
      order_id: 7,
      status: 'paid',
      paid_at: '2026-09-24T12:00:00Z',
      transaction_id: 'wx-mock-tx-1',
    });
    fakeOrder.cancelOrder.mockResolvedValue({ ok: true });
  });

  it('onLoad(orderId, amount) → 写入组件 data', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7, amount: '388' });
    await w.vm.mounted();
    expect(w.vm.orderId).toBe(7);
    expect(w.vm.amount).toBe(388);
    expect(w.vm.amountText).toBe('388.00');
  });

  it('mounted 调 api.payOrder(orderId) → 写入 payUrl / expireAt / amount', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    expect(fakePay.payOrder).toHaveBeenCalledWith(7, { channel: 'wechat' });
    expect(w.vm.payUrl).toBe('https://sandbox.example.com/wechat/pay?token=mock');
    expect(w.vm.expireAt).toBeTruthy();
    expect(w.vm.amount).toBe(388);
    expect(w.find('[data-test="pay-url"]').exists()).toBe(true);
  });

  it('「立即支付」点击（未 expired） → toast「已生成支付 URL」', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="pay-btn"]').trigger('click');
    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && /已生成支付 URL/.test(c[0].title),
    );
    expect(toastArg).toBeTruthy();
  });

  it('「我已支付」点击 → getPayStatus 轮询 → paid → redirectTo 详情页', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="paid-btn"]').trigger('click');
    // 轮询 1 次即 paid（mock 直接返回 paid）
    await flushPromises();
    // 等 setTimeout 1s 收尾
    await new Promise((r) => setTimeout(r, 1500));
    await flushPromises();

    expect(fakePay.getPayStatus).toHaveBeenCalledWith(7);
    expect(mockUni.redirectTo).toHaveBeenCalledTimes(1);
    expect(mockUni.redirectTo.mock.calls[0][0].url).toBe('/pages/order/detail?orderId=7');
  });

  it('轮询 3 次仍 pending → toast「暂未收到支付结果」', async () => {
    fakePay.getPayStatus.mockResolvedValue({ order_id: 7, status: 'pending' });

    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="paid-btn"]').trigger('click');
    await flushPromises();
    await new Promise((r) => setTimeout(r, 2500));
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && /暂未收到支付结果/.test(c[0].title),
    );
    expect(toastArg).toBeTruthy();
    expect(mockUni.redirectTo).not.toHaveBeenCalled();
  });

  it('超时态：expired=true → 「支付已超时」+ 取消订单按钮显示 + 立即支付 disabled', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await w.setData({ expired: true });
    await flushPromises();

    expect(w.find('[data-test="expired-text"]').exists()).toBe(true);
    expect(w.find('[data-test="cancel-btn"]').exists()).toBe(true);
    expect(w.find('[data-test="pay-btn"]').attributes('disabled')).toBeDefined();

    // 取消订单点击
    await w.find('[data-test="cancel-btn"]').trigger('click');
    await flushPromises();

    expect(fakeOrder.cancelOrder).toHaveBeenCalledWith(7, '支付超时');
    expect(mockUni.navigateBack).toHaveBeenCalledTimes(1);
  });

  it('缺 orderId → mounted 直接 return，不调 payOrder', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakePay.payOrder).not.toHaveBeenCalled();
    expect(w.vm.payUrl).toBe('');
  });
});