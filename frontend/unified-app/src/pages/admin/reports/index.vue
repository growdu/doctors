<script setup lang="ts">
/**
 * admin/reports/index.vue — 数据报表（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「总订单/总金额/今日订单/今日 GMV」卡片
 *   → getReportOverview() 拉 5 项关键指标
 *   → 「导出报表」按钮（占位 + UniModal 反馈）
 *
 * 设计要点：
 *   - 简化版：复用 dashboard 的 6 卡片布局但改为大卡（5 项）
 *   - v2 admin-service 未暴露趋势/分类报表，本期仅做 overview
 */
import { ref, onMounted } from 'vue';
import { getReportOverview } from '@/api/admin';
import type { ReportOverview } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';

const overview = ref<ReportOverview | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    overview.value = await getReportOverview();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toLocaleString()}`;
}

function onExport() {
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: '导出功能开发中', icon: 'none' });
  }
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-reports-page">
    <view v-if="loading && !overview" data-testid="admin-reports-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error && !overview" data-testid="admin-reports-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-reports__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="overview" data-testid="admin-reports-content">
      <view class="admin-reports__header">
        <text class="admin-reports__title">数据报表</text>
        <text class="admin-reports__subtitle">业务核心指标概览</text>
      </view>

      <view class="admin-reports__grid" data-testid="admin-reports-grid">
        <UiCard data-testid="reports-stat-total-orders">
          <view class="admin-reports__stat admin-reports__stat--primary">
            <text class="admin-reports__stat-icon">📋</text>
            <text class="admin-reports__stat-value">{{ overview.total_orders.toLocaleString() }}</text>
            <text class="admin-reports__stat-label">总订单数</text>
          </view>
        </UiCard>

        <UiCard data-testid="reports-stat-total-amount">
          <view class="admin-reports__stat admin-reports__stat--error">
            <text class="admin-reports__stat-icon">💰</text>
            <text class="admin-reports__stat-value">{{ formatAmount(overview.total_amount) }}</text>
            <text class="admin-reports__stat-label">总金额</text>
          </view>
        </UiCard>

        <UiCard data-testid="reports-stat-today-orders">
          <view class="admin-reports__stat admin-reports__stat--success">
            <text class="admin-reports__stat-icon">📈</text>
            <text class="admin-reports__stat-value">{{ overview.today_orders.toLocaleString() }}</text>
            <text class="admin-reports__stat-label">今日订单</text>
          </view>
        </UiCard>

        <UiCard data-testid="reports-stat-today-amount">
          <view class="admin-reports__stat admin-reports__stat--warning">
            <text class="admin-reports__stat-icon">💵</text>
            <text class="admin-reports__stat-value">{{ formatAmount(overview.today_amount) }}</text>
            <text class="admin-reports__stat-label">今日 GMV</text>
          </view>
        </UiCard>

        <UiCard data-testid="reports-stat-online-escorts">
          <view class="admin-reports__stat admin-reports__stat--info">
            <text class="admin-reports__stat-icon">👨‍⚕️</text>
            <text class="admin-reports__stat-value">{{ overview.online_escorts.toLocaleString() }}</text>
            <text class="admin-reports__stat-label">在线陪诊师</text>
          </view>
        </UiCard>
      </view>

      <view class="admin-reports__actions">
        <UiButton type="primary" block data-testid="admin-reports-export-btn" @click="onExport">
          导出报表
        </UiButton>
      </view>
    </view>
  </view>
</template>

<style scoped>
.admin-reports__header {
  padding: var(--ui-space-base) 0;
}

.admin-reports__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-reports__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-reports__grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-reports__stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--ui-space-sm);
}

.admin-reports__stat-icon {
  font-size: 32px;
  margin-bottom: var(--ui-space-sm);
}

.admin-reports__stat-value {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-bold);
  display: block;
}

.admin-reports__stat-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-reports__stat--primary .admin-reports__stat-value { color: var(--ui-color-primary); }
.admin-reports__stat--error .admin-reports__stat-value { color: var(--ui-color-error); }
.admin-reports__stat--success .admin-reports__stat-value { color: var(--ui-color-success); }
.admin-reports__stat--warning .admin-reports__stat-value { color: var(--ui-color-warning); }
.admin-reports__stat--info .admin-reports__stat-value { color: var(--ui-color-info); }

.admin-reports__actions {
  margin-top: var(--ui-space-base);
}

.admin-reports__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>