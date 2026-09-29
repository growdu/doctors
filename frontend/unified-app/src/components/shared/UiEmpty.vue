<script setup lang="ts">
/**
 * UiEmpty 空状态组件（v2 unified-app）。
 *
 * 设计原则：
 *   - 单色 icon（emoji 或字符，避免图片依赖）
 *   - title + description 二级文本
 *   - action slot 提供「重试 / 去看看」等行动按钮
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.2
 */
import { uiColorTextSecondary } from '@/styles/tokens';

withDefaults(
  defineProps<{
    /** 图标：单字符或 emoji（避免图片资源依赖） */
    icon?: string;
    title?: string;
    description?: string;
  }>(),
  {
    icon: '📭',
    title: '暂无数据',
    description: '',
  },
);
</script>

<template>
  <view class="ui-empty" data-testid="ui-empty">
    <view class="ui-empty__icon" data-testid="ui-empty-icon">{{ icon }}</view>
    <view v-if="title" class="ui-empty__title" data-testid="ui-empty-title">{{ title }}</view>
    <view v-if="description" class="ui-empty__description" data-testid="ui-empty-description">
      {{ description }}
    </view>
    <view v-if="$slots.action" class="ui-empty__action">
      <slot name="action" />
    </view>
  </view>
</template>

<style lang="css" scoped>

.ui-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--ui-space-xxl) var(--ui-space-base);
  color: var(--ui-color-text-disabled);
}

.ui-empty__icon {
  font-size: 48px;
  margin-bottom: var(--ui-space-md);
  opacity: 0.6;
}

.ui-empty__title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  margin-bottom: var(--ui-space-xs);
}

.ui-empty__description {
  font-size: var(--ui-font-sm);
  color: v-bind('uiColorTextSecondary');
  margin-bottom: var(--ui-space-md);
  text-align: center;
}

.ui-empty__action {
  margin-top: var(--ui-space-sm);
}
</style>
