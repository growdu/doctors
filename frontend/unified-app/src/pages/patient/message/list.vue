<script setup lang="ts">
/**
 * patient/message/list.vue — 站内信列表（v2 unified-app · patient 域）。
 *
 * 入口：patient 域「消息中心」
 *   → onMounted 调 api/message.listMessages({ read: undefined })
 *   → 渲染消息卡：类型 + 标题 + 时间 + 未读红点
 *
 * 设计要点：
 *   - 顶部统计：未读数 + 全部消息数
 *   - 点击消息跳详情（标记已读由后端处理）
 */
import { ref, onMounted } from 'vue';
import { listMessages } from '@/api/message';
import type { Message, MessageType } from '@/api/message';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const messages = ref<Message[]>([]);
const unread = ref(0);
const loading = ref(false);
const error = ref<string | null>(null);

const TYPE_LABEL: Record<MessageType, string> = {
  system: '系统',
  order: '订单',
  payment: '支付',
  sos: 'SOS',
  review: '评价',
};

const TYPE_ICON: Record<MessageType, string> = {
  system: '🔔',
  order: '📋',
  payment: '💰',
  sos: '🚨',
  review: '⭐',
};

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listMessages();
    messages.value = r.items;
    unread.value = r.unread;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const isSameDay = d.toDateString() === now.toDateString();
  if (isSameDay) {
    return iso.split('T')[1]?.substring(0, 5) || iso;
  }
  return iso.split('T')[0]?.substring(5) || iso;
}

function onDetail(m: Message) {
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: `/pages/patient/message/detail?id=${m.id}` });
  }
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-message-list">
    <view class="message-list__summary" data-testid="message-summary">
      未读 <text class="message-list__unread">{{ unread }}</text> / 共 {{ messages.length }} 条
    </view>

    <view v-if="loading && messages.length === 0" data-testid="message-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="message-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="message-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="messages.length === 0" data-testid="message-empty">
      <UiEmpty icon="📭" title="暂无消息" />
    </view>

    <view v-else data-testid="message-list">
      <view
        v-for="m in messages"
        :key="m.id"
        class="message-list__item"
        :class="{ 'message-list__item--unread': !m.read }"
        :data-testid="`message-item-${m.id}`"
        @click="onDetail(m)"
      >
        <view class="message-list__item-icon">{{ TYPE_ICON[m.type] }}</view>
        <view class="message-list__item-body">
          <view class="message-list__item-row">
            <text class="message-list__item-type">[{{ TYPE_LABEL[m.type] }}]</text>
            <text class="message-list__item-title">{{ m.title }}</text>
            <view v-if="!m.read" class="message-list__item-dot" data-testid="message-unread-dot" />
          </view>
          <text class="message-list__item-time">{{ formatDate(m.created_at) }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.message-list__summary {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  padding: var(--ui-space-sm) 0;
}

.message-list__unread {
  color: var(--ui-color-error);
  font-weight: var(--ui-font-weight-semibold);
  margin: 0 4px;
}

.message-list__item {
  display: flex;
  align-items: center;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  margin-bottom: var(--ui-space-sm);
  box-shadow: var(--ui-shadow-sm);
  cursor: pointer;
  transition: background var(--ui-duration-fast);
}

.message-list__item:active {
  background: var(--ui-color-bg-hover);
}

.message-list__item--unread {
  background: rgba(22, 119, 255, 0.04);
  border-left: 3px solid var(--ui-color-primary);
}

.message-list__item-icon {
  font-size: 28px;
  margin-right: var(--ui-space-md);
}

.message-list__item-body {
  flex: 1;
  min-width: 0;
}

.message-list__item-row {
  display: flex;
  align-items: center;
  gap: var(--ui-space-xs);
  margin-bottom: 4px;
}

.message-list__item-type {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  flex-shrink: 0;
}

.message-list__item-title {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.message-list__item-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ui-color-error);
  flex-shrink: 0;
}

.message-list__item-time {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
}

.message-list__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>