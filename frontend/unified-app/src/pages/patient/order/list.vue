<script setup lang="ts">
/**
 * patient/order/list.vue — 订单列表（v2 unified-app · patient 域）。
 *
 * 入口：patient 域「个人中心」→ 我的订单 / 首页「我的订单」
 *   → onMounted 调 orderStore.fetchList({ role: 'patient' })
 *   → 渲染订单卡：医院 + 套餐 + 状态 + 金额 + 「查看详情」
 *
 * 设计要点：
 *   - 状态标签颜色映射（5 状态）
 *   - 点击订单跳详情
 */
import { onMounted } from 'vue';
import { useOrderStore } from '@/store/order';
import type { Order, OrderStatus } from '@/api/orders';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const store = useOrderStore();

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending_escort: '待陪诊师接单',
  escort_confirmed: '已确认',
  in_service: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

const STATUS_CLASS: Record<OrderStatus, string> = {
  pending_escort: 'order-list__status--pending',
  escort_confirmed: 'order-list__status--confirmed',
  in_service: 'order-list__status--active',
  completed: 'order-list__status--done',
  cancelled: 'order-list__status--cancel',
};

async function onLoad() {
  await store.fetchList({ role: 'patient' });
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

function onDetail(o: Order) {
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: `/pages/patient/order/detail?id=${o.id}` });
  }
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-order-list">
    <view v-if="store.loading && store.orders.length === 0" data-testid="order-list-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="store.error && store.orders.length === 0" data-testid="order-list-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
        <template #action>
          <text class="order-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="store.orders.length === 0" data-testid="order-list-empty">
      <UiEmpty icon="📋" title="还没有订单" description="快去下单体验吧">
        <template #action>
          <text class="order-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else data-testid="order-list">
      <UiCard
        v-for="o in store.orders"
        :key="o.id"
        :title="`订单 #${o.id}`"
        :data-testid="`order-card-${o.id}`"
        @click="onDetail(o)"
      >
        <view class="order-list__row">
          <text class="order-list__hospital">医院 #{{ o.hospital_id }}</text>
          <text :class="['order-list__status', STATUS_CLASS[o.status]]" :data-testid="`order-status-${o.id}`">
            {{ STATUS_LABEL[o.status] }}
          </text>
        </view>
        <text class="order-list__time">📅 {{ formatDate(o.appointment_time) }}</text>
        <text class="order-list__address">📍 {{ o.address }}</text>
        <template #footer>
          <view class="order-list__footer">
            <text class="order-list__amount">{{ formatAmount(o.total_amount) }}</text>
            <text class="order-list__view" :data-testid="`view-order-${o.id}`">查看详情 →</text>
          </view>
        </template>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.order-list__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.order-list__hospital {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.order-list__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.order-list__status--pending { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }
.order-list__status--confirmed { background: var(--ui-color-primary); color: var(--ui-color-text-inverse); }
.order-list__status--active { background: var(--ui-color-success); color: var(--ui-color-text-inverse); }
.order-list__status--done { background: var(--ui-color-text-disabled); color: var(--ui-color-text-inverse); }
.order-list__status--cancel { background: var(--ui-color-error); color: var(--ui-color-text-inverse); }

.order-list__time,
.order-list__address {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.order-list__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.order-list__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.order-list__view {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
}

.order-list__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>