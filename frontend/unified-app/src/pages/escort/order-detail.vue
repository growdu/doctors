<script setup lang="ts">
/**
 * escort/order-detail.vue — 任务详情（v2 unified-app · escort 域）。
 *
 * 入口：escort/orders 点击订单卡
 *   → orderStore.fetchDetail(id) 拉详情
 *   → escort 专属操作：「开始服务」（escort_confirmed → in_service）+ 「完成任务」（in_service → completed）
 *
 * 设计要点：
 *   - 复用 patient/order-detail 渲染样式
 *   - RoleGuard 限制 escort 角色
 *   - 状态机：
 *     pending_escort  → 不允许（escort 已接单才进此页）
 *     escort_confirmed → 显示「开始服务」按钮
 *     in_service       → 显示「完成任务」按钮
 *     completed/cancelled → 仅展示
 */
import { ref, onMounted } from 'vue';
import RoleGuard from '@/components/shared/RoleGuard.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { useOrderStore } from '@/store/order';
import { finishOrder, confirmAccept } from '@/api/orders';
import type { Order, OrderStatus } from '@/api/orders';

const store = useOrderStore();
const orderId = ref<number | null>(null);
const submitting = ref(false);

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending_escort: '待陪诊师接单',
  escort_confirmed: '待服务',
  in_service: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

const STATUS_CLASS: Record<OrderStatus, string> = {
  pending_escort: 'escort-order-detail__status--pending',
  escort_confirmed: 'escort-order-detail__status--confirmed',
  in_service: 'escort-order-detail__status--active',
  completed: 'escort-order-detail__status--done',
  cancelled: 'escort-order-detail__status--cancel',
};

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.id) orderId.value = Number(opts.id);
}

async function onLoad() {
  parseQuery();
  if (!orderId.value) return;
  try {
    await store.fetchDetail(orderId.value);
  } catch {
    // ignore
  }
}

function canStart(o: Order): boolean {
  return o.status === 'escort_confirmed';
}

function canFinish(o: Order): boolean {
  return o.status === 'in_service';
}

async function onStart() {
  if (!store.currentOrder) return;
  submitting.value = true;
  try {
    await confirmAccept(store.currentOrder.id);
    await store.fetchDetail(store.currentOrder.id);
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

async function onFinish() {
  if (!store.currentOrder) return;
  submitting.value = true;
  try {
    await finishOrder(store.currentOrder.id);
    await store.fetchDetail(store.currentOrder.id);
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <RoleGuard :required="['escort']" fallback-title="需要 escort 角色">
    <view class="ui-page" data-testid="escort-order-detail-page">
      <view v-if="store.loading && !store.currentOrder" data-testid="escort-order-detail-loading">
        <UiLoading text="加载中..." />
      </view>

      <view v-else-if="store.error && !store.currentOrder" data-testid="escort-order-detail-error">
        <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
          <template #action>
            <text class="escort-order-detail__retry" @click="onLoad">点击重试</text>
          </template>
        </UiEmpty>
      </view>

      <template v-else-if="store.currentOrder">
        <UiCard :title="`订单 #${store.currentOrder.id}`" data-testid="escort-order-detail-header">
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">状态</text>
            <text :class="['escort-order-detail__status', STATUS_CLASS[store.currentOrder.status]]" data-testid="escort-order-detail-status">
              {{ STATUS_LABEL[store.currentOrder.status] }}
            </text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">医院 ID</text>
            <text class="escort-order-detail__value">#{{ store.currentOrder.hospital_id }}</text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">套餐 ID</text>
            <text class="escort-order-detail__value">{{ store.currentOrder.package_id ?? '-' }}</text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">预约时间</text>
            <text class="escort-order-detail__value">{{ formatDate(store.currentOrder.appointment_time) }}</text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">服务地址</text>
            <text class="escort-order-detail__value">{{ store.currentOrder.address }}</text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">患者 ID</text>
            <text class="escort-order-detail__value">#{{ store.currentOrder.patient_id }}</text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">服务金额</text>
            <text class="escort-order-detail__value escort-order-detail__amount">
              {{ formatYuan(store.currentOrder.total_amount) }}
            </text>
          </view>
        </UiCard>

        <!-- escort 操作 -->
        <view v-if="canStart(store.currentOrder) || canFinish(store.currentOrder)" class="escort-order-detail__actions" data-testid="escort-order-detail-actions">
          <UiButton
            v-if="canStart(store.currentOrder)"
            type="primary"
            block
            :loading="submitting"
            data-testid="escort-order-detail-start"
            @click="onStart"
          >
            开始服务
          </UiButton>
          <UiButton
            v-if="canFinish(store.currentOrder)"
            type="primary"
            block
            :loading="submitting"
            data-testid="escort-order-detail-finish"
            @click="onFinish"
          >
            完成任务
          </UiButton>
        </view>
      </template>
    </view>
  </RoleGuard>
</template>

<style scoped>
.escort-order-detail__row {
  display: flex;
  align-items: center;
  margin-bottom: var(--ui-space-sm);
}

.escort-order-detail__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  width: 80px;
  flex-shrink: 0;
}

.escort-order-detail__value {
  flex: 1;
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
}

.escort-order-detail__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.escort-order-detail__status--pending { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }
.escort-order-detail__status--confirmed { background: var(--ui-color-primary); color: var(--ui-color-text-inverse); }
.escort-order-detail__status--active { background: var(--ui-color-success); color: var(--ui-color-text-inverse); }
.escort-order-detail__status--done { background: var(--ui-color-text-disabled); color: var(--ui-color-text-inverse); }
.escort-order-detail__status--cancel { background: var(--ui-color-error); color: var(--ui-color-text-inverse); }

.escort-order-detail__amount {
  color: var(--ui-color-success);
  font-weight: var(--ui-font-weight-semibold);
}

.escort-order-detail__actions {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-md);
}

.escort-order-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>