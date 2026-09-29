// src/pages/support/index/index.test.js
//
// 客服 / 帮助中心单测 —— @vue/test-utils mount + mock uni。
//
// 测试点（brief 要求 4-5 个 it）：
//   1. mounted → 渲染电话卡 + FAQ（默认收起）+ 留言表单
//   2. FAQ 点击 toggle → openKeys 增删 + answer 显隐
//   3. 「一键拨打」点击 → toast
//   4. 留言长度 10-500 校验 → canSubmitMessage
//   5. 留言提交 → toast「留言成功」+ 清空 textarea

import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { jest } from '@jest/globals';

const mockUni = {
  showToast: jest.fn(),
  showModal: jest.fn(),
  navigateTo: jest.fn(),
  redirectTo: jest.fn(),
  reLaunch: jest.fn(),
  makePhoneCall: jest.fn(),
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

describe('support/index/index.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('mounted → 渲染电话卡 + FAQ（默认全收起） + 留言表单', async () => {
    const w = await mountPage();
    await flushPromises();

    expect(w.find('[data-test="support-page"]').exists()).toBe(true);
    expect(w.find('[data-test="phone-number"]').text()).toBe('400-100-1234');
    expect(w.find('[data-test="call-btn"]').exists()).toBe(true);

    // FAQ 5 项 + 默认全收起
    expect(w.find('[data-test="faq-how_to_book"]').exists()).toBe(true);
    expect(w.find('[data-test="faq-cancel_order"]').exists()).toBe(true);
    expect(w.findAll('[data-test^="answer-"]')).toHaveLength(0);

    // 留言表单
    expect(w.find('[data-test="message-input"]').exists()).toBe(true);
    expect(w.find('[data-test="submit-msg-btn"]').exists()).toBe(true);
  });

  it('FAQ 点击 toggle → openKeys 增删 + answer 显隐', async () => {
    const w = await mountPage();
    await flushPromises();

    // 初始不开
    expect(w.find('[data-test="faq-how_to_book"]').attributes('data-open')).toBe('false');
    expect(w.find('[data-test="answer-how_to_book"]').exists()).toBe(false);

    // 点击 question 区域
    await w.find('[data-test="toggle-how_to_book"]').trigger('click');
    expect(w.vm.openKeys).toContain('how_to_book');
    expect(w.find('[data-test="faq-how_to_book"]').attributes('data-open')).toBe('true');
    expect(w.find('[data-test="answer-how_to_book"]').exists()).toBe(true);

    // 多开：再点另一个 FAQ
    await w.find('[data-test="toggle-cancel_order"]').trigger('click');
    expect(w.vm.openKeys).toHaveLength(2);

    // 再点第一个 → 收起
    await w.find('[data-test="toggle-how_to_book"]').trigger('click');
    expect(w.vm.openKeys).not.toContain('how_to_book');
    expect(w.vm.openKeys).toHaveLength(1);
  });

  it('「一键拨打」点击 → toast（v1 mock）', async () => {
    const w = await mountPage();
    await flushPromises();

    await w.find('[data-test="call-btn"]').trigger('click');
    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: expect.stringContaining('400-100-1234') }),
    );
  });

  it('留言长度 10-500 校验：<10 → disabled；>=10 → enabled', async () => {
    const w = await mountPage();
    await flushPromises();

    // 默认空 → 不满足
    expect(w.vm.canSubmitMessage).toBe(false);
    const btn = w.find('[data-test="submit-msg-btn"]');
    expect(btn.attributes('disabled')).toBeDefined();

    // 8 字 → 仍 disabled
    w.vm.message = '12345678';
    await w.vm.$nextTick();
    expect(w.vm.canSubmitMessage).toBe(false);

    // 10 字 → enabled
    w.vm.message = '1234567890';
    await w.vm.$nextTick();
    expect(w.vm.canSubmitMessage).toBe(true);

    // 501 字 → disabled
    w.vm.message = 'a'.repeat(501);
    await w.vm.$nextTick();
    expect(w.vm.canSubmitMessage).toBe(false);
  });

  it('留言提交 → toast「留言成功」+ 清空 textarea', async () => {
    const w = await mountPage();
    await flushPromises();

    w.vm.message = '我的订单没法取消';
    await w.vm.$nextTick();

    await w.find('[data-test="submit-msg-btn"]').trigger('click');
    await flushPromises();

    expect(mockUni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: expect.stringContaining('留言成功') }),
    );
    expect(w.vm.message).toBe('');
  });
});
</content>
</invoke>