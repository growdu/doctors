<script setup lang="ts">
/**
 * escort/invitations/index.vue — 我的邀请（v2.1 选人模式 · escort 域）。
 *
 * §48.S v2.1 重写：之前是 v1 抢单池（fetchFeed lobby）；
 * 现在是 v2.1 选人模式邀请列表（GET /api/v1/orders?role=escort&status=invitations）。
 *
 * 入口：escort home「我的邀请」卡片
 *   → orderStore.list 拉邀请订单列表（status='escort_pending_acceptance'）
 *   → 按 escort_pending_expire_at ASC 排序（最近到期的最优先）
 *   → 每个 item 显示订单 ID / 医院 / 预约时间 / 倒计时 / 接单 / 拒接
 *
 * 设计要点：
 *   - RoleGuard 限制 escort 角色
 *   - 每行嵌入 CountdownBadge（与 order-detail 一致）
 *   - 拒接弹 UiModal 收原因（必填）
 *   - 操作成功后从列表移除该项
 */
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import RoleGuard from '@/components/shared/RoleGuard.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { listOrders, confirmAccept, rejectAccept } from '@/api/orders';
import type { Order, OrderStatus } from '@/api/orders';

const items = ref<Order[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const showRejectModal = ref(false);
const currentOrderId = ref<number | null>(null);
const rejectReason = ref('');
const submitting = ref(false);

const now = ref(Date.now());
let tickHandle: number | null = null;

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listOrders({ role: 'escort', status: 'invitations' as OrderStatus });
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function remainingSec(o: Order): number {
  if (!o.escort_pending_expire_at) return 0;
  const expire = new Date(o.escort_pending_expire_at).getTime();
  return Math.max(0, Math.floor((expire - now.value) / 1000));
}

function countdownDisplay(o: Order): string {
  const s = remainingSec(o);
  if (s <= 0) return '已超时';
  return `剩余 ${s} 秒`;
}

function countdownUrgent(o: Order): boolean {
  return remainingSec(o) > 0 && remainingSec(o) < 60;
}

const hasAny = computed(() => items.value.length > 0);
const allExpired = computed(() => items.value.every((o) => remainingSec(o) <= 0));

async function onAccept(o: Order) {
  submitting.value = true;
  try {
    await confirmAccept(o.id);
    items.value = items.value.filter((x) => x.id !== o.id);
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

function onAskReject(o: Order) {
  currentOrderId.value = o.id;
  rejectReason.value = '';
  showRejectModal.value = true;
}

async function onConfirmReject() {
  if (!currentOrderId.value || !rejectReason.value.trim()) return;
  submitting.value = true;
  try {
    await rejectAccept(currentOrderId.value, rejectReason.value.trim());
    items.value = items.value.filter((x) => x.id !== currentOrderId.value!);
    showRejectModal.value = false;
    rejectReason.value = '';
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

function onCancelReject() {
  showRejectModal.value = false;
  rejectReason.value = '';
  currentOrderId.value = null;
}

/** 点击 card body（不点按钮）跳详情 */
function onCardClick(o: Order) {
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: `/pages/escort/order-detail?id=${o.id}` });
  }
}

function formatDate(iso?: string): string {
  if (!iso) return '—';
  const [date, time] = iso.split('T');
  return date + ' ' + (time?.substring(0, 5) || '');
}

onMounted(() => {
  onLoad();
  tickHandle = window.setInterval(() => { now.value = Date.now(); }, 1000);
});

onBeforeUnmount(() => {
  if (tickHandle !== null) window.clearInterval(tickHandle);
});
</script>

<template>
  <RoleGuard :required="['escort']" fallback-title="需要 escort 角色">
    <view class="ui-page" data-testid="escort-invitations-page">
      <view class="escort-invitations__header">
        <text class="escort-invitations__title">📩 我的邀请</text>
        <text class="escort-invitations__subtitle">v2.1 选人模式 · 30s 倒计时确认</text>
      </view>

      <view v-if="loading" data-testid="escort-invitations-loading">
        <UiLoading text="加载中..." />
      </view>

      <view v-else-if="error" data-testid="escort-invitations-error">
        <UiEmpty icon="⚠️" title="加载失败" :description="error">
          <template #action>
            <text class="escort-invitations__retry" @click="onLoad">点击重试</text>
          </template>
        </UiEmpty>
      </view>

      <view v-else-if="!hasAny" data-testid="escort-invitations-empty">
        <UiEmpty icon="📭" title="暂无邀请" description="患者选择您后订单会出现在此" />
      </view>

      <view v-else-if="allExpired" data-testid="escort-invitations-all-expired">
        <UiEmpty icon="⏰" title="邀请已全部超时" description="请等待新邀请或刷新">
          <template #action>
            <text class="escort-invitations__retry" @click="onLoad">刷新</text>
          </template>
        </UiEmpty>
      </view>

      <view v-else data-testid="escort-invitations-list">
        <UiCard
          v-for="o in items.filter((x) => remainingSec(x) > 0)"
          :key="o.id"
          :data-testid="`escort-invitations-card-${o.id}`"
          @click="onCardClick(o)"
        >
          <view class="escort-invitations__row">
            <text class="escort-invitations__order-id">订单 #{{ o.id }}</text>
            <text
              :class="['escort-invitations__countdown', countdownUrgent(o) ? 'escort-invitations__countdown--urgent' : 'escort-invitations__countdown--normal']"
              :data-testid="`escort-invitations-countdown-${o.id}`"
            >
              ⏱ {{ countdownDisplay(o) }}
            </text>
          </view>
          <text class="escort-invitations__hospital">🏥 医院 #{{ o.hospital_id }}</text>
          <text class="escort-invitations__time">📅 预约 {{ formatDate(o.service_start_at) }}</text>
          <text class="escort-invitations__amount">💰 服务金额 ¥{{ (o.amount / 100).toFixed(2) }}</text>
          <view class="escort-invitations__actions">
            <UiButton
              type="primary"
              size="sm"
              :data-testid="`escort-invitations-accept-${o.id}`"
              @click.stop="onAccept(o)"
            >
              接受邀请
            </UiButton>
            <UiButton
              type="danger"
              size="sm"
              :data-testid="`escort-invitations-reject-${o.id}`"
              @click.stop="onAskReject(o)"
            >
              拒绝邀请
            </UiButton>
          </view>
        </UiCard>
      </view>

      <!-- 拒接 Modal -->
      <UiModal
        :visible="showRejectModal"
        title="拒绝邀请"
        content=""
        @update:visible="(v: boolean) => !v && onCancelReject()"
      >
        <view data-testid="escort-invitations-reject-modal">
          <UiInput
            v-model="rejectReason"
            label="拒接原因"
            placeholder="例：已接其他单"
            type="textarea"
            :maxlength="200"
            data-testid="escort-invitations-reject-reason"
          />
          <view class="escort-invitations__modal-actions">
            <UiButton type="default" block data-testid="escort-invitations-reject-cancel" @click="onCancelReject">返回</UiButton>
            <UiButton
              type="danger"
              block
              :loading="submitting"
              :disabled="!rejectReason.trim()"
              data-testid="escort-invitations-reject-confirm"
              @click="onConfirmReject"
            >
              确认拒接
            </UiButton>
          </view>
        </view>
      </UiModal>
    </view>
  </RoleGuard>
</template>

<script lang="ts">
// §48.S: 占位（保留 script 块以备后续扩展；原 useRouter 已被 uni.navigateTo 替代）
</script>

<style scoped>
.escort-invitations__header {
  padding: var(--ui-space-base) 0;
}

.escort-invitations__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-invitations__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-invitations__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.escort-invitations__order-id {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.escort-invitations__countdown {
  font-size: var(--ui-font-sm);
  font-weight: var(--ui-font-weight-semibold);
  padding: 4px 12px;
  border-radius: var(--ui-radius-md);
  font-variant-numeric: tabular-nums;
}
.escort-invitations__countdown--normal { background: #e6f7ff; color: #1677ff; }
.escort-invitations__countdown--urgent { background: #fff1f0; color: #f5222d; }

.escort-invitations__hospital,
.escort-invitations__time,
.escort-invitations__amount {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-invitations__actions {
  display: flex;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-sm);
}

.escort-invitations__modal-actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.escort-invitations__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>
