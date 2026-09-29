<script setup lang="ts">
/**
 * escort/invitations/index.vue — 抢单池（v2 unified-app · escort 域）。
 *
 * 入口：escort home「抢单池」卡片
 *   → fetchFeed() 拉候选订单列表（match-service，按 score + distance 排序）
 *   → 每个 item 显示医院 ID / 预约时间 / 剩余席位 / 接单数 / 截止时间
 *   → 行内「接单」/「拒单」操作
 *
 * 设计要点：
 *   - RoleGuard 限制 escort 角色
 *   - 拒单弹 UiModal 收原因（必填）
 *   - 操作成功后从列表移除该项
 */
import { ref, onMounted } from 'vue';
import RoleGuard from '@/components/shared/RoleGuard.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { fetchFeed } from '@/api/match';
import type { MatchFeedItem } from '@/api/match';
import { confirmAccept, rejectAccept } from '@/api/orders';

const items = ref<MatchFeedItem[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const showRejectModal = ref(false);
const currentOrderId = ref<number | null>(null);
const rejectReason = ref('');
const submitting = ref(false);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await fetchFeed();
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onAccept(o: MatchFeedItem) {
  try {
    await confirmAccept(o.order_id);
    items.value = items.value.filter((x) => x.order_id !== o.order_id);
  } catch {
    // ignore
  }
}

function onAskReject(o: MatchFeedItem) {
  currentOrderId.value = o.order_id;
  rejectReason.value = '';
  showRejectModal.value = true;
}

async function onConfirmReject() {
  if (!currentOrderId.value || !rejectReason.value.trim()) return;
  submitting.value = true;
  try {
    await rejectAccept(currentOrderId.value, rejectReason.value.trim());
    items.value = items.value.filter((x) => x.order_id !== currentOrderId.value!);
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

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <RoleGuard :required="['escort']" fallback-title="需要 escort 角色">
    <view class="ui-page" data-testid="escort-invitations-page">
      <view class="escort-invitations__header">
        <text class="escort-invitations__title">🎯 抢单池</text>
        <text class="escort-invitations__subtitle">可接候选订单</text>
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

      <view v-else-if="items.length === 0" data-testid="escort-invitations-empty">
        <UiEmpty icon="🎯" title="暂无可接订单" />
      </view>

      <view v-else data-testid="escort-invitations-list">
        <UiCard
          v-for="o in items"
          :key="o.order_id"
          :data-testid="`escort-invitations-card-${o.order_id}`"
        >
          <view class="escort-invitations__row">
            <text class="escort-invitations__order-id">订单 #{{ o.order_id }}</text>
            <text class="escort-invitations__badge" data-testid="escort-invitations-slots">
              剩余 {{ o.remaining_slots }} / {{ o.total_candidates }}
            </text>
          </view>
          <text class="escort-invitations__hospital">🏥 医院 #{{ o.hospital_id }}</text>
          <text class="escort-invitations__time">📅 预约 {{ formatDate(o.appointment_time) }}</text>
          <text class="escort-invitations__accepted">👥 已收到 {{ o.accepted_count }} 单接单申请</text>
          <text class="escort-invitations__deadline">⏰ 截止 {{ formatDate(o.deadline) }}</text>
          <view class="escort-invitations__actions">
            <UiButton
              type="primary"
              size="sm"
              :data-testid="`escort-invitations-accept-${o.order_id}`"
              @click="onAccept(o)"
            >
              接单
            </UiButton>
            <UiButton
              type="danger"
              size="sm"
              :data-testid="`escort-invitations-reject-${o.order_id}`"
              @click="onAskReject(o)"
            >
              拒单
            </UiButton>
          </view>
        </UiCard>
      </view>

      <!-- 拒单 Modal -->
      <UiModal
        :visible="showRejectModal"
        title="拒绝接单"
        content=""
        @update:visible="(v: boolean) => !v && onCancelReject()"
      >
        <view data-testid="escort-invitations-reject-modal">
          <UiInput
            v-model="rejectReason"
            label="拒单原因"
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
              确认拒单
            </UiButton>
          </view>
        </view>
      </UiModal>
    </view>
  </RoleGuard>
</template>

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

.escort-invitations__badge {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.escort-invitations__hospital,
.escort-invitations__time,
.escort-invitations__accepted,
.escort-invitations__deadline {
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