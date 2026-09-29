<script setup lang="ts">
/**
 * admin/escorts/pending-audit.vue — 陪诊师待审核队列（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「在线陪诊师」卡片 / escorts/index 「待审核」入口
 *   → 调 listPendingEscorts() 拉待审列表
 *   → 行操作：通过 / 拒绝（拒绝打开 UiModal 收原因）
 *
 * 设计要点：
 *   - 复用 patient/order STATUS 颜色映射思路，audit_status='pending' → 橙
 *   - 拒绝强制 UiModal 收集原因；通过直接调 API
 *   - 操作成功后从列表移除（乐观更新）
 */
import { ref, onMounted } from 'vue';
import { listPendingEscorts, approveEscort, rejectEscort } from '@/api/admin';
import type { PendingEscort } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const items = ref<PendingEscort[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const showRejectModal = ref(false);
const rejectTarget = ref<PendingEscort | null>(null);
const rejectReason = ref('');
const rejecting = ref(false);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listPendingEscorts();
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function onDetail(e: PendingEscort) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/admin/escorts/detail?id=${e.id}` });
}

function onAskApprove(e: PendingEscort) {
  void doApprove(e);
}

async function doApprove(e: PendingEscort) {
  try {
    await approveEscort(e.id);
    items.value = items.value.filter((x) => x.id !== e.id);
  } catch {
    // ignore; 失败保留列表项
  }
}

function onAskReject(e: PendingEscort) {
  rejectTarget.value = e;
  rejectReason.value = '';
  showRejectModal.value = true;
}

async function onConfirmReject() {
  if (!rejectTarget.value || !rejectReason.value.trim()) return;
  rejecting.value = true;
  try {
    await rejectEscort(rejectTarget.value.id, rejectReason.value.trim());
    items.value = items.value.filter((x) => x.id !== rejectTarget.value!.id);
    showRejectModal.value = false;
    rejectReason.value = '';
  } catch {
    // ignore
  } finally {
    rejecting.value = false;
  }
}

function onCancelReject() {
  showRejectModal.value = false;
  rejectReason.value = '';
  rejectTarget.value = null;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-escorts-audit-page">
    <view v-if="loading" data-testid="audit-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="audit-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-escorts-audit__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="audit-empty">
      <UiEmpty icon="✅" title="暂无待审核陪诊师" />
    </view>

    <view v-else data-testid="audit-list">
      <UiCard
        v-for="e in items"
        :key="e.id"
        :data-testid="`audit-card-${e.id}`"
      >
        <view class="admin-escorts-audit__row">
          <view class="admin-escorts-audit__info">
            <text class="admin-escorts-audit__name">#{{ e.id }} {{ e.nickname }}</text>
            <text class="admin-escorts-audit__phone">📱 {{ e.phone }}</text>
            <text class="admin-escorts-audit__time">📅 {{ formatDate(e.submitted_at) }}</text>
          </view>
          <text class="admin-escorts-audit__badge admin-escorts-audit__badge--pending" :data-testid="`audit-status-${e.id}`">
            待审核
          </text>
        </view>
        <view class="admin-escorts-audit__actions">
          <UiButton type="default" size="sm" :data-testid="`audit-detail-${e.id}`" @click="onDetail(e)">
            查看详情
          </UiButton>
          <UiButton type="primary" size="sm" :data-testid="`audit-approve-${e.id}`" @click="onAskApprove(e)">
            通过
          </UiButton>
          <UiButton type="danger" size="sm" :data-testid="`audit-reject-${e.id}`" @click="onAskReject(e)">
            拒绝
          </UiButton>
        </view>
      </UiCard>
    </view>

    <!-- 拒绝原因 Modal -->
    <UiModal
      :visible="showRejectModal"
      title="拒绝陪诊师审核"
      content=""
      @update:visible="(v: boolean) => !v && onCancelReject()"
    >
      <view data-testid="audit-reject-modal">
        <UiInput
          v-model="rejectReason"
          label="拒绝原因"
          placeholder="例：健康证已过期"
          type="textarea"
          :maxlength="200"
          data-testid="audit-reject-reason"
        />
        <view class="admin-escorts-audit__modal-actions">
          <UiButton type="default" block data-testid="audit-reject-cancel" @click="onCancelReject">返回</UiButton>
          <UiButton
            type="danger"
            block
            :loading="rejecting"
            :disabled="!rejectReason.trim()"
            data-testid="audit-reject-confirm"
            @click="onConfirmReject"
          >
            确认拒绝
          </UiButton>
        </view>
      </view>
    </UiModal>
  </view>
</template>

<style scoped>
.admin-escorts-audit__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.admin-escorts-audit__info {
  flex: 1;
}

.admin-escorts-audit__name {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-escorts-audit__phone,
.admin-escorts-audit__time {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-escorts-audit__badge {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.admin-escorts-audit__badge--pending {
  background: var(--ui-color-warning);
  color: var(--ui-color-text-inverse);
}

.admin-escorts-audit__actions {
  display: flex;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-sm);
}

.admin-escorts-audit__modal-actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-escorts-audit__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>