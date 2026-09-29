<!--
  src/pages/login/index/index.vue

  登录页 v2 —— 短信 / 微信 / 6 个 demo 账号一键填（plan v1.1）
  （spec §5.1 + 替代/并行 src/pages/auth/login.vue）

  页面流向：
    入口：uni.reLaunch(/pages/login/index)（utils/request.js 401 拦截 / auth.logout）
       → 本页
       → 填手机号 + 点「获取验证码」→ uni-app 60s 倒计时（复用 CountdownBadge）
       → 填 6 位验证码 → 「登录」→ auth.loginByPhone → 跳 /pages/index/index
       → 「微信登录」→ auth.loginByWechat(mock) → 同上
       → 「6 个 demo 账号」一键 fill（v1 提效：QA / 开发免去手填手机号）

  模板要点：
    - 顶部 <u-navbar>「登录」+ 自动返回（隐藏）
    - Logo + 标题区
    - 手机号输入（11 位中国大陆手机号）
    - 验证码输入（6 位）+ 「获取验证码」按钮（60s 倒计时复用 CountdownBadge）
    - 「登录」主按钮
    - 「微信登录」次按钮（v1 占位「WechatAuth」mock）
    - 6 个 demo 账号列表（一键填手机号 / 验证码）

  行为：
    - onLoad(query)：从 query 读 redirect（登录后跳回哪）
    - sendCode：60s 倒计时（复用 CountdownBadge）；调用 auth.sendSmsCode
    - loginByPhone：调 auth.loginByPhone（store）→ 跳 redirect || /pages/index/index
    - loginByWechat：调 auth.loginByWechat（store）→ 同上
    - 失败 → toast 提示

  数据来源：
    - auth store：useAuthStore().loginByPhone / loginByWechat

  测试覆盖：src/pages/login/index/index.test.js
-->
<template>
  <view class="page-login-v2" data-test="login-page">
    <u-navbar title="登录" :auto-back="false" />

    <view class="page-login-v2__hero">
      <text class="page-login-v2__title">陪诊患者端 v2</text>
      <text class="page-login-v2__subtitle">首次登录自动注册账号</text>
    </view>

    <view class="page-login-v2__form" data-test="login-form">
      <!-- 手机号 -->
      <view class="page-login-v2__field">
        <text class="page-login-v2__label">手机号</text>
        <input
          v-model="phone"
          class="page-login-v2__input"
          type="number"
          maxlength="11"
          placeholder="请输入 11 位手机号"
          data-test="phone-input"
        />
      </view>

      <!-- 验证码 -->
      <view class="page-login-v2__field page-login-v2__field--code">
        <text class="page-login-v2__label">验证码</text>
        <input
          v-model="code"
          class="page-login-v2__input page-login-v2__input--code"
          type="number"
          maxlength="6"
          placeholder="6 位验证码"
          data-test="code-input"
        />
        <view class="page-login-v2__send-btn-wrap">
          <u-button
            v-if="smsCooldownSeconds <= 0"
            type="primary"
            size="small"
            plain
            :disabled="!canSendSms"
            data-test="send-code-btn"
            @click="onSendCode"
          >获取验证码</u-button>
          <view v-else class="page-login-v2__cooldown" data-test="cooldown">
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
        class="page-login-v2__wechat"
        data-test="wechat-btn"
        @click="onWechatLogin"
      >微信登录（v1 mock）</u-button>
    </view>

    <!-- 6 个 demo 账号一键填 -->
    <view class="page-login-v2__demo" data-test="demo-card">
      <text class="page-login-v2__demo-title">Demo 账号（v1 一键填）</text>
      <view class="page-login-v2__demo-list">
        <view
          v-for="acc in DEMO_ACCOUNTS"
          :key="acc.phone"
          class="page-login-v2__demo-row"
          :data-test="'demo-' + acc.label"
          :data-phone="acc.phone"
          @click="onFillDemo(acc)"
        >
          <text class="page-login-v2__demo-label">{{ acc.label }}</text>
          <text class="page-login-v2__demo-phone">{{ acc.phone }}</text>
        </view>
      </view>
    </view>

    <!-- 协议占位 -->
    <view class="page-login-v2__agreement" data-test="agreement">
      <text class="page-login-v2__agreement-text">登录即代表同意《服务协议》与《隐私政策》</text>
    </view>
  </view>
</template>

<script>
// login/index 页 —— Vue 3 Options API（与 src/pages/auth/login.vue 风格统一）。
//
// 关键设计（v1.1 增量）：
//   - 复用 src/pages/auth/login.vue 的短信 / 微信登录逻辑
//   - 新增 6 个 demo 账号一键填（针对 QA + 开发提效）
//   - 路由独立：/pages/login/index 而非 /pages/auth/login——便于外部测试链接固定

import { useAuthStore } from '@/stores/auth.js';
import CountdownBadge from '@/components/CountdownBadge.vue';

// 短信冷却秒数（v1 固定 60；后续按后端 ttl 走）
const SMS_COOLDOWN_SECONDS = 60;

// 6 个 demo 账号：覆盖各档位用户类型 + 国家号（CN/海外）；点击自动填 phone + code=123456
const DEMO_ACCOUNTS = [
  { label: '普通用户',   phone: '13800138001' },
  { label: 'VIP 用户',   phone: '13800138002' },
  { label: '陪诊师',     phone: '13800138003' },
  { label: '客服',       phone: '13800138004' },
  { label: '未实名用户', phone: '13800138005' },
  { label: '海外测试',   phone: '8613800138666' }, // 11 位起始 86（中国区号）
];

export default {
  name: 'LoginIndexPage',
  components: { CountdownBadge },
  data() {
    return {
      DEMO_ACCOUNTS,
      phone: '',
      code: '',
      smsCooldownExpireAt: '',
      smsCooldownSeconds: 0,
      redirect: '',
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
        const expire = new Date(Date.now() + SMS_COOLDOWN_SECONDS * 1000).toISOString();
        this.smsCooldownExpireAt = expire;
        this.smsCooldownSeconds = SMS_COOLDOWN_SECONDS;
        this._toast('验证码已发送（v1 mock：123456）');
      } catch (e) {
        this._toast(this._errMsg(e, '验证码下发失败'));
      }
    },

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

    /**
     * v1.1: 点击 demo 账号 → 自动填手机号 + 验证码（123456）。
     * 不直接登录，方便用户核对 UI + 验证码状态。
     */
    onFillDemo(acc) {
      if (!acc || !acc.phone) return;
      this.phone = acc.phone;
      this.code = '123456';
      this._toast(`已填入 demo「${acc.label}」（验证码 123456）`);
    },

    _goAfterLogin() {
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
.page-login-v2 {
  min-height: 100vh;
  padding: 32px 24px;
  background: #ffffff;
  box-sizing: border-box;
}

.page-login-v2__hero {
  margin-top: 24px;
  margin-bottom: 32px;
}

.page-login-v2__title {
  display: block;
  font-size: 28px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-login-v2__subtitle {
  display: block;
  margin-top: 8px;
  font-size: 14px;
  color: #8c8c8c;
}

.page-login-v2__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-login-v2__field {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #e5e5e5;
  padding: 12px 0;
}

.page-login-v2__field--code {
  gap: 8px;
}

.page-login-v2__label {
  width: 64px;
  font-size: 14px;
  color: #595959;
}

.page-login-v2__input {
  flex: 1;
  font-size: 16px;
  color: #1f1f1f;
}

.page-login-v2__input--code {
  flex: 1;
}

.page-login-v2__send-btn-wrap {
  margin-left: 8px;
}

.page-login-v2__cooldown {
  display: flex;
  align-items: center;
}

.page-login-v2__wechat {
  margin-top: 8px;
}

.page-login-v2__demo {
  margin-top: 32px;
  padding: 12px 0;
  border-top: 1px dashed #f0f0f0;
}

.page-login-v2__demo-title {
  display: block;
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}

.page-login-v2__demo-list {
  display: flex;
  flex-direction: column;
}

.page-login-v2__demo-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f5f5f5;
}

.page-login-v2__demo-label {
  font-size: 13px;
  color: #606266;
}

.page-login-v2__demo-phone {
  font-size: 13px;
  color: #1989fa;
  font-variant-numeric: tabular-nums;
}

.page-login-v2__agreement {
  margin-top: 24px;
  text-align: center;
}

.page-login-v2__agreement-text {
  font-size: 12px;
  color: #8c8c8c;
}
</style>
</content>
</invoke>