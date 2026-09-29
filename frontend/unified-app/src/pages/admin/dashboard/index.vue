<script setup lang="ts">
/**
 * admin/dashboard/index.vue — 管理后台仪表盘（v2 unified-app · admin 域）。
 *
 * 入口：admin 域首页
 *   → 拉 api/admin.getReportOverview() 拿 6 项关键指标
 *   → 6 卡片栅格（4 + 2）+ 点击跳对应列表
 *
 * 设计要点：
 *   - 6 卡片：总订单 / 总金额 / 今日订单 / 今日 GMV / 在线陪诊师 / 用户数（占位）
 *   - 数字格式化 + 颜色映射（金额 error 红 / 订单 primary）
 *   - 卡片点击跳 admin/orders/list（统一入口）
 */
import { ref, onMounted } from 'vue';
import { getReportOverview } from '@/api/admin';
import type { ReportOverview } from '@/api/admin';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

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

interface Stat {
  key: string;
  label: string;
  icon: string;
  value: string;
  colorClass: string;
  path: string;
}

const STATS = ref<Stat[]>([]);

function rebuildStats() {
  if (!overview.value) return;
  STATS.value = [
    {
      key: 'total-orders',
      label: '总订单数',
      icon: '📋',
      value: overview.value.total_orders.toLocaleString(),
      colorClass: 'admin-dashboard__stat--primary',
      path: '/pages/admin/orders/list',
    },
    {
      key: 'total-amount',
      label: '总金额',
      icon: '💰',
      value: `¥${(overview.value.total_amount / 100).toLocaleString()}`,
      colorClass: 'admin-dashboard__stat--error',
      path: '/pages/admin/finance/index',
    },
    {
      key: 'today-orders',
      label: '今日订单',
      icon: '📈',
      value: overview.value.today_orders.toLocaleString(),
      colorClass: 'admin-dashboard__stat--success',
      path: '/pages/admin/orders/list',
    },
    {
      key: 'today-amount',
      label: '今日 GMV',
      icon: '💵',
      value: `¥${(overview.value.today_amount / 100).toLocaleString()}`,
      colorClass: 'admin-dashboard__stat--warning',
      path: '/pages/admin/finance/index',
    },
    {
      key: 'online-escorts',
      label: '在线陪诊师',
      icon: '👨‍⚕️',
      value: overview.value.online_escorts.toLocaleString(),
      colorClass: 'admin-dashboard__stat--info',
      path: '/pages/admin/escorts/index',
    },
    {
      key: 'work-orders',
      label: '待处理工单',
      icon: '🎫',
      value: '0',
      colorClass: 'admin-dashboard__stat--disabled',
      path: '/pages/admin/work-orders/index',
    },
  ];
}

function onCardClick(stat: Stat) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: stat.path });
}

onMounted(async () => {
  await onLoad();
  rebuildStats();
});
</script>

<template>
  <view class="ui-page" data-testid="admin-dashboard">
    <view v-if="loading && !overview" data-testid="dashboard-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error && !overview" data-testid="dashboard-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-dashboard__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="overview" data-testid="dashboard-content">
      <view class="admin-dashboard__header">
        <text class="admin-dashboard__title">仪表盘</text>
        <text class="admin-dashboard__subtitle">实时业务监控</text>
      </view>

      <view class="admin-dashboard__grid" data-testid="dashboard-grid">
        <view
          v-for="stat in STATS"
          :key="stat.key"
          class="admin-dashboard__stat"
          :class="stat.colorClass"
          :data-testid="`dashboard-stat-${stat.key}`"
          @click="onCardClick(stat)"
        >
          <text class="admin-dashboard__stat-icon">{{ stat.icon }}</text>
          <text class="admin-dashboard__stat-value">{{ stat.value }}</text>
          <text class="admin-dashboard__stat-label">{{ stat.label }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.admin-dashboard__header {
  padding: var(--ui-space-base) 0;
}

.admin-dashboard__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-dashboard__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-dashboard__grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ui-space-md);
}

.admin-dashboard__stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-base);
  box-shadow: var(--ui-shadow-sm);
  cursor: pointer;
  transition: transform var(--ui-duration-fast);
}

.admin-dashboard__stat:active {
  transform: scale(0.97);
}

.admin-dashboard__stat-icon {
  font-size: 32px;
  margin-bottom: var(--ui-space-sm);
}

.admin-dashboard__stat-value {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  display: block;
  margin-bottom: var(--ui-space-xs);
}

.admin-dashboard__stat-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.admin-dashboard__stat--primary .admin-dashboard__stat-value { color: var(--ui-color-primary); }
.admin-dashboard__stat--error .admin-dashboard__stat-value { color: var(--ui-color-error); }
.admin-dashboard__stat--success .admin-dashboard__stat-value { color: var(--ui-color-success); }
.admin-dashboard__stat--warning .admin-dashboard__stat-value { color: var(--ui-color-warning); }
.admin-dashboard__stat--info .admin-dashboard__stat-value { color: var(--ui-color-info); }
.admin-dashboard__stat--disabled .admin-dashboard__stat-value { color: var(--ui-color-text-disabled); }

.admin-dashboard__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>