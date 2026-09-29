<script setup lang="ts">
/**
 * escort/login.vue — 陪诊师登录页（v2 unified-app · escort 域）。
 *
 * 入口：escort splash 未登录跳转 / 主动重新登录
 *   → 输入手机号 + 6 位验证码
 *   → 调 sendSmsCode + loginByPhone（authService）
 *   → 成功后 refreshMe + 跳 /pages/escort/invitations
 *
 * 设计要点：
 *   - 复用 authStore.login（patient/login 已有逻辑）
 *   - esc 域特定：默认发送 SMS 后假设激活角色为 escort（运营分发用户时已绑定）
 *   - 验证码倒计时 60s
 */
import { ref } from 'vue';
import { useAuthStore } from '@/store/auth';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';
import { sendSmsCode } from '@/api/auth';

const auth = useAuthStore();

const phone = ref('');
const code = ref('');
const sendingCode = ref(false);
const submitting = ref(false);
const countdown = ref(0);
const errorMsg = ref<string | null>(null);

function validatePhone(p: string): boolean {
  return /^\d{11}$/.test(p);
}

function startCountdown() {
  countdown.value = 60;
  const interval = setInterval(() => {
    countdown.value -= 1;
    if (countdown.value <= 0) clearInterval(interval);
  }, 1000);
}

async function onSendCode() {
  errorMsg.value = null;
  if (!validatePhone(phone.value.trim())) {
    errorMsg.value = '请输入 11 位手机号';
    return;
  }
  sendingCode.value = true;
  try {
    await sendSmsCode(phone.value.trim());
    startCountdown();
  } catch (e) {
    errorMsg.value = `发送失败：${(e as Error).message}`;
  } finally {
    sendingCode.value = false;
  }
}

async function onLogin() {
  errorMsg.value = null;
  if (!validatePhone(phone.value.trim())) {
    errorMsg.value = '请输入 11 位手机号';
    return;
  }
  if (code.value.trim().length !== 6) {
    errorMsg.value = '请输入 6 位验证码';
    return;
  }
  submitting.value = true;
  try {
    await auth.login(phone.value.trim(), code.value.trim());
    if (typeof uni !== 'undefined') {
      uni.reLaunch({ url: '/pages/escort/invitations/index' });
    }
  } catch (e) {
    errorMsg.value = `登录失败：${(e as Error).message}`;
  } finally {
    submitting.value = false;
  }
}

function onGoHome() {
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/home/index' });
  }
}
</script>

<template>
  <view class="ui-page" data-testid="escort-login-page">
    <view class="escort-login__header">
      <text class="escort-login__title">🚑 陪诊师登录</text>
      <text class="escort-login__subtitle">使用手机号 + 短信验证码</text>
    </view>

    <UiCard data-testid="escort-login-card">
      <UiInput
        v-model="phone"
        label="手机号"
        placeholder="11 位手机号"
        type="number"
        :maxlength="11"
        data-testid="escort-login-phone"
      />

      <view class="escort-login__code-row">
        <view class="escort-login__code-input">
          <UiInput
            v-model="code"
            label="验证码"
            placeholder="6 位数字"
            type="number"
            :maxlength="6"
            data-testid="escort-login-code"
          />
        </view>
        <UiButton
          size="sm"
          :loading="sendingCode"
          :disabled="countdown > 0"
          data-testid="escort-login-send-code"
          @click="onSendCode"
        >
          {{ countdown > 0 ? `重新获取 (${countdown}s)` : '获取验证码' }}
        </UiButton>
      </view>

      <view v-if="errorMsg" class="escort-login__error" data-testid="escort-login-error">
        <text class="escort-login__error-text">⚠️ {{ errorMsg }}</text>
      </view>
    </UiCard>

    <view class="escort-login__actions">
      <UiButton
        type="primary"
        block
        :loading="submitting"
        data-testid="escort-login-submit"
        @click="onLogin"
      >
        登录
      </UiButton>
      <UiButton
        type="default"
        block
        data-testid="escort-login-back-home"
        @click="onGoHome"
      >
        返回首页选择其他角色
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.escort-login__header {
  padding: var(--ui-space-base) 0;
  text-align: center;
}

.escort-login__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-login__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-login__code-row {
  display: flex;
  align-items: flex-end;
  gap: var(--ui-space-sm);
}

.escort-login__code-input {
  flex: 1;
}

.escort-login__error {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-error);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-top: var(--ui-space-sm);
}

.escort-login__error-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
}

.escort-login__actions {
  margin-top: var(--ui-space-md);
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}
</style>