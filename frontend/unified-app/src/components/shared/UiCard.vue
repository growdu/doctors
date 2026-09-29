<script setup lang="ts">
/**
 * UiCard 容器组件（v2 unified-app）。
 *
 * 设计原则：
 *   - 统一卡片样式（背景 / 圆角 / 阴影 / 内边距）
 *   - 可选 title（卡头）：左侧文字 + 右侧 extra slot
 *   - 默认插槽（卡体）：业务内容
 *   - 可选 footer slot（卡底）：操作按钮等
 *   - shadow 等级：sm / md / lg
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.2
 */
withDefaults(
  defineProps<{
    title?: string;
    /** 阴影等级：sm / md / lg */
    shadow?: 'sm' | 'md' | 'lg' | 'none';
    /** 是否去掉内边距（用于嵌入图片 / 列表） */
    noPadding?: boolean;
  }>(),
  {
    title: '',
    shadow: 'sm',
    noPadding: false,
  },
);
</script>

<template>
  <view
    class="ui-card"
    :class="[`ui-card--shadow-${shadow}`, { 'ui-card--no-padding': noPadding }]"
    data-testid="ui-card"
  >
    <view v-if="title || $slots.extra" class="ui-card__header" data-testid="ui-card-header">
      <view v-if="title" class="ui-card__title" data-testid="ui-card-title">{{ title }}</view>
      <view v-if="$slots.extra" class="ui-card__extra" data-testid="ui-card-extra">
        <slot name="extra" />
      </view>
    </view>
    <view class="ui-card__body" data-testid="ui-card-body">
      <slot />
    </view>
    <view v-if="$slots.footer" class="ui-card__footer" data-testid="ui-card-footer">
      <slot name="footer" />
    </view>
  </view>
</template>

<style lang="css" scoped>

.ui-card {
  background-color: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  overflow: hidden;
  margin-bottom: var(--ui-space-md);
}

.ui-card--shadow-sm {
  box-shadow: var(--ui-shadow-sm);
}

.ui-card--shadow-md {
  box-shadow: var(--ui-shadow-md);
}

.ui-card--shadow-lg {
  box-shadow: var(--ui-shadow-lg);
}

.ui-card--shadow-none {
  box-shadow: none;
}

.ui-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--ui-space-md) var(--ui-space-base);
  border-bottom: 1px solid var(--ui-color-divider);
}

.ui-card__title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
}

.ui-card__extra {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
}

.ui-card__body {
  padding: var(--ui-space-base);
}

.ui-card--no-padding .ui-card__body {
  padding: 0;
}

.ui-card__footer {
  padding: var(--ui-space-md) var(--ui-space-base);
  border-top: 1px solid var(--ui-color-divider);
}
</style>
