<script setup lang="ts">
/**
 * UiInput 表单输入组件（v2 unified-app）。
 *
 * 设计原则：
 *   - v-model 双向绑定（modelValue / update:modelValue）
 *   - 5 种 type：text / tel / number / password / textarea
 *   - 可选 label slot（左侧标签，垂直对齐）
 *   - clearable 模式（输入有内容时右侧显示清除按钮）
 *   - error 状态（红色边框 + 错误信息文本）
 *   - disabled / readonly 状态
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.2
 */
import { computed } from 'vue';

type InputType = 'text' | 'tel' | 'number' | 'password' | 'textarea';

const props = withDefaults(
  defineProps<{
    modelValue?: string | number;
    type?: InputType;
    placeholder?: string;
    label?: string;
    maxlength?: number;
    disabled?: boolean;
    readonly?: boolean;
    clearable?: boolean;
    error?: string;
  }>(),
  {
    modelValue: '',
    type: 'text',
    placeholder: '',
    label: '',
    maxlength: 140,
    disabled: false,
    readonly: false,
    clearable: false,
    error: '',
  },
);

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void;
  (e: 'blur', ev: FocusEvent): void;
  (e: 'focus', ev: FocusEvent): void;
}>();

const stringValue = computed(() => String(props.modelValue ?? ''));
const isMultiline = computed(() => props.type === 'textarea');
const showClear = computed(() => props.clearable && stringValue.value.length > 0 && !props.disabled && !props.readonly);

function onInput(ev: Event) {
  const target = ev.target as HTMLInputElement | HTMLTextAreaElement;
  emit('update:modelValue', target.value);
}

function onClear() {
  emit('update:modelValue', '');
}

function onBlur(ev: FocusEvent) {
  emit('blur', ev);
}

function onFocus(ev: FocusEvent) {
  emit('focus', ev);
}
</script>

<template>
  <view class="ui-input" :class="{ 'ui-input--error': !!error, 'ui-input--disabled': disabled, 'ui-input--multiline': isMultiline }" data-testid="ui-input">
    <view v-if="label || $slots.label" class="ui-input__label" data-testid="ui-input-label">
      <slot name="label">{{ label }}</slot>
    </view>
    <view class="ui-input__field" data-testid="ui-input-field">
      <textarea
        v-if="isMultiline"
        class="ui-input__textarea"
        :value="stringValue"
        :placeholder="placeholder"
        :maxlength="maxlength"
        :disabled="disabled"
        :readonly="readonly"
        data-testid="ui-input-textarea"
        @input="onInput"
        @blur="onBlur"
        @focus="onFocus"
      />
      <input
        v-else
        class="ui-input__inner"
        :type="type"
        :value="stringValue"
        :placeholder="placeholder"
        :maxlength="maxlength"
        :disabled="disabled"
        :readonly="readonly"
        data-testid="ui-input-inner"
        @input="onInput"
        @blur="onBlur"
        @focus="onFocus"
      />
      <button
        v-if="showClear"
        type="button"
        class="ui-input__clear"
        data-testid="ui-input-clear"
        @click="onClear"
      >
        ✕
      </button>
    </view>
    <view v-if="error" class="ui-input__error" data-testid="ui-input-error">
      {{ error }}
    </view>
  </view>
</template>

<style lang="scss" scoped>
@import '@/styles/tokens.scss';

.ui-input {
  display: flex;
  flex-direction: column;
  margin-bottom: $ui-space-md;
}

.ui-input__label {
  font-size: $ui-font-sm;
  color: $ui-color-text-secondary;
  margin-bottom: $ui-space-xs;
}

.ui-input__field {
  position: relative;
  display: flex;
  align-items: center;
}

.ui-input__inner,
.ui-input__textarea {
  flex: 1;
  width: 100%;
  padding: $ui-space-sm $ui-space-md;
  font-size: $ui-font-base;
  color: $ui-color-text-primary;
  background-color: $ui-color-bg-card;
  border: 1px solid $ui-color-border;
  border-radius: $ui-radius-sm;
  outline: none;
  transition: border-color $ui-duration-fast ease;
  box-sizing: border-box;
}

.ui-input__textarea {
  resize: vertical;
  min-height: 80px;
  font-family: inherit;
}

.ui-input__inner:focus,
.ui-input__textarea:focus {
  border-color: $ui-color-primary;
}

.ui-input__inner::placeholder,
.ui-input__textarea::placeholder {
  color: $ui-color-text-disabled;
}

.ui-input--disabled .ui-input__inner,
.ui-input--disabled .ui-input__textarea {
  background-color: $ui-color-bg-hover;
  color: $ui-color-text-disabled;
  cursor: not-allowed;
}

.ui-input--error .ui-input__inner,
.ui-input--error .ui-input__textarea {
  border-color: $ui-color-error;
}

.ui-input__clear {
  position: absolute;
  right: $ui-space-sm;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  padding: 0;
  font-size: 10px;
  color: $ui-color-text-disabled;
  background-color: #d9d9d9;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.ui-input__clear:hover {
  background-color: #bfbfbf;
}

.ui-input__error {
  margin-top: $ui-space-xs;
  font-size: $ui-font-xs;
  color: $ui-color-error;
}
</style>