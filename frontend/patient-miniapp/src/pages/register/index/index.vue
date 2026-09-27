<!--
  src/pages/register/index/index.vue

  注册页 —— 短信验证码 + 协议勾选（plan v1.1）
  （spec §5.1）

  页面流向：
    入口：登录页「没有账号？」/ 外部落地页
       → 本页
       → 填手机号 + 验证码 + 勾选协议 → 「注册」
       → register API（v1 mock：直接登录走 loginByPhone）
       → 跳 /pages/index/index

  模板要点：
    - 顶部 <u-navbar>「注册」+ 自动返回
    - 手机号 + 验证码（复用 CountdownBadge 60s 冷却）
    - 协议 checkbox（勾选后才允许「注册」）
    - 「注册」主按钮（disabled when：phone 无效 / code 缺位 / 未勾选协议）

  行为：
    - onLoad(query)：从 query 读 redirect
    - onSendCode：60s 倒计时
    - onRegister：v1 mock 直接调 store.loginByPhone（首次登录自动注册）

  数据来源：
    - auth store：useAuthStore().loginByPhone / sendSmsCode

  测试覆盖：src/pages/register/index/index.test.js
-->
<template>
  <view class="page-register" data-test="register-page">
    <u-navbar title="注册" :auto-back="true" />

    <view class="page-register__hero">
      <text class="page-register__title">创建账号</text>
      <text class="page-register__subtitle">用手机号一键注册</text>
    </view>

    <view class="page-register__form" data-test="register-form">
      <view class="page-register__field">
        <text class="page-register__label">手机号</text>
        <input
          v-model="phone"
          class="page-register__input"
          type="number"
          maxlength="11"
          placeholder="请输入 11 位手机号"
          data-test="phone-input"
        />
      </view>
      <view class="page-register__field page-register__field--code">
        <text class="page-register__label">验证码</text>
        <input
          v-model="code"
          class="page-register__input page-register__input--code"
          type="number"
          maxlength="6"
          placeholder="6 位验证码"
          data-test="code-input"
        />
        <view class="page-register__send-btn-wrap">
          <u-button
            v-if="smsCooldownSeconds <= 0"
            type="primary"
            size="small"
            plain
            :disabled="!canSendSms"
            data-test="send-code-btn"
            @click="onSendCode"
          >获取验证码</u-button>
          <view v-else class="page-register__cooldown" data-test="cooldown">
            <CountdownBadge
              :expire-at="smsCooldownExpireAt"
              label=""
            />
          </view>
        </view>
      </view>

      <!-- 协议 -->
      <view class="page-register__agreement" data-test="agreement-row">
        <view
          class="page-register__checkbox"
          :class="{ 'page-register__checkbox--checked': agreed }"
          :data-test="agreed ? 'agreed-checkbox' : 'disagree-checkbox'"
          :data-agreed="agreed"
          @click="agreed = !agreed"
        >{{ agreed ? '✓' : '' }}</view>
        <text class="page-register__agreement-text">
          我已阅读并同意
          <text class="page-register__agreement-link" data-test="agreement-link">《服务协议》</text>
          与
          <text class="page-register__agreement-link" data-test="privacy-link">《隐私政策》</text>
        </text>
      </view>

      <u-button
        type="primary"
        :disabled="!canRegister"
        data-test="register-btn"
        @click="onRegister"
      >{{ submitting ? '注册中' : '注册' }}</u-button>
    </view>
  </view>
</template>

<script>
// register 页 —— Vue 3 Options API（与 src/pages/auth/login.vue 风格统一）。
//
// 关键设计：
//   - v1 后端首次登录自动建账号；本页面复用 store.loginByPhone 作为 register 入口
//   - 协议必须勾选 → canRegister 才为 true
//   - onLoad 读 ?redirect=xxx → 注册后跳回

import { useAuthStore } from '@/stores/auth.js';
import CountdownBadge from '@/components/CountdownBadge.vue';

const SMS_COOLDOWN_SECONDS = 60;

export default {
  name: 'RegisterPage',
  components: { CountdownBadge },
  data() {
    return {
      phone: '',
      code: '',
      smsCooldownExpireAt: '',
      smsCooldownSeconds: 0,
      agreed: false,
      redirect: '',
      submitting: false,
    };
  },
  computed: {
    canSendSms() {
      return /^1\d{10}$/.test(String(this.phone || ''));
    },
    canRegister() {
      return this.canSendSms
        && /^\d{6}$/.test(String(this.code || ''))
        && this.agreed
        && !this.submitting;
    },
    authStore() {
      return useAuthStore();
    },
  },
  methods: {
    onLoad(query) {
      this.redirect = (query && query.redirect) || '';
    },

    async onSendCode() {
      if (!this.canSendSms) {
        this._toast('请输入正确的手机号');
        return;
      }
      try {
        await this.authStore.sendSmsCode(this.phone);
        this.smsCooldownExpireAt = new Date(Date.now() + SMS_COOLDOWN_SECONDS * 1000).toISOString();
        this.smsCooldownSeconds = SMS_COOLDOWN_SECONDS;
        this._toast('验证码已发送（v1 mock：123456）');
      } catch (e) {
        this._toast(this._errMsg(e, '验证码下发失败'));
      }
    },

    /**
     * v1 注册：复用 loginByPhone（首次登录自动注册）。
     * 后续 plan 接 register API 后改走 /auth/register。
     */
    async onRegister() {
      if (!this.canRegister) return;
      this.submitting = true;
      try {
        await this.authStore.loginByPhone(this.phone, this.code);
        this._toast('注册成功');
        this._goAfterRegister();
      } catch (e) {
        this._toast(this._errMsg(e, '注册失败，请重试'));
      } finally {
        this.submitting = false;
      }
    },

    _goAfterRegister() {
      const url = this.redirect || '/pages/index/index';
      if (typeof uni !== 'undefined' && typeof uni.reLaunch === 'function') {
        uni.reLaunch({ url });
      }
    },

    _toast(title) {
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title, icon: 'none' });
      }
    },

    _errMsg(e, fallback) {
      if (e && e.message) return e.message;
      return fallback;
    },
  },
};
</script>

<style lang="scss" scoped>
.page-register {
  min-height: 100vh;
  padding: 32px 24px;
  background: #ffffff;
  box-sizing: border-box;
}

.page-register__hero {
  margin-top: 24px;
  margin-bottom: 32px;
}

.page-register__title {
  display: block;
  font-size: 28px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-register__subtitle {
  display: block;
  margin-top: 8px;
  font-size: 14px;
  color: #8c8c8c;
}

.page-register__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-register__field {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #e5e5e5;
  padding: 12px 0;
}

.page-register__field--code {
  gap: 8px;
}

.page-register__label {
  width: 64px;
  font-size: 14px;
  color: #595959;
}

.page-register__input {
  flex: 1;
  font-size: 16px;
  color: #1f1f1f;
}

.page-register__input--code {
  flex: 1;
}

.page-register__send-btn-wrap {
  margin-left: 8px;
}

.page-register__cooldown {
  display: flex;
  align-items: center;
}

.page-register__agreement {
  display: flex;
  align-items: flex-start;
  padding: 8px 0;
  gap: 8px;
}

.page-register__checkbox {
  width: 18px;
  height: 18px;
  border: 1px solid #dcdfe6;
  border-radius: 3px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: #fff;
  background: #fff;
}

.page-register__checkbox--checked {
  background: #1989fa;
  border-color: #1989fa;
}

.page-register__agreement-text {
  font-size: 12px;
  color: #606266;
  line-height: 18px;
  flex: 1;
}

.page-register__agreement-link {
  color: #1989fa;
  text-decoration: underline;
}
</style>
</content>
</invoke>