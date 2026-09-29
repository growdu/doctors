<script setup lang="ts">
/**
 * admin/orders/list.vue — admin 订单列表（v2 unified-app · admin 域）。
 *
 * 入口：admin orders 概览卡片点击
 *   → query 携带 ?status=xxx 过滤（OrderStatus）
 *   → 调 orderStore.fetchList({ role: 'patient', status })
 *
 * 设计要点：
 *   - 5 状态过滤 tab（pending / confirmed / in_service / completed / cancelled）
 *   - 复用 patient/order/list 渲染样式 + 订单卡
 */
import { ref, onMounted } from 'vue';
import { useOrderStore } from '@/store/order';
import type { Order, OrderStatus } from '@/api/orders';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const store = useOrderStore();

const STATUS_TABS: Array<{ status: OrderStatus | 'all'; label: string }> = [
  { status: 'all', label: '全部' },
  { status: 'pending_escort', label: '待接单' },
  { status: 'escort_confirmed', label: '已确认' },
  { status: 'in_service', label: '服务中' },
  { status: 'completed', label: '已完成' },
  { status: 'cancelled', label: '已取消' },
];

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending_escort: '待陪诊师接单',
  escort_confirmed: '已确认',
  in_service: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

const STATUS_CLASS: Record<OrderStatus, string> = {
  pending_escort: 'admin-orders-list__status--pending',
  escort_confirmed: 'admin-orders-list__status--confirmed',
  in_service: 'admin-orders-list__status--active',
  completed: 'admin-orders-list__status--done',
  cancelled: 'admin-orders-list__status--cancel',
};

const activeTab = ref<OrderStatus | 'all'>('all');
const initialStatus = ref<OrderStatus | null>(null);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.status) {
    initialStatus.value = opts.status as OrderStatus;
    activeTab.value = opts.status as OrderStatus;
  }
}

async function onLoad() {
  await store.fetchList({ role: 'patient' });
}

async function onSwitchTab(status: OrderStatus | 'all') {
  activeTab.value = status;
  await store.fetchList({ role: 'patient' });
}

function onDetail(o: Order) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/admin/orders/detail?id=${o.id}` });
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(async () => {
  parseQuery();
  await onLoad();
});
</script>

<template>
  <view class="ui-page" data-testid="admin-orders-list">
    <!-- tab 过滤 -->
    <view class="admin-orders-list__tabs" data-testid="orders-list-tabs">
      <view
        v-for="t in STATUS_TABS"
        :key="t.status"
        class="admin-orders-list__tab"
        :class="{ 'admin-orders-list__tab--active': activeTab === t.status }"
        :data-testid="`orders-list-tab-${t.status}`"
        @click="onSwitchTab(t.status)"
      >
        {{ t.label }}
      </view>
    </view>

    <view v-if="store.loading && store.orders.length === 0" data-testid="orders-list-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="store.error && store.orders.length === 0" data-testid="orders-list-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
        <template #action>
          <text class="admin-orders-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="store.orders.length === 0" data-testid="orders-list-empty">
      <UiEmpty icon="📋" title="暂无订单" />
    </view>

    <view v-else data-testid="orders-list">
      <UiCard
        v-for="o in store.orders"
        :key="o.id"
        :data-testid="`orders-list-card-${o.id}`"
        @click="onDetail(o)"
      >
        <view class="admin-orders-list__row">
          <text class="admin-orders-list__order-id">订单 #{{ o.id }}</text>
          <text :class="['admin-orders-list__status', STATUS_CLASS[o.status]]" :data-testid="`orders-list-status-${o.id}`">
            {{ STATUS_LABEL[o.status] }}
          </text>
        </view>
        <text class="admin-orders-list__hospital">医院 #{{ o.hospital_id }}</text>
        <text class="admin-orders-list__time">📅 {{ formatDate(o.appointment_time) }}</text>
        <text class="admin-orders-list__address">📍 {{ o.address }}</text>
        <template #footer>
          <view class="admin-orders-list__footer">
            <text class="admin-orders-list__amount">{{ formatAmount(o.total_amount) }}</text>
            <text class="admin-orders-list__view">查看详情 →</text>
          </view>
        </template>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-orders-list__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.admin-orders-list__tab {
  flex: 1;
  min-width: 64px;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.admin-orders-list__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.admin-orders-list__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.admin-orders-list__order-id {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.admin-orders-list__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.admin-orders-list__status--pending { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }
.admin-orders-list__status--confirmed { background: var(--ui-color-primary); color: var(--ui-color-text-inverse); }
.admin-orders-list__status--active { background: var(--ui-color-success); color: var(--ui-color-text-inverse); }
.admin-orders-list__status--done { background: var(--ui-color-text-disabled); color: var(--ui-color-text-inverse); }
.admin-orders-list__status--cancel { background: var(--ui-color-error); color: var(--ui-color-text-inverse); }

.admin-orders-list__hospital,
.admin-orders-list__time,
.admin-orders-list__address {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-orders-list__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.admin-orders-list__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.admin-orders-list__view {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
}

.admin-orders-list__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>