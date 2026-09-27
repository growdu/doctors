<!--
  src/pages/auth/real-name.vue

  实名认证页 —— 身份证号 + 姓名 + mock 提交（v1 不接 OCR）
  （spec §5.2 + plan Task 7 v1.1 增量）

  页面流向：
    入口：profile 入口 / 订单创建前置（spec §4.1 要求下单前完成实名）
       → 本页
       → 填身份证号 + 姓名 → 「提交认证」
       → auth.submitRealName（v1 mock：服务端不接 OCR，仅校验格式）→ 调回上一页
       → onLoad 支持 query.redirect 跳回指定页（订单创建场景）

  模板要点：
    - 顶部 <u-navbar>「实名认证」+ 自动返回
    - 顶部说明文案：v1 mock 不接 OCR，请手工填写
    - 姓名输入（最多 32 字符）
    - 身份证号输入（18 位）
    - 「提交认证」主按钮（disabled 条件：name 为空 / idCard 非 18 位）
    - loading 态：按钮 loading + 禁用重复提交

  行为：
    - onLoad(query)：从 query 读 redirect；不带时回退到上一页（uni.navigateBack）
    - onSubmit：调 auth.store.submitRealName → 成功后 uni.navigateBack（无 redirect 时）或 uni.redirectTo(redirect)
    - 失败 → toast 提示，保留输入

  数据来源：
    - auth store：useAuthStore().submitRealName / fetchMe

  测试覆盖：src/pages/auth/real-name.test.js
-->
<template>
  <view class="page-real-name" data-test="real-name-page">
    <u-navbar title="实名认证" :auto-back="true" />

    <view class="page-real-name__hero" data-test="hero">
      <text class="page-real-name__title">请填写实名信息</text>
      <text class="page-real-name__subtitle">v1 mock：不接 OCR，请手工填写身份证号</text>
    </view>

    <view class="page-real-name__form" data-test="form">
      <view class="page-real-name__field">
        <text class="page-real-name__label">姓名</text>
        <input
          v-model="name"
          class="page-real-name__input"
          type="text"
          maxlength="32"
          placeholder="请输入真实姓名"
          data-test="name-input"
        />
      </view>

      <view class="page-real-name__field">
        <text class="page-real-name__label">身份证号</text>
        <input
          v-model="idCard"
          class="page-real-name__input"
          type="idcard"
          maxlength="18"
          placeholder="18 位身份证号"
          data-test="idcard-input"
        />
      </view>

      <u-button
        type="primary"
        :loading="submitting"
        :disabled="!canSubmit"
        data-test="submit-btn"
        @click="onSubmit"
      >提交认证</u-button>
    </view>
  </view>
</template>

<script>
// real-name 页 —— Vue 3 Options API（与项目既有页面风格统一）。
//
// 关键设计：
//   - onLoad 放在 methods 内（uni-app 支持 + 测试可通过 vm.onLoad 直接调用）
//   - 18 位身份证号正则：/^\d{17}[\dXx]$/（末位可为数字或 X/x）
//   - 提交后跳回策略：query.redirect 优先 → 否则 uni.navigateBack()
//
// 测试策略见 real-name.test.js。

import { useAuthStore } from '@/stores/auth.js';

const IDCARD_RE = /^\d{17}[\dXx]$/;

export default {
  name: 'RealNamePage',
  data() {
    return {
      name: '',
      idCard: '',
      redirect: '',
      submitting: false,
    };
  },
  computed: {
    idCardValid() {
      return IDCARD_RE.test(String(this.idCard || ''));
    },
    canSubmit() {
      return String(this.name || '').trim().length > 0
        && this.idCardValid
        && !this.submitting;
    },
    authStore() {
      return useAuthStore();
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     * 从 query 读 redirect（订单创建前置场景会用）。
     * @param {object} query
     */
    onLoad(query) {
      this.redirect = (query && query.redirect) || '';
    },

    /**
     * 「提交认证」点击：调 store.submitRealName → 成功后跳回。
     */
    async onSubmit() {
      if (!this.canSubmit) return;
      this.submitting = true;
      try {
        await this.authStore.submitRealName({
          name: this.name.trim(),
          id_card: this.idCard.trim(),
        });
        // 刷新 user（real_name_verified 字段会被后端置 true）
        try {
          await this.authStore.fetchMe();
        } catch (_e) {
          // fetchMe 失败不阻塞流程（提交已成功）
        }
        this._toast('实名认证成功');
        this._goBack();
      } catch (e) {
        this._toast(this._errMsg(e, '认证失败，请重试'));
      } finally {
        this.submitting = false;
      }
    },

    /** 跳回：redirect 优先 → 否则 navigateBack */
    _goBack() {
      if (typeof uni === 'undefined') return;
      if (this.redirect) {
        if (typeof uni.redirectTo === 'function') {
          uni.redirectTo({ url: this.redirect });
        }
        return;
      }
      if (typeof uni.navigateBack === 'function') {
        uni.navigateBack({ delta: 1 });
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
.page-real-name {
  min-height: 100vh;
  padding: 24px;
  background: #ffffff;
  box-sizing: border-box;
}

.page-real-name__hero {
  margin-bottom: 24px;
}

.page-real-name__title {
  display: block;
  font-size: 22px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-real-name__subtitle {
  display: block;
  margin-top: 8px;
  font-size: 13px;
  color: #8c8c8c;
}

.page-real-name__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-real-name__field {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #e5e5e5;
  padding: 12px 0;
}

.page-real-name__label {
  width: 80px;
  font-size: 14px;
  color: #595959;
}

.page-real-name__input {
  flex: 1;
  font-size: 16px;
  color: #1f1f1f;
}
</style>