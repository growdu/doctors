// src/pages/order/candidates/index.test.js
//
// candidates 页单测 —— @vue/test-utils mount + 模拟 uni-app 生命周期 + mock store。
//
// 测试点（5 个 it —— brief 要求覆盖 onLoad/列表渲染/selectEscortBy/空状态/错误 toast）：
//   1. onLoad → 调 useOrderStore().loadCandidates(orderId)
//   2. 候选列表渲染：candidates.length > 0 → N 张 EscortCandidateCard
//   3. 点击 EscortCandidateCard → confirm → selectEscortBy(orderId, escortId) → redirectTo 详情页
//   4. 空状态：candidates=[] → u-empty-stub 可见 + EscortCandidateCard 不渲染
//   5. 错误 toast：loadCandidates 抛错 → uni.showToast「加载失败，请重试」被调用
//
// 测试策略：
//   - jest.doMock('@/stores/order.js')：注入 fake store（jest 的 moduleNameMapper 会把
//     '@/stores/order.js' 映射到 src/stores/order.js，与生产代码 import 路径一致）
//   - jest.doMock('@/components/EscortCandidateCard.vue', () => Stub)：用最小 stub
//     替换子组件，避免 @dcloudio/uni-app / uView Plus 等子依赖在 jsdom 下报错
//   - 在 beforeEach 中挂载全局 `uni`（uni.showToast/showModal/redirectTo 的 mock）
//
// 备注：
//   - 当前 jest.config.js 未挂 vue-jest，本测试文件仅作为单测约定；待 devDependencies
//     增补 vue-jest + @vue/test-utils 后即可跑（任务外步骤）。
//   - onLoad / onMounted 是 uni-app 扩展生命周期：在测试里通过 wrapper.vm.onLoad(query)
//     手动触发（uni-app 编译器会把它们当作 method 挂到组件实例）。
//   - confirm dialog 用 stub.showModal 直接返回 { confirm: true }，跳过手点确认。

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { jest } from '@jest/globals';

// ---- 全局 uni mock：showToast / showModal / redirectTo / showLoading
const mockUni = {
  showToast: jest.fn(),
  showModal: jest.fn(),
  redirectTo: jest.fn(),
  navigateTo: jest.fn(),
  reLaunch: jest.fn(),
  showLoading: jest.fn(),
  hideLoading: jest.fn(),
};

beforeAll(() => {
  global.uni = mockUni;
});

afterEach(() => {
  // 清 mock 但不删 global.uni（每个 case 重新构造）
  jest.clearAllMocks();
});

// ---- 子组件 stub：EscortCandidateCard 用最小化模板替换（避免拉 uView Plus）
const EscortCandidateCardStub = {
  name: 'EscortCandidateCard',
  props: ['candidate', 'loading', 'disabled'],
  emits: ['select'],
  template:
    '<button class="escort-candidate-card-stub" :data-escort-id="candidate && candidate.escortId" :data-disabled="!!disabled" :data-loading="!!loading" @click="$emit(\'select\', candidate && candidate.escortId)">{{ candidate && candidate.nickname }}</button>',
};

// ---- uView Plus 组件 stub（页面模板用到 u-navbar / u-empty / u-skeleton / u-button）
const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"><slot /></div>' },
  'u-empty': {
    props: ['text', 'mode'],
    template: '<div class="u-empty-stub" :data-text="text"></div>',
  },
  'u-skeleton': {
    props: ['rows', 'title', 'avatar', 'avatarShape', 'avatarSize'],
    template: '<div class="u-skeleton-stub"></div>',
  },
  'u-button': {
    props: ['type', 'size', 'loading', 'disabled', 'plain'],
    template:
      '<button class="u-button-stub" :disabled="!!disabled" :data-loading="!!loading" @click="$emit(\'click\')"><slot /></button>',
  },
};

// ---- fake store：每个 case 在 beforeEach 重置 mock 返回值
const fakeStore = {
  candidates: [],
  loadCandidates: jest.fn(),
  selectEscortBy: jest.fn(),
};

jest.doMock('@/stores/order.js', () => ({
  useOrderStore: () => fakeStore,
}));

// ---- 子组件 mock（用 jest.doMock 让 page 解析 EscortCandidateCard 时拿到 stub）
jest.doMock(
  '@/components/EscortCandidateCard.vue',
  () => ({ default: EscortCandidateCardStub }),
  { virtual: true },
);

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('candidates/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeStore.loadCandidates.mockReset();
    fakeStore.selectEscortBy.mockReset();
    // 默认：loadCandidates 返回两条候选
    fakeStore.loadCandidates.mockResolvedValue({
      items: [
        { escortId: 11, nickname: '张三', rating: 4.9, completedOrders: 220, distanceKm: 1.2, tags: ['耐心'] },
        { escortId: 12, nickname: '李四', rating: 4.7, completedOrders: 90, distanceKm: 3.4, tags: [] },
      ],
      generated_at: '2026-09-24T15:30:00+08:00',
    });
    fakeStore.selectEscortBy.mockResolvedValue({
      orderId: 7,
      status: 'escortPendingAcceptance',
      escortPendingExpireAt: '2026-09-24T15:30:30+08:00',
    });
    // 默认 showModal 视为用户点确认
    mockUni.showModal.mockImplementation((opts) => {
      if (opts && typeof opts.success === 'function') opts.success({ confirm: true });
    });
  });

  it('onLoad(orderId) → mounted 触发 loadCandidates(orderId)', async () => {
    const w = await mountPage();
    // 模拟 uni-app 导航：本页被以 ?orderId=7 打开
    await w.vm.onLoad({ orderId: 7 });
    // mounted 在 mount 时已触发（此时 orderId=null，未拉取）；这里再显式调一次
    // （与生产一致：onLoad 先到 → orderId 写入 → mounted 再读）
    await w.vm.mounted();
    await flushPromises();

    expect(fakeStore.loadCandidates).toHaveBeenCalledWith(7);
    // 候选已写入组件 data
    expect(w.vm.candidates).toHaveLength(2);
    expect(w.vm.candidates[0].escortId).toBe(11);
  });

  it('候选列表渲染：candidates.length>0 时渲染 N 张 EscortCandidateCard', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    const cards = w.findAll('.escort-candidate-card-stub');
    expect(cards).toHaveLength(2);
    expect(cards[0].attributes('data-escort-id')).toBe('11');
    expect(cards[1].attributes('data-escort-id')).toBe('12');
    // 空状态 / skeleton 不应同时存在
    expect(w.find('.u-empty-stub').exists()).toBe(false);
    expect(w.find('.u-skeleton-stub').exists()).toBe(false);
  });

  it('点击 EscortCandidateCard → confirm → selectEscortBy → 跳详情页', async () => {
    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    // 点击第 1 张卡片
    await w.findAll('.escort-candidate-card-stub')[0].trigger('click');
    // confirm dialog → confirm=true → doSelect 触发
    expect(mockUni.showModal).toHaveBeenCalledTimes(1);
    await flushPromises();

    expect(fakeStore.selectEscortBy).toHaveBeenCalledTimes(1);
    expect(fakeStore.selectEscortBy).toHaveBeenCalledWith(7, 11);
    // 跳详情页
    expect(mockUni.redirectTo).toHaveBeenCalledTimes(1);
    expect(mockUni.redirectTo.mock.calls[0][0].url).toMatch(/\/pages\/order\/detail\?orderId=7/);
  });

  it('空状态：candidates=[] 时显示 u-empty-stub + 不渲染卡片', async () => {
    // 覆盖默认 mock：本订单无候选
    fakeStore.loadCandidates.mockResolvedValueOnce({ items: [], generated_at: '' });

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 99 });
    await w.vm.mounted();
    await flushPromises();

    // 空状态 u-empty-stub 出现，text=「暂无候选陪诊师」
    const empty = w.find('.u-empty-stub');
    expect(empty.exists()).toBe(true);
    expect(empty.attributes('data-text')).toBe('暂无候选陪诊师');
    // 没有任何卡片
    expect(w.findAll('.escort-candidate-card-stub')).toHaveLength(0);
  });

  it('错误 toast：loadCandidates 抛错 → uni.showToast「加载失败，请重试」', async () => {
    // 本次 loadCandidates 模拟网络/服务端错误
    fakeStore.loadCandidates.mockRejectedValueOnce(new Error('network down'));

    const w = await mountPage();
    await w.vm.onLoad({ orderId: 7 });
    await w.vm.mounted();
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalled();
    const toastArg = mockUni.showToast.mock.calls.find(
      (c) => c[0] && c[0].title === '加载失败，请重试',
    );
    expect(toastArg).toBeTruthy();
    expect(toastArg[0].icon).toBe('none');
  });
});