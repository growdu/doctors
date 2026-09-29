<script setup lang="ts">
/**
 * HomeShell 主页壳（v2 unified-app · Phase 3.0.6 完整闭环）。
 *
 * 三大模块：
 *   1. 用户卡 —显示当前手机号 / active_role / roles[]
 *   2. 角色切换 —按钮打开 RoleSwitcherModal
 *   3. 域切换 —DomainSwitcher 卡片网格，点击跳转域首页
 *
 * 流程：
 *   - 未登录：显示 SMS 登录入口（Phase 3.1 patient 域迁移后接入真登录页）
 *   - 已登录：渲染用户卡 + 角色切换按钮 + 域切换器
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.6
 */
import { computed, ref } from 'vue';
import { useAuthStore } from '@/store/auth';
import UiButton from '@/components/shared/UiButton.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import DomainSwitcher from '@/components/shared/DomainSwitcher.vue';
import RoleSwitcherModal from '@/components/shared/RoleSwitcherModal.vue';
import type { Role } from '@/types/auth';

const auth = useAuthStore();

const phone = ref('13800138000');
const code = ref('');
const smsSent = ref(false);
const loginLoading = ref(false);
const loginError = ref('');

const switcherVisible = ref(false);
const domain = ref<'patient' | 'escort' | 'admin' | undefined>(undefined);

const availableRoles = computed<Role[]>(() => Array.from(new Set(auth.roles as Role[])));

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

async function onSendSms() {
  smsSent.value = true;
}

async function onLogin() {
  if (!phone.value || !code.value) {
    loginError.value = '手机号 + 验证码必填';
    return;
  }
  loginLoading.value = true;
  loginError.value = '';
  try {
    await auth.login(phone.value, code.value);
  } catch (e) {
    loginError.value = (e as Error).message || '登录失败';
  } finally {
    loginLoading.value = false;
  }
}

function onDomainSelect(d: 'patient' | 'escort' | 'admin') {
  domain.value = d;
  // 跳转域首页（Phase 3.1 patient 域迁移后接入真路径）
  const path = `/pages/${d}/index`;
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: path });
  }
}

function onLogout() {
  auth.onUnauthorized();
}
</script>

<template>
  <view class="ui-page">
    <!-- 顶部品牌卡 -->
    <UiCard title="Doctors 统一 App" shadow="sm" data-testid="home-brand">
      <view class="home-brand__sub">3 端合并 · 多角色 · 角色切换</view>
    </UiCard>

    <!-- 未登录：SMS 登录 -->
    <view v-if="!auth.isAuthed" data-testid="home-login-card">
      <UiCard title="SMS 登录" shadow="sm">
        <UiInput v-model="phone" label="手机号" placeholder="13800138000" clearable data-testid="home-login-phone" />
        <UiInput v-model="code" label="验证码" placeholder="6 位 mock code（看 auth-service 日志）" :maxlength="6" data-testid="home-login-code" />
        <view v-if="loginError" class="home-login__error" data-testid="home-login-error">{{ loginError }}</view>
        <view class="home-login__actions">
          <UiButton type="default" size="sm" @click="onSendSms" data-testid="home-sms-btn">发验证码</UiButton>
          <UiButton type="primary" :loading="loginLoading" @click="onLogin" data-testid="home-login-btn">登录</UiButton>
        </view>
      </UiCard>
    </view>

    <!-- 已登录：用户卡 -->
    <view v-else data-testid="home-user-card">
      <UiCard shadow="sm">
        <view class="home-user__row">
          <view class="home-user__label">手机号</view>
          <view class="home-user__value" data-testid="home-user-phone">{{ auth.user?.phone }}</view>
        </view>
        <view class="home-user__row">
          <view class="home-user__label">激活角色</view>
          <view class="home-user__value home-user__value--primary" data-testid="home-user-active">
            {{ roleIcons[auth.activeRole as Role] || '🔘' }} {{ auth.activeRole }}
          </view>
        </view>
        <view class="home-user__row">
          <view class="home-user__label">全部角色</view>
          <view class="home-user__value" data-testid="home-user-roles">
            {{ availableRoles.map((r) => roleIcons[r]).join(' ') }}
          </view>
        </view>
        <view class="home-user__actions">
          <UiButton type="primary" size="sm" block @click="switcherVisible = true" data-testid="home-role-switch-btn">
            切换激活角色（{{ availableRoles.length }}）
          </UiButton>
          <UiButton type="default" size="sm" block @click="onLogout" data-testid="home-logout-btn">退出登录</UiButton>
        </view>
      </UiCard>
    </view>

    <!-- 域切换器（仅已登录显示） -->
    <view v-if="auth.isAuthed" data-testid="home-domain-section">
      <view class="home-section__title">选择进入域</view>
      <DomainSwitcher :current-domain="domain" @select="onDomainSelect" />
    </view>
  </view>

  <!-- 角色切换弹层 -->
  <RoleSwitcherModal v-model:visible="switcherVisible" />
</template>

<style lang="css" scoped>

.home-brand__sub {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.home-login__error {
  margin-top: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
}

.home-login__actions {
  display: flex;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.home-user__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--ui-space-sm) 0;
  border-bottom: 1px solid var(--ui-color-divider);
}

.home-user__row:last-of-type {
  border-bottom: none;
}

.home-user__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.home-user__value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
}

.home-user__value--primary {
  color: var(--ui-color-primary);
}

.home-user__actions {
  margin-top: var(--ui-space-md);
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.home-section__title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  margin-bottom: var(--ui-space-md);
  margin-top: var(--ui-space-base);
}
</style>
