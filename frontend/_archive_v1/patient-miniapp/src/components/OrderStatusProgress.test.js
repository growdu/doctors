// src/components/OrderStatusProgress.test.js
//
// OrderStatusProgress 单文件组件单测 —— @vue/test-utils mount + 节点状态断言。
//
// 测试点（brief 要求 4 个 it）：
//   1. 6 步骤渲染：默认 steps 下渲染 6 个 .osp-step，数据属性 step-key 与对应状态字符串一致
//   2. 当前节点高亮：currentIndex 节点 data-step-state=current，对应 label 加 .osp-step__label--current
//   3. 步骤完成态：currentIndex 之前的节点 data-step-state=completed（绿色 ✓），之后是 future（灰色）
//   4. 自定义 steps：传入 3 个状态，组件只渲染 3 个节点；currentStep 用 data-step-state 校验
//
// 设计要点：
//   - <u-icon> 是 uView Plus 组件，jsdom 下未注册 → stub 为标记 .u-icon-stub
//   - <view> / <text> 是 uni-app 自定义组件 → stub 为 div / span
//   - 通过 data-step-key / data-step-state / data-step-index 数据属性校验节点状态

import { mount } from '@vue/test-utils';
import OrderStatusProgress from './OrderStatusProgress.vue';

const UNI_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-icon': {
    props: ['name', 'color', 'size'],
    template: '<i class="u-icon-stub" :data-name="name" :data-color="color"></i>',
  },
};

const mountProgress = (props) =>
  mount(OrderStatusProgress, {
    props,
    global: { stubs: UNI_STUBS },
  });

describe('OrderStatusProgress.vue', () => {
  it('6 步骤渲染：默认 steps 下渲染 6 个 .osp-step', () => {
    const w = mountProgress({ currentStatus: 'inService' });

    const steps = w.findAll('.osp-step');
    expect(steps).toHaveLength(6);

    // 每个 step 的 data-step-key 与默认 steps 顺序一致
    const keys = steps.map((s) => s.attributes('data-step-key'));
    expect(keys).toEqual([
      'paid',
      'selectingEscort',
      'escortPendingAcceptance',
      'accepted',
      'inService',
      'completed',
    ]);

    // 中文 label 全部出现
    const html = w.html();
    expect(html).toContain('已支付');
    expect(html).toContain('待选陪诊师');
    expect(html).toContain('待陪诊师确认');
    expect(html).toContain('已接单');
    expect(html).toContain('服务中');
    expect(html).toContain('已完成');
  });

  it('当前节点高亮：data-step-state=current + label 加 --current 类', () => {
    const w = mountProgress({ currentStatus: 'escortPendingAcceptance' });

    const current = w.findAll('.osp-step').find(
      (s) => s.attributes('data-step-state') === 'current',
    );
    expect(current).toBeTruthy();
    expect(current.attributes('data-step-key')).toBe('escortPendingAcceptance');

    // label 上有 .osp-step__label--current（蓝色高亮）
    const label = current.find('.osp-step__label');
    expect(label.exists()).toBe(true);
    expect(label.classes()).toContain('osp-step__label--current');
  });

  it('步骤完成态：currentIndex 之前为 completed，之后为 future', () => {
    // 当前为 accepted（index=3）：paid/selectingEscort/escortPendingAcceptance 已完成，
    // accepted 当前，inService/completed 未来
    const w = mountProgress({ currentStatus: 'accepted' });

    const steps = w.findAll('.osp-step');
    expect(steps[0].attributes('data-step-state')).toBe('completed');
    expect(steps[1].attributes('data-step-state')).toBe('completed');
    expect(steps[2].attributes('data-step-state')).toBe('completed');
    expect(steps[3].attributes('data-step-state')).toBe('current');
    expect(steps[3].attributes('data-step-key')).toBe('accepted');
    expect(steps[4].attributes('data-step-state')).toBe('future');
    expect(steps[5].attributes('data-step-state')).toBe('future');

    // 已完成节点渲染 u-icon-stub（checkmark-circle-fill）
    const completedStep = steps[0];
    const icon = completedStep.find('.u-icon-stub');
    expect(icon.exists()).toBe(true);
    expect(icon.attributes('data-name')).toBe('checkmark-circle-fill');
    // 未来节点不应有 u-icon（应渲染数字序号）
    expect(steps[5].find('.u-icon-stub').exists()).toBe(false);
    expect(steps[5].find('.osp-step__index').exists()).toBe(true);
  });

  it('自定义 steps：传入 3 节点数组，组件严格按传入顺序渲染', () => {
    const customSteps = ['selected', 'confirmed', 'arrived'];
    const w = mountProgress({
      currentStatus: 'confirmed',
      steps: customSteps,
    });

    const steps = w.findAll('.osp-step');
    expect(steps).toHaveLength(3);
    expect(steps[0].attributes('data-step-key')).toBe('selected');
    expect(steps[1].attributes('data-step-key')).toBe('confirmed');
    expect(steps[2].attributes('data-step-key')).toBe('arrived');

    expect(steps[0].attributes('data-step-state')).toBe('completed');
    expect(steps[1].attributes('data-step-state')).toBe('current');
    expect(steps[2].attributes('data-step-state')).toBe('future');
  });
});
