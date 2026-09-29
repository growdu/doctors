<script setup lang="ts">
/**
 * patient/auth/login.vue — SMS 登录 + 微信登录占位（v2 unified-app）。
 *
 * 入口：profile 退出登录后 / 应用启动未登录态
 *   → 默认 SMS 登录：手机号 + 验证码
 *   → 微信登录：占位（v2.1 接 wxlogin）
 *   → 成功：跳 patient 域首页
 *
 * 设计要点：
 *   - 复用 authStore.login()（v2 多角色）
 *   - 倒计时 60s 防重发
 */
import { ref, computed, onUnmounted } from 'vue';
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

const canLogin = computed(() => /^1\d{10}$/.test(phone.value) && code.value.length >= 4 && !loginLoading.value);

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
  if (!canLogin.value) return;
  loginLoading.value = true;
  error.value = '';
  try {
    await auth.login(phone.value, code.value);
    if (typeof uni !== 'undefined') {
      uni.reLaunch({ url: '/pages/patient/index' });
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loginLoading.value = false;
  }
}

function onWxLogin() {
  // v2.1 接 wxlogin
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: '微信登录即将上线', icon: 'none' });
  }
}

onUnmounted(() => {
  if (timer !== null) clearInterval(timer);
});
</script>

<template>
  <view class="ui-page" data-testid="patient-auth-login">
    <view class="login__brand">
      <text class="login__brand-icon">🏥</text>
      <text class="login__brand-name">Doctors 统一 App</text>
      <text class="login__brand-subtitle">登录开启陪诊服务</text>
    </view>

    <UiCard>
      <UiInput
        v-model="phone"
        label="手机号"
        placeholder="11 位手机号"
        type="tel"
        :maxlength="11"
        clearable
        data-testid="auth-login-phone"
      />

      <view class="login__code-row">
        <view class="login__code-input">
          <UiInput
            v-model="code"
            label="验证码"
            placeholder="6 位验证码"
            type="number"
            :maxlength="6"
            data-testid="auth-login-code"
          />
        </view>
        <view class="login__code-btn-wrap">
          <UiButton
            type="default"
            size="sm"
            :loading="smsSending"
            :disabled="countdown > 0 || smsSending"
            data-testid="auth-send-sms-btn"
            @click="onSendSms"
          >
            {{ countdown > 0 ? `${countdown}s 后重发` : '发送验证码' }}
          </UiButton>
        </view>
      </view>

      <view v-if="error" class="login__error" data-testid="auth-login-error">{{ error }}</view>

      <view class="login__actions">
        <UiButton
          type="primary"
          block
          :loading="loginLoading"
          :disabled="!canLogin"
          data-testid="auth-login-btn"
          @click="onLogin"
        >
          登录
        </UiButton>
        <UiButton
          type="default"
          block
          data-testid="auth-wxlogin-btn"
          @click="onWxLogin"
        >
          💚 微信登录（即将上线）
        </UiButton>
      </view>
    </UiCard>
  </view>
</template>

<style scoped>
.login__brand {
  text-align: center;
  padding: var(--ui-space-xl) 0 var(--ui-space-lg);
}

.login__brand-icon {
  font-size: 56px;
  display: block;
}

.login__brand-name {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: var(--ui-space-sm);
}

.login__brand-subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.login__code-row {
  display: flex;
  align-items: flex-start;
  gap: var(--ui-space-md);
}

.login__code-input {
  flex: 1;
  min-width: 0;
}

.login__code-btn-wrap {
  flex-shrink: 0;
  padding-top: var(--ui-space-base);
}

.login__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-top: var(--ui-space-md);
}

.login__actions {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}
</style>