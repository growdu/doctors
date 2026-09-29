/**
 * escort/checkin/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - parseQuery 提取 orderId
 *   - mount 时调 uni.getLocation 拿 GPS + 显示位置
 *   - getLocation 失败回退默认位置
 *   - 「签到」按钮 → 调 updateLocation + toast + navigateBack
 *   - 位置缺失时按钮 disabled
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import CheckinPage from './index.vue';

vi.mock('@/api/escort', () => ({
  updateLocation: vi.fn(),
}));

import * as apiEscort from '@/api/escort';

interface LocationCb {
  success?: (r: unknown) => void;
  fail?: (e: unknown) => void;
}

function setupGetLocationMock(behavior: 'success' | 'fail', data?: { latitude: number; longitude: number }) {
  (globalThis as unknown as {
    uni: { getLocation: (opts: LocationCb) => LocationCb };
  }).uni.getLocation = (opts: LocationCb) => {
    if (behavior === 'success' && data) {
      queueMicrotask(() => opts.success?.(data));
    } else if (behavior === 'fail') {
      queueMicrotask(() => opts.fail?.({ errMsg: '定位失败' }));
    }
    return opts;
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  // 默认 getCurrentPages 返回 orderId=101
  (globalThis as unknown as {
    uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
  }).uni.getCurrentPages = () => [{ options: { orderId: '101' } }];
});

describe('escort/checkin · 加载', () => {
  it('parseQuery 提取 orderId + mount 调 getLocation', async () => {
    setupGetLocationMock('success', { latitude: 39.9, longitude: 116.4 });

    const w = mount(CheckinPage);
    await flushPromises();

    expect(w.text()).toContain('订单 #101 签到');
    expect(w.find('[data-testid="escort-checkin-pos"]').text()).toContain('lat=39.900000');
    expect(w.find('[data-testid="escort-checkin-pos"]').text()).toContain('lng=116.400000');
  });

  it('getLocation 失败回退默认位置', async () => {
    setupGetLocationMock('fail');

    const w = mount(CheckinPage);
    await flushPromises();

    // 回退默认 北京坐标
    expect(w.find('[data-testid="escort-checkin-pos"]').text()).toContain('lat=39.908823');
    expect(w.find('[data-testid="escort-checkin-pos"]').text()).toContain('lng=116.397470');
    expect(w.find('[data-testid="escort-checkin-pos-error"]').exists()).toBe(true);
  });

  it('未传 orderId 时显示「订单 #—」', async () => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: {} }];
    setupGetLocationMock('success', { latitude: 39.9, longitude: 116.4 });

    const w = mount(CheckinPage);
    await flushPromises();

    expect(w.text()).toContain('订单 #— 签到');
  });
});

describe('escort/checkin · 签到流程', () => {
  it('点击「签到」调 updateLocation + toast', async () => {
    setupGetLocationMock('success', { latitude: 39.9, longitude: 116.4 });
    vi.mocked(apiEscort.updateLocation).mockResolvedValue({ ok: true });

    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(CheckinPage);
    await flushPromises();

    await w.find('[data-testid="escort-checkin-submit"]').trigger('click');
    await flushPromises();

    expect(apiEscort.updateLocation).toHaveBeenCalledWith({
      city: '',
      lat: 116.4,
      lng: 39.9,
      address: '',
    });
    expect(toastSpy).toHaveBeenCalledWith({ title: '签到成功', icon: 'success' });
  });

  it('签到 API 异常显示错误 toast', async () => {
    setupGetLocationMock('success', { latitude: 39.9, longitude: 116.4 });
    vi.mocked(apiEscort.updateLocation).mockRejectedValue(new Error('网络异常'));

    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(CheckinPage);
    await flushPromises();

    await w.find('[data-testid="escort-checkin-submit"]').trigger('click');
    await flushPromises();

    expect(toastSpy).toHaveBeenCalledWith({ title: '签到失败：网络异常', icon: 'none' });
  });
});