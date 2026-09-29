<script setup lang="ts">
/**
 * admin/work-orders/index.vue — 客服工单管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「工单」相关入口
 *   → 3 状态过滤 tab（open / in_progress / closed）+ 列表
 *   → 「创建工单」按钮 → UiModal 收 3 字段（type / title / content）
 *   → v2 admin-service 未暴露 assign/close 端点，本期仅做 list + create
 *
 * 设计要点：
 *   - 复用 patient/order/list 渲染样式
 *   - 创建成功后乐观插入到列表首部
 */
import { ref, onMounted } from 'vue';
import { listWorkOrders, createWorkOrder } from '@/api/admin';
import type { WorkOrder } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

type WorkOrderStatus = WorkOrder['status'];
type WorkOrderType = WorkOrder['type'];

const STATUS_TABS: Array<{ status: WorkOrderStatus | 'all'; label: string }> = [
  { status: 'all', label: '全部' },
  { status: 'open', label: '待处理' },
  { status: 'in_progress', label: '处理中' },
  { status: 'closed', label: '已关闭' },
];

const STATUS_LABEL: Record<WorkOrderStatus, string> = {
  open: '待处理',
  in_progress: '处理中',
  closed: '已关闭',
};

const STATUS_CLASS: Record<WorkOrderStatus, string> = {
  open: 'admin-work-orders__status--open',
  in_progress: 'admin-work-orders__status--progress',
  closed: 'admin-work-orders__status--closed',
};

const TYPE_LABEL: Record<WorkOrderType, string> = {
  complaint: '投诉',
  refund: '退款',
  consult: '咨询',
  other: '其他',
};

const items = ref<WorkOrder[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const activeTab = ref<WorkOrderStatus | 'all'>('all');

const showCreateModal = ref(false);
const createType = ref<WorkOrderType>('complaint');
const createTitle = ref('');
const createContent = ref('');
const submitting = ref(false);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const query: { status?: WorkOrderStatus } = {};
    if (activeTab.value !== 'all') query.status = activeTab.value;
    const r = await listWorkOrders(query);
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onSwitchTab(status: WorkOrderStatus | 'all') {
  activeTab.value = status;
  await onLoad();
}

function onAskCreate() {
  createType.value = 'complaint';
  createTitle.value = '';
  createContent.value = '';
  showCreateModal.value = true;
}

async function onConfirmCreate() {
  if (!createTitle.value.trim() || !createContent.value.trim()) return;
  submitting.value = true;
  try {
    const w = await createWorkOrder({
      type: createType.value,
      title: createTitle.value.trim(),
      content: createContent.value.trim(),
    });
    items.value = [w, ...items.value];
    showCreateModal.value = false;
  } catch {
    // ignore
  } finally {
    submitting.value = false;
  }
}

function onCancelCreate() {
  showCreateModal.value = false;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-work-orders-page">
    <view class="admin-work-orders__tabs" data-testid="admin-work-orders-tabs">
      <view
        v-for="t in STATUS_TABS"
        :key="t.status"
        class="admin-work-orders__tab"
        :class="{ 'admin-work-orders__tab--active': activeTab === t.status }"
        :data-testid="`admin-work-orders-tab-${t.status}`"
        @click="onSwitchTab(t.status)"
      >
        {{ t.label }}
      </view>
    </view>

    <view class="admin-work-orders__actions">
      <UiButton
        type="primary"
        block
        data-testid="admin-work-orders-create-btn"
        @click="onAskCreate"
      >
        + 创建工单
      </UiButton>
    </view>

    <view v-if="loading" data-testid="admin-work-orders-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="admin-work-orders-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-work-orders__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="admin-work-orders-empty">
      <UiEmpty icon="🎫" title="暂无工单" />
    </view>

    <view v-else data-testid="admin-work-orders-list">
      <UiCard
        v-for="w in items"
        :key="w.id"
        :data-testid="`admin-work-orders-card-${w.id}`"
      >
        <view class="admin-work-orders__row">
          <text class="admin-work-orders__id">工单 #{{ w.id }}</text>
          <text :class="['admin-work-orders__status', STATUS_CLASS[w.status]]" :data-testid="`admin-work-orders-status-${w.id}`">
            {{ STATUS_LABEL[w.status] }}
          </text>
        </view>
        <view class="admin-work-orders__type-row">
          <text class="admin-work-orders__type">{{ TYPE_LABEL[w.type] }}</text>
          <text class="admin-work-orders__ref" v-if="w.ref_type">
            · {{ w.ref_type }} #{{ w.ref_id }}
          </text>
        </view>
        <text class="admin-work-orders__title">{{ w.title }}</text>
        <text class="admin-work-orders__content">{{ w.content }}</text>
        <text class="admin-work-orders__time">📅 {{ formatDate(w.created_at) }}</text>
        <text v-if="w.assignee" class="admin-work-orders__assignee">👤 负责人 #{{ w.assignee }}</text>
      </UiCard>
    </view>

    <!-- 创建工单 Modal -->
    <UiModal
      :visible="showCreateModal"
      title="创建工单"
      content=""
      @update:visible="(v: boolean) => !v && onCancelCreate()"
    >
      <view data-testid="admin-work-orders-create-modal">
        <view class="admin-work-orders__type-row" data-testid="admin-work-orders-create-types">
          <view
            v-for="t in (['complaint', 'refund', 'consult', 'other'] as WorkOrderType[])"
            :key="t"
            class="admin-work-orders__type-chip"
            :class="{ 'admin-work-orders__type-chip--active': createType === t }"
            :data-testid="`admin-work-orders-create-type-${t}`"
            @click="createType = t"
          >
            {{ TYPE_LABEL[t] }}
          </view>
        </view>
        <UiInput
          v-model="createTitle"
          label="工单标题"
          placeholder="一句话描述问题"
          data-testid="admin-work-orders-create-title"
        />
        <UiInput
          v-model="createContent"
          label="详细内容"
          placeholder="请详细描述问题"
          type="textarea"
          :maxlength="500"
          data-testid="admin-work-orders-create-content"
        />
        <view class="admin-work-orders__modal-actions">
          <UiButton type="default" block data-testid="admin-work-orders-create-cancel" @click="onCancelCreate">取消</UiButton>
          <UiButton
            type="primary"
            block
            :loading="submitting"
            :disabled="!createTitle.trim() || !createContent.trim()"
            data-testid="admin-work-orders-create-confirm"
            @click="onConfirmCreate"
          >
            提交
          </UiButton>
        </view>
      </view>
    </UiModal>
  </view>
</template>

<style scoped>
.admin-work-orders__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.admin-work-orders__tab {
  flex: 1;
  min-width: 64px;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.admin-work-orders__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.admin-work-orders__actions {
  margin-bottom: var(--ui-space-md);
}

.admin-work-orders__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.admin-work-orders__id {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.admin-work-orders__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.admin-work-orders__status--open { background: var(--ui-color-warning); color: var(--ui-color-text-inverse); }
.admin-work-orders__status--progress { background: var(--ui-color-primary); color: var(--ui-color-text-inverse); }
.admin-work-orders__status--closed { background: var(--ui-color-text-disabled); color: var(--ui-color-text-inverse); }

.admin-work-orders__type-row {
  display: flex;
  align-items: center;
  gap: var(--ui-space-xs);
  margin-top: 2px;
}

.admin-work-orders__type {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
}

.admin-work-orders__ref {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.admin-work-orders__title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-work-orders__content {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
  white-space: pre-wrap;
}

.admin-work-orders__time,
.admin-work-orders__assignee {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-work-orders__type-chip {
  font-size: var(--ui-font-sm);
  padding: var(--ui-space-xs) var(--ui-space-sm);
  background: var(--ui-color-bg-hover);
  border-radius: var(--ui-radius-sm);
  color: var(--ui-color-text-secondary);
  cursor: pointer;
}

.admin-work-orders__type-chip--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.admin-work-orders__modal-actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-work-orders__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>