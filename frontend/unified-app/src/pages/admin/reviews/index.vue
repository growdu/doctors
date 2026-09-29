<script setup lang="ts">
/**
 * admin/reviews/index.vue — 评价管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「评价」相关入口
 *   → v2 admin-service 未暴露评价管理端点
 *   → 本页为占位说明：跳转 patient 域的评价列表查看数据
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
    key: 'moderate',
    icon: '🛡️',
    title: '评价审核',
    desc: 'v2 后端暂未上线',
    path: '',
    available: false,
  },
  {
    key: 'patient-view',
    icon: '⭐',
    title: '用户视角评价',
    desc: '查看 patient 域评价（user-service /api/v1/reviews）',
    path: '/pages/patient/reviews/index',
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
  <view class="ui-page" data-testid="admin-reviews-page">
    <view class="admin-reviews__header">
      <text class="admin-reviews__title">评价管理</text>
      <text class="admin-reviews__subtitle">v2 admin-service 暂未暴露审核端点</text>
    </view>

    <view class="admin-reviews__note" data-testid="admin-reviews-note">
      <text class="admin-reviews__note-text">
        💡 当前 admin 端只能查看用户视角的评价。评价审核 / 删除 / 屏蔽等功能需等 admin-service 上线。
      </text>
    </view>

    <view class="admin-reviews__list" data-testid="admin-reviews-list">
      <UiCard
        v-for="e in ENTRIES"
        :key="e.key"
        :data-testid="`admin-reviews-entry-${e.key}`"
        @click="onNavigate(e)"
      >
        <view class="admin-reviews__entry" :class="{ 'admin-reviews__entry--disabled': !e.available }">
          <text class="admin-reviews__entry-icon">{{ e.icon }}</text>
          <view class="admin-reviews__entry-info">
            <text class="admin-reviews__entry-title">{{ e.title }}</text>
            <text class="admin-reviews__entry-desc">{{ e.desc }}</text>
          </view>
          <text class="admin-reviews__entry-arrow">{{ e.available ? '→' : '🚧' }}</text>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-reviews__header {
  padding: var(--ui-space-base) 0;
}

.admin-reviews__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-reviews__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-reviews__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-warning);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-md);
}

.admin-reviews__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.admin-reviews__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.admin-reviews__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.admin-reviews__entry--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.admin-reviews__entry-icon {
  font-size: 32px;
}

.admin-reviews__entry-info {
  flex: 1;
}

.admin-reviews__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-reviews__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-reviews__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}
</style>