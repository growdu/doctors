// src/pages/order/index.test.js
//
// 订单列表页单测 —— @vue/test-utils mount + 模拟 uni-app 生命周期 + mock store。
//
// 测试点（brief 要求 5 个 it）：
//   1. onLoad+loadList：传入 ?status=selectingEscort → mounted 调
//      useOrderStore().loadList({ status: 'selectingEscort' })；默认无 status 走 {}
//   2. 状态 tab 切换：模拟 u-tabs click → 再次调 loadList(新 status)
//   3. OrderListItem 渲染：orders.length>0 → N 张 OrderListItem-stub
//   4. 点击跳 detail：OrderListItem emit view → uni.navigateTo('/pages/order/detail?orderId=xxx')
//   5. 空状态：orders=[] → u-empty-stub 可见 + OrderListItem-stub 不渲染
//
// 测试策略：
//   - jest.doMock('@/stores/order.js') 注入 fake store
//   - jest.doMock('@/components/OrderListItem.vue', ...) 用最小 stub 替换
//     （避免依赖 uView Plus 等子组件）
//   - uView Plus 组件全部 stub
//   - onLoad 是 uni-app 扩展生命周期：通过 wrapper.vm.onLoad(query) 手动触发
//
// 备注：
//   - u-tabs 的 @click 会传 { index, ... } 入参；模拟时传标准入参

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
  cancel: jest.fn(),
};

jest.doMock('@/stores/order.js', () => ({
  useOrderStore: () => fakeStore,
}));

// ---- OrderListItem stub：render 时输出可识别的 data-test + 模拟 view/select/cancel emit
const OrderListItemStub = {
  name: 'OrderListItem',
  props: ['order', 'statusActions'],
  emits: ['view', 'select', 'cancel'],
  template:
    '<div class="order-list-item-stub" :data-order-id="order.id" :data-order-status="order.status" @click="$emit(\'view\', order.id)"><button class="stub-select-btn" @click.stop="$emit(\'select\', order.id)">选陪诊师</button><button class="stub-cancel-btn" @click.stop="$emit(\'cancel\', order.id)">取消</button></div>',
};

jest.doMock(
  '@/components/OrderListItem.vue',
  () => ({ default: OrderListItemStub }),
  { virtual: true },
);

// ---- uView Plus stub
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': {
    props: ['title', 'autoBack'],
    template: '<div class="u-navbar-stub" :data-title="title"></div>',
  },
  'u-tabs': {
    props: ['list', 'current', 'lineColor', 'activeStyle', 'inactiveColor'],
    // emit 'click' 时附带 { index, ...item } 入参
    template:
      '<div class="u-tabs-stub"><button v-for="(item, idx) in list" :key="item.key" class="u-tab-item" :data-tab-index="idx" :data-tab-key="item.key" @click="$emit(\'click\', { index: idx, key: item.key, label: item.label })">{{ item.label }}</button></div>',
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
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('order/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadList.mockReset();
    fakeStore.cancel.mockReset();
    // 默认：loadList 返回两条订单
    fakeStore.loadList.mockResolvedValue([
      {
        id: 7,
        status: 'selectingEscort',
        amount: 38800,
        hospitalName: '北京协和医院',
        appointmentAt: '2026-09-25T09:30:00+08:00',
      },
      {
        id: 8,
        status: 'inService',
        amount: 20000,
        hospitalId: 102,
        appointmentAt: '2026-09-26T10:00:00+08:00',
      },
    ]);
    fakeStore.cancel.mockResolvedValue(undefined);
  });

  it('onLoad+mounted → loadList({}) 默认无 status', async () => {
    const w = await mountPage();
    // 不传 status —— 默认走全部
    await w.vm.onLoad({});
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);
    // 默认入参：空对象（不传 status 过滤）
    expect(fakeStore.loadList).toHaveBeenCalledWith({});
    // orders 已写入
    expect(w.vm.orders).toHaveLength(2);
    expect(w.vm.orders[0].id).toBe(7);
  });

  it('onLoad 传 ?status=selectingEscort → 加载时按 status 过滤', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ status: 'selectingEscort' });
    await w.vm.mounted();
    await flushPromises();

    // loadList 应收到 status 参数
    expect(fakeStore.loadList).toHaveBeenCalledWith({ status: 'selectingEscort' });
    // currentTabIndex 切到 selectingEscort 那一档（index=2）
    expect(w.vm.currentTabIndex).toBe(2);
    expect(w.vm.currentStatus).toBe('selectingEscort');
  });

  it('状态 tab 切换 → loadList 用新 status 重拉', async () => {
    const w = await mountPage();
    await w.vm.onLoad({});
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadList).toHaveBeenCalledTimes(1);

    // 模拟点击 u-tabs 第二档（已支付 / paid，index=1）
    const paidTab = w.findAll('.u-tab-item')[1];
    expect(paidTab.exists()).toBe(true);
    expect(paidTab.attributes('data-tab-key')).toBe('paid');
    await paidTab.trigger('click');
    await flushPromises();

    // 第二次 loadList 携带 status: 'paid'
    expect(fakeStore.loadList).toHaveBeenCalledTimes(2);
    expect(fakeStore.loadList).toHaveBeenNthCalledWith(2, { status: 'paid' });
    expect(w.vm.currentStatus).toBe('paid');
    expect(w.vm.currentTabIndex).toBe(1);
  });

  it('OrderListItem 渲染：orders.length>0 → N 张卡片 stub', async () => {
    const w = await mountPage();
    await w.vm.onLoad({});
    await w.vm.mounted();
    await flushPromises();

    const cards = w.findAll('.order-list-item-stub');
    expect(cards).toHaveLength(2);
    expect(cards[0].attributes('data-order-id')).toBe('7');
    expect(cards[1].attributes('data-order-id')).toBe('8');
    // 空状态 / skeleton 不应同时存在
    expect(w.find('.u-empty-stub').exists()).toBe(false);
    expect(w.find('.u-skeleton-stub').exists()).toBe(false);
  });

  it('点击 OrderListItem → emit view → uni.navigateTo detail', async () => {
    const w = await mountPage();
    await w.vm.onLoad({});
    await w.vm.mounted();
    await flushPromises();

    // 点击第一张卡片 → OrderListItem-stub 在根元素 @click="$emit('view', order.id)"
    await w.findAll('.order-list-item-stub')[0].trigger('click');
    expect(mockUni.navigateTo).toHaveBeenCalledTimes(1);
    expect(mockUni.navigateTo.mock.calls[0][0].url).toMatch(
      /\/pages\/order\/detail\?orderId=7/,
    );
  });

  it('空状态：loadList 返回 [] → u-empty-stub 可见 + 不渲染卡片', async () => {
    // 覆盖默认 mock：本订单为空
    fakeStore.loadList.mockResolvedValueOnce([]);

    const w = await mountPage();
    await w.vm.onLoad({});
    await w.vm.mounted();
    await flushPromises();

    // 空状态 u-empty-stub 出现，text=「暂无订单」
    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无订单');
    // 没有卡片
    expect(w.findAll('.order-list-item-stub')).toHaveLength(0);
  });
});