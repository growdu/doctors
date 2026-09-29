// src/pages/message/detail.test.js
//
// 站内信详情页单测 —— @vue/test-utils mount + jest.doMock 注入 fake message store。
//
// 测试点（5 个 it）：
//   1. onLoad(id) → 写入组件 data
//   2. mounted 调 store.loadDetail(id) + 渲染 from / to / time / content / title
//   3. category chip 显示
//   4. 「查看详情」点击（带 link） → navigateTo(msg.link)
//   5. loadDetail 抛错 → error-state + 重试按钮

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
  current: null,
  loadingDetail: false,
  loadDetail: jest.fn(async (id) => {
    fakeMessageStore.current = {
      id,
      category: 'order',
      title: '订单 #7 创建成功',
      from: '订单系统',
      to: '我',
      content: '您的订单 #7 已创建，请耐心等待陪诊师接单。\n如有疑问请联系客服。',
      read_at: new Date().toISOString(),
      created_at: '2026-09-22T10:00:00Z',
      link: '/pages/order/detail?orderId=7',
    };
    return fakeMessageStore.current;
  }),
  loadList: jest.fn(),
  markAllRead: jest.fn(),
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
  const mod = await import('./detail.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('message/detail.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeMessageStore.current = null;
    fakeMessageStore.loadDetail.mockClear();
  });

  it('onLoad(id) → 写入组件 data', async () => {
    const w = await mountPage();
    w.vm.onLoad({ id: 7 });
    await w.vm.mounted();
    expect(w.vm.id).toBe(7);
  });

  it('mounted 调 store.loadDetail(id) + 渲染 from / to / time / content / title', async () => {
    const w = await mountPage();
    w.vm.onLoad({ id: 7 });
    await w.vm.mounted();
    await flushPromises();

    expect(fakeMessageStore.loadDetail).toHaveBeenCalledWith(7);

    // 卡片渲染
    expect(w.find('[data-test="detail-card"]').exists()).toBe(true);
    expect(w.find('[data-test="title"]').text()).toBe('订单 #7 创建成功');
    expect(w.find('[data-test="from-value"]').text()).toBe('订单系统');
    expect(w.find('[data-test="to-value"]').text()).toBe('我');
    expect(w.find('[data-test="time-value"]').text()).toBe('2026-09-22 18:00:00');
    // content 是多行文本
    expect(w.find('[data-test="content"]').text()).toContain('订单 #7 已创建');
  });

  it('category chip 显示', async () => {
    const w = await mountPage();
    w.vm.onLoad({ id: 7 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="category-chip"]').exists()).toBe(true);
    expect(w.find('[data-test="category-chip"]').text()).toBe('order');
  });

  it('「查看详情」点击（带 link） → navigateTo(msg.link)', async () => {
    const w = await mountPage();
    w.vm.onLoad({ id: 7 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="link-btn"]').exists()).toBe(true);
    await w.find('[data-test="link-btn"]').trigger('click');

    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toBe(
      '/pages/order/detail?orderId=7',
    );
  });

  it('无 link → 「查看详情」按钮不显示', async () => {
    fakeMessageStore.loadDetail.mockImplementationOnce(async (id) => {
      fakeMessageStore.current = {
        id,
        category: 'system',
        title: '系统升级',
        from: '系统',
        to: '我',
        content: '平台升级通知',
        read_at: null,
        created_at: '2026-09-22T10:00:00Z',
        // 无 link
      };
      return fakeMessageStore.current;
    });

    const w = await mountPage();
    w.vm.onLoad({ id: 8 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="link-btn"]').exists()).toBe(false);
  });

  it('loadDetail 抛错 → error-state + 重试按钮', async () => {
    fakeMessageStore.loadDetail.mockRejectedValueOnce(new Error('network down'));

    const w = await mountPage();
    w.vm.onLoad({ id: 7 });
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="error-state"]').exists()).toBe(true);
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);
  });

  it('loadDetail 返回 null → missing state', async () => {
    fakeMessageStore.loadDetail.mockImplementationOnce(async () => {
      fakeMessageStore.current = null;
      return null;
    });

    const w = await mountPage();
    w.vm.onLoad({ id: 999 });
    await w.vm.mounted();
    await flushPromises();

    // loadDetail 已 resolve 但 current=null → loadError=true → error-state
    // （与「消息不存在」missing 二选一：当前实现优先 error-state）
    expect(
      w.find('[data-test="error-state"]').exists() || w.find('[data-test="missing"]').exists(),
    ).toBe(true);
  });
});