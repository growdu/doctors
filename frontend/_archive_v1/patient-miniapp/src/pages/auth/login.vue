<!--
  src/pages/auth/login.vue

  登录页 —— 短信验证码 + 微信入口（v1 mock）
  （spec §5.1 + plan Task 7 v1.1 增量）

  页面流向：
    入口：uni.reLaunch(/pages/auth/login)（utils/request.js 401 拦截 / auth.logout）
       → 本页
       → 填手机号 + 点「获取验证码」→ uni-app 60s 倒计时（复用 CountdownBadge）
       → 填 6 位验证码 → 「登录」→ auth.loginByPhone → 跳 /pages/index/index
       → 「微信登录」→ auth.loginByWechat(mock) → 跳 /pages/index/index

  模板要点：
    - 顶部 <u-navbar>「登录」+ 自动返回（隐藏）
    - Logo + 标题区
    - 手机号输入（11 位中国大陆手机号）
    - 验证码输入（6 位）+ 「获取验证码」按钮（60s 倒计时复用 CountdownBadge）
    - 「登录」主按钮（disabled 条件：phone 无效 / code 不全）
    - 「微信登录」次按钮（v1 占位「WechatAuth」mock）
    - 「服务协议」+「隐私政策」占位

  行为：
    - onLoad(query)：从 query 读 redirect（登录后跳回哪）
    - methods.onLoad：在测试中可显式调用（vue3-jest 不会把 top-level onLoad 暴露到 vm）
    - sendCode：60s 倒计时（复用 CountdownBadge）；调用 auth.sendSmsCode
    - loginByPhone：调 auth.loginByPhone（store）→ 跳 redirect 或 /pages/index/index
    - loginByWechat：调 auth.loginByWechat（store）→ 同上
    - 失败 → toast 提示，不跳转

  数据来源：
    - auth store：useAuthStore().loginByPhone / loginByWechat

  测试覆盖：src/pages/auth/login.test.js
-->
<template>
  <view class="page-login" data-test="login-page">
    <u-navbar title="登录" :auto-back="false" />

    <view class="page-login__hero">
      <text class="page-login__title">陪诊患者端</text>
      <text class="page-login__subtitle">首次登录自动注册账号</text>
    </view>

    <view class="page-login__form" data-test="login-form">
      <!-- 手机号 -->
      <view class="page-login__field">
        <text class="page-login__label">手机号</text>
        <input
          v-model="phone"
          class="page-login__input"
          type="number"
          maxlength="11"
          placeholder="请输入 11 位手机号"
          data-test="phone-input"
        />
      </view>

      <!-- 验证码 -->
      <view class="page-login__field page-login__field--code">
        <text class="page-login__label">验证码</text>
        <input
          v-model="code"
          class="page-login__input page-login__input--code"
          type="number"
          maxlength="6"
          placeholder="6 位验证码"
          data-test="code-input"
        />
        <view class="page-login__send-btn-wrap">
          <u-button
            v-if="smsCooldownSeconds <= 0"
            type="primary"
            size="small"
            plain
            :disabled="!canSendSms"
            data-test="send-code-btn"
            @click="onSendCode"
          >获取验证码</u-button>
          <view v-else class="page-login__cooldown" data-test="cooldown">
            <CountdownBadge
              :expire-at="smsCooldownExpireAt"
              label=""
            />
          </view>
        </view>
      </view>

      <!-- 登录按钮 -->
      <u-button
        type="primary"
        :disabled="!canLogin"
        data-test="login-btn"
        @click="onLogin"
      >登录</u-button>

      <!-- 微信登录 -->
      <u-button
        type="success"
        plain
        class="page-login__wechat"
        data-test="wechat-btn"
        @click="onWechatLogin"
      >微信登录（v1 mock）</u-button>

      <!-- 协议占位 -->
      <view class="page-login__agreement" data-test="agreement">
        <text class="page-login__agreement-text">登录即代表同意《服务协议》与《隐私政策》</text>
      </view>
    </view>
  </view>
</template>

<script>
// login 页 —— Vue 3 Options API（与项目既有页面风格统一）。
//
// 关键设计：
//   - onLoad 放在 methods 内（uni-app 支持 + 测试可通过 vm.onLoad 直接调用）
//   - 短信 60s 倒计时复用 CountdownBadge：服务端下发后本地立刻写入 smsCooldownExpireAt
//   - 登录成功后：useAuthStore().loginByPhone → uni.reLaunch(redirect || /pages/index/index)
//
// 测试策略见 login.test.js。

import { useAuthStore } from '@/stores/auth.js';
import CountdownBadge from '@/components/CountdownBadge.vue';

// 短信冷却秒数（v1 固定 60；后续按后端 ttl 走）
const SMS_COOLDOWN_SECONDS = 60;

export default {
  name: 'LoginPage',
  components: { CountdownBadge },
  data() {
    return {
      phone: '',
      code: '',
      // 倒计时截止时刻（ISO8601），>= smsCooldownExpireAt 时禁用发送按钮
      smsCooldownExpireAt: '',
      smsCooldownSeconds: 0,
      // uni-app Page 钩子：从 query 读 redirect
      redirect: '',
      // 防重复提交锁
      _submitting: false,
    };
  },
  computed: {
    canSendSms() {
      return /^1\d{10}$/.test(String(this.phone || ''));
    },
    canLogin() {
      return this.canSendSms && /^\d{6}$/.test(String(this.code || '')) && !this._submitting;
    },
    authStore() {
      return useAuthStore();
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     * 放在 methods 内 → 测试可通过 wrapper.vm.onLoad({ redirect }) 直接调用。
     * @param {object} query
     */
    onLoad(query) {
      this.redirect = (query && query.redirect) || '';
    },

    /**
     * 「获取验证码」点击：调 store.sendSmsCode 触发后端下发，60s 冷却。
     */
    async onSendCode() {
      if (!this.canSendSms) {
        this._toast('请输入正确的手机号');
        return;
      }
      try {
        await this.authStore.sendSmsCode(this.phone);
        // 写入 60s 倒计时截止时刻
        const expire = new Date(Date.now() + SMS_COOLDOWN_SECONDS * 1000).toISOString();
        this.smsCooldownExpireAt = expire;
        this.smsCooldownSeconds = SMS_COOLDOWN_SECONDS;
        this._toast('验证码已发送（v1 mock：123456）');
      } catch (e) {
        this._toast(this._errMsg(e, '验证码下发失败'));
      }
    },

    /**
     * 「登录」点击：调 store.loginByPhone → 成功跳 redirect || /pages/index/index
     */
    async onLogin() {
      if (!this.canLogin) return;
      this._submitting = true;
      try {
        await this.authStore.loginByPhone(this.phone, this.code);
        this._goAfterLogin();
      } catch (e) {
        this._toast(this._errMsg(e, '登录失败，请重试'));
      } finally {
        this._submitting = false;
      }
    },

    /**
     * 「微信登录」点击：v1 mock「WechatAuth」入口 → store.loginByWechat
     */
    async onWechatLogin() {
      this._submitting = true;
      try {
        await this.authStore.loginByWechat({
          code: 'WechatAuth',
          nickname: '微信用户',
          avatar: '',
        });
        this._goAfterLogin();
      } catch (e) {
        this._toast(this._errMsg(e, '微信登录失败'));
      } finally {
        this._submitting = false;
      }
    },

    /** 登录成功 → 跳 redirect 或首页 */
    _goAfterLogin() {
      const url = this.redirect || '/pages/index/index';
      if (typeof uni !== 'undefined' && typeof uni.reLaunch === 'function') {
        uni.reLaunch({ url });
      }
    },

    /** toast 兜底（jest 环境可能没有 uni） */
    _toast(title) {
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title, icon: 'none' });
      }
    },

    /** 从异常对象里取 message */
    _errMsg(e, fallback) {
      if (e && e.message) return e.message;
      return fallback;
    },
  },
};
</script>

<style lang="scss" scoped>
.page-login {
  min-height: 100vh;
  padding: 32px 24px;
  background: #ffffff;
  box-sizing: border-box;
}

.page-login__hero {
  margin-top: 24px;
  margin-bottom: 32px;
}

.page-login__title {
  display: block;
  font-size: 28px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-login__subtitle {
  display: block;
  margin-top: 8px;
  font-size: 14px;
  color: #8c8c8c;
}

.page-login__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-login__field {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #e5e5e5;
  padding: 12px 0;
}

.page-login__field--code {
  gap: 8px;
}

.page-login__label {
  width: 64px;
  font-size: 14px;
  color: #595959;
}

.page-login__input {
  flex: 1;
  font-size: 16px;
  color: #1f1f1f;
}

.page-login__input--code {
  flex: 1;
}

.page-login__send-btn-wrap {
  margin-left: 8px;
}

.page-login__cooldown {
  display: flex;
  align-items: center;
}

.page-login__wechat {
  margin-top: 8px;
}

.page-login__agreement {
  margin-top: 16px;
  text-align: center;
}

.page-login__agreement-text {
  font-size: 12px;
  color: #8c8c8c;
}
</style>