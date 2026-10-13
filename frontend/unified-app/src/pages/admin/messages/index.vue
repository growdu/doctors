<script setup lang="ts">
/**
 * admin/messages/index.vue — 站内信管理（v2 unified-app · admin 域 · v2.1 升级）。
 *
 * v2.1 升级：从"占位 toast"升级为"实时拉取消息 + 类型聚合 dashboard"。
 * 后端 listMessages 已可用，本页展示全量站内信（按 type 聚合）。
 *
 * 设计要点：
 *   - 5 张类型卡：system / order / payment / sos / review
 *   - 列表：标题 / 类型徽章 / 创建时间 / 已读状态
 *   - 区块级 catch：失败不阻塞整体渲染
 */
import { ref, computed, onMounted } from 'vue';
import { listMessages } from '@/api/message';
import type { Message, MessageType } from '@/api/message';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const messages = ref<Message[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listMessages({ page_size: 100 });
    messages.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

const stats = computed(() => {
  const byType: Record<MessageType, number> = {
    system: 0, order: 0, payment: 0, sos: 0, review: 0,
  };
  let unread = 0;
  for (const m of messages.value) {
    byType[m.type] = (byType[m.type] ?? 0) + 1;
    if (!m.read) unread++;
  }
  return { total: messages.value.length, unread, byType };
});

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

function formatDate(iso?: string): string {
  if (!iso) return '—';
  const t = iso.split('T');
  return t[0] + ' ' + (t[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-messages-page">
    <view class="admin-messages__header">
      <text class="admin-messages__title">站内信管理</text>
      <text class="admin-messages__subtitle">实时统计 · 拉自 message-service</text>
    </view>

    <view v-if="loading && messages.length === 0" data-testid="admin-messages-loading">
      <UiLoading text="加载消息..." />
    </view>

    <view v-else-if="error && messages.length === 0" data-testid="admin-messages-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <UiButton type="primary" size="sm" @click="onLoad" data-testid="admin-messages-retry">重试</UiButton>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <!-- 5 张类型卡 -->
      <view class="admin-messages__stats" data-testid="admin-messages-stats">
        <view class="admin-messages__stat" data-testid="admin-messages-stat-total">
          <text class="admin-messages__stat-value">{{ stats.total }}</text>
          <text class="admin-messages__stat-label">总消息</text>
        </view>
        <view class="admin-messages__stat admin-messages__stat--unread" data-testid="admin-messages-stat-unread">
          <text class="admin-messages__stat-value">{{ stats.unread }}</text>
          <text class="admin-messages__stat-label">未读</text>
        </view>
      </view>

      <view class="admin-messages__types" data-testid="admin-messages-types">
        <view
          v-for="(count, type) in stats.byType"
          :key="type"
          class="admin-messages__type-chip"
          :data-testid="`admin-messages-type-${type}`"
        >
          <text class="admin-messages__type-icon">{{ TYPE_ICON[type] }}</text>
          <text class="admin-messages__type-label">{{ TYPE_LABEL[type] }}</text>
          <text class="admin-messages__type-count">{{ count }}</text>
        </view>
      </view>

      <!-- 消息列表 -->
      <view class="admin-messages__section-title">📨 最近消息</view>
      <view v-if="messages.length === 0" data-testid="admin-messages-empty">
        <UiEmpty icon="📭" title="暂无消息" />
      </view>
      <view v-else data-testid="admin-messages-list">
        <UiCard
          v-for="m in messages.slice(0, 20)"
          :key="m.id"
          :data-testid="`admin-message-${m.id}`"
        >
          <view class="admin-message__row">
            <text class="admin-message__type-icon">{{ TYPE_ICON[m.type] }}</text>
            <view class="admin-message__body">
              <text class="admin-message__title">{{ m.title }}</text>
              <text class="admin-message__content">{{ m.content }}</text>
              <text class="admin-message__meta">{{ TYPE_LABEL[m.type] }} · {{ formatDate(m.created_at) }}</text>
            </view>
            <view v-if="!m.read" class="admin-message__unread-dot" :data-testid="`admin-message-unread-${m.id}`" />
          </view>
        </UiCard>
      </view>
    </template>
  </view>
</template>

<style scoped>
.admin-messages__header {
  padding: var(--ui-space-base) 0 var(--ui-space-sm);
}

.admin-messages__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-messages__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-messages__stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ui-space-md);
  margin-bottom: var(--ui-space-md);
}

.admin-messages__stat {
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: var(--ui-shadow-sm);
}

.admin-messages__stat-value {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-primary);
  display: block;
}

.admin-messages__stat--unread .admin-messages__stat-value { color: var(--ui-color-error); }

.admin-messages__stat-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  margin-top: var(--ui-space-xs);
}

.admin-messages__types {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ui-space-sm);
  margin-bottom: var(--ui-space-md);
}

.admin-messages__type-chip {
  display: flex;
  align-items: center;
  gap: var(--ui-space-xs);
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-pill);
  padding: 4px 12px;
  box-shadow: var(--ui-shadow-sm);
}

.admin-messages__type-icon { font-size: var(--ui-font-base); }
.admin-messages__type-label { font-size: var(--ui-font-sm); color: var(--ui-color-text-primary); }
.admin-messages__type-count {
  font-size: var(--ui-font-xs);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-primary);
  background: var(--ui-color-bg-hover);
  padding: 1px 6px;
  border-radius: var(--ui-radius-pill);
}

.admin-messages__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.admin-message__row {
  display: flex;
  align-items: flex-start;
  gap: var(--ui-space-sm);
  position: relative;
}

.admin-message__type-icon { font-size: 24px; }

.admin-message__body { flex: 1; }

.admin-message__title {
  font-size: var(--ui-font-base);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-message__content {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-message__meta {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: 4px;
}

.admin-message__unread-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ui-color-error);
  margin-top: 6px;
  flex-shrink: 0;
}
</style>