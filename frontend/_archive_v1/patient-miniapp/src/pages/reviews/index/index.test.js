// src/pages/reviews/index/index.test.js
//
// 我的评价列表页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 4-5 个 it）：
//   1. mounted → 调 review store loadList({ page: 1, page_size: 20 })
//   2. 评价卡片渲染：mine.length>0 → N 张卡片（含订单号 / 星级 / 摘要）
//   3. tab 切换：默认已评价 tab；切到「待评价」tab → 列表清空显示 empty
//   4. 点击卡片 → uni.navigateTo /pages/order/detail?orderId=xxx
//   5. 空状态：loadList 返回 [] → u-empty-stub「暂无评价」

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

// ---- fake review store
const fakeStore = {
  mine: [],
  current: null,
  loading: false,
  submitting: false,
  error: null,
  lastSubmittedId: null,
  loadList: jest.fn(),
  loadDetail: jest.fn(),
  submit: jest.fn(),
};

jest.doMock('@/stores/review.js', () => ({
  useReviewStore: () => fakeStore,
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
    inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" :disabled="disabled" @click.stop="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('reviews/index/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadList.mockReset();
    fakeStore.mine = [];

    // 默认：loadList 返回 2 条评价
    fakeStore.loadList.mockImplementation(async () => {
      fakeStore.mine = [
        { id: 11, order_id: 7, escort_id: 99, rating: 5, comment: '陪诊师非常专业，强烈推荐', reply: '感谢您的好评', created_at: '2026-09-25T10:00:00+08:00' },
        { id: 12, order_id: 8, escort_id: 99, rating: 4, comment: '服务态度好', reply: '', created_at: '2026-09-26T11:00:00+08:00' },
      ];
      return fakeStore.mine;
    });
    fakeStore.loading = false;
  });

  it('mounted → 调 loadList({ page: 1, page_size: 20 })', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);
    expect(fakeStore.loadList).toHaveBeenCalledWith({ page: 1, page_size: 20 });
    expect(w.vm.reviews).toHaveLength(2);
  });

  it('评价卡片渲染：mine.length>0 → N 张卡片 + 5 星 / 4 星 + 已回复 badge', async () => {
    const w = await mountPage();
    await flushPromises();

    const cards = w.findAll('[data-test^="review-card-"]');
    expect(cards).toHaveLength(2);
    expect(cards[0].attributes('data-order-id')).toBe('7');
    // 5 星 + 4 星
    expect(cards[0].text()).toContain('★');
    expect(cards[1].text()).toContain('★');
    // 第一条有 reply → 已回复 badge
    expect(w.find('[data-test="review-reply-badge"]').exists()).toBe(true);
    // 第一条评论文本渲染
    expect(cards[0].text()).toContain('陪诊师非常专业');
  });

  it('tab 切换：切到「待评价」 → 列表清空 + empty state 显示', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(w.vm.currentTab).toBe('reviewed');
    expect(w.findAll('[data-test^="review-card-"]')).toHaveLength(2);

    // 切到「待评价」
    await w.find('[data-test="tab-pending"]').trigger('click');
    expect(w.vm.currentTab).toBe('pending');
    // 列表清空
    expect(w.findAll('[data-test^="review-card-"]')).toHaveLength(0);
    // empty stub 显示「没有待评价订单」
    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('没有待评价订单');
  });

  it('点击评价卡片 → uni.navigateTo /pages/order/detail?orderId=xxx', async () => {
    const w = await mountPage();
    await flushPromises();

    const first = w.find('[data-test="review-card-11"]');
    expect(first.exists()).toBe(true);
    await first.trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/order\/detail\?orderId=7/,
    );
  });

  it('空状态：loadList 返回 [] → u-empty-stub「暂无评价」 + 不渲染卡片', async () => {
    fakeStore.loadList.mockImplementationOnce(async () => {
      fakeStore.mine = [];
      return [];
    });

    const w = await mountPage();
    await flushPromises();

    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无评价');
    expect(w.findAll('[data-test^="review-card-"]')).toHaveLength(0);
  });
});
</content>
</invoke>