<script setup lang="ts">
/**
 * admin/finance/index.vue — 财务管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「总金额」/「今日 GMV」卡片
 *   → listBillings() 按 type 分类拉取（income / refund / withdraw）
 *   → 3 卡片概览（今日 / 本月 / 总金额）+ 列表展示
 *
 * 设计要点：
 *   - 简化为单页聚合：3 类型 tab + 概览卡 + 列表
 *   - 复用 admin/orders 的金额格式化（分→元）
 */
import { ref, onMounted } from 'vue';
import { listBillings } from '@/api/admin';
import type { Billing } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

type BillingType = Billing['type'];

const TYPE_TABS: Array<{ key: BillingType | 'all'; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'income', label: '收入' },
  { key: 'refund', label: '退款' },
  { key: 'withdraw', label: '提现' },
];

const TYPE_LABEL: Record<BillingType, string> = {
  income: '收入',
  refund: '退款',
  withdraw: '提现',
};

const TYPE_CLASS: Record<BillingType, string> = {
  income: 'admin-finance__type--income',
  refund: 'admin-finance__type--refund',
  withdraw: 'admin-finance__type--withdraw',
};

const items = ref<Billing[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const activeTab = ref<BillingType | 'all'>('all');

const totalIncome = ref(0);
const totalRefund = ref(0);
const totalWithdraw = ref(0);

async function loadCount(type: BillingType): Promise<number> {
  try {
    const r = await listBillings({ type, page: 1, page_size: 1 });
    return r.total;
  } catch {
    return 0;
  }
}

async function loadOverview() {
  const [i, r, w] = await Promise.all([
    loadCount('income'),
    loadCount('refund'),
    loadCount('withdraw'),
  ]);
  totalIncome.value = i;
  totalRefund.value = r;
  totalWithdraw.value = w;
}

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const query: { type?: BillingType } = {};
    if (activeTab.value !== 'all') query.type = activeTab.value;
    const r = await listBillings(query);
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onSwitchTab(t: BillingType | 'all') {
  activeTab.value = t;
  await onLoad();
}

function formatYuan(cents: number): string {
  const sign = cents < 0 ? '-' : '';
  return `${sign}¥${(Math.abs(cents) / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(async () => {
  loading.value = true;
  await Promise.all([loadOverview(), onLoad()]);
});
</script>

<template>
  <view class="ui-page" data-testid="admin-finance-page">
    <!-- 概览 -->
    <view class="admin-finance__overview" data-testid="admin-finance-overview">
      <UiCard data-testid="admin-finance-overview-income">
        <view class="admin-finance__overview-card admin-finance__overview-card--income">
          <text class="admin-finance__overview-label">收入</text>
          <text class="admin-finance__overview-value">{{ totalIncome }}</text>
        </view>
      </UiCard>
      <UiCard data-testid="admin-finance-overview-refund">
        <view class="admin-finance__overview-card admin-finance__overview-card--refund">
          <text class="admin-finance__overview-label">退款</text>
          <text class="admin-finance__overview-value">{{ totalRefund }}</text>
        </view>
      </UiCard>
      <UiCard data-testid="admin-finance-overview-withdraw">
        <view class="admin-finance__overview-card admin-finance__overview-card--withdraw">
          <text class="admin-finance__overview-label">提现</text>
          <text class="admin-finance__overview-value">{{ totalWithdraw }}</text>
        </view>
      </UiCard>
    </view>

    <!-- type 过滤 tab -->
    <view class="admin-finance__tabs" data-testid="admin-finance-tabs">
      <view
        v-for="t in TYPE_TABS"
        :key="t.key"
        class="admin-finance__tab"
        :class="{ 'admin-finance__tab--active': activeTab === t.key }"
        :data-testid="`admin-finance-tab-${t.key}`"
        @click="onSwitchTab(t.key)"
      >
        {{ t.label }}
      </view>
    </view>

    <view v-if="loading" data-testid="admin-finance-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="admin-finance-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-finance__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="admin-finance-empty">
      <UiEmpty icon="💰" title="暂无账单" />
    </view>

    <view v-else data-testid="admin-finance-list">
      <UiCard
        v-for="b in items"
        :key="b.id"
        :data-testid="`admin-finance-card-${b.id}`"
      >
        <view class="admin-finance__row">
          <text class="admin-finance__id">账单 #{{ b.id }}</text>
          <text :class="['admin-finance__type', TYPE_CLASS[b.type]]" :data-testid="`admin-finance-type-${b.id}`">
            {{ TYPE_LABEL[b.type] }}
          </text>
        </view>
        <text class="admin-finance__order">订单 #{{ b.order_id }}</text>
        <text class="admin-finance__amount">{{ formatYuan(b.amount) }}</text>
        <text class="admin-finance__time">📅 {{ formatDate(b.created_at) }}</text>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-finance__overview {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--ui-space-md);
  margin-bottom: var(--ui-space-md);
}

.admin-finance__overview-card {
  text-align: center;
}

.admin-finance__overview-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
}

.admin-finance__overview-value {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-finance__overview-card--income .admin-finance__overview-value { color: var(--ui-color-success); }
.admin-finance__overview-card--refund .admin-finance__overview-value { color: var(--ui-color-error); }
.admin-finance__overview-card--withdraw .admin-finance__overview-value { color: var(--ui-color-warning); }

.admin-finance__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.admin-finance__tab {
  flex: 1;
  min-width: 64px;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.admin-finance__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.admin-finance__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.admin-finance__id {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.admin-finance__type {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.admin-finance__type--income { background: var(--ui-color-success); color: var(--ui-color-text-inverse); }
.admin-finance__type--refund { background: var(--ui-color-error); color: var(--ui-color-text-inverse); }
.admin-finance__type--withdraw { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }

.admin-finance__order,
.admin-finance__time {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-finance__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-finance__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>