<script setup lang="ts">
/**
 * patient/refund/apply.vue — 申请退款（v2 unified-app）。
 *
 * 入口：order/detail（paid 状态后）「申请退款」
 *   → 必填：原因
 *   → 可选：退款金额（不填则全额）
 *   → 调 api/payment.refundPayment(payment_id, { amount?, reason })
 *
 * 设计要点：
 *   - 显示订单金额 + 提示全额/部分退款
 *   - 金额限制：≤ 订单金额
 *   - 原因必填，≤ 200 字
 */
import { ref, computed, onMounted } from 'vue';
import { refundPayment, getPayment } from '@/api/payment';
import UiInput from '@/components/shared/UiInput.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';

const orderId = ref<number | null>(null);
const paymentId = ref<number | null>(null);
const orderAmount = ref(0);
const refundAmount = ref<number | undefined>(undefined);
const reason = ref('');
const submitting = ref(false);
const success = ref(false);
const error = ref<string | null>(null);

const canSubmit = computed(() => {
  if (!reason.value.trim()) return false;
  if (refundAmount.value !== undefined) {
    if (refundAmount.value <= 0) return false;
    if (refundAmount.value > orderAmount.value) return false;
  }
  return true;
});

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.orderId) orderId.value = Number(opts.orderId);
  if (opts?.paymentId) paymentId.value = Number(opts.paymentId);
  if (opts?.amount) orderAmount.value = Number(opts.amount);
}

async function onLoadPayment() {
  if (!paymentId.value) return;
  try {
    const p = await getPayment(paymentId.value);
    orderAmount.value = p.amount;
  } catch (e) {
    error.value = (e as Error).message;
  }
}

async function onSubmit() {
  if (!canSubmit.value || !paymentId.value) return;
  submitting.value = true;
  error.value = null;
  try {
    await refundPayment(paymentId.value, {
      amount: refundAmount.value,
      reason: reason.value.trim(),
    });
    success.value = true;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    submitting.value = false;
  }
}

function onBack() {
  if (typeof uni !== 'undefined') {
    uni.navigateBack({ delta: 1 });
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

onMounted(() => {
  parseQuery();
  onLoadPayment();
});
</script>

<template>
  <view class="ui-page" data-testid="patient-refund-apply">
    <view v-if="success" data-testid="refund-success">
      <UiEmpty icon="✅" title="退款申请已提交" description="客服会在 1-3 个工作日内审核">
        <template #action>
          <UiButton type="primary" size="sm" data-testid="refund-back-btn" @click="onBack">返回订单详情</UiButton>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <UiCard title="退款信息">
        <view class="refund-apply__row">
          <text class="refund-apply__label">订单金额</text>
          <text class="refund-apply__value" data-testid="refund-order-amount">{{ formatYuan(orderAmount) }}</text>
        </view>
        <view class="refund-apply__row">
          <text class="refund-apply__label">退款类型</text>
          <text class="refund-apply__value">不填金额 = 全额退款</text>
        </view>

        <UiInput
          v-model="refundAmount"
          label="退款金额（分）"
          placeholder="留空 = 全额"
          type="number"
          :maxlength="10"
          data-testid="refund-amount"
        />

        <UiInput
          v-model="reason"
          label="退款原因"
          placeholder="请描述退款原因"
          type="textarea"
          :maxlength="200"
          data-testid="refund-reason"
        />

        <view v-if="error" class="refund-apply__error" data-testid="refund-error">{{ error }}</view>
      </UiCard>

      <view class="refund-apply__actions">
        <UiButton
          type="primary"
          block
          :loading="submitting"
          :disabled="!canSubmit"
          data-testid="refund-submit-btn"
          @click="onSubmit"
        >
          提交退款申请
        </UiButton>
      </view>
    </template>
  </view>
</template>

<style scoped>
.refund-apply__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--ui-space-sm) 0;
  border-bottom: 1px solid var(--ui-color-divider);
}

.refund-apply__row:last-of-type {
  border-bottom: none;
}

.refund-apply__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.refund-apply__value {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
}

.refund-apply__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-top: var(--ui-space-md);
}

.refund-apply__actions {
  margin-top: var(--ui-space-base);
}
</style>