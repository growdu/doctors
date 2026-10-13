/**
 * patient/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * patient/index 是 alias path → patient/index/index（tabBar 真首页）。
 * onLoad 时调 uni.reLaunch('/pages/patient/index/index')。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';

const reLaunchSpy = vi.fn();

vi.mock('@dcloudio/uni-app', () => ({
  onLoad: (cb: () => void) => cb(),
}));

beforeEach(() => {
  reLaunchSpy.mockReset();
  (globalThis as unknown as { uni: { reLaunch: typeof reLaunchSpy } }).uni.reLaunch = reLaunchSpy;
});

import IndexPage from './index.vue';

describe('patient/index · alias 重定向', () => {
  it('mount → onLoad → reLaunch /pages/patient/index/index', () => {
    mount(IndexPage);
    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/patient/index/index' });
  });
});