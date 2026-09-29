<script setup lang="ts">
/**
 * escort/availability/index.vue — 陪诊师可接单时段管理（v2 unified-app · escort 域）。
 *
 * 入口：escort home 或聚合菜单「空余时段」入口
 *   → listMyAvailabilities() 拉我的时段列表
 *   → 「新增时段」按钮 → UiModal 收 start_at / end_at / remark
 *   → 每行「删除」按钮调 removeAvailability(id)
 *
 * 设计要点：
 *   - 简化版：时段以 ISO 字符串输入（uni-app 无原生 datetime picker，简单用文本框）
 *   - 校验：end_at > start_at
 *   - 删除成功后从列表移除
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { listMyAvailabilities, addAvailability, removeAvailability } from '@/api/escort';
import type { Availability } from '@/api/escort';

const items = ref<Availability[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const showCreateModal = ref(false);
const startAt = ref('');
const endAt = ref('');
const remark = ref('');
const submitting = ref(false);
const createError = ref<string | null>(null);

const removingId = ref<number | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listMyAvailabilities();
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function onAskCreate() {
  startAt.value = '';
  endAt.value = '';
  remark.value = '';
  createError.value = null;
  showCreateModal.value = true;
}

async function onConfirmCreate() {
  createError.value = null;
  if (!startAt.value.trim() || !endAt.value.trim()) {
    createError.value = '请输入开始和结束时间';
    return;
  }
  if (new Date(endAt.value) <= new Date(startAt.value)) {
    createError.value = '结束时间必须晚于开始时间';
    return;
  }
  submitting.value = true;
  try {
    const a = await addAvailability({
      start_at: startAt.value.trim(),
      end_at: endAt.value.trim(),
      remark: remark.value.trim(),
    });
    items.value = [...items.value, a];
    showCreateModal.value = false;
  } catch (e) {
    createError.value = `创建失败：${(e as Error).message}`;
  } finally {
    submitting.value = false;
  }
}

function onCancelCreate() {
  showCreateModal.value = false;
  createError.value = null;
}

async function onRemove(a: Availability) {
  removingId.value = a.id;
  try {
    await removeAvailability(a.id);
    items.value = items.value.filter((x) => x.id !== a.id);
  } catch {
    // ignore
  } finally {
    removingId.value = null;
  }
}

function formatDate(iso: string): string {
  return iso.replace('T', ' ').substring(0, 16);
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="escort-availability-page">
    <view class="escort-availability__actions">
      <UiButton
        type="primary"
        block
        data-testid="escort-availability-create-btn"
        @click="onAskCreate"
      >
        + 新增时段
      </UiButton>
    </view>

    <view v-if="loading" data-testid="escort-availability-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="escort-availability-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="escort-availability__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="escort-availability-empty">
      <UiEmpty icon="📅" title="暂未设置时段" description="点上方「+ 新增时段」添加" />
    </view>

    <view v-else data-testid="escort-availability-list">
      <UiCard
        v-for="a in items"
        :key="a.id"
        :data-testid="`escort-availability-card-${a.id}`"
      >
        <view class="escort-availability__row">
          <text class="escort-availability__time" :data-testid="`escort-availability-time-${a.id}`">
            🕐 {{ formatDate(a.start_at) }} → {{ formatDate(a.end_at) }}
          </text>
          <UiButton
            type="danger"
            size="sm"
            :loading="removingId === a.id"
            :data-testid="`escort-availability-remove-${a.id}`"
            @click="onRemove(a)"
          >
            删除
          </UiButton>
        </view>
        <text v-if="a.remark" class="escort-availability__remark">📝 {{ a.remark }}</text>
      </UiCard>
    </view>

    <!-- 创建 Modal -->
    <UiModal
      :visible="showCreateModal"
      title="新增时段"
      content=""
      @update:visible="(v: boolean) => !v && onCancelCreate()"
    >
      <view data-testid="escort-availability-create-modal">
        <UiInput
          v-model="startAt"
          label="开始时间"
          placeholder="例：2026-10-01T09:00:00Z"
          data-testid="escort-availability-create-start"
        />
        <UiInput
          v-model="endAt"
          label="结束时间"
          placeholder="例：2026-10-01T17:00:00Z"
          data-testid="escort-availability-create-end"
        />
        <UiInput
          v-model="remark"
          label="备注（可选）"
          placeholder="例：周末优先"
          data-testid="escort-availability-create-remark"
        />
        <view v-if="createError" class="escort-availability__error" data-testid="escort-availability-create-error">
          <text class="escort-availability__error-text">⚠️ {{ createError }}</text>
        </view>
        <view class="escort-availability__modal-actions">
          <UiButton type="default" block data-testid="escort-availability-create-cancel" @click="onCancelCreate">取消</UiButton>
          <UiButton
            type="primary"
            block
            :loading="submitting"
            data-testid="escort-availability-create-confirm"
            @click="onConfirmCreate"
          >
            添加
          </UiButton>
        </view>
      </view>
    </UiModal>
  </view>
</template>

<style scoped>
.escort-availability__actions {
  margin-bottom: var(--ui-space-md);
}

.escort-availability__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.escort-availability__time {
  flex: 1;
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.escort-availability__remark {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-availability__error {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-error);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-md);
}

.escort-availability__error-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
}

.escort-availability__modal-actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.escort-availability__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>