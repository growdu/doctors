<!--
  src/pages/refund/apply.vue

  退款申请页 —— 退款理由选择 + 金额显示 + 提交
  （spec §4.2 + plan Task P6）

  页面流向：
    入口：订单详情「申请退款」
       → 本页（?orderId=xxx&amount=xxx）
       → mounted 渲染：订单号 + 退款金额 + 6 类理由（u-radio-group）
       → 「提交申请」→ api.applyRefund(orderId, { reason, reason_code })
       → 成功 → toast「申请已提交，请等待审核」 → uni.navigateBack
       → 失败 → toast 错误信息

  模板要点：
    - 顶部 <u-navbar>「申请退款」+ 自动返回
    - 金额卡：¥xxx.xx（红色大数字）+ 订单号
    - 退款理由：6 个 radio 项（spec §4.2 表 4-2）
        1. 服务未开始就取消（reason_code=CANCEL_BEFORE_START）
        2. 陪诊师迟到（reason_code=ESCORT_LATE）
        3. 服务态度问题（reason_code=BAD_SERVICE）
        4. 信息填写错误（reason_code=INFO_WRONG）
        5. 临时有事改约（reason_code=RESCHEDULE）
        6. 其他（reason_code=OTHER → 弹 textarea 让用户填具体原因）
    - 「提交申请」主按钮（disabled 条件：未选 / OTHER 模式未填）
    - 备注 textarea（OTHER 模式展开）

  行为：
    - onLoad(query)：从 query 读 orderId + amount
    - onSubmit：调 api.applyRefund → 成功后 navigateBack

  数据来源：
    - api/refund.js 直接调（v1 mock；后续接入 refund store）

  测试覆盖：src/pages/refund/apply.test.js
-->
<template>
  <view class="page-refund" data-test="refund-page">
    <u-navbar title="申请退款" :auto-back="true" />

    <view class="page-refund__amount-card" data-test="amount-card">
      <text class="page-refund__amount-label">退款金额</text>
      <text class="page-refund__amount-value" data-test="amount-value">¥{{ amountText }}</text>
      <text class="page-refund__order-id">订单号：{{ orderId || '-' }}</text>
    </view>

    <view class="page-refund__reason-card" data-test="reason-card">
      <text class="page-refund__reason-title">请选择退款理由</text>
      <view
        v-for="opt in REASON_OPTIONS"
        :key="opt.code"
        class="page-refund__reason-item"
        :class="{ 'page-refund__reason-item--active': selectedCode === opt.code }"
        :data-test="'reason-' + opt.code"
        :data-reason-code="opt.code"
        @click="onSelectReason(opt)"
      >
        <text class="page-refund__reason-text">{{ opt.label }}</text>
        <text v-if="selectedCode === opt.code" class="page-refund__reason-tick">✓</text>
      </view>

      <view v-if="showOtherTextarea" class="page-refund__other-wrap" data-test="other-wrap">
        <textarea
          v-model="otherReason"
          class="page-refund__other-textarea"
          maxlength="200"
          placeholder="请填写具体原因（10-200 字）"
          data-test="other-textarea"
        />
      </view>
    </view>

    <view class="page-refund__footer">
      <u-button
        type="primary"
        size="large"
        :loading="submitting"
        :disabled="!canSubmit"
        data-test="submit-btn"
        @click="onSubmit"
      >提交申请</u-button>
    </view>
  </view>
</template>

<script>
// refund/apply 页 —— Vue 3 Options API。
//
// 关键设计：
//   - onLoad 放在 methods 内（uni-app 支持 + 测试可通过 vm 调用）
//   - 6 类退款理由 + OTHER 模式弹 textarea
//   - 后端校验：reason 长度 10-200 字符；OTHER 模式强制必填

import { applyRefund } from '@/api/refund.js';

// 退款理由枚举（spec §4.2 表 4-2）
const REASON_OPTIONS = [
  { code: 'CANCEL_BEFORE_START', label: '服务未开始就取消' },
  { code: 'ESCORT_LATE', label: '陪诊师迟到' },
  { code: 'BAD_SERVICE', label: '服务态度问题' },
  { code: 'INFO_WRONG', label: '信息填写错误' },
  { code: 'RESCHEDULE', label: '临时有事改约' },
  { code: 'OTHER', label: '其他' },
];

export default {
  name: 'RefundApplyPage',
  data() {
    return {
      orderId: null,
      amount: null,
      amountText: '0.00',
      selectedCode: '',
      otherReason: '',
      submitting: false,
    };
  },
  computed: {
    REASON_OPTIONS() {
      return REASON_OPTIONS;
    },
    showOtherTextarea() {
      return this.selectedCode === 'OTHER';
    },
    canSubmit() {
      if (this.submitting) return false;
      if (!this.selectedCode) return false;
      if (this.selectedCode === 'OTHER') {
        const len = String(this.otherReason || '').trim().length;
        return len >= 10 && len <= 200;
      }
      return true;
    },
    /** 最终 reason 字符串：OTHER 用 otherReason，否则用 option label */
    reasonText() {
      if (!this.selectedCode) return '';
      if (this.selectedCode === 'OTHER') return String(this.otherReason || '').trim();
      const opt = REASON_OPTIONS.find((o) => o.code === this.selectedCode);
      return opt ? opt.label : '';
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     */
    onLoad(query) {
      this.orderId = (query && query.orderId) || null;
      if (query && query.amount) {
        this.amount = Number(query.amount);
        this.amountText = this.amount.toFixed(2);
      }
    },

    /**
     * 选择退款理由
     * @param {{ code: string, label: string }} opt
     */
    onSelectReason(opt) {
      this.selectedCode = opt.code;
      if (opt.code !== 'OTHER') {
        this.otherReason = '';
      }
    },

    /**
     * 「提交申请」点击：调 api.applyRefund → 成功 navigateBack
     */
    async onSubmit() {
      if (!this.canSubmit) return;
      this.submitting = true;
      try {
        await applyRefund(this.orderId, {
          reason: this.reasonText,
          reason_code: this.selectedCode,
        });
        this._toast('申请已提交，请等待审核');
        if (typeof uni !== 'undefined' && typeof uni.navigateBack === 'function') {
          uni.navigateBack({ delta: 1 });
        }
      } catch (e) {
        this._toast(this._errMsg(e, '提交失败，请重试'));
      } finally {
        this.submitting = false;
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
.page-refund {
  min-height: 100vh;
  padding: 16px;
  padding-bottom: 96px;
  background: #f5f5f5;
  box-sizing: border-box;
}

.page-refund__amount-card {
  background: #ffffff;
  padding: 24px 16px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.page-refund__amount-label {
  font-size: 14px;
  color: #8c8c8c;
}

.page-refund__amount-value {
  font-size: 32px;
  font-weight: 700;
  color: #ff4d4f;
}

.page-refund__order-id {
  font-size: 12px;
  color: #8c8c8c;
}

.page-refund__reason-card {
  background: #ffffff;
  padding: 16px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.page-refund__reason-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f1f1f;
  margin-bottom: 8px;
}

.page-refund__reason-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  cursor: pointer;
}

.page-refund__reason-item--active {
  border-color: #1677ff;
  background: #e6f4ff;
}

.page-refund__reason-text {
  font-size: 14px;
  color: #1f1f1f;
}

.page-refund__reason-tick {
  color: #1677ff;
  font-size: 18px;
  font-weight: 700;
}

.page-refund__other-wrap {
  margin-top: 8px;
}

.page-refund__other-textarea {
  width: 100%;
  min-height: 80px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  padding: 8px;
  font-size: 14px;
  box-sizing: border-box;
}

.page-refund__footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 12px 16px;
  background: #ffffff;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
}
</style>