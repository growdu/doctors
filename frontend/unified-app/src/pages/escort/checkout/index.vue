<script setup lang="ts">
/**
 * escort/checkout/index.vue — 订单签出 / 服务完成小结（v2 unified-app · escort 域）。
 *
 * 入口：escort/order-detail「完成任务」之前（in_service → 完成）
 *   → 填写备注（服务小结，可选）
 *   → 「完成签出」按钮 → 调 finishOrder + navigateBack
 *
 * 设计要点：
 *   - 复用 orderService.finishOrder（escort-only，in_service → completed）
 *   - 备注目前只是本地记录；v2 后端不接收（后端 finishOrder 无 body）
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { useOrderStore } from '@/store/order';
import { finishOrder } from '@/api/orders';

const orderId = ref<number | null>(null);
const note = ref('');
const submitting = ref(false);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.orderId) orderId.value = Number(opts.orderId);
}

async function onCheckout() {
  if (!orderId.value) return;
  submitting.value = true;
  try {
    await finishOrder(orderId.value);
    // 同步刷新订单 store
    const store = useOrderStore();
    await store.fetchDetail(orderId.value);
    if (typeof uni !== 'undefined') {
      uni.showToast({ title: '已完成', icon: 'success' });
      setTimeout(() => uni.navigateBack(), 800);
    }
  } catch (e) {
    if (typeof uni !== 'undefined') uni.showToast({ title: `完成失败：${(e as Error).message}`, icon: 'none' });
  } finally {
    submitting.value = false;
  }
}

onMounted(parseQuery);
</script>

<template>
  <view class="ui-page" data-testid="escort-checkout-page">
    <UiCard :title="`订单 #${orderId ?? '—'} 签出`" data-testid="escort-checkout-header">
      <view v-if="!orderId" data-testid="escort-checkout-loading">
        <UiLoading text="加载订单 ID..." />
      </view>

      <UiInput
        v-model="note"
        label="服务小结（可选）"
        placeholder="例：陪同完成检查，患者已回家"
        type="textarea"
        :maxlength="200"
        data-testid="escort-checkout-note"
      />
    </UiCard>

    <view class="escort-checkout__actions">
      <UiButton
        type="primary"
        block
        :loading="submitting"
        :disabled="!orderId"
        data-testid="escort-checkout-submit"
        @click="onCheckout"
      >
        完成签出
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.escort-checkout__actions {
  margin-top: var(--ui-space-md);
}
</style>