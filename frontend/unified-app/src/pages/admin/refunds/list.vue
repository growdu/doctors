<script setup lang="ts">
/**
 * admin/refunds/list.vue — 退款审批列表（v2 unified-app · admin 域）。
 *
 * 入口：admin/refunds/index「进入列表」
 *   → listRefunds 按 status 过滤
 *   → pending 项可「通过」/「驳回」（驳回必填原因）
 *   → 任意项可「详情」跳 detail
 *
 * 设计要点：
 *   - 3 状态过滤 tab（pending / approved / rejected）
 *   - 行内按钮：pending 显示审批按钮；其他状态显示详情按钮
 *   - 通过弹 UiModal 收「审批备注」（可选）；驳回弹 UiModal 收原因（必填）
 *   - 操作成功后从列表移除该项
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

const STATUS_TABS: Array<{ status: RefundStatus | 'all'; label: string }> = [
  { status: 'all', label: '全部' },
  { status: 'pending', label: '待审' },
  { status: 'approved', label: '已通过' },
  { status: 'rejected', label: '已驳回' },
];

const STATUS_LABEL: Record<RefundStatus, string> = {
  pending: '待审',
  approved: '已通过',
  rejected: '已驳回',
};

const STATUS_CLASS: Record<RefundStatus, string> = {
  pending: 'admin-refunds-list__status--pending',
  approved: 'admin-refunds-list__status--approved',
  rejected: 'admin-refunds-list__status--rejected',
};

const items = ref<Refund[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const activeTab = ref<RefundStatus | 'all'>('all');

const showApproveModal = ref(false);
const showRejectModal = ref(false);
const current = ref<Refund | null>(null);
const approveNote = ref('');
const rejectReason = ref('');
const submitting = ref(false);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const query: { status?: RefundStatus } = {};
    if (activeTab.value !== 'all') query.status = activeTab.value;
    const r = await listRefunds(query);
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onSwitchTab(status: RefundStatus | 'all') {
  activeTab.value = status;
  await onLoad();
}

function onDetail(r: Refund) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/admin/refunds/detail?id=${r.id}` });
}

function onAskApprove(r: Refund) {
  current.value = r;
  approveNote.value = '';
  showApproveModal.value = true;
}

function onAskReject(r: Refund) {
  current.value = r;
  rejectReason.value = '';
  showRejectModal.value = true;
}

async function onConfirmApprove() {
  if (!current.value) return;
  submitting.value = true;
  try {
    await approveRefund(current.value.id, approveNote.value.trim() || undefined);
    items.value = items.value.filter((x) => x.id !== current.value!.id);
    showApproveModal.value = false;
    approveNote.value = '';
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

async function onConfirmReject() {
  if (!current.value || !rejectReason.value.trim()) return;
  submitting.value = true;
  try {
    await rejectRefund(current.value.id, rejectReason.value.trim());
    items.value = items.value.filter((x) => x.id !== current.value!.id);
    showRejectModal.value = false;
    rejectReason.value = '';
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

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-refunds-list">
    <!-- tab 过滤 -->
    <view class="admin-refunds-list__tabs" data-testid="refunds-list-tabs">
      <view
        v-for="t in STATUS_TABS"
        :key="t.status"
        class="admin-refunds-list__tab"
        :class="{ 'admin-refunds-list__tab--active': activeTab === t.status }"
        :data-testid="`refunds-list-tab-${t.status}`"
        @click="onSwitchTab(t.status)"
      >
        {{ t.label }}
      </view>
    </view>

    <view v-if="loading" data-testid="refunds-list-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="refunds-list-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-refunds-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="refunds-list-empty">
      <UiEmpty icon="📋" title="暂无退款工单" />
    </view>

    <view v-else data-testid="refunds-list">
      <UiCard
        v-for="r in items"
        :key="r.id"
        :data-testid="`refunds-list-card-${r.id}`"
      >
        <view class="admin-refunds-list__row">
          <text class="admin-refunds-list__id">退款 #{{ r.id }}</text>
          <text :class="['admin-refunds-list__status', STATUS_CLASS[r.status]]" :data-testid="`refunds-list-status-${r.id}`">
            {{ STATUS_LABEL[r.status] }}
          </text>
        </view>
        <text class="admin-refunds-list__order">订单 #{{ r.order_id }} · 支付 #{{ r.payment_id }}</text>
        <text class="admin-refunds-list__reason">📝 {{ r.reason }}</text>
        <text class="admin-refunds-list__amount">{{ formatYuan(r.amount) }}</text>
        <text class="admin-refunds-list__time">📅 {{ formatDate(r.created_at) }}</text>
        <view class="admin-refunds-list__actions">
          <UiButton type="default" size="sm" :data-testid="`refunds-list-detail-${r.id}`" @click="onDetail(r)">
            详情
          </UiButton>
          <template v-if="r.status === 'pending'">
            <UiButton type="primary" size="sm" :data-testid="`refunds-list-approve-${r.id}`" @click="onAskApprove(r)">
              通过
            </UiButton>
            <UiButton type="danger" size="sm" :data-testid="`refunds-list-reject-${r.id}`" @click="onAskReject(r)">
              驳回
            </UiButton>
          </template>
        </view>
      </UiCard>
    </view>

    <!-- 通过 Modal -->
    <UiModal
      :visible="showApproveModal"
      title="通过退款"
      content=""
      @update:visible="(v: boolean) => !v && onCancelModal()"
    >
      <view data-testid="refunds-approve-modal">
        <UiInput
          v-model="approveNote"
          label="审批备注（可选）"
          placeholder="例：已联系财务，全额退款"
          type="textarea"
          :maxlength="120"
          data-testid="refunds-approve-note"
        />
        <view class="admin-refunds-list__modal-actions">
          <UiButton type="default" block data-testid="refunds-approve-cancel" @click="onCancelModal">返回</UiButton>
          <UiButton
            type="primary"
            block
            :loading="submitting"
            data-testid="refunds-approve-confirm"
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
      <view data-testid="refunds-reject-modal">
        <UiInput
          v-model="rejectReason"
          label="驳回原因"
          placeholder="例：患者已在其他渠道退款"
          type="textarea"
          :maxlength="200"
          data-testid="refunds-reject-reason"
        />
        <view class="admin-refunds-list__modal-actions">
          <UiButton type="default" block data-testid="refunds-reject-cancel" @click="onCancelModal">返回</UiButton>
          <UiButton
            type="danger"
            block
            :loading="submitting"
            :disabled="!rejectReason.trim()"
            data-testid="refunds-reject-confirm"
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
.admin-refunds-list__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.admin-refunds-list__tab {
  flex: 1;
  min-width: 64px;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.admin-refunds-list__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.admin-refunds-list__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.admin-refunds-list__id {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.admin-refunds-list__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.admin-refunds-list__status--pending { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }
.admin-refunds-list__status--approved { background: var(--ui-color-success); color: var(--ui-color-text-inverse); }
.admin-refunds-list__status--rejected { background: var(--ui-color-error); color: var(--ui-color-text-inverse); }

.admin-refunds-list__order,
.admin-refunds-list__reason,
.admin-refunds-list__time {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-refunds-list__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-refunds-list__actions {
  display: flex;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-sm);
}

.admin-refunds-list__modal-actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-refunds-list__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>