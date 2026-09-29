<script setup lang="ts">
/**
 * admin/sos/index.vue — SOS 紧急工单管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「SOS」相关入口
 *   → v2 admin-service 未暴露 SOS 管理端点（sos 由 sos-service 自治）
 *   → 本页为占位说明：跳转 patient 域的 SOS 触发页（演示视角）
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
    key: 'list',
    icon: '🚨',
    title: 'SOS 工单列表',
    desc: 'v2 后端暂未上线（sos-service 自治告警）',
    path: '',
    available: false,
  },
  {
    key: 'patient-view',
    icon: '🆘',
    title: 'SOS 触发演示',
    desc: '查看 patient 域 SOS 触发页（演示视角）',
    path: '/pages/patient/sos/trigger',
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
  <view class="ui-page" data-testid="admin-sos-page">
    <view class="admin-sos__header">
      <text class="admin-sos__title">SOS 管理</text>
      <text class="admin-sos__subtitle">v2 SOS 紧急工单由 sos-service 自治</text>
    </view>

    <view class="admin-sos__note" data-testid="admin-sos-note">
      <text class="admin-sos__note-text">
        💡 SOS 工单走 sos-service，不通过 admin-service。管理功能需等 sos-service 暴露查询端点。
      </text>
    </view>

    <view class="admin-sos__list" data-testid="admin-sos-list">
      <UiCard
        v-for="e in ENTRIES"
        :key="e.key"
        :data-testid="`admin-sos-entry-${e.key}`"
        @click="onNavigate(e)"
      >
        <view class="admin-sos__entry" :class="{ 'admin-sos__entry--disabled': !e.available }">
          <text class="admin-sos__entry-icon">{{ e.icon }}</text>
          <view class="admin-sos__entry-info">
            <text class="admin-sos__entry-title">{{ e.title }}</text>
            <text class="admin-sos__entry-desc">{{ e.desc }}</text>
          </view>
          <text class="admin-sos__entry-arrow">{{ e.available ? '→' : '🚧' }}</text>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-sos__header {
  padding: var(--ui-space-base) 0;
}

.admin-sos__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-sos__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-sos__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-error);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-md);
}

.admin-sos__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.admin-sos__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.admin-sos__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.admin-sos__entry--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.admin-sos__entry-icon {
  font-size: 32px;
}

.admin-sos__entry-info {
  flex: 1;
}

.admin-sos__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-sos__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-sos__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}
</style>