<script setup lang="ts">
/**
 * RoleSwitcherModal 角色切换弹层（v2 unified-app）。
 *
 * 设计原则：
 *   - 复用 UiModal 组件（统一交互）
 *   - 列出用户所有 roles + 当前 active 高亮
 *   - 点击角色：调用 authStore.switchRole() + 关闭弹层
 *   - loading 状态：切换按钮显示 spinner + 禁用
 *
 * 关键场景：
 *   - 用户只有 1 个 role：弹层提示"当前账号仅 1 个角色，无需切换"
 *   - 用户有多个 role：列表展示，点击切换
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.6
 */
import { computed, ref } from 'vue';
import { useAuthStore } from '@/store/auth';
import UiModal from './UiModal.vue';
import UiButton from './UiButton.vue';
import type { Role } from '@/types/auth';

const auth = useAuthStore();

const visible = defineModel<boolean>('visible', { default: false });
const switching = ref<string | null>(null);

const roleIcons: Record<Role, string> = {
  patient: '🩺',
  escort: '🚑',
  super_admin: '🛡️',
  order_admin: '📋',
  refund_admin: '💰',
  cs: '🎧',
  audit_admin: '🔍',
  viewer: '👁️',
};

const roleLabels: Record<Role, string> = {
  patient: '患者',
  escort: '陪诊师',
  super_admin: '超级管理员',
  order_admin: '订单管理员',
  refund_admin: '退款管理员',
  cs: '客服',
  audit_admin: '审核管理员',
  viewer: '只读账号',
};

const availableRoles = computed<Role[]>(() => Array.from(new Set(auth.roles as Role[])));
const singleRole = computed(() => availableRoles.value.length <= 1);

async function onPick(role: Role) {
  if (role === auth.activeRole) {
    visible.value = false;
    return;
  }
  switching.value = role;
  try {
    await auth.switchRole(role);
    visible.value = false;
  } catch (e) {
    // 错误已在 store.error；调用方可读
  } finally {
    switching.value = null;
  }
}
</script>

<template>
  <UiModal
    v-model:visible="visible"
    title="切换激活角色"
    content=""
    cancel-text="关闭"
    confirm-text=""
    :close-on-mask="true"
    data-testid="role-switcher-modal"
  >
    <view v-if="singleRole" class="role-switcher-modal__single" data-testid="role-switcher-single">
      <view class="role-switcher-modal__msg">
        当前账号仅 1 个角色（{{ roleLabels[auth.activeRole as Role] || auth.activeRole }}），无需切换。
      </view>
      <UiButton type="primary" block @click="visible = false" data-testid="role-switcher-ack">
        知道了
      </UiButton>
    </view>

    <view v-else class="role-switcher-modal__list" data-testid="role-switcher-list">
      <view
        v-for="r in availableRoles"
        :key="r"
        class="role-switcher-modal__item"
        :class="{
          'role-switcher-modal__item--active': r === auth.activeRole,
          'role-switcher-modal__item--loading': switching === r,
        }"
        :data-testid="`role-option-${r}`"
        @click="onPick(r)"
      >
        <view class="role-switcher-modal__icon">{{ roleIcons[r] }}</view>
        <view class="role-switcher-modal__label">{{ roleLabels[r] }}</view>
        <view v-if="r === auth.activeRole" class="role-switcher-modal__check" data-testid="role-option-active">
          ✓
        </view>
        <view v-else-if="switching === r" class="role-switcher-modal__loading" data-testid="role-option-loading">
          切换中…
        </view>
      </view>
    </view>
  </UiModal>
</template>

<style lang="css" scoped>

.role-switcher-modal__single {
  padding: var(--ui-space-md) 0;
}

.role-switcher-modal__msg {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-secondary);
  text-align: center;
  margin-bottom: var(--ui-space-base);
}

.role-switcher-modal__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.role-switcher-modal__item {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
  padding: var(--ui-space-md);
  background-color: var(--ui-color-bg-hover);
  border: 2px solid transparent;
  border-radius: var(--ui-radius-md);
  cursor: pointer;
  transition: all var(--ui-duration-fast) ease;
}

.role-switcher-modal__item--active {
  border-color: var(--ui-color-primary);
  background-color: rgba(22, 119, 255, 0.04);
}

.role-switcher-modal__item--loading {
  opacity: 0.6;
}

.role-switcher-modal__icon {
  font-size: 24px;
}

.role-switcher-modal__label {
  flex: 1;
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
}

.role-switcher-modal__check {
  color: var(--ui-color-primary);
  font-weight: var(--ui-font-weight-bold);
}

.role-switcher-modal__loading {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}
</style>
