<script setup lang="ts">
/**
 * admin/login/index.vue — admin 后台登录（v2 unified-app · admin 域）。
 *
 * 与 patient/auth/login 区别：
 *   - 多角色提示：admin 后台 6 个角色一键 demo 账号
 *   - 登录后跳 admin 域（多角色自动激活其中一个）
 *
 * 设计要点：
 *   - 6 个 demo 账号一键填入（与 admin-web v1 LoginPage 对齐）
 *   - 真实 SMS 登录（authService）+ 多角色用户（auth-store.roles 包含 admin_*）
 */
import { ref } from 'vue';
import { useAuthStore } from '@/store/auth';
import { sendSmsCode } from '@/api/auth';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';

const auth = useAuthStore();

const phone = ref('');
const code = ref('');
const smsSending = ref(false);
const countdown = ref(0);
const loginLoading = ref(false);
const error = ref('');

let timer: number | null = null;

interface DemoAccount {
  username: string;
  label: string;
  role: string;
  color: string;
}

const DEMO_ACCOUNTS: DemoAccount[] = [
  { username: 'super', label: '超级管理员', role: 'super_admin', color: 'red' },
  { username: 'order', label: '订单管理员', role: 'order_admin', color: 'orange' },
  { username: 'refund', label: '退款管理员', role: 'refund_admin', color: 'gold' },
  { username: 'audit', label: '审核管理员', role: 'audit_admin', color: 'green' },
  { username: 'cs', label: '客服', role: 'cs', color: 'blue' },
  { username: 'viewer', label: '只读观察', role: 'viewer', color: 'default' },
];

async function onSendSms() {
  if (!/^1\d{10}$/.test(phone.value)) {
    error.value = '请输入 11 位手机号';
    return;
  }
  smsSending.value = true;
  error.value = '';
  try {
    await sendSmsCode(phone.value);
    countdown.value = 60;
    timer = setInterval(() => {
      countdown.value -= 1;
      if (countdown.value <= 0 && timer !== null) {
        clearInterval(timer);
        timer = null;
      }
    }, 1000) as unknown as number;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    smsSending.value = false;
  }
}

async function onLogin() {
  if (!/^1\d{10}$/.test(phone.value) || code.value.length < 4) {
    error.value = '请输入完整的手机号与验证码';
    return;
  }
  loginLoading.value = true;
  error.value = '';
  try {
    await auth.login(phone.value, code.value);
    if (typeof uni !== 'undefined') {
      uni.reLaunch({ url: '/pages/admin/dashboard/index' });
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loginLoading.value = false;
  }
}

function onFillDemo(acct: DemoAccount) {
  // v1 admin-web demo: 直接填入测试手机号 + mock code 「000000」
  // 后端 dev seed 0015 提供对应用户（参 dev.md §15）
  phone.value = `1380013${acct.username === 'super' ? '0001' : '8000'}`.slice(0, 11).padEnd(11, '0');
  code.value = '000000';
}
</script>

<template>
  <view class="ui-page" data-testid="admin-login">
    <view class="admin-login__brand">
      <text class="admin-login__brand-icon">🛡️</text>
      <text class="admin-login__brand-name">Doctors Admin</text>
      <text class="admin-login__brand-subtitle">管理后台登录</text>
    </view>

    <UiCard>
      <UiInput
        v-model="phone"
        label="手机号"
        placeholder="11 位手机号"
        type="tel"
        :maxlength="11"
        clearable
        data-testid="admin-login-phone"
      />

      <view class="admin-login__code-row">
        <view class="admin-login__code-input">
          <UiInput
            v-model="code"
            label="验证码"
            placeholder="6 位验证码"
            type="number"
            :maxlength="6"
            data-testid="admin-login-code"
          />
        </view>
        <view class="admin-login__code-btn-wrap">
          <UiButton
            type="default"
            size="sm"
            :loading="smsSending"
            :disabled="countdown > 0 || smsSending"
            data-testid="admin-send-sms-btn"
            @click="onSendSms"
          >
            {{ countdown > 0 ? ` ${countdown}s` : '发送验证码' }}
          </UiButton>
        </view>
      </view>

      <view v-if="error" class="admin-login__error" data-testid="admin-login-error">{{ error }}</view>

      <view class="admin-login__actions">
        <UiButton
          type="primary"
          block
          :loading="loginLoading"
          data-testid="admin-login-btn"
          @click="onLogin"
        >
          登录
        </UiButton>
      </view>
    </UiCard>

    <!-- Demo 账号快捷填充 -->
    <UiCard title="Demo 账号快捷填充" shadow="sm" data-testid="admin-login-demo">
      <view class="admin-login__demo-grid">
        <view
          v-for="acct in DEMO_ACCOUNTS"
          :key="acct.username"
          class="admin-login__demo-item"
          :class="`admin-login__demo-item--${acct.color}`"
          :data-testid="`admin-demo-${acct.username}`"
          @click="onFillDemo(acct)"
        >
          <text class="admin-login__demo-label">{{ acct.label }}</text>
          <text class="admin-login__demo-role">{{ acct.role }}</text>
        </view>
      </view>
    </UiCard>
  </view>
</template>

<style scoped>
.admin-login__brand {
  text-align: center;
  padding: var(--ui-space-lg) 0;
}

.admin-login__brand-icon {
  font-size: 56px;
  display: block;
}

.admin-login__brand-name {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: var(--ui-space-sm);
}

.admin-login__brand-subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-login__code-row {
  display: flex;
  align-items: flex-start;
  gap: var(--ui-space-md);
}

.admin-login__code-input {
  flex: 1;
  min-width: 0;
}

.admin-login__code-btn-wrap {
  flex-shrink: 0;
  padding-top: var(--ui-space-base);
}

.admin-login__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-top: var(--ui-space-md);
}

.admin-login__actions {
  margin-top: var(--ui-space-md);
}

.admin-login__demo-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ui-space-sm);
}

.admin-login__demo-item {
  padding: var(--ui-space-sm) var(--ui-space-md);
  background: var(--ui-color-bg-hover);
  border-radius: var(--ui-radius-md);
  border-left: 4px solid var(--ui-color-primary);
  cursor: pointer;
  transition: background var(--ui-duration-fast);
}

.admin-login__demo-item:active {
  background: var(--ui-color-divider);
}

.admin-login__demo-item--red { border-left-color: #ff4d4f; }
.admin-login__demo-item--orange { border-left-color: #fa8c16; }
.admin-login__demo-item--gold { border-left-color: #faad14; }
.admin-login__demo-item--green { border-left-color: #52c41a; }
.admin-login__demo-item--blue { border-left-color: #1677ff; }
.admin-login__demo-item--default { border-left-color: var(--ui-color-text-disabled); }

.admin-login__demo-label {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
  display: block;
}

.admin-login__demo-role {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
  font-family: monospace;
}
</style>