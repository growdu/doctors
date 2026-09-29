<script setup lang="ts">
/**
 * admin/messages/index.vue — 站内信管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「站内信」相关入口
 *   → v2 admin-service 未暴露站内信 CRUD 端点
 *   → 本页为占位说明：跳转 patient 域的消息列表查看数据
 *
 * 设计要点：
 *   - 简单页：2 个入口卡（推送消息占位 + 用户视角查看）
 */
import { onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';

interface Entry {
  key: string;
  icon: string;
  title: string;
  desc: string;
  path: string;
  available: boolean;
}

const ENTRIES: Entry[] = [
  {
    key: 'broadcast',
    icon: '📢',
    title: '群发消息',
    desc: 'v2 后端暂未上线',
    path: '',
    available: false,
  },
  {
    key: 'patient-view',
    icon: '📬',
    title: '用户视角消息',
    desc: '查看 patient 域站内信（user-service /api/v1/messages）',
    path: '/pages/patient/message/list',
    available: true,
  },
];

function onNavigate(entry: Entry) {
  if (!entry.available) {
    if (typeof uni !== 'undefined') uni.showToast({ title: '功能开发中', icon: 'none' });
    return;
  }
  if (typeof uni !== 'undefined') uni.navigateTo({ url: entry.path });
}

onMounted(() => {
  // 占位：不发起 API 调用
});
</script>

<template>
  <view class="ui-page" data-testid="admin-messages-page">
    <view class="admin-messages__header">
      <text class="admin-messages__title">站内信管理</text>
      <text class="admin-messages__subtitle">v2 admin-service 暂未暴露群发端点</text>
    </view>

    <view class="admin-messages__note" data-testid="admin-messages-note">
      <text class="admin-messages__note-text">
        💡 当前 admin 端只能查看用户视角的站内信。群发 / 模板管理等功能需等 admin-service 上线。
      </text>
    </view>

    <view class="admin-messages__list" data-testid="admin-messages-list">
      <UiCard
        v-for="e in ENTRIES"
        :key="e.key"
        :data-testid="`admin-messages-entry-${e.key}`"
        @click="onNavigate(e)"
      >
        <view
          class="admin-messages__entry"
          :class="{ 'admin-messages__entry--disabled': !e.available }"
        >
          <text class="admin-messages__entry-icon">{{ e.icon }}</text>
          <view class="admin-messages__entry-info">
            <text class="admin-messages__entry-title">{{ e.title }}</text>
            <text class="admin-messages__entry-desc">{{ e.desc }}</text>
          </view>
          <text class="admin-messages__entry-arrow">{{ e.available ? '→' : '🚧' }}</text>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-messages__header {
  padding: var(--ui-space-base) 0;
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

.admin-messages__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-warning);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-md);
}

.admin-messages__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.admin-messages__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.admin-messages__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.admin-messages__entry--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.admin-messages__entry-icon {
  font-size: 32px;
}

.admin-messages__entry-info {
  flex: 1;
}

.admin-messages__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-messages__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-messages__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}
</style>