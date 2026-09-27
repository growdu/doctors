// src/pages/sos/trigger.test.js
//
// SOS 紧急呼救页单测 —— @vue/test-utils mount + jest.doMock 注入 fake api/sos。
//
// 测试点（5 个 it）：
//   1. onLoad(orderId) → 写入组件 data；mounted 设置 mock 位置
//   2. 长按 touchstart → state='triggering' + progress > 0
//   3. 长按未到 1.5s 松开 → state='idle' + progress=0
//   4. 长按完成 1.5s → _onConfirm → api.triggerSos(payload) → state='triggered'
//   5. triggered 状态下「我按错了」点击 → api.cancelSos → state='idle'
//
// 测试策略：
//   - jest.doMock('@/api/sos.js', virtual) 注入 fake
//   - 用 jest.useFakeTimers() + advanceTimersByTime 模拟 setInterval
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
  jest.useRealTimers();
});

// ---- fake api/sos
const fakeSos = {
  triggerSos: jest.fn(),
  cancelSos: jest.fn(),
  getSosDetail: jest.fn(),
};

jest.doMock(
  '@/api/sos.js',
  () => fakeSos,
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
        inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" :disabled="!!disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./trigger.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('sos/trigger.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeSos.triggerSos.mockReset();
    fakeSos.cancelSos.mockReset();
    fakeSos.triggerSos.mockResolvedValue({
      id: 'sos-1',
      user_id: 1,
      lat: 39.9129,
      lng: 116.4148,
      status: 'pending',
      created_at: '2026-09-24T15:00:00Z',
    });
    fakeSos.cancelSos.mockResolvedValue({
      id: 'sos-1',
      status: 'cancelled',
    });
  });

  it('onLoad(orderId) → 写入组件 data；mounted 设置 mock 位置', async () => {
    const w = await mountPage();
    w.vm.onLoad({ orderId: 7 });
    expect(w.vm.orderId).toBe(7);

    expect(w.vm.location.lat).toBeCloseTo(39.9129);
    expect(w.vm.location.lng).toBeCloseTo(116.4148);
  });

  it('长按 touchstart → state=triggering + progress > 0', async () => {
    jest.useFakeTimers();
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.vm.state).toBe('idle');
    expect(w.find('[data-test="idle-state"]').exists()).toBe(true);

    w.vm.onPressStart();
    expect(w.vm.state).toBe('triggering');

    // 推进 500ms
    jest.advanceTimersByTime(500);
    expect(w.vm.progress).toBeGreaterThan(0);
    expect(w.vm.progress).toBeLessThan(100);

    // 清理：松开
    w.vm.onPressEnd();
    expect(w.vm.state).toBe('idle');
    expect(w.vm.progress).toBe(0);
  });

  it('长按未到 1.5s 松开 → state=idle + progress=0（不触发 triggerSos）', async () => {
    jest.useFakeTimers();
    const w = await mountPage();

    w.vm.onPressStart();
    jest.advanceTimersByTime(500); // 33%
    w.vm.onPressEnd();

    expect(w.vm.state).toBe('idle');
    expect(w.vm.progress).toBe(0);
    expect(fakeSos.triggerSos).not.toHaveBeenCalled();
  });

  it('长按完成 1.5s → _onConfirm → api.triggerSos(payload) → state=triggered', async () => {
    jest.useFakeTimers();
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    w.vm.onPressStart();
    // 推进到 ≥ 1500ms
    jest.advanceTimersByTime(1600);
    // 触发是 async，需要等 Promise 解析
    await flushPromises();

    expect(fakeSos.triggerSos).toHaveBeenCalledTimes(1);
    expect(fakeSos.triggerSos).toHaveBeenCalledWith(expect.objectContaining({
      lat: 39.9129,
      lng: 116.4148,
      order_id: null,
    }));
    expect(w.vm.state).toBe('triggered');
    expect(w.vm.sosEventId).toBe('sos-1');
    expect(w.find('[data-test="triggered-state"]').exists()).toBe(true);
    expect(w.find('[data-test="countdown-text"]').text()).toContain('10 秒');
  });

  it('triggered 状态下「我按错了」点击 → api.cancelSos → state=idle', async () => {
    jest.useFakeTimers();
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    // 先触发
    w.vm.onPressStart();
    jest.advanceTimersByTime(1600);
    await flushPromises();
    expect(w.vm.state).toBe('triggered');

    // 取消
    await w.find('[data-test="cancel-sos-btn"]').trigger('click');
    await flushPromises();

    expect(fakeSos.cancelSos).toHaveBeenCalledTimes(1);
    expect(fakeSos.cancelSos).toHaveBeenCalledWith('sos-1', 'user_cancelled');
    expect(w.vm.state).toBe('idle');
  });

  it('triggerSos 抛错 → state=idle + toast', async () => {
    jest.useFakeTimers();
    fakeSos.triggerSos.mockRejectedValueOnce(new Error('位置上报失败'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    w.vm.onPressStart();
    jest.advanceTimersByTime(1600);
    await flushPromises();

    expect(w.vm.state).toBe('idle');
    expect(w.vm.progress).toBe(0);
    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && c[0].title === '位置上报失败',
    );
    expect(toastArg).toBeTruthy();
  });

  it('triggered 倒计时归零 → state=done + 「返回」按钮显示', async () => {
    jest.useFakeTimers();
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    w.vm.onPressStart();
    jest.advanceTimersByTime(1600);
    await flushPromises();
    expect(w.vm.state).toBe('triggered');

    // 推进 10s 倒计时
    jest.advanceTimersByTime(10000);
    await flushPromises();
    expect(w.vm.state).toBe('done');
    expect(w.find('[data-test="done-state"]').exists()).toBe(true);

    // 返回按钮点击
    await w.find('[data-test="back-btn"]').trigger('click');
    expect(mockUni.navigateBack).toHaveBeenCalledTimes(1);
  });
});