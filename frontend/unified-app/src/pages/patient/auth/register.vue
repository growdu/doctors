<script setup lang="ts">
/**
 * patient/auth/register.vue — 注册（v2 unified-app）。
 *
 * 入口：auth/login「没有账号？立即注册」
 *   → SMS 验证 + 设置密码（可选）+ 同意协议
 *   → 调 authStore.login()（v2 后端注册即登录：role 自动 patient）
 *
 * 简化策略：v1 patient-miniapp 注册与登录合二为一（SMS 登录即注册）。
 * 本页提供「密码」字段占位（v2 后续接密码注册）。
 */
import { ref, computed } from 'vue';
import { useAuthStore } from '@/store/auth';
import { sendSmsCode } from '@/api/auth';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';

const auth = useAuthStore();

const phone = ref('');
const code = ref('');
const password = ref('');
const confirmPassword = ref('');
const agreedTerms = ref(false);
const smsSending = ref(false);
const countdown = ref(0);
const registerLoading = ref(false);
const error = ref('');

let timer: number | null = null;

const PHONE_RE = /^1\d{10}$/;
const canRegister = computed(() =>
  PHONE_RE.test(phone.value) &&
  code.value.length >= 4 &&
  password.value.length >= 6 &&
  password.value === confirmPassword.value &&
  agreedTerms.value &&
  !registerLoading.value,
);

async function onSendSms() {
  if (!PHONE_RE.test(phone.value)) {
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

async function onRegister() {
  if (!canRegister.value) return;
  registerLoading.value = true;
  error.value = '';
  try {
    // v2 简化：注册 = SMS 登录（后端自动建 patient role）
    await auth.login(phone.value, code.value);
    if (typeof uni !== 'undefined') {
      uni.reLaunch({ url: '/pages/patient/index' });
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    registerLoading.value = false;
  }
}
</script>

<template>
  <view class="ui-page" data-testid="patient-auth-register">
    <UiCard title="注册新账号">
      <UiInput
        v-model="phone"
        label="手机号"
        placeholder="11 位手机号"
        type="tel"
        :maxlength="11"
        clearable
        data-testid="register-phone"
      />

      <view class="register__code-row">
        <view class="register__code-input">
          <UiInput
            v-model="code"
            label="验证码"
            placeholder="6 位验证码"
            type="number"
            :maxlength="6"
            data-testid="register-code"
          />
        </view>
        <view class="register__code-btn-wrap">
          <UiButton
            type="default"
            size="sm"
            :loading="smsSending"
            :disabled="countdown > 0 || smsSending"
            data-testid="register-send-sms-btn"
            @click="onSendSms"
          >
            {{ countdown > 0 ? `${countdown}s` : '发送验证码' }}
          </UiButton>
        </view>
      </view>

      <UiInput
        v-model="password"
        label="设置密码"
        placeholder="至少 6 位"
        type="password"
        :maxlength="32"
        data-testid="register-password"
      />
      <UiInput
        v-model="confirmPassword"
        label="确认密码"
        placeholder="再次输入密码"
        type="password"
        :maxlength="32"
        data-testid="register-confirm-password"
      />

      <view class="register__terms">
        <switch :checked="agreedTerms" @change="(e: any) => (agreedTerms = e.detail.value)" data-testid="register-terms-switch" />
        <text class="register__terms-text" @click="agreedTerms = !agreedTerms">
          我已阅读并同意《服务协议》《隐私政策》
        </text>
      </view>

      <view v-if="error" class="register__error" data-testid="register-error">{{ error }}</view>

      <view class="register__actions">
        <UiButton
          type="primary"
          block
          :loading="registerLoading"
          :disabled="!canRegister"
          data-testid="register-submit-btn"
          @click="onRegister"
        >
          注册并登录
        </UiButton>
      </view>
    </UiCard>
  </view>
</template>

<style scoped>
.register__code-row {
  display: flex;
  align-items: flex-start;
  gap: var(--ui-space-md);
}

.register__code-input {
  flex: 1;
  min-width: 0;
}

.register__code-btn-wrap {
  flex-shrink: 0;
  padding-top: var(--ui-space-base);
}

.register__terms {
  display: flex;
  align-items: center;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-md);
}

.register__terms-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  flex: 1;
}

.register__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-top: var(--ui-space-md);
}

.register__actions {
  margin-top: var(--ui-space-base);
}
</style>