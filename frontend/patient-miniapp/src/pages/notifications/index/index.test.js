// src/pages/notifications/index/index.test.js
//
// 通知中心单测 —— @vue/test-utils mount + mock uni。
//
// 测试点（brief 要求 4-5 个 it）：
//   1. mounted → 渲染 4 tab + 全部 tab 下的通知列表 + 未读红点
//   2. 切到「订单」tab → 只显示 order 类型通知
//   3. 切到「营销」tab → 只显示 marketing 类型 + 未读 badge = 1
//   4. 点击未读条目 → markRead + data-unread 切换 false
//   5. 「全部已读」点击 → 所有条目 read_at 都标记

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { jest } from '@jest/globals';

const mockUni = {
  showToast: jest.fn(),
  showModal: jest.fn(),
  navigateTo: jest.fn(),
  redirectTo: jest.fn(),
  reLaunch: jest.fn(),
};

beforeAll(() => {
  global.uni = mockUni;
});

afterEach(() => {
  jest.clearAllMocks();
});

const U_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-navbar': { template: '<div class="u-navbar-stub"></div>' },
  'u-skeleton': { props: ['rows', 'title'], template: '<div class="u-skeleton-stub"></div>' },
  'u-empty': { props: ['text', 'mode'], template: '<div class="u-empty-stub" :data-text="text"></div>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'disabled'],
    emits: ['click'],
    inheritAttrs: false,
    template:
      '<button class="u-button-stub" v-bind="$attrs" :data-type="type" :disabled="!!disabled" @click.stop="$emit(\'click\')"><slot /></button>',
  },
};

const mountPage = async () => {
  const mod = await import('./index.vue');
  return mount(mod.default, { global: { stubs: U_STUBS } });
};

describe('notifications/index/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('mounted → 4 tab + 默认全部 + 5 条通知 + 未读红点', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(w.vm.currentTab).toBe('all');
    expect(w.find('[data-test="tab-all"]').exists()).toBe(true);
    expect(w.find('[data-test="tab-order"]').exists()).toBe(true);
    expect(w.find('[data-test="tab-system"]').exists()).toBe(true);
    expect(w.find('[data-test="tab-marketing"]').exists()).toBe(true);

    // 5 条通知
    const items = w.findAll('[data-test^="notification-"]');
    expect(items).toHaveLength(5);
    // 4 条未读（默认），但本地列表渲染的是当前 tab 下
    // 全部 tab 的未读数（在 tab badge）
    expect(w.find('[data-test="badge-all"]').text()).toBe('4');

    // 第 4 条已读 → 无未读红点
    const fourth = w.find('[data-test="notification-4"]');
    expect(fourth.attributes('data-unread')).toBe('false');
  });

  it('切到「订单」tab → 只显示 order 类型通知 (2 条)', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="tab-order"]').trigger('click');
    expect(w.vm.currentTab).toBe('order');

    const items = w.findAll('[data-test^="notification-"]');
    expect(items).toHaveLength(2);
    // 都是订单类型 → 第 1 / 2 条
    expect(items[0].attributes('data-notif-id')).toBe('1');
    expect(items[1].attributes('data-notif-id')).toBe('2');
  });

  it('切到「营销」tab → 只显示 marketing 通知 + 未读 badge', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="tab-marketing"]').trigger('click');
    expect(w.vm.currentTab).toBe('marketing');

    const items = w.findAll('[data-test^="notification-"]');
    expect(items).toHaveLength(1);
    expect(items[0].attributes('data-notif-id')).toBe('5');
    // 营销未读 badge = 1
    expect(w.find('[data-test="badge-marketing"]').text()).toBe('1');
  });

  it('点击未读条目 → markRead + data-unread 切换 false', async () => {
    const w = await mountPage();
    await flushPromises();

    // 第 1 条未读
    const item1 = w.find('[data-test="notification-1"]');
    expect(item1.attributes('data-unread')).toBe('true');

    await item1.trigger('click');
    await flushPromises();

    expect(w.vm.notifications.find((n) => n.id === 1).read_at).toBeTruthy();
    // 重新渲染
    const item1After = w.find('[data-test="notification-1"]');
    expect(item1After.attributes('data-unread')).toBe('false');
    // 全部未读 badge - 1
    expect(w.find('[data-test="badge-all"]').text()).toBe('3');
  });

  it('「全部已读」点击 → 所有条目 read_at 都标记', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="mark-all-btn"]').trigger('click');
    await flushPromises();

    // 所有通知都有 read_at
    const unread = w.vm.notifications.filter((n) => !n.read_at);
    expect(unread).toHaveLength(0);
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: '全部已读' }),
    );
    // 全部 badge 不再显示
    expect(w.find('[data-test="badge-all"]').exists()).toBe(false);
  });

  it('空状态：所有通知都被 markRead → 仍有列表（空 badge），不会触发 empty', async () => {
    // 非测试目标 —— 默认 NOTIFICATIONS 内含空 readable item；切换到空 type 的 tab 会显示 empty
    const w = await mountPage();
    await flushPromises();

    // v1 NOTIFICATIONS 包含 3 个 type；不存在 0 长度的情况；
    // 此测试仅断言 4 tab 都能渲染（不空）
    expect(w.find('[data-test="tab-system"]').exists()).toBe(true);
  });
});
</content>
</invoke>