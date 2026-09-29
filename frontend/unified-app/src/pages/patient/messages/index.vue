<script setup lang="ts">
/**
 * patient/messages/index.vue — patient 域消息聚合页（v2 unified-app）。
 *
 * 入口：patient 域「消息中心」聚合入口
 *   → 复用 message/list 内容 + 顶部 Tab（全部 / 系统 / 订单 / 支付）
 *   → 简化版：仅展示 patient 域关心的 3 类型（system / order / payment）
 */
import { ref, computed, onMounted } from 'vue';
import { listMessages } from '@/api/message';
import type { Message } from '@/api/message';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const messages = ref<Message[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const activeTab = ref<'all' | 'system' | 'order' | 'payment'>('all');

const filtered = computed(() => {
  if (activeTab.value === 'all') return messages.value;
  return messages.value.filter((m) => m.type === activeTab.value);
});

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listMessages();
    messages.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatDate(iso: string): string {
  return iso.split('T')[0]?.substring(5) || iso;
}

function onDetail(m: Message) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/patient/message/detail?id=${m.id}` });
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-messages">
    <view class="messages__tabs" data-testid="messages-tabs">
      <view
        v-for="t in (['all', 'system', 'order', 'payment'] as const)"
        :key="t"
        class="messages__tab"
        :class="{ 'messages__tab--active': activeTab === t }"
        :data-testid="`messages-tab-${t}`"
        @click="activeTab = t"
      >
        {{ t === 'all' ? '全部' : t === 'system' ? '系统' : t === 'order' ? '订单' : '支付' }}
      </view>
    </view>

    <view v-if="loading && filtered.length === 0" data-testid="messages-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="messages-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="messages__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="filtered.length === 0" data-testid="messages-empty">
      <UiEmpty icon="📭" title="暂无消息" />
    </view>

    <view v-else data-testid="messages-list">
      <view
        v-for="m in filtered"
        :key="m.id"
        class="messages__item"
        :class="{ 'messages__item--unread': !m.read }"
        :data-testid="`messages-item-${m.id}`"
        @click="onDetail(m)"
      >
        <view class="messages__item-body">
          <text class="messages__item-title">{{ m.title }}</text>
          <text class="messages__item-time">{{ formatDate(m.created_at) }}</text>
        </view>
        <view v-if="!m.read" class="messages__item-dot" />
      </view>
    </view>
  </view>
</template>

<style scoped>
.messages__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
}

.messages__tab {
  flex: 1;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
}

.messages__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.messages__item {
  display: flex;
  align-items: center;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  margin-bottom: var(--ui-space-sm);
  box-shadow: var(--ui-shadow-sm);
}

.messages__item--unread {
  border-left: 3px solid var(--ui-color-primary);
  background: rgba(22, 119, 255, 0.04);
}

.messages__item-body {
  flex: 1;
  min-width: 0;
}

.messages__item-title {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.messages__item-time {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: 2px;
}

.messages__item-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ui-color-error);
  flex-shrink: 0;
  margin-left: var(--ui-space-sm);
}

.messages__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>