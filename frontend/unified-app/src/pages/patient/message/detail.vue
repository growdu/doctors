<script setup lang="ts">
/**
 * patient/message/detail.vue — 站内信详情（v2 unified-app）。
 *
 * 入口：message/list 点击「查看」→ 本页（?id=xxx）
 *   → 调 api/message.getMessage(id)
 *   → 渲染完整内容
 */
import { ref, onMounted } from 'vue';
import { getMessage } from '@/api/message';
import type { Message } from '@/api/message';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const messageId = ref<number | null>(null);
const message = ref<Message | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const current = pages[pages.length - 1];
  const opts = current?.options;
  if (opts?.id) messageId.value = Number(opts.id);
}

async function onLoad() {
  parseQuery();
  if (!messageId.value) {
    error.value = '缺少消息 ID';
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

function formatDate(iso: string): string {
  return iso.replace('T', ' ').substring(0, 19);
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-message-detail">
    <view v-if="loading" data-testid="message-detail-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="message-detail-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="message-detail__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else-if="message">
      <UiCard :title="message.title" data-testid="message-detail-header">
        <view class="message-detail__meta">
          <text class="message-detail__time">{{ formatDate(message.created_at) }}</text>
          <text v-if="message.ref_id" class="message-detail__ref">关联资源 #{{ message.ref_id }}</text>
        </view>
        <view class="message-detail__divider" />
        <text class="message-detail__content" data-testid="message-detail-content">{{ message.content }}</text>
      </UiCard>
    </template>
  </view>
</template>

<style scoped>
.message-detail__meta {
  display: flex;
  justify-content: space-between;
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  margin-bottom: var(--ui-space-md);
}

.message-detail__divider {
  height: 1px;
  background: var(--ui-color-divider);
  margin: var(--ui-space-md) 0;
}

.message-detail__content {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  line-height: 1.7;
  white-space: pre-wrap;
}

.message-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>