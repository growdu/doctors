// src/pages/auth/login.test.js
//
// 登录页单测 —— @vue/test-utils mount + jest.doMock 注入 fake auth store。
//
// 测试点（5 个 it）：
//   1. onLoad(redirect) → redirect 写入组件 data
//   2. 60s 倒计时：点击「获取验证码」→ smsCooldownSeconds=60 + CountdownBadge 渲染
//   3. 「登录」点击 → store.loginByPhone(phone, code) → reLaunch 跳 redirect 或首页
//   4. 「微信登录」点击 → store.loginByWechat({code:'WechatAuth'}) → reLaunch 首页
//   5. canSendSms / canLogin 计算属性：手机号格式 + 6 位验证码
//
// 测试策略：
//   - jest.doMock('@/stores/auth.js') 注入 fake store
//   - jest.doMock('@/components/CountdownBadge.vue', virtual: true) 注入最小 stub
//   - uView Plus 组件全部 stub
//   - 全局 uni mock

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

// ---- CountdownBadge stub（避免拉 vue3-jest 编译子组件时遇到 onMounted 副作用）
const CountdownBadgeStub = {
  name: 'CountdownBadge',
  props: ['expireAt', 'label'],
  template:
    '<div class="countdown-badge-stub" :data-expire="expireAt" :data-label="label"></div>',
};

// ---- fake auth store
const fakeAuthStore = {
  sendSmsCode: jest.fn(),
  loginByPhone: jest.fn(),
  loginByWechat: jest.fn(),
};

jest.doMock('@/stores/auth.js', () => ({
  useAuthStore: () => fakeAuthStore,
}));

// ---- uView Plus stub（含 CountdownBadge）
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled'],
    emits: ['click'],
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="!!disabled" @click="$emit(\'click\')"><slot /></button>',
  },
  // CountdownBadge 通过 mount 的 stubs 选项注入，避免 vue3-jest 处理 .vue 时与 jest.doMock 冲突
  CountdownBadge: CountdownBadgeStub,
};

const mountPage = async () => {
  const mod = await import('./login.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('auth/login.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAuthStore.sendSmsCode.mockReset();
    fakeAuthStore.loginByPhone.mockReset();
    fakeAuthStore.loginByWechat.mockReset();

    fakeAuthStore.sendSmsCode.mockResolvedValue({ sent: true, ttl: 60 });
    fakeAuthStore.loginByPhone.mockResolvedValue({
      access_token: 't-abc-123',
      user: { id: 1, phone: '13800138000' },
    });
    fakeAuthStore.loginByWechat.mockResolvedValue({
      access_token: 't-wx-456',
      user: { id: 1, nickname: '微信用户' },
    });
  });

  it('onLoad(redirect) → 写入组件 data', async () => {
    const w = await mountPage();
    expect(w.vm.redirect).toBe('');
    w.vm.onLoad({ redirect: '/pages/order/index' });
    expect(w.vm.redirect).toBe('/pages/order/index');
  });

  it('60s 倒计时：发送验证码 → smsCooldownSeconds=60 + CountdownBadge 渲染', async () => {
    const w = await mountPage();
    w.vm.phone = '13800138000';
    await w.vm.onSendCode();
    await flushPromises();

    expect(fakeAuthStore.sendSmsCode).toHaveBeenCalledWith('13800138000');
    expect(w.vm.smsCooldownSeconds).toBe(60);
    expect(w.vm.smsCooldownExpireAt).toBeTruthy();
    // 倒计时组件渲染
    expect(w.find('.countdown-badge-stub').exists()).toBe(true);
    expect(w.find('.countdown-badge-stub').attributes('data-expire')).toBe(
      w.vm.smsCooldownExpireAt,
    );
  });

  it('「登录」点击 → store.loginByPhone(phone, code) → reLaunch 跳 redirect 或首页', async () => {
    const w = await mountPage();
    w.vm.onLoad({ redirect: '/pages/order/index' });
    // 用 setData 而非 w.vm.x 直写 —— vue-test-utils v2 + Options API 直写不触发模板重渲染
    await w.setData({ phone: '13800138000', code: '123456' });

    expect(w.vm.canLogin).toBe(true);
    await w.find('[data-test="login-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAuthStore.loginByPhone).toHaveBeenCalledWith('13800138000', '123456');
    expect(mockUni.reLaunch).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch.mock.calls[0][0].url).toBe('/pages/order/index');

    // 无 redirect → 默认跳首页
    w.vm.redirect = '';
    mockUni.reLaunch.mockClear();
    await w.find('[data-test="login-btn"]').trigger('click');
    await flushPromises();
    expect(mockUni.reLaunch.mock.calls[0][0].url).toBe('/pages/index/index');
  });

  it('「微信登录」点击 → store.loginByWechat(WechatAuth) → reLaunch 首页', async () => {
    const w = await mountPage();
    w.vm.onLoad({});

    await w.find('[data-test="wechat-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAuthStore.loginByWechat).toHaveBeenCalledTimes(1);
    const arg = fakeAuthStore.loginByWechat.mock.calls[0][0];
    expect(arg.code).toBe('WechatAuth');
    expect(mockUni.reLaunch).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch.mock.calls[0][0].url).toBe('/pages/index/index');
  });

  it('canSendSms / canLogin 计算属性：手机号格式 + 6 位验证码', async () => {
    const w = await mountPage();
    // 初始：均不可
    expect(w.vm.canSendSms).toBe(false);
    expect(w.vm.canLogin).toBe(false);

    // 手机号有效 → 可发送
    w.vm.phone = '13800138000';
    expect(w.vm.canSendSms).toBe(true);
    expect(w.vm.canLogin).toBe(false); // code 还没填

    // 填 6 位验证码 → 可登录
    w.vm.code = '123456';
    expect(w.vm.canLogin).toBe(true);

    // 验证码不是 6 位 → 不可登录
    w.vm.code = '1234';
    expect(w.vm.canLogin).toBe(false);

    // 手机号非法 → 不可发送
    w.vm.phone = '1234';
    expect(w.vm.canSendSms).toBe(false);
    expect(w.vm.canLogin).toBe(false);
  });

  it('登录失败 → uni.showToast「登录失败，请重试」', async () => {
    fakeAuthStore.loginByPhone.mockRejectedValueOnce(new Error('验证码错误'));

    const w = await mountPage();
    await w.setData({ phone: '13800138000', code: '000000' });

    await w.find('[data-test="login-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && c[0].title === '验证码错误',
    );
    expect(toastArg).toBeTruthy();
    // 不跳转
    expect(mockUni.reLaunch).not.toHaveBeenCalled();
  });
});