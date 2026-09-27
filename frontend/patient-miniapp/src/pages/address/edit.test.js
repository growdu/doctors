// src/pages/address/edit.test.js
//
// 地址编辑页单测 —— @vue/test-utils mount + jest.doMock 注入 fake address store。
//
// 测试点（6 个 it）：
//   1. onLoad(addressId) → 写入 mode + addressId
//   2. 编辑模式 mounted → loadList + findById 写入表单
//   3. canSave 计算属性：联系人 + 11 位手机 + 详细地址 ≥ 5 字
//   4. 「保存」点击（新增模式） → store.add(payload) → navigateBack
//   5. 「保存」点击（编辑模式） → store.update(id, payload) → navigateBack
//   6. 「设为默认」点击（编辑模式） → store.setDefault(id) → navigateBack
//   7. 保存失败 → toast + 不跳转

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { reactive } from 'vue';
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

// ---- fake address store
const fakeAddressStore = reactive({
  list: [],
  defaultId: null,
  loadList: jest.fn(async () => {
    fakeAddressStore.list = [
      { id: 1, recipient: '张三', phone: '13800138000', detail: '北京市东城区某街道 1 号', is_default: true },
      { id: 2, recipient: '李四', phone: '13900139000', detail: '北京市海淀区某街道 2 号', is_default: false },
    ];
    fakeAddressStore.defaultId = 1;
    return fakeAddressStore.list;
  }),
  add: jest.fn(async (payload) => ({
    id: 99, ...payload,
  })),
  update: jest.fn(async (id, payload) => ({ id, ...payload })),
  setDefault: jest.fn(async (id) => ({ id, is_default: true })),
  remove: jest.fn(),
});

jest.doMock('@/stores/address.js', () => ({
  useAddressStore: () => fakeAddressStore,
}));

// ---- uView Plus stub（含 u-switch）
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-switch': {
    props: ['modelValue', 'disabled'],
    emits: ['update:modelValue'],
    template:
      '<button class="u-switch-stub" :data-value="!!modelValue" :disabled="!!disabled" @click="$emit(\'update:modelValue\', !modelValue)"></button>',
  },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled', 'loading'],
    emits: ['click'],
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="!!disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./edit.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('address/edit.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAddressStore.list = [];
    fakeAddressStore.defaultId = null;
    fakeAddressStore.loadList.mockClear();
    fakeAddressStore.add.mockClear();
    fakeAddressStore.update.mockClear();
    fakeAddressStore.setDefault.mockClear();
  });

  it('onLoad(addressId) → 写入 mode + addressId', async () => {
    const w = await mountPage();

    // 无 addressId → 新增模式
    w.vm.onLoad({});
    expect(w.vm.mode).toBe('create');
    expect(w.vm.addressId).toBeNull();

    // 有 addressId → 编辑模式
    w.vm.onLoad({ addressId: 1 });
    expect(w.vm.mode).toBe('edit');
    expect(w.vm.addressId).toBe(1);
  });

  it('编辑模式 mounted → loadList + findById 写入表单', async () => {
    const w = await mountPage();
    w.vm.onLoad({ addressId: 2 });
    await w.vm.mounted();
    await flushPromises();

    expect(fakeAddressStore.loadList).toHaveBeenCalledTimes(1);
    expect(w.vm.recipient).toBe('李四');
    expect(w.vm.phone).toBe('13900139000');
    expect(w.vm.detail).toBe('北京市海淀区某街道 2 号');
    expect(w.vm.isCurrentDefault).toBe(false);
  });

  it('编辑模式 mounted → 当前默认地址 → isCurrentDefault=true + switch disabled', async () => {
    const w = await mountPage();
    w.vm.onLoad({ addressId: 1 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.vm.isCurrentDefault).toBe(true);
    expect(w.find('[data-test="default-switch"]').attributes('disabled')).toBeDefined();
  });

  it('canSave 计算属性：联系人 + 11 位手机 + 详细地址 ≥ 5 字', async () => {
    const w = await mountPage();
    w.vm.onLoad({}); // create mode

    expect(w.vm.canSave).toBe(false);

    await w.setData({ recipient: '张三' });
    expect(w.vm.canSave).toBe(false);

    await w.setData({ phone: '123' });
    expect(w.vm.canSave).toBe(false);

    await w.setData({ phone: '13800138000', detail: '1234' });
    expect(w.vm.canSave).toBe(false);

    await w.setData({ detail: '北京市东城区某街道 1 号' });
    expect(w.vm.canSave).toBe(true);
  });

  it('「保存」点击（新增模式） → store.add(payload) → navigateBack', async () => {
    const w = await mountPage();
    w.vm.onLoad({});
    await w.setData({
      recipient: '张三',
      phone: '13800138000',
      detail: '北京市东城区某街道 1 号',
      isDefault: true,
    });

    await w.find('[data-test="save-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAddressStore.add).toHaveBeenCalledTimes(1);
    expect(fakeAddressStore.add).toHaveBeenCalledWith({
      recipient: '张三',
      phone: '13800138000',
      detail: '北京市东城区某街道 1 号',
      is_default: true,
    });
    expect(mockUni.navigateBack).toHaveBeenCalledTimes(1);
  });

  it('「保存」点击（编辑模式） → store.update(id, payload) → navigateBack', async () => {
    const w = await mountPage();
    w.vm.onLoad({ addressId: 2 });
    await w.vm.mounted();
    await flushPromises();
    // 修改某字段
    await w.setData({ recipient: '李四（更新）' });

    await w.find('[data-test="save-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAddressStore.update).toHaveBeenCalledTimes(1);
    expect(fakeAddressStore.update).toHaveBeenCalledWith(2, expect.objectContaining({
      recipient: '李四（更新）',
      phone: '13900139000',
    }));
    expect(mockUni.navigateBack).toHaveBeenCalledTimes(1);
  });

  it('「设为默认」点击（编辑模式 + 非默认） → store.setDefault(id) → navigateBack', async () => {
    const w = await mountPage();
    w.vm.onLoad({ addressId: 2 });
    await w.vm.mounted();
    await flushPromises();

    // 当前非默认 → 「设为默认」按钮显示
    expect(w.find('[data-test="set-default-btn"]').exists()).toBe(true);

    await w.find('[data-test="set-default-btn"]').trigger('click');
    await flushPromises();

    expect(fakeAddressStore.setDefault).toHaveBeenCalledTimes(1);
    expect(fakeAddressStore.setDefault).toHaveBeenCalledWith(2);
    expect(mockUni.navigateBack).toHaveBeenCalledTimes(1);
  });

  it('「设为默认」按钮在「当前默认地址」时不显示', async () => {
    const w = await mountPage();
    w.vm.onLoad({ addressId: 1 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="set-default-btn"]').exists()).toBe(false);
  });

  it('保存失败 → toast + 不跳转', async () => {
    fakeAddressStore.add.mockRejectedValueOnce(new Error('地址数量已达上限（5 条）'));

    const w = await mountPage();
    w.vm.onLoad({});
    await w.setData({
      recipient: '张三',
      phone: '13800138000',
      detail: '北京市东城区某街道 1 号',
    });

    await w.find('[data-test="save-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && c[0].title === '地址数量已达上限（5 条）',
    );
    expect(toastArg).toBeTruthy();
    expect(mockUni.navigateBack).not.toHaveBeenCalled();
  });
});