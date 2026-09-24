<!--
  src/components/OrderListItem.vue

  OrderListItem —— 订单列表卡片（uView Plus u-card 风格）。

  用途：
    在「我的订单」页（pages/order/index）循环渲染每条订单。
    单一职责：渲染单个订单卡片 + 状态徽标 + 操作按钮。
    不参与业务状态：列表拉取 / 详情跳转 / 取消调用 API 由父页面承载。

  Props：
    - order         (Object, 必填) 订单对象，形状：
                      { id, status, amount, hospitalId|hospitalName,
                        appointmentAt, selectedEscortId, ... }
    - statusActions (Object, 默认 {}) 状态 → 操作 key 列表：
                      { paid: ['view','select'], selectingEscort: ['view','select'],
                        escortPendingAcceptance: ['view'], accepted: ['view','cancel'],
                        inService: ['view'], completed: ['view'], cancelled: ['view'] }
                      父页面负责传入；本组件按 order.status 取对应数组渲染按钮。

  Emits：
    - view(orderId)   用户点击「详情」按钮 或 卡片本身（不含按钮区域） → 跳详情页
    - select(orderId) 用户点击「选陪诊师」按钮 → 跳 candidates 选陪诊师
    - cancel(orderId) 用户点击「取消」按钮 → 弹 confirm → 调 store.cancel

  行为：
    - 状态徽标按 status 映射友好文案（paid → 已支付 等）
    - 操作按钮按 statusActions[status] 渲染；空数组 → 只展示「详情」兜底
    - 卡片整体可点击（emit view），按钮 click 会 stop propagation 避免双触发

  样式：uView Plus u-card 风格
    - 背景 #fff、圆角 12px、内边距 16px、轻微阴影
    - 头部：订单号 + 状态徽标
    - 中部：医院 + 就诊时间
    - 底部：金额 + 操作按钮组

  测试覆盖：src/components/OrderListItem.test.js（4 个 it）
-->
<template>
  <view class="order-list-item" data-test="order-list-item">
    <!-- 头部：订单号 + 状态 -->
    <view class="order-list-item__header">
      <text class="order-list-item__id">订单号 #{{ order.id }}</text>
      <text
        class="order-list-item__status"
        :class="['order-list-item__status--' + statusBadgeType]"
        data-test="status-badge"
      >{{ statusText }}</text>
    </view>

    <!-- 中部：医院 + 就诊时间（可点击 → view） -->
    <view
      class="order-list-item__body"
      data-test="card-body"
      @click="onCardClick"
    >
      <view class="order-list-item__row">
        <text class="order-list-item__label">医院</text>
        <text
          class="order-list-item__value"
          data-test="hospital"
        >{{ hospitalText }}</text>
      </view>
      <view class="order-list-item__row">
        <text class="order-list-item__label">就诊时间</text>
        <text
          class="order-list-item__value"
          data-test="appointment-at"
        >{{ formattedAppointmentAt }}</text>
      </view>
    </view>

    <!-- 底部：金额 + 操作按钮 -->
    <view class="order-list-item__footer">
      <text
        class="order-list-item__amount"
        data-test="amount"
      >{{ formattedAmount }}</text>
      <view class="order-list-item__actions">
        <u-button
          v-for="action in actions"
          :key="action.key"
          size="mini"
          :type="action.btnType"
          :plain="action.plain"
          :data-test="'action-' + action.key"
          @click.stop="onActionClick(action.key)"
        >{{ action.label }}</u-button>
      </view>
    </view>
  </view>
</template>

<script>
// OrderListItem —— Vue 3 Options API（与既有组件 CountdownBadge / EscortCandidateCard 风格一致）。
//
// 为何选 Options API：
//   - props 简单（2 个），computed 只读，无 setup 拆分必要
//   - 与本项目其它组件保持风格统一
//
// 设计要点：
//   - 状态映射、动作定义均为纯常量 + 内部 computed：组件对外只暴露 props/emits，
//     不暴露「每个状态该显示什么按钮」的硬编码，便于父页面控制权限（取消/选陪诊师）
//   - 卡片点击 → emit view；按钮点击 → emit 对应 action（stop propagation 防双触发）

import { formatMoney, formatDateTime } from '@/utils/format.js';

// 状态码 → 友好文案（与 OrderStatusProgress 一致；cancelled/canceled 同义兼容）
const STATUS_LABEL = {
  paid: '已支付',
  selectingEscort: '待选陪诊师',
  escortPendingAcceptance: '待陪诊师确认',
  accepted: '已接单',
  inService: '服务中',
  completed: '已完成',
  cancelled: '已取消',
  canceled: '已取消',
};

// 状态码 → 徽标颜色类型（primary/success/warning/info/default）
const STATUS_BADGE_TYPE = {
  paid: 'info',
  selectingEscort: 'warning',
  escortPendingAcceptance: 'warning',
  accepted: 'primary',
  inService: 'primary',
  completed: 'success',
  cancelled: 'default',
  canceled: 'default',
};

// 操作 key → 按钮定义（label / btnType / plain）
const ACTION_DEFS = {
  view:   { key: 'view',   label: '详情',     btnType: 'default', plain: true  },
  select: { key: 'select', label: '选陪诊师', btnType: 'primary', plain: false },
  cancel: { key: 'cancel', label: '取消订单', btnType: 'error',   plain: true  },
};

export default {
  name: 'OrderListItem',
  props: {
    order: {
      type: Object,
      required: true,
    },
    statusActions: {
      type: Object,
      default: () => ({}),
    },
  },
  emits: ['view', 'select', 'cancel'],
  computed: {
    statusText() {
      const s = this.order && this.order.status;
      return STATUS_LABEL[s] || s || '-';
    },
    statusBadgeType() {
      const s = this.order && this.order.status;
      return STATUS_BADGE_TYPE[s] || 'default';
    },
    formattedAmount() {
      const amt = this.order && this.order.amount;
      if (typeof amt === 'number') {
        try {
          return formatMoney(amt);
        } catch (_e) {
          return String(amt);
        }
      }
      return '-';
    },
    formattedAppointmentAt() {
      return formatDateTime(this.order && this.order.appointmentAt);
    },
    // 医院展示名：优先 hospitalName → 兜底 hospitalId
    hospitalText() {
      const o = this.order || {};
      return o.hospitalName || o.hospitalId || '-';
    },
    // 当前订单状态对应的操作按钮数组（按 statusActions[status] 渲染）
    actions() {
      const s = this.order && this.order.status;
      const keys = (this.statusActions && this.statusActions[s]) || [];
      return keys
        .map((k) => ACTION_DEFS[k])
        .filter(Boolean);
    },
  },
  methods: {
    /** 卡片中部点击 → emit view（不含按钮区域，按钮 click 已 stop propagation） */
    onCardClick() {
      this.$emit('view', this.order.id);
    },
    /** 操作按钮点击 → 按 key 路由到对应事件 */
    onActionClick(key) {
      if (key === 'view') {
        this.$emit('view', this.order.id);
      } else if (key === 'select') {
        this.$emit('select', this.order.id);
      } else if (key === 'cancel') {
        this.$emit('cancel', this.order.id);
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.order-list-item {
  background: #ffffff;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.order-list-item__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px dashed #f0f0f0;
  margin-bottom: 12px;
}

.order-list-item__id {
  font-size: 13px;
  color: #606266;
  font-variant-numeric: tabular-nums;
}

.order-list-item__status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  border-style: solid;
  border-width: 1px;
  line-height: 1.4;
}
.order-list-item__status--primary { color: #1677ff; border-color: #1677ff; background: #e8f3ff; }
.order-list-item__status--success { color: #52c41a; border-color: #52c41a; background: #f6ffed; }
.order-list-item__status--warning { color: #fa8c16; border-color: #fa8c16; background: #fff7e6; }
.order-list-item__status--info    { color: #909399; border-color: #909399; background: #f4f4f5; }
.order-list-item__status--default { color: #909399; border-color: #dcdfe6; background: #fafafa; }

.order-list-item__body {
  margin-bottom: 12px;
  // 卡片中部可点击区
  cursor: pointer;
}

.order-list-item__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
}

.order-list-item__label {
  font-size: 13px;
  color: #909399;
  flex-shrink: 0;
}

.order-list-item__value {
  font-size: 14px;
  color: #303133;
  max-width: 65%;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.order-list-item__footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.order-list-item__amount {
  font-size: 18px;
  font-weight: 600;
  color: #ff4d4f;
  font-variant-numeric: tabular-nums;
}

.order-list-item__actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
</style>