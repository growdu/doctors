<script setup lang="ts">
/**
 * patient/notifications/index.vue — patient 域通知聚合（v2 unified-app）。
 *
 * 入口：patient 域 Tab「通知」
 *   → 系统通知（mock 静态数据）
 *   → 营销通知（mock）
 *
 * v2 简化：通知页用本地 mock 列表（接入消息中心后续接后端 api/message.broadcast）
 */
import { ref } from 'vue';

interface Notice {
  id: number;
  icon: string;
  title: string;
  content: string;
  time: string;
  read: boolean;
}

const notices = ref<Notice[]>([
  {
    id: 1,
    icon: '🎉',
    title: '欢迎使用 Doctors 统一 App',
    content: 'v2.0 已上线：3 端合并 + 多角色 + 角色切换',
    time: '2 天前',
    read: false,
  },
  {
    id: 2,
    icon: '💝',
    title: '限时优惠：新人首单立减 30 元',
    content: '完成首单即可领取优惠券，限 7 天内使用',
    time: '3 天前',
    read: false,
  },
  {
    id: 3,
    icon: '🔔',
    title: '实名认证可解锁全部服务',
    content: '下单、SOS 一键联系、陪诊师优先匹配',
    time: '5 天前',
    read: true,
  },
  {
    id: 4,
    icon: '⚡',
    title: 'VIP 陪诊师上线',
    content: '资深医护背景陪诊师已上线，可优先匹配',
    time: '1 周前',
    read: true,
  },
]);

function onMarkAllRead() {
  notices.value.forEach((n) => (n.read = true));
}
</script>

<template>
  <view class="ui-page" data-testid="patient-notifications">
    <view class="notifications__actions" data-testid="notifications-actions">
      <text class="notifications__count">共 {{ notices.length }} 条通知</text>
      <text class="notifications__mark-read" data-testid="notifications-mark-read" @click="onMarkAllRead">
        全部已读
      </text>
    </view>

    <view data-testid="notifications-list">
      <view
        v-for="n in notices"
        :key="n.id"
        class="notifications__item"
        :class="{ 'notifications__item--unread': !n.read }"
        :data-testid="`notifications-item-${n.id}`"
      >
        <text class="notifications__item-icon">{{ n.icon }}</text>
        <view class="notifications__item-body">
          <view class="notifications__item-header">
            <text class="notifications__item-title">{{ n.title }}</text>
            <text class="notifications__item-time">{{ n.time }}</text>
          </view>
          <text class="notifications__item-content">{{ n.content }}</text>
        </view>
        <view v-if="!n.read" class="notifications__item-dot" data-testid="notifications-unread-dot" />
      </view>
    </view>
  </view>
</template>

<style scoped>
.notifications__actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--ui-space-sm) var(--ui-space-xs);
  margin-bottom: var(--ui-space-sm);
}

.notifications__count {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.notifications__mark-read {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  cursor: pointer;
}

.notifications__item {
  display: flex;
  align-items: flex-start;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  margin-bottom: var(--ui-space-sm);
  box-shadow: var(--ui-shadow-sm);
  gap: var(--ui-space-md);
}

.notifications__item--unread {
  border-left: 3px solid var(--ui-color-primary);
}

.notifications__item-icon {
  font-size: 28px;
  flex-shrink: 0;
}

.notifications__item-body {
  flex: 1;
  min-width: 0;
}

.notifications__item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--ui-space-xs);
}

.notifications__item-title {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
}

.notifications__item-time {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  flex-shrink: 0;
}

.notifications__item-content {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  line-height: 1.5;
  display: block;
}

.notifications__item-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ui-color-error);
  flex-shrink: 0;
  align-self: flex-start;
  margin-top: var(--ui-space-sm);
}
</style>