// src/pages/message/list.test.js
//
// 站内信列表页单测 —— @vue/test-utils mount + jest.doMock 注入 fake message store。
//
// 测试点（6 个 it）：
//   1. mounted 调 store.loadList()
//   2. 未读 banner：unread>0 显示 + 数字正确
//   3. 列表渲染：每条 message-{id} 渲染 + 未读红点
//   4. 「全部已读」点击 → store.markAllRead()
//   5. 点击条目 → store.loadDetail(id) → navigateTo 详情页
//   6. loadList 抛错 → error-state + 重试按钮
//
// 测试策略：
//   - jest.doMock('@/stores/message.js') 注入 fake store
//   - uView Plus 组件全部 stub（含 u-skeleton / u-empty / u-button / u-navbar）

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

// ---- fake message store
const fakeMessageStore = reactive({
  list: [],
  current: null,
  unread: 0,
  pagination: { total: 0, page: 1, page_size: 20 },
  loading: false,
  loadingDetail: false,
  error: null,
  loadList: jest.fn(async (query) => {
    fakeMessageStore.list = [
      { id: 1, category: 'order', title: '订单创建成功', content: '您的订单 #7 已创建', read_at: null, created_at: '2026-09-22T10:00:00Z' },
      { id: 2, category: 'escort', title: '陪诊师已接单', content: '张三已确认接单', read_at: '2026-09-22T11:00:00Z', created_at: '2026-09-22T11:00:00Z' },
      { id: 3, category: 'system', title: '系统升级通知', content: '平台将于本周日 02:00-04:00 升级', read_at: null, created_at: '2026-09-23T09:00:00Z' },
    ];
    fakeMessageStore.unread = 2;
    fakeMessageStore.pagination = { total: 3, page: (query && query.page) || 1, page_size: (query && query.page_size) || 20 };
    return { items: fakeMessageStore.list, unread: 2, ...fakeMessageStore.pagination };
  }),
  loadDetail: jest.fn(async (id) => {
    fakeMessageStore.current = fakeMessageStore.list.find((m) => String(m.id) === String(id)) || null;
    if (fakeMessageStore.current && !fakeMessageStore.current.read_at) {
      fakeMessageStore.current.read_at = new Date().toISOString();
      if (fakeMessageStore.unread > 0) fakeMessageStore.unread -= 1;
    }
    return fakeMessageStore.current;
  }),
  markAllRead: jest.fn(async () => {
    const now = new Date().toISOString();
    fakeMessageStore.list = fakeMessageStore.list.map((m) => Object.assign({}, m, { read_at: m.read_at || now }));
    fakeMessageStore.unread = 0;
    return { updated: fakeMessageStore.list.length };
  }),
  clearAll: jest.fn(),
});

jest.doMock('@/stores/message.js', () => ({
  useMessageStore: () => fakeMessageStore,
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-skeleton': { template: '<div class="u-skeleton-stub"></div>' },
  'u-empty': { template: '<div class="u-empty-stub"></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled', 'loading'],
    emits: ['click'],
        inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" :disabled="!!disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./list.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('message/list.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeMessageStore.list = [];
    fakeMessageStore.unread = 0;
    fakeMessageStore.loadList.mockClear();
    fakeMessageStore.loadDetail.mockClear();
    fakeMessageStore.markAllRead.mockClear();
  });

  it('mounted 调 store.loadList()', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();
    expect(fakeMessageStore.loadList).toHaveBeenCalledWith({ page: 1, page_size: 20 });
  });

  it('未读 banner：unread>0 显示 + 数字正确', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="unread-banner"]').exists()).toBe(true);
    expect(w.find('[data-test="unread-banner"]').text()).toContain('2 条未读');
  });

  it('列表渲染：每条 message-{id} 渲染 + 未读红点', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="message-1"]').exists()).toBe(true);
    expect(w.find('[data-test="message-2"]').exists()).toBe(true);
    expect(w.find('[data-test="message-3"]').exists()).toBe(true);

    // 未读红点：1 + 3 各一个（共 2 个），2 已读无红点
    const dots = w.findAll('[data-test="unread-dot"]');
    expect(dots.length).toBe(2);
    // 标题渲染
    expect(w.find('[data-test="message-1"]').text()).toContain('订单创建成功');
    expect(w.find('[data-test="message-2"]').text()).toContain('陪诊师已接单');
  });

  it('「全部已读」点击 → store.markAllRead() + toast', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="mark-all-btn"]').trigger('click');
    await flushPromises();

    expect(fakeMessageStore.markAllRead).toHaveBeenCalledTimes(1);
    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && c[0].title === '全部已读',
    );
    expect(toastArg).toBeTruthy();
  });

  it('点击条目 → store.loadDetail(id) → navigateTo 详情页', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    await w.find('[data-test="message-1"]').trigger('click');
    await flushPromises();

    expect(fakeMessageStore.loadDetail).toHaveBeenCalledWith(1);
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe('/pages/message/detail?id=1');
  });

  it('loadList 抛错 → error-state + 重试按钮（点击重试）', async () => {
    fakeMessageStore.loadList.mockRejectedValueOnce(new Error('network down'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="error-state"]').exists()).toBe(true);
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);

    // 重试：loadList 再调一次
    fakeMessageStore.loadList.mockRejectedValueOnce(new Error('still down'));
    await w.find('[data-test="retry-btn"]').trigger('click');
    await flushPromises();
    expect(fakeMessageStore.loadList).toHaveBeenCalledTimes(2);
  });

  it('空列表：u-empty 显示 + 全部已读按钮不渲染', async () => {
    fakeMessageStore.loadList.mockImplementationOnce(async () => {
      fakeMessageStore.list = [];
      fakeMessageStore.unread = 0;
      return { items: [], unread: 0, total: 0, page: 1, page_size: 20 };
    });

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="empty"]').exists()).toBe(true);
    expect(w.find('[data-test="mark-all-btn"]').exists()).toBe(false);
  });
});