<script setup lang="ts">
/**
 * admin/dashboard/detail.vue — 仪表盘详情页（v2 unified-app · admin 域）。
 *
 * 入口：dashboard 卡片点击「工单 / 陪诊师」等深度链接（待 v2.1 接）
 * * v2.0 占位：显示详细指标趋势 + 退款 / SOS 待办列表
 *
 * 简化版：复用 overview + 列出前 5 退款 + 前 5 工单占位
 */
import { ref, onMounted } from 'vue';
import { getReportOverview, listRefunds, listWorkOrders } from '@/api/admin';
import type { ReportOverview, Refund, WorkOrder } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const overview = ref<ReportOverview | null>(null);
const refunds = ref<Refund[]>([]);
const workOrders = ref<WorkOrder[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const [ov, rf, wo] = await Promise.all([
      getReportOverview(),
      listRefunds({ status: 'pending' }).catch(() => ({ items: [] })),
      listWorkOrders({ status: 'open' }).catch(() => ({ items: [] })),
    ]);
    overview.value = ov;
    refunds.value = rf.items.slice(0, 5);
    workOrders.value = wo.items.slice(0, 5);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-dashboard-detail">
    <view v-if="loading && !overview" data-testid="detail-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error && !overview" data-testid="detail-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-detail__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="overview" data-testid="detail-content">
      <!-- 详细指标 -->
      <UiCard title="业务指标" data-testid="detail-metrics">
        <view class="admin-detail__metric-row">
          <text class="admin-detail__metric-label">累计订单</text>
          <text class="admin-detail__metric-value">{{ overview.total_orders.toLocaleString() }}</text>
        </view>
        <view class="admin-detail__metric-row">
          <text class="admin-detail__metric-label">累计 GMV</text>
          <text class="admin-detail__metric-value admin-detail__metric-value--amount">{{ formatYuan(overview.total_amount) }}</text>
        </view>
        <view class="admin-detail__metric-row">
          <text class="admin-detail__metric-label">今日订单 / GMV</text>
          <text class="admin-detail__metric-value">
            {{ overview.today_orders }} 单 / {{ formatYuan(overview.today_amount) }}
          </text>
        </view>
        <view class="admin-detail__metric-row">
          <text class="admin-detail__metric-label">在线陪诊师</text>
          <text class="admin-detail__metric-value">{{ overview.online_escorts }} 位</text>
        </view>
      </UiCard>

      <!-- 待审批退款 -->
      <UiCard title="待审批退款（5））" data-testid="detail-refunds">
        <view v-if="refunds.length === 0" data-testid="detail-refunds-empty">
          <UiEmpty icon="💰" title="暂无待审批退款" />
        </view>
        <view v-else data-testid="detail-refunds-list">
          <view
            v-for="r in refunds"
            :key="r.id"
            class="admin-detail__item"
            :data-testid="`detail-refund-${r.id}`"
          >
            <view class="admin-detail__item-row">
              <text class="admin-detail__item-id">退款 #{{ r.id }}</text>
              <text class="admin-detail__item-amount">{{ formatYuan(r.amount) }}</text>
            </view>
            <text class="admin-detail__item-reason">{{ r.reason }}</text>
          </view>
        </view>
      </UiCard>

      <!-- 待处理工单 -->
      <UiCard title="待处理工单（5）" data-testid="detail-work-orders">
        <view v-if="workOrders.length === 0" data-testid="detail-work-orders-empty">
          <UiEmpty icon="🎫" title="暂无待处理工单" />
        </view>
        <view v-else data-testid="detail-work-orders-list">
          <view
            v-for="w in workOrders"
            :key="w.id"
            class="admin-detail__item"
            :data-testid="`detail-work-order-${w.id}`"
          >
            <view class="admin-detail__item-row">
              <text class="admin-detail__item-id">工单 #{{ w.id }} · {{ w.type }}</text>
              <text class="admin-detail__item-status">{{ w.status }}</text>
            </view>
            <text class="admin-detail__item-reason">{{ w.title }} - {{ w.content }}</text>
          </view>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-detail__metric-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--ui-space-sm) 0;
  border-bottom: 1px solid var(--ui-color-divider);
}

.admin-detail__metric-row:last-of-type {
  border-bottom: none;
}

.admin-detail__metric-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.admin-detail__metric-value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
}

.admin-detail__metric-value--amount {
  color: var(--ui-color-error);
}

.admin-detail__item {
  padding: var(--ui-space-sm) 0;
  border-bottom: 1px solid var(--ui-color-divider);
}

.admin-detail__item:last-child {
  border-bottom: none;
}

.admin-detail__item-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2px;
}

.admin-detail__item-id {
  font-size: var(--ui-font-sm);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.admin-detail__item-amount {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  font-weight: var(--ui-font-weight-semibold);
}

.admin-detail__item-status {
  font-size: var(--ui-font-xs);
  background: var(--ui-color-warning);
  color: var(--ui-color-text-inverse);
  padding: 1px 6px;
  border-radius: var(--ui-radius-sm);
}

.admin-detail__item-reason {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
}

.admin-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>