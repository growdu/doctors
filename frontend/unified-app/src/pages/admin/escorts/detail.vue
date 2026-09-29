<script setup lang="ts">
/**
 * admin/escorts/detail.vue — 陪诊师详情（v2 unified-app · admin 域）。
 *
 * 入口：admin/escorts/pending-audit 点击「查看详情」
 *   → 调 listPendingEscorts() 过滤出 id 匹配的项
 *   → 顶部：基本信息（昵称 / 手机 / 资质材料 / 提交时间）
 *   → 底部：审批操作按钮（approve / reject）
 *
 * 设计要点：
 *   - v2 unified-app 没有 escort detail 单端点（admin-service 只暴露 pending-audit），
 *     因此从 listPendingEscorts 列表里取一条（已审核的不再展示）；
 *   - 拒绝弹 UiModal 收原因
 *   - 操作成功后 uni.navigateBack 返回列表
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

const escort = ref<PendingEscort | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);
const escortId = ref<number | null>(null);

const showRejectModal = ref(false);
const rejectReason = ref('');
const rejecting = ref(false);
const approving = ref(false);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.id) escortId.value = Number(opts.id);
}

async function onLoad() {
  parseQuery();
  if (!escortId.value) {
    error.value = '未指定陪诊师 ID';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    const r = await listPendingEscorts();
    escort.value = r.items.find((x) => x.id === escortId.value) ?? null;
    if (!escort.value) error.value = '陪诊师不存在或已审核';
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function onAskReject() {
  rejectReason.value = '';
  showRejectModal.value = true;
}

async function onConfirmReject() {
  if (!escort.value || !rejectReason.value.trim()) return;
  rejecting.value = true;
  try {
    await rejectEscort(escort.value.id, rejectReason.value.trim());
    showRejectModal.value = false;
    if (typeof uni !== 'undefined') uni.navigateBack();
  } catch {
    // ignore
  } finally {
    rejecting.value = false;
  }
}

async function onApprove() {
  if (!escort.value) return;
  approving.value = true;
  try {
    await approveEscort(escort.value.id);
    if (typeof uni !== 'undefined') uni.navigateBack();
  } catch {
    // ignore
  } finally {
    approving.value = false;
  }
}

function onCancelReject() {
  showRejectModal.value = false;
  rejectReason.value = '';
}

function onBack() {
  if (typeof uni !== 'undefined') uni.navigateBack();
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-escort-detail-page">
    <view v-if="loading" data-testid="escort-detail-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error || !escort" data-testid="escort-detail-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error ?? '陪诊师不存在'">
        <template #action>
          <text class="admin-escort-detail__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <UiCard :title="`陪诊师 #${escort.id}`" data-testid="escort-detail-header">
        <view class="admin-escort-detail__row">
          <text class="admin-escort-detail__label">昵称</text>
          <text class="admin-escort-detail__value">{{ escort.nickname }}</text>
        </view>
        <view class="admin-escort-detail__row">
          <text class="admin-escort-detail__label">手机</text>
          <text class="admin-escort-detail__value">{{ escort.phone }}</text>
        </view>
        <view class="admin-escort-detail__row">
          <text class="admin-escort-detail__label">用户 ID</text>
          <text class="admin-escort-detail__value">#{{ escort.user_id }}</text>
        </view>
        <view class="admin-escort-detail__row">
          <text class="admin-escort-detail__label">提交时间</text>
          <text class="admin-escort-detail__value">{{ formatDate(escort.submitted_at) }}</text>
        </view>
        <view class="admin-escort-detail__row admin-escort-detail__row--top">
          <text class="admin-escort-detail__label">资质材料</text>
          <view class="admin-escort-detail__value">
            <view v-if="escort.qualification_urls.length === 0" data-testid="escort-detail-no-quals">
              <text class="admin-escort-detail__muted">暂无</text>
            </view>
            <view v-else data-testid="escort-detail-quals">
              <text
                v-for="(_, idx) in escort.qualification_urls"
                :key="idx"
                class="admin-escort-detail__qual"
                :data-testid="`escort-detail-qual-${idx}`"
              >
                📎 材料 {{ idx + 1 }}
              </text>
            </view>
          </view>
        </view>
      </UiCard>

      <view class="admin-escort-detail__actions" data-testid="escort-detail-actions">
        <UiButton type="default" block data-testid="escort-detail-back" @click="onBack">返回</UiButton>
        <UiButton
          type="primary"
          block
          :loading="approving"
          data-testid="escort-detail-approve"
          @click="onApprove"
        >
          通过审核
        </UiButton>
        <UiButton
          type="danger"
          block
          data-testid="escort-detail-reject"
          @click="onAskReject"
        >
          拒绝审核
        </UiButton>
      </view>
    </template>

    <!-- 拒绝原因 Modal -->
    <UiModal
      :visible="showRejectModal"
      title="拒绝陪诊师审核"
      content=""
      @update:visible="(v: boolean) => !v && onCancelReject()"
    >
      <view data-testid="escort-detail-reject-modal">
        <UiInput
          v-model="rejectReason"
          label="拒绝原因"
          placeholder="例：健康证已过期，请重新上传"
          type="textarea"
          :maxlength="200"
          data-testid="escort-detail-reject-reason"
        />
        <view class="admin-escort-detail__modal-actions">
          <UiButton type="default" block data-testid="escort-detail-reject-cancel" @click="onCancelReject">返回</UiButton>
          <UiButton
            type="danger"
            block
            :loading="rejecting"
            :disabled="!rejectReason.trim()"
            data-testid="escort-detail-reject-confirm"
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
.admin-escort-detail__row {
  display: flex;
  align-items: center;
  margin-bottom: var(--ui-space-sm);
}

.admin-escort-detail__row--top {
  align-items: flex-start;
}

.admin-escort-detail__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  width: 80px;
  flex-shrink: 0;
}

.admin-escort-detail__value {
  flex: 1;
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
}

.admin-escort-detail__qual {
  display: block;
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  padding: 2px 0;
}

.admin-escort-detail__muted {
  color: var(--ui-color-text-disabled);
}

.admin-escort-detail__actions {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-md);
}

.admin-escort-detail__modal-actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-escort-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>