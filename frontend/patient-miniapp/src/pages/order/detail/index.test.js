// src/pages/order/detail/index.test.js
//
// 订单详情页单测 —— @vue/test-utils mount + 模拟 uni-app 生命周期 + mock store。
//
// 测试点（brief 要求 6 个 it）：
//   1. onLoad+loadOrder：传入 ?orderId=7 → mounted 调 useOrderStore().loadOrder(7)
//   2. OrderStatusProgress 渲染：默认 6 节点 + 当前状态 current 高亮
//   3. selectingEscort：「去选陪诊师」按钮存在 + 点击 → uni.navigateTo 到 candidates
//   4. escortPendingAcceptance：CountdownBadge 存在且 label='陪诊师确认剩余' + 显示已选 escortId
//   5. 拒接回退 Alert：order.escortRejectReason='escort_declined' → 文案 + 重新选择按钮
//   6. 错误状态：loadOrder 抛错 → 显示 error-state + 重试按钮 + 点击重试再次调 loadOrder
//
// 测试策略：
//   - jest.doMock('@/stores/order.js') 注入 fake store
//   - jest.doMock('@/components/OrderStatusProgress.vue', ...) 替换为 stub（避免依赖 u-icon 等子组件）
//     —— 但 CountdownBadge 走真实组件，因为它本身的单测已覆盖倒计时；这里只验证 mount + 参数透传
//   - uView Plus 组件全部 stub
//   - onLoad / onUnload 是 uni-app 扩展生命周期：通过 wrapper.vm.onLoad(query) 手动触发

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
  current: null,
  loadOrder: jest.fn(),
  stopPolling: jest.fn(),
};

jest.doMock('@/stores/order.js', () => ({
  useOrderStore: () => fakeStore,
}));

// ---- 子组件 stub：OrderStatusProgress 用最小化模板替换（避免拉 uView Plus + 复杂模板）
const OrderStatusProgressStub = {
  name: 'OrderStatusProgress',
  props: ['currentStatus', 'steps'],
  template:
    '<div class="osp-stub" :data-current="currentStatus" :data-step-count="steps && steps.length"></div>',
};

// CountdownBadge 走真实组件（其本身已有单测覆盖倒计时）；本页只验证 mount + 属性透传。
jest.doMock(
  '@/components/OrderStatusProgress.vue',
  () => ({ default: OrderStatusProgressStub }),
  { virtual: true },
);

// ---- uView Plus + uni-app stub
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
    template: '<button class="u-button-stub" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('order/detail/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadOrder.mockReset();
    fakeStore.stopPolling.mockReset();
  });

  it('onLoad(orderId) → mounted 触发 loadOrder(orderId)', async () => {
    fakeStore.loadOrder.mockResolvedValue({
      id: 7, status: 'selectingEscort', amount: 200, selectedEscortId: null,
    });

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadOrder).toHaveBeenCalledWith(7);
    // 订单已写入组件本地
    expect(w.vm.orderId).toBe(7);
    expect(w.vm.order.status).toBe('selectingEscort');
  });

  it('OrderStatusProgress 渲染：默认 6 节点 + current=' + '7 状态下走对应映射', async () => {
    // accepted 状态：让 store 返回 status=escortConfirmed（详页内映射为 accepted 字符串）
    fakeStore.loadOrder.mockResolvedValue({
      id: 7, status: 'escortConfirmed', amount: 200, selectedEscortId: 11,
    });

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    // 通过 stub 的 data-current 验证：escortConfirmed → 'accepted' 的映射生效
    const osp = w.find('.osp-stub');
    expect(osp.exists()).toBe(true);
    // 进度条组件的 currentStatus prop 应为 'accepted'（view 层做了 escortConfirmed → accepted 的映射）
    expect(osp.attributes('data-current')).toBe('accepted');
    expect(osp.attributes('data-step-count')).toBe('6');
  });

  it('selectingEscort：「去选陪诊师」按钮可见 + 点击 → navigateTo candidates', async () => {
    fakeStore.loadOrder.mockResolvedValue({
      id: 7, status: 'selectingEscort', amount: 200,
    });

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    const gotoBtn = w.find('[data-test="goto-candidates-btn"]');
    expect(gotoBtn.exists()).toBe(true);
    expect(gotoBtn.text()).toContain('去选陪诊师');

    // 点击 → uni.navigateTo('/pages/order/candidates/index?orderId=7')
    await gotoBtn.trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/order\/candidates\/index\?orderId=7/,
    );
  });

  it('escortPendingAcceptance：CountdownBadge 存在 + 「已选陪诊师 #11」显示', async () => {
    fakeStore.loadOrder.mockResolvedValue({
      id: 7,
      status: 'escortPendingAcceptance',
      amount: 200,
      selectedEscortId: 11,
      escortPendingExpireAt: '2026-09-24T15:30:30+08:00',
    });

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    // CountdownBadge 真实组件应已挂载
    const badge = w.find('.countdown-badge');
    expect(badge.exists()).toBe(true);
    expect(w.text()).toContain('陪诊师确认剩余');
    expect(w.text()).toContain('已选陪诊师 #11');
  });

  it('拒接回退 Alert：escortRejectReason 非空 → 文案 + 重新选择按钮 → 跳 candidates', async () => {
    fakeStore.loadOrder.mockResolvedValue({
      id: 7,
      status: 'selectingEscort',
      amount: 200,
      escortRejectReason: 'escort_declined',
    });

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    const alert = w.find('[data-test="reject-alert"]');
    expect(alert.exists()).toBe(true);
    expect(w.text()).toContain('陪诊师主动拒接');
    expect(w.text()).toContain('重新选择陪诊师');

    // 点击「重新选择陪诊师」→ uni.navigateTo candidates
    const reselectBtn = w.find('[data-test="reselect-btn"]');
    await reselectBtn.trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/order\/candidates\/index\?orderId=7/,
    );
  });

  it('错误状态：loadOrder 抛错 → 显示重试按钮 + 点击再次调 loadOrder', async () => {
    // 首次失败
    fakeStore.loadOrder.mockRejectedValueOnce(new Error('network down'));

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    // 显示 error state
    const errState = w.find('[data-test="error-state"]');
    expect(errState.exists()).toBe(true);

    // 重试按钮存在
    const retryBtn = w.find('[data-test="retry-btn"]');
    expect(retryBtn.exists()).toBe(true);

    // 重置 mock，二次成功返回
    fakeStore.loadOrder.mockResolvedValueOnce({
      id: 7, status: 'selectingEscort', amount: 200,
    });
    await retryBtn.trigger('click');
    await flushPromises();

    // 二次调用 loadOrder(7) → 渲染恢复正常态（status-card 出现）
    expect(fakeStore.loadOrder).toHaveBeenCalledTimes(2);
    expect(fakeStore.loadOrder).toHaveBeenNthCalledWith(2, 7);
    expect(w.find('[data-test="status-card"]').exists()).toBe(true);
    // 错误态消失
    expect(w.find('[data-test="error-state"]').exists()).toBe(false);
    // 错误 toast 被调
    expect(mockUni.showToast).toHaveBeenCalled();
  });
});
