// src/pages/settings/index/index.test.js
//
// 设置页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 4-5 个 it）：
//   1. mounted → 渲染 4 tab + 基本 tab 项 + 底部退出登录
//   2. 切到「推送」tab → 渲染推送项 + 底部退出登录不显示
//   3. toggle 点击 → data-on 反向 + state 反向
//   4. 「退出登录」点击 → auth.logout + reLaunch 到 /pages/login/index
//   5. onLoad(tab='push') → 自动切到推送 tab

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { jest } from '@jest/globals';

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

const fakeAuthStore = {
  token: '',
  user: null,
  isLoggedIn: false,
  loginByPhone: jest.fn(),
  loginByWechat: jest.fn(),
  sendSmsCode: jest.fn(),
  fetchMe: jest.fn(),
  logout: jest.fn(),
  onUnauthorized: jest.fn(),
};

jest.doMock('@/stores/auth.js', () => ({
  useAuthStore: () => fakeAuthStore,
}));

const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled'],
    emits: ['click'],
    inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" :disabled="!!disabled" @click.stop="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('settings/index/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAuthStore.logout.mockReset();
    fakeAuthStore.logout.mockResolvedValue({});
  });

  it('mounted → 4 tab + 默认 basic + 渲染基础项 + 底部退出登录', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(w.vm.currentTab).toBe('basic');
    // 4 tab
    expect(w.find('[data-test="tab-basic"]').exists()).toBe(true);
    expect(w.find('[data-test="tab-pay"]').exists()).toBe(true);
    expect(w.find('[data-test="tab-sms"]').exists()).toBe(true);
    expect(w.find('[data-test="tab-push"]').exists()).toBe(true);
    // 基础 tab 下 4 项
    expect(w.find('[data-test="item-language"]').exists()).toBe(true);
    expect(w.find('[data-test="item-darkMode"]').exists()).toBe(true);
    expect(w.find('[data-test="item-biometric"]').exists()).toBe(true);
    expect(w.find('[data-test="item-timezone"]').exists()).toBe(true);
    // 底部退出登录
    expect(w.find('[data-test="logout-btn"]').exists()).toBe(true);
  });

  it('切到「推送」tab → 渲染推送项 + 底部退出登录不显示', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="tab-push"]').trigger('click');
    expect(w.vm.currentTab).toBe('push');
    // 推送 tab 下
    expect(w.find('[data-test="item-orderPush"]').exists()).toBe(true);
    expect(w.find('[data-test="item-quietHours"]').exists()).toBe(true);
    // 基础 tab 项不应显示
    expect(w.find('[data-test="item-language"]').exists()).toBe(false);
    // 底部退出登录隐藏
    expect(w.find('[data-test="logout-btn"]').exists()).toBe(false);
  });

  it('toggle 点击 → data-on 反向 + state 反向', async () => {
    const w = await mountPage();
    await flushPromises();

    // 默认 darkMode=false
    expect(w.find('[data-test="toggle-darkMode"]').attributes('data-on')).toBe('false');
    expect(w.vm.settings.darkMode).toBe(false);

    await w.find('[data-test="toggle-darkMode"]').trigger('click');
    expect(w.vm.settings.darkMode).toBe(true);
    expect(w.find('[data-test="toggle-darkMode"]').attributes('data-on')).toBe('true');

    // 再点关闭
    await w.find('[data-test="toggle-darkMode"]').trigger('click');
    expect(w.vm.settings.darkMode).toBe(false);
  });

  it('「退出登录」点击 → auth.logout + reLaunch 到 /pages/login/index', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="logout-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAuthStore.logout).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch.mock.calls[0][0].url).toBe('/pages/login/index');
  });

  it('onLoad(tab=push) → 自动切到推送 tab', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ tab: 'push' });
    expect(w.vm.currentTab).toBe('push');
  });

  it('logout 抛错 → 仍 reLaunch（兜底）', async () => {
    fakeAuthStore.logout.mockRejectedValueOnce(new Error('网络异常'));

    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="logout-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.reLaunch).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch.mock.calls[0][0].url).toBe('/pages/login/index');
  });
});
</content>
</invoke>