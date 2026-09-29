<script setup lang="ts">
/**
 * admin/refunds/index.vue — 退款管理首页（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「退款」相关入口
 *   → 聚合页：跳到列表 + 总览指标
 *
 * 简化版：核心入口是列表；其余入口占位
 */
import { ref, onMounted } from 'vue';
import { listRefunds } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';

const pendingCount = ref(0);
const approvedCount = ref(0);
const rejectedCount = ref(0);

async function loadCount(status: 'pending' | 'approved' | 'rejected'): Promise<number> {
  try {
    const r = await listRefunds({ status, page: 1, page_size: 1 });
    return r.total;
  } catch {
    return 0;
  }
}

async function onLoad() {
  const [p, a, r] = await Promise.all([
    loadCount('pending'),
    loadCount('approved'),
    loadCount('rejected'),
  ]);
  pendingCount.value = p;
  approvedCount.value = a;
  rejectedCount.value = r;
}

function onNavigate(path: string) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: path });
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-refunds-index">
    <view class="admin-refunds-index__header">
      <text class="admin-refunds-index__title">退款管理</text>
      <text class="admin-refunds-index__subtitle">退款工单审批</text>
    </view>

    <view class="admin-refunds-index__stats" data-testid="admin-refunds-stats">
      <UiCard data-testid="refunds-stat-pending">
        <view class="admin-refunds-index__stat admin-refunds-index__stat--pending">
          <text class="admin-refunds-index__stat-label">待审</text>
          <text class="admin-refunds-index__stat-value" :data-testid="`refunds-stat-pending-value`">{{ pendingCount }}</text>
        </view>
      </UiCard>
      <UiCard data-testid="refunds-stat-approved">
        <view class="admin-refunds-index__stat admin-refunds-index__stat--approved">
          <text class="admin-refunds-index__stat-label">已通过</text>
          <text class="admin-refunds-index__stat-value">{{ approvedCount }}</text>
        </view>
      </UiCard>
      <UiCard data-testid="refunds-stat-rejected">
        <view class="admin-refunds-index__stat admin-refunds-index__stat--rejected">
          <text class="admin-refunds-index__stat-label">已驳回</text>
          <text class="admin-refunds-index__stat-value">{{ rejectedCount }}</text>
        </view>
      </UiCard>
    </view>

    <view class="admin-refunds-index__actions">
      <UiButton
        type="primary"
        block
        data-testid="refunds-go-list"
        @click="() => onNavigate('/pages/admin/refunds/list')"
      >
        进入退款列表 →
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.admin-refunds-index__header {
  padding: var(--ui-space-base) 0;
}

.admin-refunds-index__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-refunds-index__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-refunds-index__stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-refunds-index__stat {
  text-align: center;
}

.admin-refunds-index__stat-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
}

.admin-refunds-index__stat-value {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-refunds-index__stat--pending .admin-refunds-index__stat-value { color: var(--ui-color-warning); }
.admin-refunds-index__stat--approved .admin-refunds-index__stat-value { color: var(--ui-color-success); }
.admin-refunds-index__stat--rejected .admin-refunds-index__stat-value { color: var(--ui-color-error); }

.admin-refunds-index__actions {
  margin-top: var(--ui-space-base);
}
</style>