// src/pages/hospitals/list.test.js
//
// 医院列表页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 5 个 it）：
//   1. mounted → 调 hospital store loadList({ keyword, level, page: 1, limit: 20 })
//   2. 等级 chips 渲染（全部 / 三甲 / 三乙 / 二甲）+ 切换 chip → loadList 用新 level
//   3. 医院卡片渲染 + 点击 → uni.navigateTo detail
//   4. 搜索框 onSearch → 立即重拉
//   5. 空状态：loadList 返回 [] → u-empty-stub 可见 + 不渲染卡片
//
// 测试策略：
//   - jest.doMock('@/stores/hospital.js') 注入 fake store
//   - uView Plus 组件全部 stub（u-search / u-skeleton / u-empty / u-button）

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

// ---- fake store
const fakeStore = {
  list: [],
  loadList: jest.fn(),
};

jest.doMock('@/stores/hospital.js', () => ({
  useHospitalStore: () => fakeStore,
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': {
    props: ['title', 'autoBack'],
    template: '<div class="u-navbar-stub" :data-title="title"></div>',
  },
  'u-search': {
    props: ['modelValue', 'placeholder', 'showAction'],
    emits: ['update:modelValue', 'search', 'clear'],
    template:
      '<input class="u-search-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" @keyup.enter="$emit(\'search\')" />',
  },
  'u-skeleton': {
    props: ['rows', 'title', 'avatar', 'avatarShape', 'avatarSize'],
    template: '<div class="u-skeleton-stub"></div>',
  },
  'u-empty': {
    props: ['text', 'mode'],
    template: '<div class="u-empty-stub" :data-text="text"></div>',
  },
  'u-button': {
    props: ['type', 'size', 'plain'],
    template:
      '<button class="u-button-stub" :data-type="type" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./list.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('hospitals/list.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadList.mockReset();
    fakeStore.loadList.mockResolvedValue({
      items: [
        { id: 101, name: '北京协和医院', level: '三甲', address: '东城区帅府园 1 号', phone: '010-69155555' },
        { id: 102, name: '华西医院', level: '三甲', address: '国学巷 37 号' },
      ],
      total: 2,
      page: 1,
      limit: 20,
    });
  });

  it('mounted → 调 loadList({ page: 1, limit: 20, keyword: "", level: "" })', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);
    expect(fakeStore.loadList).toHaveBeenCalledWith({
      page: 1,
      limit: 20,
      keyword: '',
      level: '',
    });
    expect(w.vm.hospitals).toHaveLength(2);
  });

  it('等级 chips 渲染 + 切换 → loadList 用新 level', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    // chips 渲染
    expect(w.find('[data-test="level-chip-"]').exists()).toBe(true);     // 全部
    expect(w.find('[data-test="level-chip-三甲"]').exists()).toBe(true);
    expect(w.find('[data-test="level-chip-三乙"]').exists()).toBe(true);
    expect(w.find('[data-test="level-chip-二甲"]').exists()).toBe(true);

    // 点击「三甲」
    await w.find('[data-test="level-chip-三甲"]').trigger('click');
    await flushPromises();
    expect(fakeStore.loadList).toHaveBeenCalledTimes(2);
    expect(fakeStore.loadList).toHaveBeenNthCalledWith(2, expect.objectContaining({
      level: '三甲',
    }));
    expect(w.vm.currentLevel).toBe('三甲');
  });

  it('医院卡片渲染 + 点击 → uni.navigateTo detail', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const cards = w.findAll('[data-test^="hospital-card-"]');
    expect(cards).toHaveLength(2);
    expect(cards[0].attributes('data-hospital-id')).toBe('101');

    await cards[0].trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/hospitals\/detail\?id=101/,
    );
  });

  it('搜索框 onSearch → 重拉 + 使用当前 keyword', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();
    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);

    // 直接调 onSearch（绕过 u-search input 事件）
    w.vm.keyword = '协和';
    await w.vm.onSearch();
    await flushPromises();
    expect(fakeStore.loadList).toHaveBeenCalledTimes(2);
    expect(fakeStore.loadList).toHaveBeenNthCalledWith(2, expect.objectContaining({
      keyword: '协和',
    }));
  });

  it('搜索框 onClear → 清空 keyword + 重拉', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    w.vm.keyword = '协和';
    await w.vm.onClear();
    expect(w.vm.keyword).toBe('');
    expect(fakeStore.loadList).toHaveBeenCalledTimes(2);
    expect(fakeStore.loadList).toHaveBeenNthCalledWith(2, expect.objectContaining({
      keyword: '',
    }));
  });

  it('空状态：loadList 返回 [] → u-empty-stub 可见', async () => {
    fakeStore.loadList.mockResolvedValueOnce({ items: [], total: 0, page: 1, limit: 20 });

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无可用医院');
    expect(w.findAll('[data-test^="hospital-card-"]')).toHaveLength(0);
  });

  it('错误态：loadList 抛错 → u-empty-stub「加载失败」 + 重试按钮', async () => {
    fakeStore.loadList.mockRejectedValueOnce(new Error('boom'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('加载失败');
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
  });
});