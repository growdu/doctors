<script setup lang="ts">
/**
 * admin/refunds/detail.vue — 退款详情（v2 unified-app · admin 域）。
 *
 * 入口：admin/refunds/list 点击「详情」
 *   → listRefunds 过滤出 id 匹配的 refund
 *   → pending 状态下显示「通过」/「驳回」操作
 *
 * 设计要点：
 *   - admin-service 未暴露 refund detail 单端点，统一从 listRefunds 过滤找到
 *   - 通过弹 UiModal 收「审批备注」（可选）；驳回弹 UiModal 收原因（必填）
 *   - 操作成功后 uni.navigateBack 返回列表
 */
import { ref, onMounted } from 'vue';
import { listRefunds, approveRefund, rejectRefund } from '@/api/admin';
import type { Refund } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

type RefundStatus = Refund['status'];

const STATUS_LABEL: Record<RefundStatus, string> = {
  pending: '待审',
  approved: '已通过',
  rejected: '已驳回',
};

const STATUS_CLASS: Record<RefundStatus, string> = {
  pending: 'admin-refund-detail__status--pending',
  approved: 'admin-refund-detail__status--approved',
  rejected: 'admin-refund-detail__status--rejected',
};

const refund = ref<Refund | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);
const refundId = ref<number | null>(null);

const showApproveModal = ref(false);
const showRejectModal = ref(false);
const approveNote = ref('');
const rejectReason = ref('');
const submitting = ref(false);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.id) refundId.value = Number(opts.id);
}

async function onLoad() {
  parseQuery();
  if (!refundId.value) {
    error.value = '未指定退款 ID';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    const r = await listRefunds({});
    refund.value = r.items.find((x) => x.id === refundId.value) ?? null;
    if (!refund.value) error.value = '退款工单不存在';
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function onAskApprove() {
  approveNote.value = '';
  showApproveModal.value = true;
}

function onAskReject() {
  rejectReason.value = '';
  showRejectModal.value = true;
}

async function onConfirmApprove() {
  if (!refund.value) return;
  submitting.value = true;
  try {
    await approveRefund(refund.value.id, approveNote.value.trim() || undefined);
    showApproveModal.value = false;
    if (typeof uni !== 'undefined') uni.navigateBack();
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

async function onConfirmReject() {
  if (!refund.value || !rejectReason.value.trim()) return;
  submitting.value = true;
  try {
    await rejectRefund(refund.value.id, rejectReason.value.trim());
    showRejectModal.value = false;
    if (typeof uni !== 'undefined') uni.navigateBack();
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

function onCancelModal() {
  showApproveModal.value = false;
  showRejectModal.value = false;
  approveNote.value = '';
  rejectReason.value = '';
}

function onBack() {
  if (typeof uni !== 'undefined') uni.navigateBack();
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
  <view class="ui-page" data-testid="admin-refund-detail-page">
    <view v-if="loading" data-testid="refund-detail-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error || !refund" data-testid="refund-detail-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error ?? '退款工单不存在'">
        <template #action>
          <text class="admin-refund-detail__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <UiCard :title="`退款 #${refund.id}`" data-testid="refund-detail-header">
        <view class="admin-refund-detail__row">
          <text class="admin-refund-detail__label">状态</text>
          <text :class="['admin-refund-detail__status', STATUS_CLASS[refund.status]]" data-testid="refund-detail-status">
            {{ STATUS_LABEL[refund.status] }}
          </text>
        </view>
        <view class="admin-refund-detail__row">
          <text class="admin-refund-detail__label">订单 ID</text>
          <text class="admin-refund-detail__value">#{{ refund.order_id }}</text>
        </view>
        <view class="admin-refund-detail__row">
          <text class="admin-refund-detail__label">支付 ID</text>
          <text class="admin-refund-detail__value">#{{ refund.payment_id }}</text>
        </view>
        <view class="admin-refund-detail__row admin-refund-detail__row--top">
          <text class="admin-refund-detail__label">退款原因</text>
          <text class="admin-refund-detail__value" data-testid="refund-detail-reason">{{ refund.reason }}</text>
        </view>
        <view class="admin-refund-detail__row">
          <text class="admin-refund-detail__label">退款金额</text>
          <text class="admin-refund-detail__amount" data-testid="refund-detail-amount">{{ formatYuan(refund.amount) }}</text>
        </view>
        <view class="admin-refund-detail__row">
          <text class="admin-refund-detail__label">创建时间</text>
          <text class="admin-refund-detail__value">{{ formatDate(refund.created_at) }}</text>
        </view>
      </UiCard>

      <view v-if="refund.status === 'pending'" class="admin-refund-detail__actions" data-testid="refund-detail-actions">
        <UiButton type="default" block data-testid="refund-detail-back" @click="onBack">返回列表</UiButton>
        <UiButton type="primary" block data-testid="refund-detail-approve" @click="onAskApprove">通过</UiButton>
        <UiButton type="danger" block data-testid="refund-detail-reject" @click="onAskReject">驳回</UiButton>
      </view>
      <view v-else class="admin-refund-detail__actions">
        <UiButton type="default" block data-testid="refund-detail-back" @click="onBack">返回列表</UiButton>
      </view>
    </template>

    <!-- 通过 Modal -->
    <UiModal
      :visible="showApproveModal"
      title="通过退款"
      content=""
      @update:visible="(v: boolean) => !v && onCancelModal()"
    >
      <view data-testid="refund-detail-approve-modal">
        <UiInput
          v-model="approveNote"
          label="审批备注（可选）"
          placeholder="例：已联系财务，全额退款"
          type="textarea"
          :maxlength="120"
          data-testid="refund-detail-approve-note"
        />
        <view class="admin-refund-detail__modal-actions">
          <UiButton type="default" block data-testid="refund-detail-approve-cancel" @click="onCancelModal">返回</UiButton>
          <UiButton
            type="primary"
            block
            :loading="submitting"
            data-testid="refund-detail-approve-confirm"
            @click="onConfirmApprove"
          >
            确认通过
          </UiButton>
        </view>
      </view>
    </UiModal>

    <!-- 驳回 Modal -->
    <UiModal
      :visible="showRejectModal"
      title="驳回退款"
      content=""
      @update:visible="(v: boolean) => !v && onCancelModal()"
    >
      <view data-testid="refund-detail-reject-modal">
        <UiInput
          v-model="rejectReason"
          label="驳回原因"
          placeholder="例：患者已在其他渠道退款"
          type="textarea"
          :maxlength="200"
          data-testid="refund-detail-reject-reason"
        />
        <view class="admin-refund-detail__modal-actions">
          <UiButton type="default" block data-testid="refund-detail-reject-cancel" @click="onCancelModal">返回</UiButton>
          <UiButton
            type="danger"
            block
            :loading="submitting"
            :disabled="!rejectReason.trim()"
            data-testid="refund-detail-reject-confirm"
            @click="onConfirmReject"
          >
            确认驳回
          </UiButton>
        </view>
      </view>
    </UiModal>
  </view>
</template>

<style scoped>
.admin-refund-detail__row {
  display: flex;
  align-items: center;
  margin-bottom: var(--ui-space-sm);
}

.admin-refund-detail__row--top {
  align-items: flex-start;
}

.admin-refund-detail__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  width: 80px;
  flex-shrink: 0;
}

.admin-refund-detail__value {
  flex: 1;
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
}

.admin-refund-detail__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.admin-refund-detail__status--pending { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }
.admin-refund-detail__status--approved { background: var(--ui-color-success); color: var(--ui-color-text-inverse); }
.admin-refund-detail__status--rejected { background: var(--ui-color-error); color: var(--ui-color-text-inverse); }

.admin-refund-detail__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.admin-refund-detail__actions {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-md);
}

.admin-refund-detail__modal-actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-refund-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>