<script setup lang="ts">
/**
 * patient/profile/index.vue — 患者域个人中心（v2 unified-app）。
 *
 * 入口：patient 域首页底部 Tab「我的」
 *   → 渲染用户信息卡 + 6 模块入口菜单
 *
 * 设计要点：
 *   - 顶部用户卡：头像占位 + 手机号 + 实名状态 + 当前激活角色
 *   - 6 菜单：地址管理 / 我的订单 / 我的优惠券 / 钱包 / 消息中心 / 设置
 *   - 底部退出登录按钮
 *   - 复用 UiCard 渲染菜单项
 */
import { computed, onMounted } from 'vue';
import { useAuthStore } from '@/store/auth';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';
import { ref } from 'vue';

const auth = useAuthStore();
const showLogoutModal = ref(false);

const ROLE_LABEL: Record<string, string> = {
  patient: '患者',
  escort: '陪诊师',
  super_admin: '超级管理员',
  order_admin: '订单管理员',
  refund_admin: '退款管理员',
  cs: '客服',
  audit_admin: '审核管理员',
  viewer: '只读账号',
};

const activeRole = computed(() => auth.activeRole || '');
const activeRoleLabel = computed(() => ROLE_LABEL[activeRole.value] || activeRole.value);

async function onLoad() {
  // 刷新 me 信息
  try {
    await auth.refreshMe();
  } catch (e) {
    // store.error 已记录
  }
}

interface MenuItem {
  icon: string;
  label: string;
  path: string;
  testId: string;
}

const MENU: MenuItem[] = [
  { icon: '📍', label: '地址管理', path: '/pages/patient/address/list', testId: 'profile-menu-address' },
  { icon: '📋', label: '我的订单', path: '/pages/patient/order/list', testId: 'profile-menu-orders' },
  { icon: '🎁', label: '我的优惠券', path: '/pages/patient/coupons/index', testId: 'profile-menu-coupons' },
  { icon: '💰', label: '我的钱包', path: '/pages/patient/wallet/index', testId: 'profile-menu-wallet' },
  { icon: '💬', label: '消息中心', path: '/pages/patient/message/list', testId: 'profile-menu-messages' },
  { icon: '⚙️', label: '设置', path: '/pages/patient/settings/index', testId: 'profile-menu-settings' },
];

function onMenuClick(m: MenuItem) {
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: m.path });
  }
}

function onSwitchRole() {
  // 复用 HomeShell 的 RoleSwitcherModal（打开）
  // 这里直接跳回 home shell 即可
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/home/index' });
  }
}

function onLogoutAsk() {
  showLogoutModal.value = true;
}

function onLogoutConfirm() {
  showLogoutModal.value = false;
  auth.onUnauthorized();
}

function onLogoutCancel() {
  showLogoutModal.value = false;
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-profile">
    <!-- 用户卡 -->
    <UiCard v-if="auth.user" data-testid="profile-user-card">
      <view class="profile__user-row">
        <view class="profile__avatar" data-testid="profile-avatar">
          {{ (auth.user.phone || '?').slice(-4) }}
        </view>
        <view class="profile__user-info">
          <text class="profile__phone" data-testid="profile-phone">{{ auth.user.phone }}</text>
          <view class="profile__badges">
            <text
              v-if="auth.user.real_name_verified"
              class="profile__badge profile__badge--verified"
              data-testid="profile-verified-badge"
            >
              ✓ 已实名
            </text>
            <text v-else class="profile__badge profile__badge--unverified" data-testid="profile-unverified-badge">
              未实名
            </text>
            <text class="profile__badge profile__badge--role" data-testid="profile-role-badge">
              {{ activeRoleLabel }}
            </text>
          </view>
        </view>
      </view>
      <view v-if="auth.roles.length > 1" class="profile__switch-row">
        <UiButton
          type="default"
          size="sm"
          data-testid="profile-switch-role-btn"
          @click="onSwitchRole"
        >
          🔄 切换激活角色（{{ auth.roles.length }}）
        </UiButton>
      </view>
    </UiCard>

    <!-- 菜单列表 -->
    <view class="profile__menu" data-testid="profile-menu">
      <UiCard v-for="m in MENU" :key="m.path" :no-padding="true">
        <view class="profile__menu-item" :data-testid="m.testId" @click="onMenuClick(m)">
          <text class="profile__menu-icon">{{ m.icon }}</text>
          <text class="profile__menu-label">{{ m.label }}</text>
          <text class="profile__menu-arrow">›</text>
        </view>
      </UiCard>
    </view>

    <!-- 退出登录 -->
    <view class="profile__logout">
      <UiButton
        type="danger"
        block
        data-testid="profile-logout-btn"
        @click="onLogoutAsk"
      >
        退出登录
      </UiButton>
    </view>

    <!-- 退出确认弹层 -->
    <UiModal
      :visible="showLogoutModal"
      title="退出登录"
      content="确定要退出当前账号吗？"
      @update:visible="(v: boolean) => !v && onLogoutCancel()"
    >
      <template #footer>
        <view style="display: flex; gap: 12px;">
          <UiButton type="default" block @click="onLogoutCancel" data-testid="logout-cancel">取消</UiButton>
          <UiButton type="danger" block @click="onLogoutConfirm" data-testid="logout-confirm">退出</UiButton>
        </view>
      </template>
    </UiModal>
  </view>
</template>

<style scoped>
.profile__user-row {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.profile__avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.profile__user-info {
  flex: 1;
  min-width: 0;
}

.profile__phone {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
  margin-bottom: var(--ui-space-xs);
}

.profile__badges {
  display: flex;
  gap: var(--ui-space-xs);
  flex-wrap: wrap;
}

.profile__badge {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
}

.profile__badge--verified {
  background: var(--ui-color-success);
  color: var(--ui-color-text-inverse);
}

.profile__badge--unverified {
  background: var(--ui-color-text-disabled);
  color: var(--ui-color-text-inverse);
}

.profile__badge--role {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
}

.profile__switch-row {
  margin-top: var(--ui-space-md);
}

.profile__menu-item {
  display: flex;
  align-items: center;
  padding: var(--ui-space-md) var(--ui-space-base);
  cursor: pointer;
  transition: background var(--ui-duration-fast);
}

.profile__menu-item:active {
  background: var(--ui-color-bg-hover);
}

.profile__menu-icon {
  font-size: 22px;
  margin-right: var(--ui-space-md);
  width: 28px;
  text-align: center;
}

.profile__menu-label {
  flex: 1;
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
}

.profile__menu-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}

.profile__logout {
  margin-top: var(--ui-space-lg);
}
</style>