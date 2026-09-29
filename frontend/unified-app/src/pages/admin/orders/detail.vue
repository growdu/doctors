<script setup lang="ts">
/**
 * admin/orders/detail.vue — admin 订单详情（v2 unified-app · admin 域）。
 *
 * 入口：admin/orders/list 点击「查看详情」
 *   → 拉 orderStore.fetchDetail(id) + 显示完整信息
 *   → admin 专属操作：「强制取消订单」+ 「查看陪诊师」
 *
 * 设计要点：
 *   - 复用 patient/orderDetail 渲染 + 添加管理操作按钮
 *   - 强制取消 UiModal 确认（输入原因）
 */
import { ref, onMounted } from 'vue';
import { useOrderStore } from '@/store/order';
import { forceCancelOrder } from '@/api/admin';
import type { Order, OrderStatus } from '@/api/orders';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const store = useOrderStore();
const orderId = ref<number | null>(null);
const showCancelModal = ref(false);
const cancelReason = ref('');
const cancelling = ref(false);

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
  const opts = pages[pages.length - 1]?.options;
  if (opts?.id) orderId.value = Number(opts.id);
}

async function onLoad() {
  parseQuery();
  if (!orderId.value) return;
  try {
    await store.fetchDetail(orderId.value);
  } catch (e) {
    // ignore
  }
}

function onAskCancel() {
  showCancelModal.value = true;
}

async function onConfirmCancel() {
  if (!store.currentOrder || !cancelReason.value.trim()) return;
  cancelling.value = true;
  try {
    await forceCancelOrder(store.currentOrder.id, cancelReason.value.trim());
    await store.fetchDetail(store.currentOrder.id);
    cancelReason.value = '';
    showCancelModal.value = false;
  } catch (e) {
    // store.error 已记录
  } finally {
    cancelling.value = false;
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

function canForceCancel(o: Order): boolean {
  return o.status === 'pending_escort' || o.status === 'escort_confirmed' || o.status === 'in_service';
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-order-detail">
    <view v-if="store.loading && !store.currentOrder" data-testid="admin-order-detail-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="store.error && !store.currentOrder" data-testid="admin-order-detail-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
        <template #action>
          <text class="admin-order-detail__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else-if="store.currentOrder">
      <UiCard :title="`订单 #${store.currentOrder.id}`" data-testid="admin-order-detail-header">
        <view class="admin-order-detail__status-bar">
          <text class="admin-order-detail__status-label">订单状态</text>
          <text class="admin-order-detail__status-value">{{ STATUS_LABEL[store.currentOrder.status] }}</text>
        </view>
        <view class="admin-order-detail__info">
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">医院 ID</text>
            <text class="admin-order-detail__value">{{ store.currentOrder.hospital_id }}</text>
          </view>
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">套餐 ID</text>
            <text class="admin-order-detail__value">{{ store.currentOrder.package_id ?? '-' }}</text>
          </view>
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">预约时间</text>
            <text class="admin-order-detail__value">{{ formatDate(store.currentOrder.appointment_time) }}</text>
          </view>
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">服务地址</text>
            <text class="admin-order-detail__value">{{ store.currentOrder.address }}</text>
          </view>
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">患者 ID</text>
            <text class="admin-order-detail__value">#{{ store.currentOrder.patient_id }}</text>
          </view>
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">陪诊师 ID</text>
            <text class="admin-order-detail__value">#{{ store.currentOrder.escort_id ?? '未分配' }}</text>
          </view>
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">金额</text>
            <text class="admin-order-detail__value admin-order-detail__value--amount">
              {{ formatYuan(store.currentOrder.total_amount) }}
            </text>
          </view>
          <view class="admin-order-detail__row">
            <text class="admin-order-detail__label">下单时间</text>
            <text class="admin-order-detail__value">{{ formatDate(store.currentOrder.created_at) }}</text>
          </view>
        </view>
      </UiCard>

      <!-- admin 专属操作 -->
      <view v-if="canForceCancel(store.currentOrder)" class="admin-order-detail__actions" data-testid="admin-order-detail-actions">
        <UiButton
          type="danger"
          block
          data-testid="admin-order-detail-force-cancel-btn"
          @click="onAskCancel"
        >
          强制取消订单
        </UiButton>
      </view>
    </template>

    <!-- 强制取消弹层 -->
    <UiModal
      :visible="showCancelModal"
      title="强制取消订单"
      content=""
      @update:visible="(v: boolean) => !v && (showCancelModal = false)"
    >
      <view data-testid="admin-order-cancel-modal">
        <UiInput
          v-model="cancelReason"
          label="取消原因"
          placeholder="请输入强制取消的原因"
          type="textarea"
          :maxlength="200"
          data-testid="admin-order-cancel-reason"
        />
        <view style="display: flex; gap: 12px; margin-top: 12px;">
          <UiButton type="default" block @click="showCancelModal = false" data-testid="admin-order-cancel-back">返回</UiButton>
          <UiButton
            type="danger"
            block
            :loading="cancelling"
            :disabled="!cancelReason.trim()"
            data-testid="admin-order-cancel-confirm"
            @click="onConfirmCancel"
          >
            确认强制取消
          </UiButton>
        </view>
      </view>
    </UiModal>
  </view>
</template>

<style scoped>
.admin-order-detail__status-bar {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
  padding: var(--ui-space-md);
  background: var(--ui-color-bg-hover);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-base);
}

.admin-order-detail__status-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.admin-order-detail__status-value {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-primary);
}

.admin-order-detail__info {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.admin-order-detail__row {
  display: flex;
  align-items: flex-start;
  gap: var(--ui-space-md);
}

.admin-order-detail__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  flex-shrink: 0;
  width: 80px;
}

.admin-order-detail__value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  flex: 1;
}

.admin-order-detail__value--amount {
  color: var(--ui-color-error);
  font-weight: var(--ui-font-weight-semibold);
  font-size: var(--ui-font-lg);
}

.admin-order-detail__actions {
  margin-top: var(--ui-space-base);
}

.admin-order-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>