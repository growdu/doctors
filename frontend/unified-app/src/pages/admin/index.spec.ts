/**
 * admin/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * admin/index 是 alias path → admin/dashboard/index（admin 域真首页）。
 * onLoad 时调 uni.reLaunch('/pages/admin/dashboard/index')。
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

describe('admin/index · alias 重定向', () => {
  it('mount → onLoad → reLaunch /pages/admin/dashboard/index', () => {
    mount(IndexPage);
    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/admin/dashboard/index' });
  });
});