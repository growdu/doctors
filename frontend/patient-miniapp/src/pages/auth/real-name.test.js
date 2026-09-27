// src/pages/auth/real-name.test.js
//
// 实名认证页单测 —— @vue/test-utils mount + jest.doMock 注入 fake auth store。
//
// 测试点（4 个 it）：
//   1. onLoad(redirect) → 写入组件 data
//   2. canSubmit 计算属性：姓名非空 + 18 位身份证（末位 X 允许）
//   3. 「提交」点击 → store.submitRealName({name, id_card}) → navigateBack
//   4. 「提交」点击带 redirect → redirectTo(redirect) 替代 navigateBack
//
// 测试策略：
//   - jest.doMock('@/stores/auth.js') 注入 fake store
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
  submitRealName: jest.fn(),
  fetchMe: jest.fn(),
};

jest.doMock('@/stores/auth.js', () => ({
  useAuthStore: () => fakeAuthStore,
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled', 'loading'],
    emits: ['click'],
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="!!disabled" :data-loading="!!loading" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./real-name.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('auth/real-name.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAuthStore.submitRealName.mockReset();
    fakeAuthStore.fetchMe.mockReset();
    fakeAuthStore.submitRealName.mockResolvedValue({
      id: 1, phone: '13800138000', real_name: '张三', real_name_verified: true,
    });
    fakeAuthStore.fetchMe.mockResolvedValue({
      id: 1, phone: '13800138000', real_name: '张三', real_name_verified: true,
    });
  });

  it('onLoad(redirect) → 写入组件 data', async () => {
    const w = await mountPage();
    expect(w.vm.redirect).toBe('');
    w.vm.onLoad({ redirect: '/pages/order/create' });
    expect(w.vm.redirect).toBe('/pages/order/create');
  });

  it('canSubmit 计算属性：姓名非空 + 18 位身份证（末位 X 允许）', async () => {
    const w = await mountPage();
    // 全空 → false
    expect(w.vm.canSubmit).toBe(false);

    // 仅姓名 → false
    await w.setData({ name: '张三' });
    expect(w.vm.canSubmit).toBe(false);

    // 姓名 + 17 位身份证 → false
    await w.setData({ idCard: '11010119900101123' });
    expect(w.vm.canSubmit).toBe(false);

    // 姓名 + 18 位数字身份证 → true
    await w.setData({ idCard: '110101199001011234' });
    expect(w.vm.canSubmit).toBe(true);

    // 姓名 + 18 位身份证末位 X → true
    await w.setData({ idCard: '11010119900101123X' });
    expect(w.vm.canSubmit).toBe(true);

    // 姓名 + 18 位身份证末位 x → true
    await w.setData({ idCard: '11010119900101123x' });
    expect(w.vm.canSubmit).toBe(true);

    // 身份证含非法字符 → false
    await w.setData({ idCard: '1101011990010112AB' });
    expect(w.vm.canSubmit).toBe(false);

    // 姓名为纯空格 → false
    await w.setData({ name: '   ', idCard: '110101199001011234' });
    expect(w.vm.canSubmit).toBe(false);
  });

  it('「提交」点击 → store.submitRealName({name, id_card}) → navigateBack', async () => {
    const w = await mountPage();
    await w.setData({ name: '张三', idCard: '110101199001011234' });

    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAuthStore.submitRealName).toHaveBeenCalledWith({
      name: '张三',
      id_card: '110101199001011234',
    });
    // fetchMe 二次刷新
    expect(fakeAuthStore.fetchMe).toHaveBeenCalledTimes(1);
    // 无 redirect → 走 navigateBack
    expect(mockUni.navigateBack).toHaveBeenCalledTimes(1);
    expect(mockUni.redirectTo).not.toHaveBeenCalled();
  });

  it('「提交」点击带 redirect → redirectTo(redirect) 替代 navigateBack', async () => {
    const w = await mountPage();
    w.vm.onLoad({ redirect: '/pages/order/create' });
    await w.setData({ name: '张三', idCard: '110101199001011234' });

    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAuthStore.submitRealName).toHaveBeenCalledTimes(1);
    expect(mockUni.redirectTo).toHaveBeenCalledTimes(1);
    expect(mockUni.redirectTo.mock.calls[0][0].url).toBe('/pages/order/create');
    expect(mockUni.navigateBack).not.toHaveBeenCalled();
  });

  it('提交失败 → uni.showToast「认证失败」+ 不跳转', async () => {
    fakeAuthStore.submitRealName.mockRejectedValueOnce(new Error('身份证号格式错误'));

    const w = await mountPage();
    await w.setData({ name: '张三', idCard: '110101199001011234' });

    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && c[0].title === '身份证号格式错误',
    );
    expect(toastArg).toBeTruthy();
    expect(mockUni.navigateBack).not.toHaveBeenCalled();
    expect(mockUni.redirectTo).not.toHaveBeenCalled();
  });
});