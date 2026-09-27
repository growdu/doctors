// src/pages/refund/apply.test.js
//
// 退款申请页单测 —— @vue/test-utils mount + jest.doMock 注入 fake api/refund。
//
// 测试点（5 个 it）：
//   1. onLoad(orderId, amount) → 写入组件 data
//   2. 6 个理由 radio 项渲染（data-test=reason-{code}）
//   3. 点击理由 → selectedCode 写入 + 高亮（--active class）
//   4. canSubmit 计算属性：OTHER 模式要求 textarea ≥ 10 字符
//   5. 「提交」点击 → api.applyRefund(orderId, {reason, reason_code}) → navigateBack
//
// 测试策略：
//   - jest.doMock('@/api/refund.js', virtual) 注入 fake
//   - uView Plus 组件全部 stub

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

// ---- fake api/refund
const fakeRefund = {
  applyRefund: jest.fn(),
  getRefundStatus: jest.fn(),
};

jest.doMock(
  '@/api/refund.js',
  () => fakeRefund,
  { virtual: true },
);

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled', 'loading'],
    emits: ['click'],
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="!!disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./apply.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('refund/apply.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeRefund.applyRefund.mockReset();
    fakeRefund.applyRefund.mockResolvedValue({
      id: 1, order_id: 7, amount: 388, reason: '服务未开始就取消',
      status: 'pending', created_at: '2026-09-24T15:00:00Z', updated_at: '2026-09-24T15:00:00Z',
    });
  });

  it('onLoad(orderId, amount) → 写入组件 data', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7, amount: '388' });
    expect(w.vm.orderId).toBe(7);
    expect(w.vm.amount).toBe(388);
    expect(w.vm.amountText).toBe('388.00');
  });

  it('6 个理由 radio 项渲染（data-test=reason-{code}）', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7, amount: '388' });
    await flushPromises();

    const codes = ['CANCEL_BEFORE_START', 'ESCORT_LATE', 'BAD_SERVICE', 'INFO_WRONG', 'RESCHEDULE', 'OTHER'];
    codes.forEach((c) => {
      expect(w.find(`[data-test="reason-${c}"]`).exists()).toBe(true);
    });
    // 金额显示
    expect(w.find('[data-test="amount-value"]').text()).toBe('¥388.00');
  });

  it('点击理由 → selectedCode 写入 + 高亮（--active class）', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });

    // 点击「服务态度问题」
    await w.find('[data-test="reason-BAD_SERVICE"]').trigger('click');
    expect(w.vm.selectedCode).toBe('BAD_SERVICE');
    expect(w.find('[data-test="reason-BAD_SERVICE"]').classes()).toContain(
      'page-refund__reason-item--active',
    );
    // 其他项不亮
    expect(w.find('[data-test="reason-OTHER"]').classes()).not.toContain(
      'page-refund__reason-item--active',
    );
  });

  it('OTHER 模式 → 弹 textarea + canSubmit 要求 ≥ 10 字符', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });

    await w.find('[data-test="reason-OTHER"]').trigger('click');
    expect(w.vm.selectedCode).toBe('OTHER');
    expect(w.find('[data-test="other-wrap"]').exists()).toBe(true);

    // 未填 → 不可提交
    expect(w.vm.canSubmit).toBe(false);

    // 填 9 字符 → 不可提交
    await w.setData({ otherReason: '123456789' });
    expect(w.vm.canSubmit).toBe(false);

    // 填 10 字符 → 可提交
    await w.setData({ otherReason: '1234567890' });
    expect(w.vm.canSubmit).toBe(true);

    // 填 200 字符 → 可提交
    await w.setData({ otherReason: 'a'.repeat(200) });
    expect(w.vm.canSubmit).toBe(true);

    // 填 201 字符 → 不可提交
    await w.setData({ otherReason: 'a'.repeat(201) });
    expect(w.vm.canSubmit).toBe(false);

    // 切回非 OTHER → textarea 消失 + otherReason 清空
    await w.find('[data-test="reason-BAD_SERVICE"]').trigger('click');
    expect(w.vm.selectedCode).toBe('BAD_SERVICE');
    expect(w.find('[data-test="other-wrap"]').exists()).toBe(false);
    expect(w.vm.otherReason).toBe('');
  });

  it('「提交」点击 → api.applyRefund(orderId, {reason, reason_code}) → navigateBack', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7, amount: '388' });

    await w.find('[data-test="reason-CANCEL_BEFORE_START"]').trigger('click');
    expect(w.vm.canSubmit).toBe(true);

    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(fakeRefund.applyRefund).toHaveBeenCalledTimes(1);
    expect(fakeRefund.applyRefund).toHaveBeenCalledWith(7, {
      reason: '服务未开始就取消',
      reason_code: 'CANCEL_BEFORE_START',
    });
    expect(mockUni.navigateBack).toHaveBeenCalledTimes(1);
  });

  it('OTHER 提交 → reason 用 otherReason（trim）', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    await w.find('[data-test="reason-OTHER"]').trigger('click');
    await w.setData({ otherReason: '   我要退款因为临时有事   ' });

    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(fakeRefund.applyRefund).toHaveBeenCalledWith(7, {
      reason: '我要退款因为临时有事',
      reason_code: 'OTHER',
    });
  });

  it('提交失败 → toast + 不跳转', async () => {
    fakeRefund.applyRefund.mockRejectedValueOnce(new Error('订单已陪诊，无法退款'));

    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    await w.find('[data-test="reason-BAD_SERVICE"]').trigger('click');
    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && c[0].title === '订单已陪诊，无法退款',
    );
    expect(toastArg).toBeTruthy();
    expect(mockUni.navigateBack).not.toHaveBeenCalled();
  });
});