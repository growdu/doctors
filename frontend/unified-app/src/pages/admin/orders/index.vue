<script setup lang="ts">
/**
 * admin/orders/index.vue — admin 订单概览（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「订单」相关入口
 *   → 3 关键指标（pending / in_service / completed）+ 跳列表
 *
 * 简化版：admin 域订单概览，提供快速跳转 list / detail
 */
import { ref, onMounted } from 'vue';
import { listOrders } from '@/api/orders';
import type { OrderStatus } from '@/api/orders';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const loading = ref(false);

const pendingCount = ref(0);
const inServiceCount = ref(0);
const completedCount = ref(0);

const STATUS_LIST: Array<{ status: OrderStatus; label: string; colorClass: string }> = [
  { status: 'pending_escort', label: '待陪诊师接单', colorClass: 'admin-orders__stat--pending' },
  { status: 'in_service', label: '服务中', colorClass: 'admin-orders__stat--active' },
  { status: 'completed', label: '已完成', colorClass: 'admin-orders__stat--done' },
];

async function loadStatusCount(status: OrderStatus): Promise<number> {
  try {
    const r = await listOrders({ status, page: 1, page_size: 1 });
    return r.total;
  } catch (e) {
    return 0;
  }
}

async function onLoad() {
  loading.value = true;
  try {
    const [p, i, c] = await Promise.all([
      loadStatusCount('pending_escort'),
      loadStatusCount('in_service'),
      loadStatusCount('completed'),
    ]);
    pendingCount.value = p;
    inServiceCount.value = i;
    completedCount.value = c;
  } finally {
    loading.value = false;
  }
}

function onViewList(status?: OrderStatus) {
  const qs = status ? `?status=${status}` : '';
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/admin/orders/list${qs}` });
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-orders-overview">
    <view v-if="loading" data-testid="orders-overview-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else data-testid="orders-overview-content">
      <UiCard v-for="s in STATUS_LIST" :key="s.status" :data-testid="`orders-stat-${s.status}`">
        <view class="admin-orders__stat" :class="s.colorClass">
          <view class="admin-orders__stat-info">
            <text class="admin-orders__stat-label">{{ s.label }}</text>
            <text class="admin-orders__stat-value">
              {{ s.status === 'pending_escort' ? pendingCount : s.status === 'in_service' ? inServiceCount : completedCount }}
            </text>
          </view>
          <UiButton
            type="primary"
            size="sm"
            :data-testid="`orders-stat-btn-${s.status}`"
            @click="onViewList(s.status)"
          >
            查看 →
          </UiButton>
        </view>
      </UiCard>

      <view class="admin-orders__all">
        <UiButton
          type="default"
          block
          data-testid="orders-stat-btn-all"
          @click="() => onViewList()"
        >
          查看全部订单
        </UiButton>
      </view>
    </view>
  </view>
</template>

<style scoped>
.admin-orders__stat {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.admin-orders__stat-info {
  flex: 1;
}

.admin-orders__stat-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
}

.admin-orders__stat-value {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-orders__stat--pending .admin-orders__stat-value { color: var(--ui-color-warning); }
.admin-orders__stat--active .admin-orders__stat-value { color: var(--ui-color-success); }
.admin-orders__stat--done .admin-orders__stat-value { color: var(--ui-color-text-disabled); }

.admin-orders__all {
  margin-top: var(--ui-space-base);
}
</style>