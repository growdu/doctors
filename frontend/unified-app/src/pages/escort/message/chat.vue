<script setup lang="ts">
/**
 * escort/message/chat.vue — 陪诊师消息详情（v2 unified-app · escort 域）。
 *
 * 入口：escort/message/list 点击消息卡
 *   → getMessage(id) 拉消息详情
 *   → 显示标题 / 内容 / 类型 / 时间 / 关联资源
 *   → 当前消息若是通知类（如订单状态变更），关联资源链接跳转订单详情
 *
 * 设计要点：
 *   - v2 message-service 暂未暴露 1:1 chat 端点，本页仅展示「通知/详情」
 *   - 通用页：与 patient/message/detail 渲染一致，仅入口不同
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { getMessage } from '@/api/message';
import type { Message, MessageType } from '@/api/message';

const TYPE_LABEL: Record<MessageType, string> = {
  system: '系统通知',
  order: '订单通知',
  payment: '支付通知',
  sos: 'SOS 通知',
  review: '评价通知',
};

const message = ref<Message | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);
const messageId = ref<number | null>(null);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.id) messageId.value = Number(opts.id);
}

async function onLoad() {
  parseQuery();
  if (!messageId.value) {
    error.value = '未指定消息 ID';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    message.value = await getMessage(messageId.value);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function onRelatedClick() {
  if (!message.value?.ref_id || !message.value.ref_type) return;
  if (message.value.ref_type === 'order') {
    if (typeof uni !== 'undefined') {
      uni.navigateTo({ url: `/pages/escort/order-detail?id=${message.value.ref_id}` });
    }
  } else {
    if (typeof uni !== 'undefined') {
      uni.showToast({ title: '相关资源跳转暂未实现', icon: 'none' });
    }
  }
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="escort-message-chat-page">
    <view v-if="loading" data-testid="escort-message-chat-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error || !message" data-testid="escort-message-chat-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error ?? '消息不存在'">
        <template #action>
          <text class="escort-message-chat__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <UiCard data-testid="escort-message-chat-header">
        <view class="escort-message-chat__row">
          <text class="escort-message-chat__type">{{ TYPE_LABEL[message.type] }}</text>
          <text class="escort-message-chat__time">{{ formatDate(message.created_at) }}</text>
        </view>
        <text class="escort-message-chat__title" data-testid="escort-message-chat-title">{{ message.title }}</text>
        <text class="escort-message-chat__content" data-testid="escort-message-chat-content">{{ message.content }}</text>
        <view v-if="message.ref_id && message.ref_type" class="escort-message-chat__ref-row">
          <text class="escort-message-chat__ref-label">相关：</text>
          <text
            class="escort-message-chat__ref-link"
            :data-testid="`escort-message-chat-ref-${message.ref_type}`"
            @click="onRelatedClick"
          >
            {{ message.ref_type }} #{{ message.ref_id }}
          </text>
        </view>
      </UiCard>

      <view class="escort-message-chat__note" data-testid="escort-message-chat-note">
        <text class="escort-message-chat__note-text">
          💡 v2 message-service 暂未暴露 1:1 对话端点；本详情页仅展示通知类消息。
        </text>
      </view>
    </template>
  </view>
</template>

<style scoped>
.escort-message-chat__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.escort-message-chat__type {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.escort-message-chat__time {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.escort-message-chat__title {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-message-chat__content {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: var(--ui-space-sm);
  white-space: pre-wrap;
}

.escort-message-chat__ref-row {
  margin-top: var(--ui-space-md);
  padding-top: var(--ui-space-sm);
  border-top: 1px solid var(--ui-color-border);
  display: flex;
  align-items: center;
}

.escort-message-chat__ref-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.escort-message-chat__ref-link {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  text-decoration: underline;
  cursor: pointer;
}

.escort-message-chat__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-info);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin: var(--ui-space-md) 0;
}

.escort-message-chat__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.escort-message-chat__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>