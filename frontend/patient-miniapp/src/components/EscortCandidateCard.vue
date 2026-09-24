<!--
  src/components/EscortCandidateCard.vue

  EscortCandidateCard —— 候选陪诊师卡片（uView Plus u-card 风格）

  用途：
    在「选择陪诊师」页（pages/order/candidates）循环渲染每位候选陪诊师。
    单一职责：渲染单个候选卡片 + 「选择TA」点击事件。
    不参与业务状态：候选列表拉取 / 选定调 API 由父页面承载。

  Props：
    - candidate (Object, 必填) 形状：{ escortId, nickname, rating, completedOrders, distanceKm, tags[] }
    - loading   (Boolean, 默认 false) 仅当前卡片「选择TA」触发 selectEscortBy 进行中 → true
    - disabled  (Boolean, 默认 false) 父级正在执行其他候选的选择（如防重复点击） → true

  Emits：
    - select(escortId)  用户点击「选择TA」按钮：父级弹 confirm → 调 selectEscortBy

  行为：
    - loading=true → 按钮 disabled + uView Plus loading 图标
    - disabled=true → 卡片加 .is-disabled 类（半透明 + 禁用交互）；点按钮也不 emit
    - tags 为空时不渲染 u-tag 区

  样式：uView Plus u-card 风格
    - 背景 #fff、圆角 12px、内边距 16px、轻微阴影
    - 头部：昵称 + u-rate 评分 + 已完成订单数（右侧次级文案）
    - 中部：u-tag 标签列表（plain、mini）
    - 底部：距离（次级文案） + u-button「选择TA」（primary、small）

  测试覆盖：src/components/EscortCandidateCard.test.js（4 个 it）
-->
<template>
  <view class="escort-candidate-card" :class="{ 'is-disabled': disabled }">
    <!-- 头部：昵称 + 评分 + 已完成订单数 -->
    <view class="escort-candidate-card__header">
      <view class="escort-candidate-card__name-block">
        <text class="escort-candidate-card__nickname">{{ candidate.nickname }}</text>
        <u-rate
          :model-value="Number(candidate.rating) || 0"
          readonly
          size="14"
          active-color="#ff9900"
        />
      </view>
      <text class="escort-candidate-card__completed">已完成 {{ completedOrdersText }} 单</text>
    </view>

    <!-- 中部：标签列表 -->
    <view
      v-if="hasTags"
      class="escort-candidate-card__tags"
      data-test="tags"
    >
      <u-tag
        v-for="tag in candidate.tags"
        :key="tag"
        :text="tag"
        type="primary"
        plain
        size="mini"
      />
    </view>

    <!-- 底部：距离 + 选择按钮 -->
    <view class="escort-candidate-card__footer">
      <text class="escort-candidate-card__distance">{{ distanceText }}</text>
      <u-button
        type="primary"
        size="small"
        :loading="loading"
        :disabled="disabled || loading"
        data-test="select-btn"
        @click="onSelectClick"
      >选择TA</u-button>
    </view>
  </view>
</template>

<script>
// EscortCandidateCard —— 单文件组件（Vue 3 Options API，与 CountdownBadge 风格一致）。
//
// 为何选 Options API：
//   - 与本项目其它组件（CountdownBadge.vue / App.vue）保持风格统一
//   - 简单展示组件，computed / methods 不需要 setup 拆分的复杂度
//   - emit 与 props 的 .sync 写法在模板里更直观

export default {
  name: 'EscortCandidateCard',
  props: {
    candidate: {
      type: Object,
      required: true,
    },
    loading: {
      type: Boolean,
      default: false,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['select'],
  computed: {
    hasTags() {
      return Array.isArray(this.candidate.tags) && this.candidate.tags.length > 0;
    },
    completedOrdersText() {
      const n = this.candidate.completedOrders;
      if (n === undefined || n === null) return 0;
      return n;
    },
    distanceText() {
      const km = this.candidate.distanceKm;
      if (km === undefined || km === null) return '距离未知';
      return `距离 ${km} km`;
    },
  },
  methods: {
    onSelectClick() {
      // disabled 时拦截（父级语义：禁止操作）
      if (this.disabled || this.loading) return;
      this.$emit('select', this.candidate.escortId);
    },
  },
};
</script>

<style lang="scss" scoped>
.escort-candidate-card {
  background: #ffffff;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  transition: opacity 0.2s ease;

  &.is-disabled {
    opacity: 0.5;
    pointer-events: none;
  }
}

.escort-candidate-card__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.escort-candidate-card__name-block {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.escort-candidate-card__nickname {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.escort-candidate-card__completed {
  font-size: 12px;
  color: #909399;
  flex-shrink: 0;
  margin-left: 12px;
}

.escort-candidate-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.escort-candidate-card__footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.escort-candidate-card__distance {
  font-size: 13px;
  color: #606266;
}
</style>