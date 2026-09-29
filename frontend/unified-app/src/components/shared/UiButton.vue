<script setup lang="ts">
/**
 * UiButton 通用按钮组件（v2 unified-app · Phase 3.0.6）。
 *
 * Phase 3.0.6 改动：
 *   - 移除 SCSS @import tokens（uni-app sass-loader 集成下解析不可靠）
 *   - 改用 CSS 变量（来自 App.vue 动态注入的 :root 块）
 *   - computed style 仍用 TS 常量（uiColorPrimary 等）驱动
 */
import { computed } from 'vue';
import { uiColorPrimary, uiColorError, uiColorTextDisabled } from '@/styles/tokens';

type ButtonType = 'primary' | 'default' | 'ghost' | 'danger';
type ButtonSize = 'sm' | 'base' | 'lg';

const props = withDefaults(
  defineProps<{
    type?: ButtonType;
    size?: ButtonSize;
    disabled?: boolean;
    loading?: boolean;
    block?: boolean;
    htmlType?: 'button' | 'submit' | 'reset';
  }>(),
  {
    type: 'default',
    size: 'base',
    disabled: false,
    loading: false,
    block: false,
    htmlType: 'button',
  },
);

const emit = defineEmits<{
  (e: 'click', ev: MouseEvent): void;
}>();

const isDisabled = computed(() => props.disabled || props.loading);

const styleBindings = computed(() => {
  const styles: Record<string, string> = {};
  if (props.type === 'primary') {
    styles['--btn-bg'] = uiColorPrimary;
    styles['--btn-color'] = '#fff';
    styles['--btn-border'] = uiColorPrimary;
  } else if (props.type === 'danger') {
    styles['--btn-bg'] = uiColorError;
    styles['--btn-color'] = '#fff';
    styles['--btn-border'] = uiColorError;
  } else if (props.type === 'ghost') {
    styles['--btn-bg'] = 'transparent';
    styles['--btn-color'] = uiColorPrimary;
    styles['--btn-border'] = uiColorPrimary;
  } else {
    styles['--btn-bg'] = '#fff';
    styles['--btn-color'] = 'rgba(0,0,0,0.88)';
    styles['--btn-border'] = '#d9d9d9';
  }
  if (props.size === 'sm') {
    styles['--btn-padding'] = '4px 12';
    styles['--btn-font-size'] = '13px';
  } else if (props.size === 'lg') {
    styles['--btn-padding'] = '10px 20';
    styles['--btn-font-size'] = '16px';
  } else {
    styles['--btn-padding'] = '6px 16';
    styles['--btn-font-size'] = '14px';
  }
  if (isDisabled.value) {
    styles['--btn-color'] = uiColorTextDisabled;
  }
  if (props.block) {
    styles['--btn-width'] = '100%';
  }
  return styles;
});

function onClick(ev: MouseEvent) {
  if (isDisabled.value) return;
  emit('click', ev);
}
</script>

<template>
  <button
    class="ui-btn"
    :class="[`ui-btn--${type}`, `ui-btn--${size}`, { 'ui-btn--disabled': isDisabled, 'ui-btn--loading': loading, 'ui-btn--block': block }]"
    :style="styleBindings"
    :type="htmlType"
    :disabled="isDisabled"
    :data-testid="`ui-btn-${type}`"
    @click="onClick"
  >
    <span v-if="loading" class="ui-btn__spinner" data-testid="ui-btn-spinner" />
    <slot />
  </button>
</template>

<style scoped>
.ui-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--ui-space-xs);
  padding: var(--btn-padding, 6px 16px);
  font-size: var(--btn-font-size, 14px);
  font-weight: var(--ui-font-weight-medium);
  line-height: var(--ui-line-height-tight);
  background: var(--btn-bg, #fff);
  color: var(--btn-color, rgba(0, 0, 0, 0.88));
  border: 1px solid var(--btn-border, #d9d9d9);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
  transition: opacity var(--ui-duration-fast) ease;
  width: var(--btn-width, auto);
}

.ui-btn--disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.ui-btn--block {
  width: 100%;
}

.ui-btn__spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: ui-btn-spin 0.8s linear infinite;
}

@keyframes ui-btn-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>