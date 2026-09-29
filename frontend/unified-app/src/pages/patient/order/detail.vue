<script setup lang="ts">
/**
 * patient/order/detail.vue — 订单详情（v2 unified-app · patient 域）。
 *
 * 入口：order/list 点击「查看详情」→ 本页（?id=xxx）
 *   → 调 orderStore.fetchDetail(id) → 渲染订单完整信息
 *   → 状态机操作按钮（仅 patient 可见）：
 *     - pending_escort / escort_confirmed → 「取消订单」
 *
 * 设计要点：
 *   - 状态机高亮（5 状态色块）
 *   - 关键时间线：创建 / 预约 / 完成
 */
import { onMounted, ref } from 'vue';
import { useOrderStore } from '@/store/order';
import type { Order, OrderStatus } from '@/api/orders';
import UiCard from '@/components/shared/UiCard.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiButton from '@/components/shared/UiButton.vue';

const store = useOrderStore();
const orderId = ref<number | null>(null);

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending_escort: '待陪诊师接单',
  escort_confirmed: '已确认',
  in_service: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const current = pages[pages.length - 1];
  const opts = current?.options;
  if (opts?.id) {
    orderId.value = Number(opts.id);
  }
}

async function onLoad() {
  parseQuery();
  if (!orderId.value) return;
  try {
    await store.fetchDetail(orderId.value);
  } catch (e) {
    // store.error 已记录
  }
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function canCancel(o: Order): boolean {
  return o.status === 'pending_escort' || o.status === 'escort_confirmed';
}

async function onCancel() {
  if (!store.currentOrder) return;
  await store.cancel(store.currentOrder.id, '用户取消');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-order-detail">
    <view v-if="store.loading && !store.currentOrder" data-testid="order-detail-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="store.error && !store.currentOrder" data-testid="order-detail-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
        <template #action>
          <text class="order-detail__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else-if="store.currentOrder">
      <UiCard :title="`订单 #${store.currentOrder.id}`" data-testid="order-detail-header">
        <view class="order-detail__status-bar">
          <text class="order-detail__status-label">订单状态</text>
          <text class="order-detail__status-value">{{ STATUS_LABEL[store.currentOrder.status] }}</text>
        </view>
        <view class="order-detail__info">
          <view class="order-detail__row">
            <text class="order-detail__label">医院 ID</text>
            <text class="order-detail__value">{{ store.currentOrder.hospital_id }}</text>
          </view>
          <view v-if="store.currentOrder.package_id" class="order-detail__row">
            <text class="order-detail__label">套餐 ID</text>
            <text class="order-detail__value">{{ store.currentOrder.package_id }}</text>
          </view>
          <view class="order-detail__row">
            <text class="order-detail__label">预约时间</text>
            <text class="order-detail__value">{{ formatDate(store.currentOrder.appointment_time) }}</text>
          </view>
          <view class="order-detail__row">
            <text class="order-detail__label">地址</text>
            <text class="order-detail__value">{{ store.currentOrder.address }}</text>
          </view>
          <view v-if="store.currentOrder.escort_id" class="order-detail__row">
            <text class="order-detail__label">陪诊师</text>
            <text class="order-detail__value">#{{ store.currentOrder.escort_id }}</text>
          </view>
          <view class="order-detail__row">
            <text class="order-detail__label">金额</text>
            <text class="order-detail__value order-detail__value--amount">
              {{ formatAmount(store.currentOrder.total_amount) }}
            </text>
          </view>
          <view class="order-detail__row">
            <text class="order-detail__label">下单时间</text>
            <text class="order-detail__value">{{ formatDate(store.currentOrder.created_at) }}</text>
          </view>
        </view>
      </UiCard>

      <view v-if="canCancel(store.currentOrder)" class="order-detail__actions" data-testid="order-detail-actions">
        <UiButton
          type="danger"
          block
          :loading="store.loading"
          data-testid="order-cancel-btn"
          @click="onCancel"
        >
          取消订单
        </UiButton>
      </view>
    </template>
  </view>
</template>

<style scoped>
.order-detail__status-bar {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
  padding: var(--ui-space-md);
  background: var(--ui-color-bg-hover);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-base);
}

.order-detail__status-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.order-detail__status-value {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-primary);
}

.order-detail__info {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.order-detail__row {
  display: flex;
  align-items: flex-start;
  gap: var(--ui-space-md);
}

.order-detail__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  flex-shrink: 0;
  width: 80px;
}

.order-detail__value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  flex: 1;
}

.order-detail__value--amount {
  color: var(--ui-color-error);
  font-weight: var(--ui-font-weight-semibold);
  font-size: var(--ui-font-lg);
}

.order-detail__actions {
  margin-top: var(--ui-space-base);
}

.order-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>