// src/pages/messages/index/index.test.js
//
// 「站内信 — 按订单 tab」列表页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 4-5 个 it）：
//   1. mounted → 调 order store loadList({})
//   2. 订单消息卡片渲染：order.length>0 → N 张卡片（含订单号 / summary）
//   3. tab 切换：默认全部；切到「未读」 → 因为 unread mock=0 → 列表空
//   4. 点击卡片 → uni.navigateTo /pages/order/detail?orderId=xxx
//   5. 空状态：loadList 返回 [] → u-empty-stub「暂无订单消息」

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

// ---- fake order store
const fakeStore = {
  list: [],
  loading: false,
  error: null,
  loadList: jest.fn(),
};

jest.doMock('@/stores/order.js', () => ({
  useOrderStore: () => fakeStore,
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
    props: ['type', 'size', 'plain'],
    emits: ['click'],
    inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" @click.stop="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('messages/index/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadList.mockReset();
    fakeStore.list = [];

    // 默认：loadList 返回 3 条订单
    fakeStore.loadList.mockImplementation(async () => {
      fakeStore.list = [
        { id: 7, status: 'inService', appointment_at: '2026-09-28T10:00:00+08:00' },
        { id: 8, status: 'selectingEscort', appointment_at: '2026-09-29T09:30:00+08:00' },
        { id: 9, status: 'completed', appointment_at: '2026-09-20T14:00:00+08:00' },
      ];
      return fakeStore.list;
    });
    fakeStore.loading = false;
  });

  it('mounted → 调 loadList({})', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);
    expect(fakeStore.loadList).toHaveBeenCalledWith({});
    expect(w.vm.cards).toHaveLength(3);
  });

  it('订单消息卡片渲染：3 条订单 → 3 张卡片 + summary + statusLabel', async () => {
    const w = await mountPage();
    await flushPromises();

    const cards = w.findAll('[data-test^="msg-card-"]');
    expect(cards).toHaveLength(3);
    expect(cards[0].attributes('data-order-id')).toBe('7');
    // summary 是按 status 推断的文本
    expect(cards[0].text()).toContain('陪诊师已到达医院');
    expect(cards[1].text()).toContain('请尽快选择陪诊师');
    expect(cards[2].text()).toContain('服务已完成');

    // statusLabel
    expect(w.findAll('[data-test="msg-summary"]')).toHaveLength(3);
  });

  it('tab 切换：切到「未读」 → 因为 unread mock=0 → 列表清空显示 empty', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(w.vm.currentTab).toBe('all');
    expect(w.findAll('[data-test^="msg-card-"]')).toHaveLength(3);

    await w.find('[data-test="tab-unread"]').trigger('click');
    expect(w.vm.currentTab).toBe('unread');

    // 没有 unread → 列表清空
    expect(w.findAll('[data-test^="msg-card-"]')).toHaveLength(0);
    // empty 显示「暂无订单消息」
    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无订单消息');
  });

  it('点击卡片 → uni.navigateTo /pages/order/detail?orderId=xxx', async () => {
    const w = await mountPage();
    await flushPromises();

    const first = w.find('[data-test="msg-card-7"]');
    expect(first.exists()).toBe(true);
    await first.trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/order\/detail\?orderId=7/,
    );
  });

  it('空状态：loadList 返回 [] → u-empty-stub「暂无订单消息」', async () => {
    fakeStore.loadList.mockImplementationOnce(async () => {
      fakeStore.list = [];
      return [];
    });

    const w = await mountPage();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无订单消息');
    expect(w.findAll('[data-test^="msg-card-"]')).toHaveLength(0);
  });
});
</content>
</invoke>