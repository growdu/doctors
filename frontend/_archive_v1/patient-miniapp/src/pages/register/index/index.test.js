// src/pages/register/index/index.test.js
//
// 注册页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 4-5 个 it）：
//   1. mounted → 渲染手机号 + 验证码 + 协议 + 注册按钮
//   2. 协议未勾选 → 注册按钮 disabled（即便 phone+code 都填）
//   3. 协议勾选 + phone+code 全 → 注册按钮 enabled → click 调 store.loginByPhone + reLaunch
//   4. onSendCode 有效手机号 → 60s 冷却
//   5. 点击协议 checkbox toggle → agreed 反向

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

const CountdownBadge = {
  name: 'CountdownBadge',
  props: ['expireAt', 'label'],
  template: '<div class="countdown-badge-stub" :data-expire="expireAt"></div>',
};

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

describe('register/index/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAuthStore.sendSmsCode.mockReset();
    fakeAuthStore.loginByPhone.mockReset();
    fakeAuthStore.sendSmsCode.mockResolvedValue({ sent: true, ttl: 60 });
    fakeAuthStore.loginByPhone.mockResolvedValue({ access_token: 'tkn' });
  });

  it('mounted → 渲染手机号 + 验证码 + 协议（未勾）+ 注册按钮', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(w.find('[data-test="register-page"]').exists()).toBe(true);
    expect(w.find('[data-test="phone-input"]').exists()).toBe(true);
    expect(w.find('[data-test="code-input"]').exists()).toBe(true);
    expect(w.find('[data-test="agreement-row"]').exists()).toBe(true);
    expect(w.find('[data-test="register-btn"]').exists()).toBe(true);

    // 协议未勾
    expect(w.vm.agreed).toBe(false);
    expect(w.find('[data-test="disagree-checkbox"]').exists()).toBe(true);
  });

  it('协议未勾选 → 注册按钮 disabled（即便 phone+code 都填）', async () => {
    const w = await mountPage();
    w.vm.phone = '13800138001';
    w.vm.code = '123456';
    expect(w.vm.agreed).toBe(false);
    expect(w.vm.canRegister).toBe(false);

    const btn = w.find('[data-test="register-btn"]');
    expect(btn.attributes('disabled')).toBeDefined();
  });

  it('协议 checkbox toggle → agreed 反向', async () => {
    const w = await mountPage();
    expect(w.vm.agreed).toBe(false);
    // 点第 1 个 checkbox
    await w.find('[data-test="disagree-checkbox"]').trigger('click');
    expect(w.vm.agreed).toBe(true);
    // 现在再点
    await w.find('[data-test="agreed-checkbox"]').trigger('click');
    expect(w.vm.agreed).toBe(false);
  });

  it('协议勾选 + phone+code 全 → 注册按钮 enabled → click → store.loginByPhone + reLaunch', async () => {
    const w = await mountPage();
    await w.vm.onLoad({});
    w.vm.phone = '13800138001';
    w.vm.code = '654321';
    await w.find('[data-test="disagree-checkbox"]').trigger('click');

    expect(w.vm.canRegister).toBe(true);
    const btn = w.find('[data-test="register-btn"]');
    expect(btn.attributes('disabled')).toBeUndefined();

    await btn.trigger('click');
    await flushPromises();

    expect(fakeAuthStore.loginByPhone).toHaveBeenCalledWith('13800138001', '654321');
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '注册成功' }),
    );
    expect(mockUni.reLaunch).toHaveBeenCalledTimes(1);
    expect(mockUni.reLaunch.mock.calls[0][0].url).toBe('/pages/index/index');
  });

  it('onSendCode 有效手机号 → 60s 冷却', async () => {
    const w = await mountPage();
    w.vm.phone = '13800138001';

    await w.vm.onSendCode();
    await flushPromises();

    expect(fakeAuthStore.sendSmsCode).toHaveBeenCalledWith('13800138001');
    expect(w.vm.smsCooldownSeconds).toBe(60);
  });

  it('注册失败 → toast 错误 + 不调 reLaunch', async () => {
    fakeAuthStore.loginByPhone.mockRejectedValueOnce(new Error('手机号已被使用'));

    const w = await mountPage();
    w.vm.phone = '13800138001';
    w.vm.code = '654321';
    await w.find('[data-test="disagree-checkbox"]').trigger('click');

    await w.vm.onRegister();
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '手机号已被使用' }),
    );
    expect(mockUni.reLaunch).not.toHaveBeenCalled();
  });
});
</content>
</invoke>