<!--
  src/components/OrderStatusProgress.vue

  OrderStatusProgress —— 订单进度条（6 节点横向 stepper）。
  单一职责：渲染订单状态机的当前进度，纯展示组件，不参与业务状态。
  为何选自定义 stepper 而非 uView Plus u-steps：
    需要逐节点控制「绿色 ✓ / 蓝色高亮 / 灰色未来」三种态的样式与图标，
    u-steps 的内置模式无法精细化每节点的 icon / color 切换，故采用
    uView Plus 视觉风格的自定义 stepper，更便于 jest 单测用 stub 验证。
  Props：
    - currentStatus (String) 当前订单状态字符串
    - steps         (Array<String>, 默认 6 节点) 进度条节点顺序
  状态判定：
    - 已完成：index < currentIndex
    - 当前  ：index === currentIndex
    - 未来  ：index > currentIndex（或 currentStatus 不在 steps 内）
  测试覆盖：src/components/OrderStatusProgress.test.js（4 个 it）
-->
<template>
  <view class="order-status-progress">
    <view
      v-for="(step, idx) in steps"
      :key="step"
      class="osp-step"
      :class="['osp-step--' + stateAt(idx)]"
      :data-step-key="step"
      :data-step-state="stateAt(idx)"
      :data-step-index="idx"
    >
      <view class="osp-step__head">
        <view class="osp-step__icon">
          <u-icon
            v-if="stateAt(idx) === 'completed'"
            name="checkmark-circle-fill"
            color="#52c41a"
            size="22"
          />
          <text
            v-else
            class="osp-step__index"
            :class="['osp-step__index--' + stateAt(idx)]"
          >{{ idx + 1 }}</text>
        </view>
        <view
          v-if="idx < steps.length - 1"
          class="osp-step__line"
          :class="['osp-line--' + lineStateAt(idx)]"
        />
      </view>
      <text
        class="osp-step__label"
        :class="['osp-step__label--' + stateAt(idx)]"
      >{{ labelOf(step) }}</text>
    </view>
  </view>
</template>

<script>
// OrderStatusProgress —— Vue 3 Options API（与既有组件 CountdownBadge / EscortCandidateCard 风格一致）。

const STATUS_LABEL_ZH = {
  paid: '已支付',
  selectingEscort: '待选陪诊师',
  escortPendingAcceptance: '待陪诊师确认',
  accepted: '已接单',
  inService: '服务中',
  completed: '已完成',
};

const DEFAULT_STEPS = [
  'paid',
  'selectingEscort',
  'escortPendingAcceptance',
  'accepted',
  'inService',
  'completed',
];

export default {
  name: 'OrderStatusProgress',
  props: {
    currentStatus: {
      type: String,
      required: true,
    },
    steps: {
      type: Array,
      default: () => DEFAULT_STEPS,
    },
  },
  computed: {
    currentIndex() {
      const idx = this.steps.indexOf(this.currentStatus);
      // 未找到 → 所有节点按「未来」处理（避免在异常状态下误标已完成）
      return idx < 0 ? -1 : idx;
    },
  },
  methods: {
    labelOf(step) {
      return STATUS_LABEL_ZH[step] || step;
    },
    stateAt(idx) {
      if (this.currentIndex < 0) return 'future';
      if (idx < this.currentIndex) return 'completed';
      if (idx === this.currentIndex) return 'current';
      return 'future';
    },
    // 连线颜色：起点节点已完成 → 连线已完成；否则为 future
    lineStateAt(idx) {
      return this.stateAt(idx) === 'completed' ? 'completed' : 'future';
    },
  },
};
</script>

<style lang="scss" scoped>
.order-status-progress {
  display: flex;
  justify-content: space-between;
  padding: 16px 8px;
  background: #ffffff;
  border-radius: 12px;
}

.osp-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.osp-step__head {
  position: relative;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.osp-step__icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  z-index: 1;
}

.osp-step__index {
  width: 24px;
  height: 24px;
  line-height: 24px;
  text-align: center;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid #dcdfe6;
  color: #909399;
  background: #f5f7fa;
}
.osp-step__index--current {
  border-color: #1677ff;
  color: #fff;
  background: #1677ff;
}
.osp-step__index--future {
  border-color: #dcdfe6;
  color: #909399;
  background: #f5f7fa;
}

.osp-step__line {
  position: absolute;
  top: 13px;
  left: 50%;
  right: -50%;
  height: 2px;
  background: #dcdfe6;
}
.osp-line--completed {
  background: #52c41a;
}

.osp-step__label {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
  text-align: center;
  line-height: 1.3;
}
.osp-step__label--completed {
  color: #52c41a;
}
.osp-step__label--current {
  color: #1677ff;
  font-weight: 600;
}
</style>
