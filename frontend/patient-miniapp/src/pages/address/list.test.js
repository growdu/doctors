// src/pages/address/list.test.js
//
// 地址管理页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 5 个 it）：
//   1. mounted → 调 address store loadList()
//   2. 地址卡片渲染 + 「默认」徽标
//   3. 「设为默认」点击 → store.setDefault(id) + 列表重拉
//   4. 「删除」点击 → store.remove(id) + 列表重拉
//   5. 「+ 新增地址」 → 打开 dialog → 填写 → 保存 → store.add

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { jest } from '@jest/globals';

// ---- 全局 uni mock
const mockUni = {
  showToast: jest.fn(),
  showModal: jest.fn((opts) => {
    // 默认 confirm=true（自动执行删除）
    if (opts && typeof opts.success === 'function') {
      opts.success({ confirm: true, cancel: false });
    }
  }),
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

// ---- fake address store
const fakeAddressStore = {
  list: [],
  defaultId: null,
  lastAddedId: null,
  loading: false,
  error: null,
  loadList: jest.fn(),
  add: jest.fn(),
  update: jest.fn(),
  remove: jest.fn(),
  setDefault: jest.fn(),
  defaultAddress: jest.fn(() => null),
};

jest.doMock('@/stores/address.js', () => ({
  useAddressStore: () => fakeAddressStore,
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': {
    props: ['title', 'autoBack'],
    template: '<div class="u-navbar-stub" :data-title="title"></div>',
  },
  'u-skeleton': {
    props: ['rows', 'title'],
    template: '<div class="u-skeleton-stub"></div>',
  },
  'u-empty': {
    props: ['text', 'mode'],
    template: '<div class="u-empty-stub" :data-text="text"></div>',
  },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled'],
    emits: ['click'],
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./list.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('address/list.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeAddressStore.loadList.mockReset();
    fakeAddressStore.setDefault.mockReset();
    fakeAddressStore.remove.mockReset();
    fakeAddressStore.add.mockReset();

    // 默认：loadList 返回 2 条地址
    fakeAddressStore.loadList.mockResolvedValue([
      { id: 1, recipient: '张三', phone: '13800138000', detail: '北京市东城区某街道 1 号', is_default: true },
      { id: 2, recipient: '李四', phone: '13900139000', detail: '北京市朝阳区某街道 2 号', is_default: false },
    ]);
    fakeAddressStore.setDefault.mockResolvedValue({});
    fakeAddressStore.remove.mockResolvedValue(undefined);
    fakeAddressStore.add.mockResolvedValue({ id: 100, recipient: 'x' });
  });

  it('mounted → 调 address store loadList()', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeAddressStore.loadList).toHaveBeenCalledTimes(1);
    expect(w.vm.addresses).toHaveLength(2);
    expect(w.vm.defaultId).toBe(1);
  });

  it('地址卡片渲染 + 「默认」徽标', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const cards = w.findAll('[data-test^="address-card-"]');
    expect(cards).toHaveLength(2);
    expect(cards[0].attributes('data-address-id')).toBe('1');

    // 默认徽标：仅 id=1 显示
    expect(w.find('[data-test="default-badge"]').exists()).toBe(true);
    // 「设为默认」按钮仅非默认项显示
    expect(w.find('[data-test="set-default-2"]').exists()).toBe(true);
    expect(w.find('[data-test="set-default-1"]').exists()).toBe(false);
  });

  it('「设为默认」点击 → store.setDefault(id) + 重拉列表', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="set-default-2"]').trigger('click');
    await flushPromises();
    expect(fakeAddressStore.setDefault).toHaveBeenCalledWith(2);
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '已设为默认' }),
    );
    // 重拉
    expect(fakeAddressStore.loadList).toHaveBeenCalledTimes(2);
  });

  it('「删除」点击 → store.remove(id) + 重拉列表', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="delete-2"]').trigger('click');
    await flushPromises();
    // mockUni.showModal 默认 confirm=true → 走删除
    expect(mockUni.showModal).toHaveBeenCalled();
    expect(fakeAddressStore.remove).toHaveBeenCalledWith(2);
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '已删除' }),
    );
  });

  it('「+ 新增地址」 → 打开 dialog → 填写 → 保存 → store.add', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    // 打开 dialog
    await w.find('[data-test="add-address-btn"]').trigger('click');
    expect(w.vm.showAddDialog).toBe(true);
    expect(w.find('[data-test="add-dialog"]').exists()).toBe(true);

    // 填写
    w.vm.draft = { recipient: '王五', phone: '13700137000', detail: '上海市浦东新区' };
    await w.vm.onAddConfirm();
    await flushPromises();
    expect(fakeAddressStore.add).toHaveBeenCalledWith({
      recipient: '王五',
      phone: '13700137000',
      detail: '上海市浦东新区',
    });
    expect(w.vm.showAddDialog).toBe(false);
  });

  it('dialog 「保存」缺字段 → 不调 add + toast 提示', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="add-address-btn"]').trigger('click');
    // 不填直接保存
    await w.vm.onAddConfirm();
    expect(fakeAddressStore.add).not.toHaveBeenCalled();
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '请填写完整' }),
    );
  });

  it('空状态：loadList 返回 [] → u-empty-stub「暂无地址」', async () => {
    fakeAddressStore.loadList.mockResolvedValueOnce([]);

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无地址');
    expect(w.findAll('[data-test^="address-card-"]')).toHaveLength(0);
  });

  it('错误态：loadList 抛错 → error-state + 重试按钮', async () => {
    fakeAddressStore.loadList.mockRejectedValueOnce(new Error('boom'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('加载失败');
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
  });
});