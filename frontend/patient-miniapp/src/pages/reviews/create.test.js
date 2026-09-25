// src/pages/reviews/create.test.js
//
// 评价创建页单测 —— @vue/test-utils mount + mock store。
//
// 测试点（brief 要求 5 个 it）：
//   1. onLoad 读 ?orderId=7&escortId=11 → 写入 orderId / escortId
//   2. 5 颗星渲染 + 点击第 3 颗 → rating=3
//   3. 未评分 → 「提交」按钮 disabled；评分后 → enabled
//   4. 评论输入 → textarea 双向绑定
//   5. 「提交」点击 → store.submit({...}) + redirectTo 订单详情
//
// 测试策略：
//   - jest.doMock('@/stores/review.js') 注入 fake store
//   - uView Plus 组件全部 stub

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
const fakeReviewStore = {
  mine: [],
  current: null,
  submitting: false,
  lastSubmittedId: null,
  submit: jest.fn(),
  loadList: jest.fn(),
  loadDetail: jest.fn(),
};

jest.doMock('@/stores/review.js', () => ({
  useReviewStore: () => fakeReviewStore,
}));

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': {
    props: ['title', 'autoBack'],
    template: '<div class="u-navbar-stub" :data-title="title"></div>',
  },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled'],
    emits: ['click'],
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
  },
  'u-loading': {
    props: ['mode', 'size'],
    template: '<div class="u-loading-stub"></div>',
  },
};

const mountPage = async () => {
  const mod = await import('./create.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('reviews/create.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeReviewStore.submit.mockReset();
    fakeReviewStore.submit.mockResolvedValue({
      id: 999,
      order_id: 7,
      escort_id: 11,
      rating: 5,
      comment: '专业',
    });
  });

  it('onLoad 读 ?orderId=7&escortId=11 → 写入 orderId / escortId', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7, escortId: 11 });
    expect(w.vm.orderId).toBe(7);
    expect(w.vm.escortId).toBe(11);
  });

  it('5 颗星渲染 + 点击第 3 颗 → rating=3', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7, escortId: 11 });

    for (let n = 1; n <= 5; n += 1) {
      expect(w.find(`[data-test="star-${n}"]`).exists()).toBe(true);
    }

    await w.find('[data-test="star-3"]').trigger('click');
    expect(w.vm.rating).toBe(3);
    // rating-text 显示「3 星」
    expect(w.find('[data-test="rating-text"]').text()).toBe('3 星');

    // 改点第 5 颗 → rating=5
    await w.find('[data-test="star-5"]').trigger('click');
    expect(w.vm.rating).toBe(5);
    expect(w.find('[data-test="rating-text"]').text()).toBe('5 星');
  });

  it('初始未评分 → 「提交」按钮 disabled', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7, escortId: 11 });

    const btn = w.find('[data-test="submit-btn"]');
    expect(btn.attributes('disabled')).toBeDefined();
    expect(w.vm.canSubmit).toBe(false);
  });

  it('评分后 → 「提交」按钮 enabled + canSubmit=true', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7, escortId: 11 });

    await w.find('[data-test="star-4"]').trigger('click');
    const btn = w.find('[data-test="submit-btn"]');
    expect(btn.attributes('disabled')).toBeUndefined();
    expect(w.vm.canSubmit).toBe(true);
  });

  it('「提交」点击 → store.submit({order_id, escort_id, rating, comment}) + redirectTo', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7, escortId: 11 });
    await w.find('[data-test="star-5"]').trigger('click');
    w.vm.comment = '非常专业，强烈推荐';

    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(fakeReviewStore.submit).toHaveBeenCalledTimes(1);
    expect(fakeReviewStore.submit).toHaveBeenCalledWith({
      order_id: 7,
      escort_id: 11,
      rating: 5,
      comment: '非常专业，强烈推荐',
    });
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '评价成功' }),
    );
    expect(mockUni.redirectTo).toHaveBeenCalledTimes(1);
    expect(mockUni.redirectTo.mock.calls[0][0].url).toBe(
      '/pages/order/detail?orderId=7',
    );
  });

  it('提交失败 → toast + 不跳转', async () => {
    fakeReviewStore.submit.mockRejectedValueOnce(new Error('rating invalid'));

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7, escortId: 11 });
    await w.find('[data-test="star-3"]').trigger('click');
    await w.find('[data-test="submit-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'rating invalid' }),
    );
    expect(mockUni.redirectTo).not.toHaveBeenCalled();
    expect(w.vm.submitting).toBe(false);
  });

  it('订单信息缺失 → toast「订单信息缺失」+ 不调 submit', async () => {
    const w = await mountPage();
    // 不调 onLoad → orderId / escortId 为 null
    await w.find('[data-test="star-5"]').trigger('click');
    await w.find('[data-test="submit-btn"]').trigger('click');

    expect(fakeReviewStore.submit).not.toHaveBeenCalled();
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '订单信息缺失' }),
    );
  });
});