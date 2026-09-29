<script setup lang="ts">
/**
 * escort/message/list.vue — 陪诊师消息列表（v2 unified-app · escort 域）。
 *
 * 入口：escort home「消息」入口
 *   → listMessages() 拉所有消息
 *   → 「未读」tab + 「全部」tab
 *   → 点击消息跳详情（patient 域消息详情共用）
 *
 * 设计要点：
 *   - 复用 message-service listMessages（@/api/message，patient 共用端点）
 *   - 未读 tab 显示 unread_count（listMessages 返回 { items, total, unread }）
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { listMessages } from '@/api/message';
import type { Message, MessageType } from '@/api/message';

const TYPE_LABEL: Record<MessageType, string> = {
  system: '系统',
  order: '订单',
  payment: '支付',
  sos: 'SOS',
  review: '评价',
};

const items = ref<Message[]>([]);
const unreadCount = ref(0);
const loading = ref(false);
const error = ref<string | null>(null);
const activeTab = ref<'all' | 'unread'>('all');

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const query = activeTab.value === 'unread' ? { read: false } : {};
    const r = await listMessages(query);
    items.value = r.items;
    unreadCount.value = r.unread;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onSwitchTab(t: 'all' | 'unread') {
  activeTab.value = t;
  await onLoad();
}

function onDetail(m: Message) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/escort/message/chat?id=${m.id}` });
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="escort-message-list-page">
    <view class="escort-message-list__tabs" data-testid="escort-message-list-tabs">
      <view
        class="escort-message-list__tab"
        :class="{ 'escort-message-list__tab--active': activeTab === 'all' }"
        data-testid="escort-message-list-tab-all"
        @click="onSwitchTab('all')"
      >
        全部
      </view>
      <view
        class="escort-message-list__tab"
        :class="{ 'escort-message-list__tab--active': activeTab === 'unread' }"
        data-testid="escort-message-list-tab-unread"
        @click="onSwitchTab('unread')"
      >
        未读 ({{ unreadCount }})
      </view>
    </view>

    <view v-if="loading" data-testid="escort-message-list-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="escort-message-list-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="escort-message-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="escort-message-list-empty">
      <UiEmpty icon="📬" title="暂无消息" />
    </view>

    <view v-else data-testid="escort-message-list">
      <UiCard
        v-for="m in items"
        :key="m.id"
        :data-testid="`escort-message-list-card-${m.id}`"
        @click="onDetail(m)"
      >
        <view class="escort-message-list__row">
          <text class="escort-message-list__type" :class="{ 'escort-message-list__type--unread': !m.read }">
            {{ TYPE_LABEL[m.type] }}
          </text>
          <text v-if="!m.read" class="escort-message-list__dot" :data-testid="`escort-message-list-dot-${m.id}`">●</text>
        </view>
        <text class="escort-message-list__title">{{ m.title }}</text>
        <text class="escort-message-list__content">{{ m.content }}</text>
        <text class="escort-message-list__time">📅 {{ formatDate(m.created_at) }}</text>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.escort-message-list__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.escort-message-list__tab {
  flex: 1;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.escort-message-list__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.escort-message-list__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.escort-message-list__type {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  background: var(--ui-color-bg-hover);
  color: var(--ui-color-text-secondary);
  font-weight: var(--ui-font-weight-medium);
}

.escort-message-list__type--unread {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
}

.escort-message-list__dot {
  font-size: 12px;
  color: var(--ui-color-error);
}

.escort-message-list__title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-message-list__content {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-message-list__time {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-message-list__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>