<script setup lang="ts">
/**
 * patient/address/list.vue — 地址管理页（v2 unified-app）。
 *
 * 页面流向：
 *   入口：patient 域「个人中心」menu → 地址管理 / 订单创建页选择地址
 *      → 本页
 *      → onLoad/mounted 调 addressStore.fetchList()
 *      → 渲染地址卡片 + 「+ 新增地址」按钮
 *      → 点击「设为默认」/「删除」/「编辑」操作
 *
 * 设计要点：
 *   - 复用 UiCard（地址卡）+ UiEmpty（空态）+ UiLoading（加载态）
 *   - 删除前确认（UiModal v-model:visible）
 *   - 「+ 新增地址」按钮 → uni.navigateTo('/pages/patient/address/edit')
 *
 * 对应：dev.md §45.5 Phase 3.1a
 */
import { ref, onMounted } from 'vue';
import { useAddressStore } from '@/store/address';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';

const store = useAddressStore();
const confirmDeleteId = ref<number | null>(null);

async function onLoad() {
  await store.fetchList();
}

onMounted(onLoad);

async function onSetDefault(id: number) {
  try {
    await store.setDefault(id);
  } catch (e) {
    // 错误已记录到 store.error
  }
}

function onEdit(id: number) {
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: `/pages/patient/address/edit?addressId=${id}` });
  }
}

function onAdd() {
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: '/pages/patient/address/edit' });
  }
}

function onAskDelete(id: number) {
  confirmDeleteId.value = id;
}

async function onConfirmDelete() {
  const id = confirmDeleteId.value;
  confirmDeleteId.value = null;
  if (!id) return;
  try {
    await store.remove(id);
  } catch (e) {
    // 错误已记录
  }
}

function onCancelDelete() {
  confirmDeleteId.value = null;
}
</script>

<template>
  <view class="ui-page" data-testid="patient-address-list">
    <!-- loading 态 -->
    <view v-if="store.loading && store.list.length === 0" data-testid="address-loading">
      <UiLoading text="加载中..." />
    </view>

    <!-- error 态 -->
    <view v-else-if="store.error && store.list.length === 0" data-testid="address-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="store.error">
        <template #action>
          <UiButton type="primary" size="sm" @click="onLoad">重试</UiButton>
        </template>
      </UiEmpty>
    </view>

    <!-- empty 态 -->
    <view v-else-if="store.list.length === 0" data-testid="address-empty">
      <UiEmpty icon="📭" title="暂无地址" description="点击下方按钮新增第一条地址">
        <template #action>
          <UiButton type="primary" size="sm" @click="onAdd">+ 新增地址</UiButton>
        </template>
      </UiEmpty>
    </view>

    <!-- list 态 -->
    <view v-else data-testid="address-list">
      <UiCard
        v-for="a in store.list"
        :key="a.id"
        :title="a.name"
        :class="{ 'address-card--default': a.is_default }"
        :data-testid="`address-card-${a.id}`"
      >
        <view class="address-card__row">
            <text class="address-card__name">{{ a.name }}</text>
            <text class="address-card__phone">{{ a.phone }}</text>
            <text v-if="a.is_default" class="address-card__badge" data-test="default-badge">默认</text>
        </view>
        <text class="address-card__detail">{{ a.province }} {{ a.city }} {{ a.district }} {{ a.detail }}</text>
        <template #footer>
          <view class="address-card__actions">
            <UiButton
              v-if="!a.is_default"
              type="primary"
              size="sm"
              :data-testid="`set-default-${a.id}`"
              @click="onSetDefault(a.id)"
            >
              设为默认
            </UiButton>
            <UiButton
              type="default"
              size="sm"
              :data-testid="`edit-${a.id}`"
              @click="onEdit(a.id)"
            >
              编辑
            </UiButton>
            <UiButton
              type="danger"
              size="sm"
              :data-testid="`delete-${a.id}`"
              @click="onAskDelete(a.id)"
            >
              删除
            </UiButton>
          </view>
        </template>
      </UiCard>
    </view>

    <!-- 底部「+ 新增地址」 -->
    <view class="address-list__bottom">
      <text class="address-list__limit-hint" data-testid="address-limit-hint">
        {{ store.count }} / 5
      </text>
      <UiButton
        type="primary"
        block
        :disabled="store.atLimit"
        :data-testid="store.atLimit ? 'add-address-btn-disabled' : 'add-address-btn'"
        @click="onAdd"
      >
        {{ store.atLimit ? '已达上限' : '+ 新增地址' }}
      </UiButton>
    </view>

    <!-- 删除确认弹层 -->
    <UiModal
      :visible="confirmDeleteId !== null"
      title="删除地址"
      content="确定要删除该地址吗？"
      @update:visible="(v: boolean) => !v && onCancelDelete()"
    >
      <template #footer>
        <view style="display: flex; gap: 12px;">
          <UiButton type="default" block @click="onCancelDelete" data-testid="delete-cancel">取消</UiButton>
          <UiButton type="danger" block @click="onConfirmDelete" data-testid="delete-confirm">删除</UiButton>
        </view>
      </template>
    </UiModal>
  </view>
</template>

<style scoped>
.address-card--default {
  border: 1px solid var(--ui-color-primary);
}

.address-card__row {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.address-card__name {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  margin-right: 12px;
}

.address-card__phone {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-secondary);
  flex: 1;
}

.address-card__badge {
  font-size: var(--ui-font-xs);
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
}

.address-card__detail {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.address-card__actions {
  display: flex;
  gap: 8px;
}

.address-list__bottom {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--ui-color-bg-card);
  padding: var(--ui-space-md) var(--ui-space-base);
  border-top: 1px solid var(--ui-color-divider);
  z-index: 10;
}

.address-list__limit-hint {
  display: block;
  text-align: right;
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  padding-bottom: 4px;
}
</style>