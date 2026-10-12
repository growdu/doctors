<script setup lang="ts">
/**
 * escort/order-detail.vue — 任务详情（v2 unified-app · escort 域）。
 *
 * 入口：escort/orders 点击订单卡
 *   → orderStore.fetchDetail(id) 拉详情
 *   → escort 专属操作：「开始服务」「完成任务」「接受 / 拒绝邀请」
 *
 * v2.1 倒计时（§48.R）：
 *   - 当 status='escort_pending_acceptance' 时显示 CountdownBadge
 *   - setInterval(1000) 1s tick；卸载 clearInterval 防内存泄漏
 *   - < 60s 红色 / >= 60s 蓝色
 *   - 接受按钮（confirm-accept）+ 拒绝按钮（reject-accept）
 *
 * 设计要点：
 *   - 复用 patient/order-detail 渲染样式
 *   - RoleGuard 限制 escort 角色
 *   - parseQuery 优先 window.location.search（h5 dev mode）
 */
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import RoleGuard from '@/components/shared/RoleGuard.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { useOrderStore } from '@/store/order';
import { finishOrder, confirmAccept, rejectAccept } from '@/api/orders';
import type { Order, OrderStatus } from '@/api/orders';

const store = useOrderStore();
const orderId = ref<number | null>(null);
const submitting = ref(false);
const now = ref(Date.now());       // §48.R: 1s tick driver for countdown
let tickHandle: number | null = null;

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending_escort: '待陪诊师接单',
  escort_pending_acceptance: '待您确认',
  escort_confirmed: '待服务',
  in_service: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

const STATUS_CLASS: Record<OrderStatus, string> = {
  pending_escort: 'escort-order-detail__status--pending',
  escort_pending_acceptance: 'escort-order-detail__status--acceptance',
  escort_confirmed: 'escort-order-detail__status--confirmed',
  in_service: 'escort-order-detail__status--active',
  completed: 'escort-order-detail__status--done',
  cancelled: 'escort-order-detail__status--cancel',
};

function parseQuery() {
  // §48.N: h5 dev mode getCurrentPages() 不带 options；优先读 window.location.search
  const search = typeof window !== 'undefined' ? window.location.search : '';
  const m = search.match(/[?&]id=(\d+)/);
  if (m) { orderId.value = Number(m[1]); return; }
  if (typeof uni !== 'undefined') {
    const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
    const opts = pages[pages.length - 1]?.options;
    if (opts?.id) orderId.value = Number(opts.id);
  }
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

// §48.R: CountdownBadge computed
const remainingSec = computed<number | null>(() => {
  if (!store.currentOrder?.escort_pending_expire_at) return null;
  const expire = new Date(store.currentOrder.escort_pending_expire_at).getTime();
  return Math.max(0, Math.floor((expire - now.value) / 1000));
});

const countdownDisplay = computed(() => {
  const s = remainingSec.value;
  if (s === null) return '';
  if (s <= 0) return '已超时';
  return `剩余 ${s} 秒`;
});

const countdownUrgent = computed(() => {
  const s = remainingSec.value;
  return s !== null && s < 60;
});

function canAccept(o: Order): boolean {
  return o.status === 'escort_pending_acceptance' && (remainingSec.value ?? 0) > 0;
}

function canReject(o: Order): boolean {
  return o.status === 'escort_pending_acceptance' && (remainingSec.value ?? 0) > 0;
}

function canStart(o: Order): boolean {
  return o.status === 'escort_confirmed';
}

function canFinish(o: Order): boolean {
  return o.status === 'in_service';
}

async function onAccept() {
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

async function onReject() {
  if (!store.currentOrder) return;
  submitting.value = true;
  try {
    await rejectAccept(store.currentOrder.id, '陪诊师拒绝');
    await store.fetchDetail(store.currentOrder.id);
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
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

function formatDate(iso?: string): string {
  if (!iso) return '—';
  const [date, time] = iso.split('T');
  return date + ' ' + (time?.substring(0, 5) || '');
}

function statusLabel(s?: string): string {
  return STATUS_LABEL[s as OrderStatus] || s || '未知';
}

function statusClass(s?: string): string {
  return STATUS_CLASS[s as OrderStatus] || '';
}

onMounted(() => {
  onLoad();
  // §48.R: 启动 1s tick 驱动倒计时
  tickHandle = window.setInterval(() => { now.value = Date.now(); }, 1000);
});

onBeforeUnmount(() => {
  if (tickHandle !== null) {
    window.clearInterval(tickHandle);
    tickHandle = null;
  }
});
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
            <text :class="['escort-order-detail__status', statusClass(store.currentOrder.status)]" data-testid="escort-order-detail-status">
              {{ statusLabel(store.currentOrder.status) }}
            </text>
          </view>

          <!-- §48.R: 30s 倒计时 CountdownBadge -->
          <view v-if="remainingSec !== null" class="escort-order-detail__row" data-testid="escort-order-detail-countdown-row">
            <text class="escort-order-detail__label">剩余确认</text>
            <text
              :class="['escort-order-detail__countdown', countdownUrgent ? 'escort-order-detail__countdown--urgent' : 'escort-order-detail__countdown--normal']"
              :data-testid="'escort-order-detail-countdown-' + (countdownUrgent ? 'urgent' : 'normal')"
            >
              ⏱ {{ countdownDisplay }}
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
            <text class="escort-order-detail__value">{{ formatDate(store.currentOrder.service_start_at) }}</text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">患者 ID</text>
            <text class="escort-order-detail__value">#{{ store.currentOrder.patient_id }}</text>
          </view>
          <view class="escort-order-detail__row">
            <text class="escort-order-detail__label">服务金额</text>
            <text class="escort-order-detail__value escort-order-detail__amount">
              {{ formatYuan(store.currentOrder.amount) }}
            </text>
          </view>
        </UiCard>

        <!-- escort 操作 -->
        <view v-if="canAccept(store.currentOrder) || canReject(store.currentOrder)" class="escort-order-detail__actions" data-testid="escort-order-detail-actions">
          <UiButton
            v-if="canAccept(store.currentOrder)"
            type="primary"
            block
            :loading="submitting"
            data-testid="escort-order-detail-accept"
            @click="onAccept"
          >
            接受邀请
          </UiButton>
          <UiButton
            v-if="canReject(store.currentOrder)"
            block
            :loading="submitting"
            data-testid="escort-order-detail-reject"
            @click="onReject"
          >
            拒绝邀请
          </UiButton>
        </view>
        <view v-if="canStart(store.currentOrder) || canFinish(store.currentOrder)" class="escort-order-detail__actions" data-testid="escort-order-detail-actions2">
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
  color: var(--var-color-text-secondary, var(--ui-color-text-secondary));
  width: 80px;
  flex-shrink: 0;
}

.escort-order-detail__value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  flex: 1;
}

.escort-order-detail__status {
  font-size: var(--ui-font-sm);
  font-weight: var(--ui-font-weight-medium);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
}
.escort-order-detail__status--acceptance { background: #fff7e6; color: #d46b08; }
.escort-order-detail__status--pending { background: #fff7e6; color: #d46b08; }
.escort-order-detail__status--confirmed { background: #e6f7ff; color: #1677ff; }
.escort-order-detail__status--active { background: #f6ffed; color: #52c41a; }
.escort-order-detail__status--done { background: #f5f5f5; color: #8c8c8c; }
.escort-order-detail__status--cancel { background: #fff1f0; color: #f5222d; }

/* §48.R: CountdownBadge 样式 */
.escort-order-detail__countdown {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  padding: 4px 12px;
  border-radius: var(--ui-radius-md);
  font-variant-numeric: tabular-nums;
}
.escort-order-detail__countdown--normal { background: #e6f7ff; color: #1677ff; }
.escort-order-detail__countdown--urgent { background: #fff1f0; color: #f5222d; }

.escort-order-detail__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.escort-order-detail__actions {
  margin-top: var(--ui-space-base);
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.escort-order-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>
