<script setup lang="ts">
/**
 * DomainSwitcher 域切换器（v2 unified-app）。
 *
 * 设计原则：
 *   - 3 域卡片（patient / escort / admin）大按钮平铺
 *   - 当前 active 域高亮（实心）+ 其他域 ghost 样式
 *   - 用户有多个 role 时，所有对应域均可访问；点击跳对应域首页
 *   - 域可用性依据：用户 active_role + roles[] 中是否包含该域代表 role
 *     - patient 域：active_role === 'patient'
 *     - escort 域：active_role === 'escort'
 *     - admin 域：active_role ∈ {super_admin, order_admin, refund_admin, cs, audit_admin, viewer}
 *
 * 注意：本组件只负责 UI，不触发角色切换；角色切换由 RoleSwitcherModal 处理。
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.5
 */
import { useAuthStore } from '@/store/auth';
import type { Role } from '@/types/auth';

interface Domain {
  id: 'patient' | 'escort' | 'admin';
  label: string;
  icon: string;
  /** 进入该域所需的 active_role（admin 任一即可） */
  entryRoles: Role[];
  /** 域首页路径 */
  homePath: string;
  /** 域描述 */
  description: string;
}

const props = defineProps<{
  /** 当前激活域（从外部传入，便于非 HomeShell 场景使用） */
  currentDomain?: 'patient' | 'escort' | 'admin';
}>();

const emit = defineEmits<{
  (e: 'select', domain: 'patient' | 'escort' | 'admin'): void;
}>();

const auth = useAuthStore();

const domains: Domain[] = [
  {
    id: 'patient',
    label: '患者域',
    icon: '🩺',
    entryRoles: ['patient'],
    homePath: '/pages/patient/index',
    description: '下单 / 订单 / 钱包 / 评价',
  },
  {
    id: 'escort',
    label: '陪诊师域',
    icon: '🚑',
    entryRoles: ['escort'],
    homePath: '/pages/escort/index',
    description: '抢单池 / 任务 / 我的钱包',
  },
  {
    id: 'admin',
    label: '管理后台',
    icon: '🛡️',
    entryRoles: ['super_admin', 'order_admin', 'refund_admin', 'cs', 'audit_admin', 'viewer'],
    homePath: '/pages/admin/index',
    description: '工单 / 报表 / 审批',
  },
];

/**
 * 判断域是否对当前用户可用：
 *   - active_role ∈ entryRoles（用户当前正在该角色）
 *   - 或者用户 roles 数组包含该域 entryRoles 任一项（可切换）
 */
function isAvailable(d: Domain): boolean {
  return d.entryRoles.some((r) => auth.hasRole(r));
}

function onSelect(d: Domain) {
  if (!isAvailable(d)) return;
  emit('select', d.id);
}
</script>

<template>
  <view class="domain-switcher" data-testid="domain-switcher">
    <view
      v-for="d in domains"
      :key="d.id"
      class="domain-switcher__card"
      :class="{
        'domain-switcher__card--active': props.currentDomain === d.id,
        'domain-switcher__card--disabled': !isAvailable(d),
      }"
      :data-testid="`domain-card-${d.id}`"
      @click="onSelect(d)"
    >
      <view class="domain-switcher__icon">{{ d.icon }}</view>
      <view class="domain-switcher__label">{{ d.label }}</view>
      <view class="domain-switcher__desc">{{ d.description }}</view>
      <view v-if="!isAvailable(d)" class="domain-switcher__lock" data-testid="domain-card-lock">
        🔒 未解锁
      </view>
    </view>
  </view>
</template>

<style lang="css" scoped>

.domain-switcher {
  display: flex;
  gap: var(--ui-space-md);
  flex-wrap: wrap;
}

.domain-switcher__card {
  position: relative;
  flex: 1 1 calc(33.33% - #{var(--ui-space-md)});
  min-width: 120px;
  padding: var(--ui-space-base);
  background-color: var(--ui-color-bg-card);
  border: 2px solid var(--ui-color-border);
  border-radius: var(--ui-radius-md);
  text-align: center;
  cursor: pointer;
  transition: all var(--ui-duration-fast) ease;
}

.domain-switcher__card--active {
  border-color: var(--ui-color-primary);
  background-color: rgba(22, 119, 255, 0.04);
}

.domain-switcher__card--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.domain-switcher__icon {
  font-size: 36px;
  margin-bottom: var(--ui-space-sm);
}

.domain-switcher__label {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  margin-bottom: var(--ui-space-xs);
}

.domain-switcher__desc {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
}

.domain-switcher__lock {
  position: absolute;
  top: var(--ui-space-xs);
  right: var(--ui-space-xs);
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
}
</style>
