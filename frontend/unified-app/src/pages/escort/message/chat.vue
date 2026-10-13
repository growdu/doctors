<script setup lang="ts">
/**
 * escort/message/chat.vue — 陪诊师消息详情（v2 unified-app · escort 域 · v2.1 升级）。
 *
 * v2.1 升级：通知详情 + 消息轨迹 timeline + 标记已读/星标/删除操作。
 * v2 message-service 暂未暴露 1:1 chat；本页只展示通知详情，但 UX 更完整。
 *
 * 设计要点：
 *   - 头部：类型徽章 / 时间 / 标题 / 内容
 *   - 消息轨迹 timeline：created → (read) → (action) → 状态点
 *   - 操作区：标记已读（本地 toggle）/ 星标收藏（本地 toggle）/ 删除（本地）
 *   - 关联资源跳转：ref_type=order 跳订单详情
 *   - 区块级 catch：失败时显示空态 + 重试
 */
import { ref, computed, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';
import { getMessage } from '@/api/message';
import type { Message, MessageType } from '@/api/message';
import { storage } from '@/utils/storage';

const TYPE_LABEL: Record<MessageType, string> = {
  system: '系统通知',
  order: '订单通知',
  payment: '支付通知',
  sos: 'SOS 通知',
  review: '评价通知',
};

const TYPE_ICON: Record<MessageType, string> = {
  system: '🔔',
  order: '📋',
  payment: '💰',
  sos: '🚨',
  review: '⭐',
};

const message = ref<Message | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);
const messageId = ref<number | null>(null);

// 本地状态：已读 + 星标 + 已删除
const localRead = ref(false);
const starred = ref(false);
const removed = ref(false);

const READ_KEY_PREFIX = 'escort.msg.read.';
const STAR_KEY_PREFIX = 'escort.msg.star.';

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.id) messageId.value = Number(opts.id);
}

function loadLocalState() {
  if (!messageId.value) return;
  localRead.value = storage.getItem(READ_KEY_PREFIX + messageId.value) === '1';
  starred.value = storage.getItem(STAR_KEY_PREFIX + messageId.value) === '1';
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
    loadLocalState();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

const isRead = computed(() => message.value?.read || localRead.value);

const timeline = computed(() => {
  if (!message.value) return [];
  const items: { ts: string; label: string; icon: string }[] = [
    { ts: message.value.created_at, label: '收到通知', icon: '📨' },
  ];
  if (isRead.value) {
    items.push({ ts: '—', label: '已读', icon: '✅' });
  }
  if (starred.value) {
    items.push({ ts: '—', label: '已收藏', icon: '⭐' });
  }
  if (removed.value) {
    items.push({ ts: '—', label: '已删除', icon: '🗑️' });
  }
  return items;
});

function onToggleRead() {
  if (!messageId.value) return;
  localRead.value = !localRead.value;
  storage.setItem(READ_KEY_PREFIX + messageId.value, localRead.value ? '1' : '0');
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: localRead.value ? '已标记已读' : '标记为未读', icon: 'none' });
  }
}

function onToggleStar() {
  if (!messageId.value) return;
  starred.value = !starred.value;
  storage.setItem(STAR_KEY_PREFIX + messageId.value, starred.value ? '1' : '0');
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: starred.value ? '已收藏' : '取消收藏', icon: 'none' });
  }
}

function onRemove() {
  removed.value = true;
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: '已删除（本地）', icon: 'none' });
    setTimeout(() => uni.navigateBack(), 600);
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

    <template v-else-if="!removed">
      <!-- 头部 -->
      <UiCard data-testid="escort-message-chat-header">
        <view class="escort-message-chat__row">
          <text class="escort-message-chat__type">
            {{ TYPE_ICON[message.type] }} {{ TYPE_LABEL[message.type] }}
          </text>
          <text v-if="!isRead" class="escort-message-chat__unread-dot" data-testid="escort-message-chat-unread">未读</text>
        </view>
        <text class="escort-message-chat__title" data-testid="escort-message-chat-title">{{ message.title }}</text>
        <text class="escort-message-chat__content" data-testid="escort-message-chat-content">{{ message.content }}</text>
        <text class="escort-message-chat__time">{{ formatDate(message.created_at) }}</text>
        <view v-if="message.ref_id && message.ref_type" class="escort-message-chat__ref-row">
          <text class="escort-message-chat__ref-label">相关：</text>
          <text
            class="escort-message-chat__ref-link"
            :data-testid="`escort-message-chat-ref-${message.ref_type}`"
            @click="onRelatedClick"
          >
            {{ message.ref_type }} #{{ message.ref_id }} →
          </text>
        </view>
      </UiCard>

      <!-- 消息轨迹 -->
      <view class="escort-message-chat__section-title">📍 消息轨迹</view>
      <view class="escort-message-chat__timeline" data-testid="escort-message-chat-timeline">
        <view
          v-for="(item, i) in timeline"
          :key="i"
          class="escort-message-chat__timeline-item"
          :data-testid="`escort-message-chat-timeline-${i}`"
        >
          <text class="escort-message-chat__timeline-icon">{{ item.icon }}</text>
          <text class="escort-message-chat__timeline-label">{{ item.label }}</text>
          <text v-if="item.ts !== '—'" class="escort-message-chat__timeline-ts">{{ formatDate(item.ts) }}</text>
        </view>
      </view>

      <!-- 操作区 -->
      <view class="escort-message-chat__actions">
        <UiButton
          v-if="!isRead"
          type="primary"
          size="sm"
          block
          data-testid="escort-message-chat-mark-read"
          @click="onToggleRead"
        >
          ✓ 标记已读
        </UiButton>
        <UiButton
          :type="starred ? 'default' : 'ghost'"
          size="sm"
          block
          data-testid="escort-message-chat-star"
          @click="onToggleStar"
        >
          {{ starred ? '★ 已收藏' : '☆ 收藏' }}
        </UiButton>
        <UiButton
          type="ghost"
          size="sm"
          block
          data-testid="escort-message-chat-remove"
          @click="onRemove"
        >
          🗑️ 删除
        </UiButton>
      </view>
    </template>

    <view v-else data-testid="escort-message-chat-removed">
      <UiEmpty icon="🗑️" title="消息已删除" description="本条消息已从本设备删除" />
    </view>
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

.escort-message-chat__unread-dot {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-pill);
  background: var(--ui-color-error);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
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
  line-height: 1.5;
}

.escort-message-chat__time {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: var(--ui-space-sm);
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

.escort-message-chat__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.escort-message-chat__timeline {
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
  box-shadow: var(--ui-shadow-sm);
}

.escort-message-chat__timeline-item {
  display: flex;
  align-items: center;
  gap: var(--ui-space-sm);
}

.escort-message-chat__timeline-icon {
  font-size: 20px;
}

.escort-message-chat__timeline-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
  flex: 1;
}

.escort-message-chat__timeline-ts {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
}

.escort-message-chat__actions {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-md);
}

.escort-message-chat__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>