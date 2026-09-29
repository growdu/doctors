<script setup lang="ts">
/**
 * UiLoading 加载状态组件（v2 unified-app）。
 *
 * 设计原则：
 *   - 纯 CSS spinner（避免 gif/webp 资源）
 *   - 3 种 size：sm (16px) / base (24px) / lg (32px)
 *   - 可选 text 提示
 *   - fullscreen 模式覆盖全屏（用于页面级 loading）
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.2
 */
import { uiColorPrimary } from '@/styles/tokens';

withDefaults(
  defineProps<{
    size?: 'sm' | 'base' | 'lg';
    color?: string;
    text?: string;
    fullscreen?: boolean;
  }>(),
  {
    size: 'base',
    color: uiColorPrimary,
    text: '',
    fullscreen: false,
  },
);
</script>

<template>
  <view
    class="ui-loading"
    :class="{ 'ui-loading--fullscreen': fullscreen }"
    :data-size="size"
    data-testid="ui-loading"
  >
    <view
      class="ui-loading__spinner"
      :class="`ui-loading__spinner--${size}`"
      :style="{ borderTopColor: color }"
      data-testid="ui-loading-spinner"
    />
    <view v-if="text" class="ui-loading__text" data-testid="ui-loading-text">{{ text }}</view>
  </view>
</template>

<style lang="scss" scoped>
@import '@/styles/tokens.scss';

.ui-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $ui-space-sm;
  padding: $ui-space-md;
}

.ui-loading--fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: $ui-z-modal;
  background-color: rgba(255, 255, 255, 0.85);
}

.ui-loading__spinner {
  display: inline-block;
  border: 2px solid #e8e8e8;
  border-top-color: $ui-color-primary;
  border-radius: 50%;
  animation: ui-loading-spin 0.8s linear infinite;
}

.ui-loading__spinner--sm {
  width: 16px;
  height: 16px;
}

.ui-loading__spinner--base {
  width: 24px;
  height: 24px;
  border-width: 3px;
}

.ui-loading__spinner--lg {
  width: 32px;
  height: 32px;
  border-width: 4px;
}

.ui-loading__text {
  font-size: $ui-font-sm;
  color: $ui-color-text-secondary;
}

@keyframes ui-loading-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>