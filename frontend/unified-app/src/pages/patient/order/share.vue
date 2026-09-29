<script setup lang="ts">
/**
 * patient/order/share.vue — 订单分享页（mobile batch 4c）。
 *
 * 入口：order/detail 点「分享给朋友」→ 本页（?id=xxx）
 *
 * 跨端行为：
 *   - h5 / app-plus：点击「分享」按钮 → shareContent() 调起系统分享面板
 *   - mp-weixin：页面级 onShareAppMessage 钩子（uni-app 自动从右上角菜单触发）
 *
 * 设计要点：
 *   - 复用 orderStore 单订单数据
 *   - 显示订单摘要（编号/金额/状态）让分享时一目了然
 *   - share.ts 内部已处理 navigator.share 失败回退 clipboard.writeText
 */
import { onMounted, ref } from 'vue';
import { onShareAppMessage } from '@dcloudio/uni-app';
import { useOrderStore } from '@/store/order';
import type { Order } from '@/api/orders';
import { shareContent } from '@/utils/share';
import { buildMpShareMessage } from '@/utils/share.mp-weixin';
import UiCard from '@/components/shared/UiCard.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiButton from '@/components/shared/UiButton.vue';

const store = useOrderStore();
const orderId = ref<number | null>(null);
const sharing = ref(false);
const lastResult = ref<{ channel?: string; errMsg?: string } | null>(null);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const current = pages[pages.length - 1];
  const opts = current?.options;
  if (opts?.id) orderId.value = Number(opts.id);
}

async function onLoad() {
  parseQuery();
  if (!orderId.value) return;
  try {
    await store.fetchDetail(orderId.value);
  } catch {
    /* store.error 已记录 */
  }
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function buildShareOptions(o: Order) {
  return {
    title: `陪诊订单 #${o.id}`,
    desc: `预约时间 ${o.appointment_time.split('T')[0]} · 金额 ${formatAmount(o.total_amount)}`,
    href: `/pages/patient/order/detail?id=${o.id}`,
  };
}

async function handleShare() {
  const order = store.currentOrder;
  if (!order) return;
  sharing.value = true;
  lastResult.value = null;
  try {
    const result = await shareContent(buildShareOptions(order));
    if (result.ok) {
      lastResult.value = { channel: result.channel };
    } else {
      lastResult.value = null;
    }
    if (typeof uni !== 'undefined') {
      const title = result.ok
        ? (result.channel === 'clipboard' ? '链接已复制' : '已分享')
        : (result.errMsg === 'user-cancelled' ? '已取消' : '分享失败');
      uni.showToast({ title });
    }
  } finally {
    sharing.value = false;
  }
}

// mp-weixin 页面级分享钩子：从右上角菜单触发，无需用户操作按钮
onShareAppMessage(() => {
  const order = store.currentOrder;
  if (!order) return { title: 'Doctors' };
  return buildMpShareMessage(buildShareOptions(order));
});

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-order-share">
    <view v-if="store.loading && !store.currentOrder" data-testid="order-share-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="store.error && !store.currentOrder" data-testid="order-share-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
        <template #action>
          <text class="order-share__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else-if="store.currentOrder">
      <UiCard :title="`订单 #${store.currentOrder.id}`" data-testid="order-share-header">
        <view class="order-share__summary">
          <view class="order-share__row">
            <text class="order-share__label">金额</text>
            <text class="order-share__amount">{{ formatAmount(store.currentOrder.total_amount) }}</text>
          </view>
          <view class="order-share__row">
            <text class="order-share__label">预约</text>
            <text class="order-share__value">{{ store.currentOrder.appointment_time.split('T')[0] }}</text>
          </view>
          <view class="order-share__row">
            <text class="order-share__label">状态</text>
            <text class="order-share__value">{{ store.currentOrder.status }}</text>
          </view>
        </view>
      </UiCard>

      <view class="order-share__actions" data-testid="order-share-actions">
        <UiButton
          type="primary"
          block
          :loading="sharing"
          data-testid="order-share-btn"
          @click="handleShare"
        >
          分享给朋友
        </UiButton>
      </view>

      <view v-if="lastResult" class="order-share__result" data-testid="order-share-result">
        <text class="order-share__result-text">
          {{ lastResult.channel === 'clipboard' ? '已复制链接到剪贴板' : lastResult.channel ? `分享渠道：${lastResult.channel}` : lastResult.errMsg }}
        </text>
      </view>
    </template>
  </view>
</template>

<style scoped>
.order-share__summary {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}
.order-share__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.order-share__label {
  color: var(--ui-color-text-secondary);
  font-size: var(--ui-font-sm);
}
.order-share__value {
  font-size: var(--ui-font-base);
}
.order-share__amount {
  font-size: var(--ui-font-lg);
  font-weight: 600;
  color: var(--ui-color-primary);
}
.order-share__actions {
  margin-top: var(--ui-space-lg);
}
.order-share__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
}
.order-share__result {
  margin-top: var(--ui-space-md);
  text-align: center;
}
.order-share__result-text {
  color: var(--ui-color-text-secondary);
  font-size: var(--ui-font-sm);
}
</style>