// src/pages/login/index/index.test.js
//
// 登录 v2 单测 —— 短信 / 微信 / 6 个 demo 账号一键填。
//
// 测试点（brief 要求 4-5 个 it）：
//   1. mounted → 渲染表单（手机号 / 验证码 / 微信登录 / 6 个 demo 行）
//   2. onSendCode → invalid phone 不调；valid → sendSmsCode + 60s 冷却
//   3. onLogin（phone+code 全）→ store.loginByPhone + reLaunch
//   4. onWechatLogin → store.loginByWechat + reLaunch
//   5. 点 demo 账号「VIP 用户」→ 自动填 phone + code=123456
//   6. 失败 → toast 提示，不调 reLaunch

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

// ---- fake auth store
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

// CountdownBadge stub
const CountdownBadge = {
  name: 'CountdownBadge',
  props: ['expireAt', 'label'],
  template: '<div class="countdown-badge-stub" :data-expire="expireAt"></div>',
};

// ---- uView Plus stub
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
  CountdownBadge,
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('login/index/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAuthStore.sendSmsCode.mockReset();
    fakeAuthStore.loginByPhone.mockReset();
    fakeAuthStore.loginByWechat.mockReset();
    fakeAuthStore.sendSmsCode.mockResolvedValue({ sent: true, ttl: 60 });
    fakeAuthStore.loginByPhone.mockResolvedValue({ access_token: 'tkn' });
    fakeAuthStore.loginByWechat.mockResolvedValue({ access_token: 'tkn-wx' });
  });

  it('mounted → 渲染手机号输入 + 验证码 + 微信登录 + 6 个 demo 行', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(w.find('[data-test="login-page"]').exists()).toBe(true);
    expect(w.find('[data-test="phone-input"]').exists()).toBe(true);
    expect(w.find('[data-test="code-input"]').exists()).toBe(true);
    expect(w.find('[data-test="login-btn"]').exists()).toBe(true);
    expect(w.find('[data-test="wechat-btn"]').exists()).toBe(true);

    // 6 个 demo 行
    const demos = w.findAll('[data-test^="demo-"]');
    expect(demos).toHaveLength(6);
    expect(w.find('[data-test="demo-VIP 用户"]').exists()).toBe(true);
    expect(w.find('[data-test="demo-陪诊师"]').exists()).toBe(true);
    expect(w.find('[data-test="demo-客服"]').exists()).toBe(true);
  });

  it('onSendCode 有效手机号 → sendSmsCode + 60s 冷却', async () => {
    const w = await mountPage();
    await w.vm.onLoad({});
    w.vm.phone = '13800138001';

    await w.vm.onSendCode();
    await flushPromises();

    expect(fakeAuthStore.sendSmsCode).toHaveBeenCalledWith('13800138001');
    expect(w.vm.smsCooldownSeconds).toBe(60);
  });

  it('onSendCode 无效手机号 → 不调 store + toast 提示', async () => {
    const w = await mountPage();
    w.vm.phone = '12345';

    await w.vm.onSendCode();
    expect(fakeAuthStore.sendSmsCode).not.toHaveBeenCalled();
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '请输入正确的手机号' }),
    );
  });

  it('onLogin（phone+code 全）→ store.loginByPhone + reLaunch', async () => {
    const w = await mountPage();
    await w.vm.onLoad({});
    w.vm.phone = '13800138002';
    w.vm.code = '654321';

    await w.vm.onLogin();
    await flushPromises();

    expect(fakeAuthStore.loginByPhone).toHaveBeenCalledWith('13800138002', '654321');
    expect(mockUni.reLaunch).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch.mock.calls[0][0].url).toBe('/pages/index/index');
  });

  it('登录失败 → toast 提示 + 不调 reLaunch', async () => {
    fakeAuthStore.loginByPhone.mockRejectedValueOnce(new Error('验证码无效'));

    const w = await mountPage();
    w.vm.phone = '13800138002';
    w.vm.code = '000000';

    await w.vm.onLogin();
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '验证码无效' }),
    );
    expect(mockUni.reLaunch).not.toHaveBeenCalled();
  });

  it('点 demo 账号「VIP 用户」 → 自动填 phone + code=123456', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="demo-VIP 用户"]').trigger('click');
    expect(w.vm.phone).toBe('13800138002');
    expect(w.vm.code).toBe('123456');
    // 提示
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: expect.stringContaining('VIP 用户') }),
    );
  });

  it('点 demo「海外测试」 → 自动填 86 开头 11 位手机号', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="demo-海外测试"]').trigger('click');
    expect(w.vm.phone).toBe('8613800138666');
  });

  it('onWechatLogin → store.loginByWechat + reLaunch', async () => {
    const w = await mountPage();
    await w.vm.onLoad({});
    await w.vm.onWechatLogin();
    await flushPromises();

    expect(fakeAuthStore.loginByWechat).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch).toHaveBeenCalledTimes(1);
  });
});
</content>
</invoke>