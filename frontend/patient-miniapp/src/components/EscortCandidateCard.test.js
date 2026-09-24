// src/components/EscortCandidateCard.test.js
//
// EscortCandidateCard 单文件组件单测 —— @vue/test-utils mount + 触发 click + 检查 emitted。
//
// 测试点（4 个 it —— brief 要求覆盖 渲染 / select 事件 / loading / disabled）：
//   1. 渲染：昵称 / 完成订单数 / 距离 / tags 全部出现在卡片内
//   2. select 事件：点击「选择TA」按钮触发 emit('select', candidate.escortId)
//   3. loading=true：按钮 disabled + uView Plus loading prop 透传
//   4. disabled=true：卡片半透明 + 按钮 disabled + 点击不 emit
//
// 依赖：
//   - @vue/test-utils v2（提供 mount / flushPromises）
//   - vue-jest（让 .vue 单文件组件可被 require）—— 当前 jest.config.js 未挂 vue-jest，
//     待 devDependencies 增补 @vue/test-utils + vue-jest 后即可跑（任务外步骤）。
//
// stub 说明：
//   - <view> / <text> 是 uni-app 自定义组件，jsdom 下未注册 → 替换为 div / span
//   - <u-rate> / <u-tag> / <u-button> 是 uView Plus 组件 → 用最小 stub 替换，
//     仅暴露测试关心的 props（modelValue / loading / disabled 等）

import { mount, flushPromises } from '@vue/test-utils';
import EscortCandidateCard from './EscortCandidateCard.vue';

const UNI_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-rate': {
    props: ['modelValue', 'readonly', 'size', 'activeColor'],
    template: '<span class="u-rate-stub" :data-value="modelValue"></span>',
  },
  'u-tag': {
    props: ['text', 'type', 'plain', 'size'],
    template: '<span class="u-tag-stub">{{ text }}</span>',
  },
  'u-button': {
    props: ['type', 'size', 'loading', 'disabled', 'plain'],
    template:
      '<button class="u-button-stub" :disabled="!!disabled" :data-loading="!!loading"><slot /></button>',
  },
};

const mountCard = (props) =>
  mount(EscortCandidateCard, {
    props,
    global: { stubs: UNI_STUBS },
  });

const baseCandidate = () => ({
  escortId: 11,
  nickname: '张三',
  rating: 4.9,
  completedOrders: 220,
  distanceKm: 1.2,
  tags: ['耐心', '三甲熟悉'],
});

describe('EscortCandidateCard.vue', () => {
  it('渲染：昵称 + 完成订单数 + 距离 + 标签 + 选择按钮', () => {
    const w = mountCard({ candidate: baseCandidate() });

    // 昵称 / 评分 / 完成订单数 / 距离
    expect(w.text()).toContain('张三');
    expect(w.text()).toContain('已完成 220 单');
    expect(w.text()).toContain('距离 1.2 km');
    expect(w.text()).toContain('选择TA');

    // 标签
    expect(w.text()).toContain('耐心');
    expect(w.text()).toContain('三甲熟悉');
    // u-tag stub 应被渲染 2 次
    expect(w.findAll('.u-tag-stub')).toHaveLength(2);

    // 评分（u-rate stub data-value 与 candidate.rating 对齐）
    const rate = w.find('.u-rate-stub');
    expect(rate.attributes('data-value')).toBe('4.9');
  });

  it('点击「选择TA」按钮 → emit select(escortId)', async () => {
    const candidate = baseCandidate();
    const w = mountCard({ candidate });

    const btn = w.find('.u-button-stub');
    expect(btn.exists()).toBe(true);
    await btn.trigger('click');

    const events = w.emitted('select');
    expect(events).toBeTruthy();
    expect(events).toHaveLength(1);
    // 事件 payload：emit('select', candidate.escortId)
    expect(events[0]).toEqual([11]);
  });

  it('loading=true → 按钮 disabled + loading prop 透传', () => {
    const w = mountCard({ candidate: baseCandidate(), loading: true });

    const btn = w.find('.u-button-stub');
    expect(btn.attributes('disabled')).toBe('true');
    expect(btn.attributes('data-loading')).toBe('true');
  });

  it('disabled=true → 卡片半透明 + 按钮 disabled + 点击不 emit', async () => {
    const w = mountCard({ candidate: baseCandidate(), disabled: true });

    // 卡片加 .is-disabled 类（半透明视觉）
    const root = w.find('.escort-candidate-card');
    expect(root.exists()).toBe(true);
    expect(root.classes()).toContain('is-disabled');

    // 按钮 disabled（loading=false 时仍 disabled）
    const btn = w.find('.u-button-stub');
    expect(btn.attributes('disabled')).toBe('true');

    // 即使点击也不应 emit（onSelectClick 内部拦截）
    await btn.trigger('click');
    await flushPromises();
    expect(w.emitted('select')).toBeFalsy();
  });
});