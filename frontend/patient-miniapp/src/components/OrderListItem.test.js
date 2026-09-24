// src/components/OrderListItem.test.js
//
// OrderListItem 单文件组件单测 —— @vue/test-utils mount + 触发 click + 检查 emitted。
//
// 测试点（brief 要求 4 个 it —— 渲染 / view 事件 / select 事件 / cancel 事件）：
//   1. 渲染：订单号 + 状态徽标 + 医院 + 时间 + 金额 + 操作按钮全部出现
//   2. view 事件：点击卡片中部 → emit('view', orderId)
//   3. select 事件：点击「选陪诊师」按钮 → emit('select', orderId)
//   4. cancel 事件：点击「取消订单」按钮 → emit('cancel', orderId)
//
// 测试策略：
//   - 沿用 EscortCandidateCard.test.js / CountdownBadge.test.js 约定：
//     <view>/<text> 是 uni-app 自定义组件 → stub 为 div/span（jsdom 下未注册）
//   - <u-button> 是 uView Plus 组件 → stub 为最小按钮，透传 click + props
//   - 当前 jest.config.js 未挂 vue-jest，本测试文件作为单测约定；
//     待 devDependencies 增补 vue-jest + @vue/test-utils 后即可跑（任务外步骤）。
//
// 备注：
//   - statusActions 控制哪些按钮渲染；本组件不硬编码状态→按钮的映射，由父页面传入。
//   - 状态徽标用 status map（paid/selectingEscort/.../cancelled/canceled），通过
//     data-test="status-badge" 读 textContent 断言。

import { mount, flushPromises } from '@vue/test-utils';
import OrderListItem from './OrderListItem.vue';

const UNI_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
  'u-button': {
    props: ['type', 'size', 'plain', 'loading', 'disabled'],
    template:
      '<button class="u-button-stub" :data-type="type" :data-size="size" :data-plain="!!plain" @click="$emit(\'click\')"><slot /></button>',
  },
};

const mountItem = (props) =>
  mount(OrderListItem, {
    props,
    global: { stubs: UNI_STUBS },
  });

// 基础订单：selectingEscort 状态，操作 = ['view','select']
const baseOrder = () => ({
  id: 7,
  status: 'selectingEscort',
  amount: 38800,
  hospitalId: 101,
  hospitalName: '北京协和医院',
  appointmentAt: '2026-09-25T09:30:00+08:00',
  selectedEscortId: null,
});

// paid → ['view','select']
// selectingEscort → ['view','select']
// escortPendingAcceptance → ['view']
// accepted → ['view','cancel']
// inService → ['view']
// completed → ['view']
// cancelled → ['view']
const STATUS_ACTIONS_ALL = {
  paid: ['view', 'select'],
  selectingEscort: ['view', 'select'],
  escortPendingAcceptance: ['view'],
  accepted: ['view', 'cancel'],
  inService: ['view'],
  completed: ['view'],
  cancelled: ['view'],
};

describe('OrderListItem.vue', () => {
  it('渲染：订单号 + 状态徽标 + 医院 + 时间 + 金额 + 操作按钮', () => {
    const order = baseOrder();
    const w = mountItem({ order, statusActions: STATUS_ACTIONS_ALL });

    // 订单号 #id
    expect(w.text()).toContain('订单号 #7');
    // 状态徽标：selectingEscort → '待选陪诊师'
    const badge = w.find('[data-test="status-badge"]');
    expect(badge.exists()).toBe(true);
    expect(badge.text()).toBe('待选陪诊师');
    // 医院（hospitalName 优先）
    expect(w.text()).toContain('北京协和医院');
    // 金额：38800 → ¥38800.00（formatMoney）
    expect(w.text()).toContain('¥38800.00');
    // 就诊时间：formatDateTime 2026-09-25 09:30
    expect(w.text()).toContain('2026-09-25 09:30');
    // 操作按钮：selectingEscort → ['view','select'] 两枚按钮
    const btns = w.findAll('.u-button-stub');
    expect(btns).toHaveLength(2);
    expect(btns[0].text()).toBe('详情');
    expect(btns[1].text()).toBe('选陪诊师');
  });

  it('点击卡片中部 → emit view(orderId)', async () => {
    const order = baseOrder();
    const w = mountItem({ order, statusActions: STATUS_ACTIONS_ALL });

    // 卡片中部 data-test="card-body"
    const body = w.find('[data-test="card-body"]');
    expect(body.exists()).toBe(true);
    await body.trigger('click');

    const events = w.emitted('view');
    expect(events).toBeTruthy();
    expect(events).toHaveLength(1);
    expect(events[0]).toEqual([7]);
  });

  it('点击「选陪诊师」按钮 → emit select(orderId) + 不 emit view', async () => {
    const order = baseOrder();
    const w = mountItem({ order, statusActions: STATUS_ACTIONS_ALL });

    // 通过 data-test="action-select" 找到 select 按钮
    const selectBtn = w.find('[data-test="action-select"]');
    expect(selectBtn.exists()).toBe(true);
    expect(selectBtn.text()).toBe('选陪诊师');

    await selectBtn.trigger('click');
    await flushPromises();

    // select 事件：emit('select', order.id)
    const selectEvents = w.emitted('select');
    expect(selectEvents).toBeTruthy();
    expect(selectEvents).toHaveLength(1);
    expect(selectEvents[0]).toEqual([7]);
    // 按钮 stop propagation：不应触发 view
    expect(w.emitted('view')).toBeFalsy();
  });

  it('点击「取消订单」按钮 → emit cancel(orderId) + 不 emit view', async () => {
    // 切到 accepted 状态：操作 = ['view','cancel']
    const order = { ...baseOrder(), status: 'accepted' };
    const w = mountItem({ order, statusActions: STATUS_ACTIONS_ALL });

    // 渲染两枚按钮：详情 + 取消订单
    const btns = w.findAll('.u-button-stub');
    expect(btns).toHaveLength(2);
    expect(btns[0].text()).toBe('详情');
    expect(btns[1].text()).toBe('取消订单');

    // 通过 data-test="action-cancel" 找到 cancel 按钮
    const cancelBtn = w.find('[data-test="action-cancel"]');
    expect(cancelBtn.exists()).toBe(true);

    await cancelBtn.trigger('click');
    await flushPromises();

    // cancel 事件：emit('cancel', order.id)
    const cancelEvents = w.emitted('cancel');
    expect(cancelEvents).toBeTruthy();
    expect(cancelEvents).toHaveLength(1);
    expect(cancelEvents[0]).toEqual([7]);
    // 按钮 stop propagation：不应触发 view
    expect(w.emitted('view')).toBeFalsy();
  });
});