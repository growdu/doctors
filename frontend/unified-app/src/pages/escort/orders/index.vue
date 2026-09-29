<script setup lang="ts">
/**
 * escort/orders/index.vue — 我的任务（v2 unified-app · escort 域）。
 *
 * 入口：escort home「我的任务」卡片
 *   → orderStore.fetchList({ role: 'escort' }) 拉 escort 已接订单
 *   → 4 状态 tab（待服务 / 服务中 / 已完成 / 已取消）
 *   → 点击订单卡跳 escort/order-detail?id=
 *
 * 设计要点：
 *   - 复用 patient/orders/list 渲染样式 + STATUS_CLASS 颜色映射
 *   - RoleGuard 限制 escort 角色
 */
import { ref, onMounted } from 'vue';
import RoleGuard from '@/components/shared/RoleGuard.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { useOrderStore } from '@/store/order';
import type { Order, OrderStatus } from '@/api/orders';

const store = useOrderStore();

const STATUS_TABS: Array<{ status: OrderStatus | 'all'; label: string }> = [
  { status: 'all', label: '全部' },
  { status: 'escort_confirmed', label: '待服务' },
  { status: 'in_service', label: '服务中' },
  { status: 'completed', label: '已完成' },
  { status: 'cancelled', label: '已取消' },
];

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending_escort: '待陪诊师接单',
  escort_confirmed: '待服务',
  in_service: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

const STATUS_CLASS: Record<OrderStatus, string> = {
  pending_escort: 'escort-orders__status--pending',
  escort_confirmed: 'escort-orders__status--confirmed',
  in_service: 'escort-orders__status--active',
  completed: 'escort-orders__status--done',
  cancelled: 'escort-orders__status--cancel',
};

const activeTab = ref<OrderStatus | 'all'>('all');

async function onLoad() {
  await store.fetchList({ role: 'escort' });
}

async function onSwitchTab(status: OrderStatus | 'all') {
  activeTab.value = status;
  await store.fetchList({ role: 'escort' });
}

function onDetail(o: Order) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/escort/order-detail?id=${o.id}` });
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <RoleGuard :required="['escort']" fallback-title="需要 escort 角色">
    <view class="ui-page" data-testid="escort-orders-page">
      <!-- tab 过滤 -->
      <view class="escort-orders__tabs" data-testid="escort-orders-tabs">
        <view
          v-for="t in STATUS_TABS"
          :key="t.status"
          class="escort-orders__tab"
          :class="{ 'escort-orders__tab--active': activeTab === t.status }"
          :data-testid="`escort-orders-tab-${t.status}`"
          @click="onSwitchTab(t.status)"
        >
          {{ t.label }}
        </view>
      </view>

      <view v-if="store.loading && store.orders.length === 0" data-testid="escort-orders-loading">
        <UiLoading text="加载中..." />
      </view>

      <view v-else-if="store.error && store.orders.length === 0" data-testid="escort-orders-error">
        <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
          <template #action>
            <text class="escort-orders__retry" @click="onLoad">点击重试</text>
          </template>
        </UiEmpty>
      </view>

      <view v-else-if="store.orders.length === 0" data-testid="escort-orders-empty">
        <UiEmpty icon="📋" title="暂无任务" />
      </view>

      <view v-else data-testid="escort-orders-list">
        <UiCard
          v-for="o in store.orders"
          :key="o.id"
          :data-testid="`escort-orders-card-${o.id}`"
          @click="onDetail(o)"
        >
          <view class="escort-orders__row">
            <text class="escort-orders__order-id">订单 #{{ o.id }}</text>
            <text :class="['escort-orders__status', STATUS_CLASS[o.status]]" :data-testid="`escort-orders-status-${o.id}`">
              {{ STATUS_LABEL[o.status] }}
            </text>
          </view>
          <text class="escort-orders__hospital">🏥 医院 #{{ o.hospital_id }}</text>
          <text class="escort-orders__time">📅 {{ formatDate(o.appointment_time) }}</text>
          <text class="escort-orders__address">📍 {{ o.address }}</text>
          <template #footer>
            <view class="escort-orders__footer">
              <text class="escort-orders__amount">{{ formatAmount(o.total_amount) }}</text>
              <text class="escort-orders__view">查看详情 →</text>
            </view>
          </template>
        </UiCard>
      </view>
    </view>
  </RoleGuard>
</template>

<style scoped>
.escort-orders__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.escort-orders__tab {
  flex: 1;
  min-width: 64px;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.escort-orders__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.escort-orders__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.escort-orders__order-id {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.escort-orders__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.escort-orders__status--pending { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }
.escort-orders__status--confirmed { background: var(--ui-color-primary); color: var(--ui-color-text-inverse); }
.escort-orders__status--active { background: var(--ui-color-success); color: var(--ui-color-text-inverse); }
.escort-orders__status--done { background: var(--ui-color-text-disabled); color: var(--ui-color-text-inverse); }
.escort-orders__status--cancel { background: var(--ui-color-error); color: var(--ui-color-text-inverse); }

.escort-orders__hospital,
.escort-orders__time,
.escort-orders__address {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-orders__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.escort-orders__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.escort-orders__view {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
}

.escort-orders__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>