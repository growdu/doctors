<script setup lang="ts">
/**
 * patient/order/pay.vue — 订单支付（v2 unified-app）。
 *
 * 入口：order/detail（escort_confirmed 状态后）「去支付」
 *   → 调 api/payment.createPayment({ order_id, channel })
 *   → 调 api/payment.completePayment(paymentId)（mock channel 同步）
 *   → 调 orderStore.fetchDetail + showToast「支付成功」
 *
 * 设计要点：
 *   - 单 channel：mock（dev 环境）
 *   - 支付状态机：create → loading → complete → success / fail
 */
import { ref, onMounted } from 'vue';
import { createPayment, completePayment } from '@/api/payment';
import type { Payment } from '@/api/payment';
import { useOrderStore } from '@/store/order';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const store = useOrderStore();
const orderId = ref<number | null>(null);
const payment = ref<Payment | null>(null);
const phase = ref<'idle' | 'creating' | 'completing' | 'success' | 'failed'>('idle');
const error = ref<string | null>(null);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.orderId) orderId.value = Number(opts.orderId);
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

async function onCreatePayment() {
  if (!orderId.value) return;
  phase.value = 'creating';
  error.value = null;
  try {
    payment.value = await createPayment({ order_id: orderId.value, channel: 'mock' });
    phase.value = 'completing';
    payment.value = await completePayment(payment.value.id);
    phase.value = 'success';
    await store.fetchDetail(orderId.value);
    if (typeof uni !== 'undefined') {
      uni.showToast({ title: '支付成功', icon: 'success' });
    }
  } catch (e) {
    error.value = (e as Error).message;
    phase.value = 'failed';
  }
}

async function onRetry() {
  phase.value = 'idle';
  await onCreatePayment();
}

function onBackToOrder() {
  if (typeof uni !== 'undefined') {
    uni.navigateBack({ delta: 1 });
  }
}

onMounted(() => {
  parseQuery();
});
</script>

<template>
  <view class="ui-page" data-testid="patient-order-pay">
    <view v-if="phase === 'idle' || phase === 'creating' || phase === 'completing'" data-testid="pay-loading">
      <UiLoading :text="phase === 'creating' ? '创建支付单…' : phase === 'completing' ? '完成支付…' : '准备支付…'" />
    </view>

    <view v-else-if="phase === 'failed' && error" data-testid="pay-error">
      <UiEmpty icon="⚠️" title="支付失败" :description="error">
        <template #action>
          <UiButton type="primary" size="sm" data-testid="pay-retry-btn" @click="onRetry">重试</UiButton>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="phase === 'success' && payment" data-testid="pay-success">
      <UiCard>
        <view class="pay-success">
          <text class="pay-success__icon">✅</text>
          <text class="pay-success__title">支付成功</text>
          <text class="pay-success__amount">{{ formatAmount(payment.amount) }}</text>
          <text class="pay-success__meta">支付单 #{{ payment.id }}</text>
          <text class="pay-success__meta">渠道：{{ payment.channel }}</text>
          <text v-if="payment.paid_at" class="pay-success__meta">{{ payment.paid_at.replace('T', ' ').substring(0, 19) }}</text>
        </view>
      </UiCard>
      <view class="pay-success__actions">
        <UiButton type="primary" block data-testid="pay-back-btn" @click="onBackToOrder">返回订单详情</UiButton>
      </view>
    </view>
  </view>
</template>

<style scoped>
.pay-success {
  text-align: center;
  padding: var(--ui-space-xl) var(--ui-space-base);
}

.pay-success__icon {
  font-size: 56px;
  display: block;
  margin-bottom: var(--ui-space-md);
}

.pay-success__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-success);
  display: block;
  margin-bottom: var(--ui-space-md);
}

.pay-success__amount {
  font-size: 36px;
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-text-primary);
  display: block;
  margin-bottom: var(--ui-space-base);
}

.pay-success__meta {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.pay-success__actions {
  margin-top: var(--ui-space-lg);
}
</style>