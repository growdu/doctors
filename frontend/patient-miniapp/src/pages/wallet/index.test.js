// src/pages/wallet/index.test.js
//
// 钱包首页单测 —— @vue/test-utils mount + jest.doMock 注入 fake wallet store。
//
// 测试点（5 个 it）：
//   1. mounted 调 store.loadWallet() + loadTransactions()
//   2. 余额 + 冻结金额渲染（蓝色大字）
//   3. 流水列表渲染：每条 tx-amount-{id} 与 color class（positive / negative）
//   4. 空流水：u-empty 显示 + 流水 list 不渲染
//   5. loadWallet 抛错 → error-state + 重试按钮（点击重试后 loadWallet 再调一次）

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

// ---- fake wallet store（reactive 包裹 + 计数）
let _loadWalletCalls = 0;
let _loadTxCalls = 0;

const fakeWalletStore = reactive({
  wallet: null,
  transactions: [],
  pagination: { total: 0, page: 1, page_size: 20 },
  loading: false,
  loadingTx: false,
  error: null,
  loadWallet: jest.fn(async () => {
    _loadWalletCalls += 1;
    fakeWalletStore.wallet = {
      user_id: 1,
      balance: 1288.5,
      frozen: 50.0,
      currency: 'CNY',
      updated_at: '2026-09-24T15:00:00Z',
    };
    return fakeWalletStore.wallet;
  }),
  loadTransactions: jest.fn(async (query) => {
    _loadTxCalls += 1;
    fakeWalletStore.transactions = [
      { id: 1, type: 'recharge', amount: 500, balance_after: 1388, description: '微信充值', created_at: '2026-09-20T10:00:00Z' },
      { id: 2, type: 'payment', amount: -388, balance_after: 1000, description: '订单 #7 支付', order_id: 7, created_at: '2026-09-22T12:30:00Z' },
      { id: 3, type: 'refund', amount: 88, balance_after: 1088, description: '订单 #5 退款', order_id: 5, created_at: '2026-09-23T09:15:00Z' },
    ];
    fakeWalletStore.pagination = { total: 3, page: (query && query.page) || 1, page_size: (query && query.page_size) || 20 };
    return { items: fakeWalletStore.transactions, ...fakeWalletStore.pagination };
  }),
  clearAll: jest.fn(() => {
    fakeWalletStore.wallet = null;
    fakeWalletStore.transactions = [];
  }),
});

jest.doMock('@/stores/wallet.js', () => ({
  useWalletStore: () => fakeWalletStore,
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
    template:
      '<button class="u-button-stub" :data-type="type" :disabled="!!disabled" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('wallet/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fakeWalletStore.wallet = null;
    fakeWalletStore.transactions = [];
    fakeWalletStore.pagination = { total: 0, page: 1, page_size: 20 };
    fakeWalletStore.loadWallet.mockClear();
    fakeWalletStore.loadTransactions.mockClear();
    fakeWalletStore.clearAll.mockClear();
    _loadWalletCalls = 0;
    _loadTxCalls = 0;
  });

  it('mounted 调 store.loadWallet() + loadTransactions()', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(fakeWalletStore.loadWallet).toHaveBeenCalledTimes(1);
    expect(fakeWalletStore.loadTransactions).toHaveBeenCalledTimes(1);
    expect(fakeWalletStore.loadTransactions).toHaveBeenCalledWith({ page: 1, page_size: 20 });
  });

  it('余额 + 冻结金额渲染（蓝色大字）', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="balance-card"]').exists()).toBe(true);
    expect(w.find('[data-test="balance-value"]').text()).toBe('¥1288.50');
    expect(w.find('[data-test="balance-frozen"]').text()).toContain('¥50.00');
  });

  it('流水列表渲染：每条 tx-amount-{id} 与 color class（positive / negative）', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.findAll('[data-test^="tx-"]').length).toBeGreaterThanOrEqual(3);

    const pos = w.find('[data-test="tx-amount-1"]');
    expect(pos.text()).toBe('+500.00');
    expect(pos.classes()).toContain('page-wallet__tx-amount--positive');

    const neg = w.find('[data-test="tx-amount-2"]');
    expect(neg.text()).toBe('-388.00');
    expect(neg.classes()).toContain('page-wallet__tx-amount--negative');

    // 时间格式化
    const desc = w.find('[data-test="tx-1"]');
    expect(desc.text()).toContain('微信充值');
    expect(desc.text()).toContain('2026-09-20');
  });

  it('空流水：u-empty 显示 + 流水 list 不渲染', async () => {
    // 覆盖默认：loadTransactions 返回空
    fakeWalletStore.loadTransactions.mockImplementationOnce(async () => {
      fakeWalletStore.transactions = [];
      fakeWalletStore.pagination = { total: 0, page: 1, page_size: 20 };
      return { items: [], total: 0, page: 1, page_size: 20 };
    });

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="tx-empty"]').exists()).toBe(true);
    expect(w.find('[data-test="tx-list"]').exists()).toBe(false);
    // 但余额卡仍渲染（loadWallet 仍 OK）
    expect(w.find('[data-test="balance-card"]').exists()).toBe(true);
  });

  it('loadWallet 抛错 → error-state + 重试按钮（点击重试后 loadWallet 再调一次）', async () => {
    fakeWalletStore.loadWallet.mockRejectedValueOnce(new Error('network down'));

    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();

    expect(w.find('[data-test="balance-error"]').exists()).toBe(true);
    expect(w.find('[data-test="retry-btn"]').exists()).toBe(true);

    // 重试按钮 → 第二次 loadWallet
    await w.find('[data-test="retry-btn"]').trigger('click');
    await flushPromises();
    expect(fakeWalletStore.loadWallet).toHaveBeenCalledTimes(2);
  });

  it('unmounted 清空 wallet + transactions', async () => {
    const w = await mountPage();
    await w.vm.mounted();
    await flushPromises();
    await w.vm.unmounted();
    expect(fakeWalletStore.clearAll).toHaveBeenCalledTimes(1);
  });
});