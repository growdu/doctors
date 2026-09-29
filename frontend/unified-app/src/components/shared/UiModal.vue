<script setup lang="ts">
/**
 * UiModal 模态弹层组件（v2 unified-app）。
 *
 * 设计原则：
 *   - v-model:visible 双向绑定（visible / update:visible）
 *   - 标题（title prop / slot）
 *   - 内容（content prop / 默认插槽）
 *   - 操作按钮：confirm / cancel 文本 + 事件
 *   - mask 点击可关闭（closeOnMask，默认 true）
 *   - 默认 confirm 点击后自动关闭
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.2
 */
const props = withDefaults(
  defineProps<{
    visible?: boolean;
    title?: string;
    content?: string;
    cancelText?: string;
    confirmText?: string;
    closeOnMask?: boolean;
    /** confirm 按钮类型：primary / danger */
    confirmType?: 'primary' | 'danger';
  }>(),
  {
    visible: false,
    title: '',
    content: '',
    cancelText: '取消',
    confirmText: '确认',
    closeOnMask: true,
    confirmType: 'primary',
  },
);

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'confirm'): void;
  (e: 'cancel'): void;
  (e: 'close'): void;
}>();

function onMaskClick() {
  if (!props.closeOnMask) return;
  emit('update:visible', false);
  emit('close');
}

function onConfirm() {
  emit('confirm');
  emit('update:visible', false);
}

function onCancel() {
  emit('cancel');
  emit('update:visible', false);
}
</script>

<template>
  <view v-if="visible" class="ui-modal" data-testid="ui-modal">
    <view class="ui-modal__mask" data-testid="ui-modal-mask" @click="onMaskClick" />
    <view class="ui-modal__panel" data-testid="ui-modal-panel" @click.stop>
      <view v-if="title || $slots.title" class="ui-modal__title" data-testid="ui-modal-title">
        <slot name="title">{{ title }}</slot>
      </view>
      <view class="ui-modal__content" data-testid="ui-modal-content">
        <slot>{{ content }}</slot>
      </view>
      <view class="ui-modal__actions" data-testid="ui-modal-actions">
        <button
          class="ui-modal__btn ui-modal__btn--cancel"
          type="button"
          data-testid="ui-modal-cancel"
          @click="onCancel"
        >
          {{ cancelText }}
        </button>
        <button
          class="ui-modal__btn"
          :class="`ui-modal__btn--${confirmType}`"
          type="button"
          data-testid="ui-modal-confirm"
          @click="onConfirm"
        >
          {{ confirmText }}
        </button>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
@import '@/styles/tokens.scss';

.ui-modal {
  position: fixed;
  inset: 0;
  z-index: $ui-z-modal;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ui-modal__mask {
  position: absolute;
  inset: 0;
  background-color: $ui-color-bg-mask;
}

.ui-modal__panel {
  position: relative;
  width: 80%;
  max-width: 360px;
  background-color: $ui-color-bg-card;
  border-radius: $ui-radius-lg;
  overflow: hidden;
  box-shadow: $ui-shadow-lg;
}

.ui-modal__title {
  padding: $ui-space-base;
  font-size: $ui-font-lg;
  font-weight: $ui-font-weight-semibold;
  text-align: center;
  border-bottom: 1px solid $ui-color-divider;
}

.ui-modal__content {
  padding: $ui-space-base;
  font-size: $ui-font-base;
  color: $ui-color-text-secondary;
  line-height: $ui-line-height-base;
  text-align: center;
  min-height: 60px;
}

.ui-modal__actions {
  display: flex;
  border-top: 1px solid $ui-color-divider;
}

.ui-modal__btn {
  flex: 1;
  padding: $ui-space-md;
  font-size: $ui-font-base;
  background-color: transparent;
  border: none;
  border-right: 1px solid $ui-color-divider;
  cursor: pointer;
}

.ui-modal__btn:last-child {
  border-right: none;
}

.ui-modal__btn--cancel {
  color: $ui-color-text-secondary;
}

.ui-modal__btn--primary {
  color: $ui-color-primary;
  font-weight: $ui-font-weight-medium;
}

.ui-modal__btn--danger {
  color: $ui-color-error;
  font-weight: $ui-font-weight-medium;
}
</style>