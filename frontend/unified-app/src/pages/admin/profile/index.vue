<script setup lang="ts">
/**
 * admin/profile/index.vue — 管理员个人中心（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「个人中心」/「退出登录」
 *   → 展示当前 admin 用户（authStore.user）
 *   → 列出已激活角色 + 全部角色
 *   → 「退出登录」清 authStore + reLaunch 到 home
 *
 * 设计要点：
 *   - 完全基于 authStore 现有字段（无额外 API 调用）
 *   - 「切换角色」入口留给 HomeShell 的 RoleSwitcherModal
 */
import { onMounted } from 'vue';
import { useAuthStore } from '@/store/auth';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';

const auth = useAuthStore();

const ROLE_LABEL: Record<string, string> = {
  patient: '患者',
  escort: '陪诊师',
  super_admin: '超级管理员',
  audit_admin: '审核管理员',
  order_admin: '订单管理员',
  refund_admin: '退款管理员',
  cs: '客服',
  viewer: '只读',
  finance_admin: '财务管理员',
};

function roleLabel(r: string): string {
  return ROLE_LABEL[r] ?? r;
}

function maskPhone(phone: string): string {
  if (phone.length !== 11) return phone;
  return `${phone.slice(0, 3)}****${phone.slice(7)}`;
}

function onLogout() {
  auth.onUnauthorized();
}

function onBackHome() {
  if (typeof uni !== 'undefined') uni.reLaunch({ url: '/pages/home/index' });
}

onMounted(() => {
  // 简化：复用 bootstrap 时已恢复的 user 状态
});
</script>

<template>
  <view class="ui-page" data-testid="admin-profile-page">
    <view v-if="auth.user" data-testid="admin-profile-content">
      <UiCard title="基本信息" data-testid="admin-profile-info-card">
        <view class="admin-profile__row">
          <text class="admin-profile__label">用户 ID</text>
          <text class="admin-profile__value">#{{ auth.user.id }}</text>
        </view>
        <view class="admin-profile__row">
          <text class="admin-profile__label">手机</text>
          <text class="admin-profile__value">{{ maskPhone(auth.user.phone) }}</text>
        </view>
        <view class="admin-profile__row">
          <text class="admin-profile__label">实名状态</text>
          <text
            class="admin-profile__value"
            :class="auth.user.real_name_verified ? 'admin-profile__verified' : 'admin-profile__unverified'"
            :data-testid="auth.user.real_name_verified ? 'admin-profile-verified' : 'admin-profile-unverified'"
          >
            {{ auth.user.real_name_verified ? '已实名' : '未实名' }}
          </text>
        </view>
      </UiCard>

      <UiCard title="角色信息" data-testid="admin-profile-roles-card">
        <view class="admin-profile__row">
          <text class="admin-profile__label">当前角色</text>
          <text class="admin-profile__value admin-profile__active-role" data-testid="admin-profile-active-role">
            {{ roleLabel(auth.user.active_role) }}
          </text>
        </view>
        <view class="admin-profile__row admin-profile__row--top">
          <text class="admin-profile__label">所有角色</text>
          <view class="admin-profile__value">
            <view
              v-for="r in auth.user.roles"
              :key="r"
              class="admin-profile__role-chip"
              :class="{ 'admin-profile__role-chip--active': r === auth.user.active_role }"
              :data-testid="`admin-profile-role-${r}`"
            >
              {{ roleLabel(r) }}
            </view>
          </view>
        </view>
      </UiCard>

      <view class="admin-profile__actions" data-testid="admin-profile-actions">
        <UiButton type="danger" block data-testid="admin-profile-logout" @click="onLogout">
          退出登录
        </UiButton>
      </view>
    </view>

    <view v-else data-testid="admin-profile-empty">
      <UiCard>
        <text class="admin-profile__empty">请先登录</text>
        <UiButton type="primary" block @click="onBackHome">
          返回首页
        </UiButton>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-profile__row {
  display: flex;
  align-items: center;
  margin-bottom: var(--ui-space-sm);
}

.admin-profile__row--top {
  align-items: flex-start;
}

.admin-profile__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  width: 80px;
  flex-shrink: 0;
}

.admin-profile__value {
  flex: 1;
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
}

.admin-profile__verified { color: var(--ui-color-success); }
.admin-profile__unverified { color: var(--ui-color-text-disabled); }

.admin-profile__active-role {
  color: var(--ui-color-primary);
  font-weight: var(--ui-font-weight-medium);
}

.admin-profile__role-chip {
  display: inline-block;
  font-size: var(--ui-font-xs);
  padding: 2px var(--ui-space-sm);
  background: var(--ui-color-bg-hover);
  border-radius: var(--ui-radius-sm);
  color: var(--ui-color-text-secondary);
  margin: 2px 4px 2px 0;
}

.admin-profile__role-chip--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.admin-profile__actions {
  margin-top: var(--ui-space-md);
}

.admin-profile__empty {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-secondary);
  display: block;
  text-align: center;
  padding: var(--ui-space-base) 0;
}
</style>