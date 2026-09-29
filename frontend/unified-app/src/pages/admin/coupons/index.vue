<script setup lang="ts">
/**
 * admin/coupons/index.vue — 优惠券管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「优惠券」相关入口
 *   → v2 admin-service 未暴露优惠券 CRUD 端点（v1 MSW-only）
 *   → 本页为占位说明：直接跳转 patient 域的优惠券页查看用户视角数据
 *
 * 设计要点：
 *   - 简单页：2 个入口卡（创建优惠券占位 + 用户视角查看）
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
    key: 'create',
    icon: '➕',
    title: '创建优惠券',
    desc: 'v2 后端暂未上线',
    path: '',
    available: false,
  },
  {
    key: 'patient-view',
    icon: '🎟️',
    title: '用户视角优惠券',
    desc: '查看 patient 域优惠券（user-service /api/v1/coupons）',
    path: '/pages/patient/coupons/index',
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
  <view class="ui-page" data-testid="admin-coupons-page">
    <view class="admin-coupons__header">
      <text class="admin-coupons__title">优惠券管理</text>
      <text class="admin-coupons__subtitle">v2 admin-service 暂未暴露 CRUD 端点</text>
    </view>

    <view class="admin-coupons__note" data-testid="admin-coupons-note">
      <text class="admin-coupons__note-text">
        💡 当前 admin 端只能查看用户视角的优惠券。优惠券创建 / 禁用等管理功能需等 admin-service 上线。
      </text>
    </view>

    <view class="admin-coupons__list" data-testid="admin-coupons-list">
      <UiCard
        v-for="e in ENTRIES"
        :key="e.key"
        :data-testid="`admin-coupons-entry-${e.key}`"
        @click="onNavigate(e)"
      >
        <view
          class="admin-coupons__entry"
          :class="{ 'admin-coupons__entry--disabled': !e.available }"
        >
          <text class="admin-coupons__entry-icon">{{ e.icon }}</text>
          <view class="admin-coupons__entry-info">
            <text class="admin-coupons__entry-title">{{ e.title }}</text>
            <text class="admin-coupons__entry-desc">{{ e.desc }}</text>
          </view>
          <text class="admin-coupons__entry-arrow">{{ e.available ? '→' : '🚧' }}</text>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-coupons__header {
  padding: var(--ui-space-base) 0;
}

.admin-coupons__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-coupons__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-coupons__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-warning);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-md);
}

.admin-coupons__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.admin-coupons__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.admin-coupons__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.admin-coupons__entry--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.admin-coupons__entry-icon {
  font-size: 32px;
}

.admin-coupons__entry-info {
  flex: 1;
}

.admin-coupons__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-coupons__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-coupons__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}
</style>