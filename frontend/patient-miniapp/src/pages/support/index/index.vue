<!--
  src/pages/support/index/index.vue

  客服 / 帮助中心 —— FAQ accordion + 留言 + 联系电话（plan v1.1）
  （spec §5.3）

  页面流向：
    入口：profile「客服」/ 设置页「联系客服」
       → 本页
       → 渲染：FAQ（accordion 展开/收起）+ 联系电话（call）+ 留言表单
       → 提交留言 → toast「留言成功」+ 清空

  模板要点：
    - 顶部 <u-navbar>「客服」+ 自动返回
    - 联系电话卡片（一键拨打）
    - FAQ accordion（点击展开/收起）
    - 留言表单（textarea） + 「提交留言」按钮

  行为：
    - onLoad(query)：占位（v1 不传参）
    - onToggleFaq(index)：展开/收起
    - onSubmitMessage()：v1 mock → toast + 清空 textarea
    - onCallPhone()：uni.makePhoneCall（v1 mock 仅 toast）

  数据来源：
    - 本地 FAQ 常量
    - 留言接口（v1 mock — 后续 plan 接 /support/messages）
-->
<template>
  <view class="page-support" data-test="support-page">
    <u-navbar title="客服" :auto-back="true" />

    <!-- 联系电话 -->
    <view class="page-support__phone-card" data-test="phone-card">
      <view class="page-support__phone-row">
        <text class="page-support__phone-label">客服热线</text>
        <text class="page-support__phone-value" data-test="phone-number">400-100-1234</text>
      </view>
      <view class="page-support__phone-row">
        <text class="page-support__phone-label">服务时间</text>
        <text class="page-support__phone-value">7×24 小时</text>
      </view>
      <u-button
        type="primary"
        size="small"
        data-test="call-btn"
        @click="onCallPhone"
      >一键拨打</u-button>
    </view>

    <!-- FAQ -->
    <view class="page-support__faq" data-test="faq-section">
      <text class="page-support__section-title">常见问题</text>
      <view
        v-for="(item, idx) in FAQS"
        :key="item.key"
        class="page-support__faq-item"
        :class="{ 'page-support__faq-item--open': openKeys.includes(item.key) }"
        :data-test="'faq-' + item.key"
        :data-faq-key="item.key"
        :data-open="openKeys.includes(item.key)"
      >
        <view
          class="page-support__faq-question"
          @click="onToggleFaq(item.key)"
        >
          <text class="page-support__faq-q">{{ item.question }}</text>
          <text
            class="page-support__faq-toggle"
            :data-test="'toggle-' + item.key"
          >{{ openKeys.includes(item.key) ? '−' : '+' }}</text>
        </view>
        <view
          v-if="openKeys.includes(item.key)"
          class="page-support__faq-answer"
          :data-test="'answer-' + item.key"
        >{{ item.answer }}</view>
      </view>
    </view>

    <!-- 留言 -->
    <view class="page-support__message" data-test="message-section">
      <text class="page-support__section-title">给我们留言</text>
      <textarea
        v-model="message"
        class="page-support__textarea"
        data-test="message-input"
        placeholder="请描述您遇到的问题（10-500 字）"
        :maxlength="500"
      />
      <text class="page-support__counter" data-test="char-counter">{{ message.length }} / 500</text>
      <u-button
        type="primary"
        :disabled="!canSubmitMessage"
        data-test="submit-msg-btn"
        @click="onSubmitMessage"
      >{{ submitting ? '提交中' : '提交留言' }}</u-button>
    </view>
  </view>
</template>

<script>
// 客服 / 帮助中心页 —— Vue 3 Options API。
//
// 关键设计：
//   - FAQ accordion 用 openKeys 数组记录已展开的 key（支持多开）
//   - 留言提交：v1 mock — 直接 toast「留言成功」+ 清空 textarea；
//     后续 plan 接 /support/messages API

const FAQS = [
  {
    key: 'how_to_book',
    question: '如何预约陪诊师？',
    answer: '在首页选择医院 → 服务包 → 填写就诊时间和联系人 → 提交订单 → 选择陪诊师即可。',
  },
  {
    key: 'cancel_order',
    question: '如何取消订单？',
    answer: '在「我的订单」中找到对应订单，点击「取消订单」按钮即可。未支付订单可随时取消，已支付订单会有少量手续费。',
  },
  {
    key: 'refund',
    question: '退款多久到账？',
    answer: '退款申请审核通过后，款项会在 1-3 个工作日内退回原支付账户。',
  },
  {
    key: 'coupon_expire',
    question: '优惠券会过期吗？',
    answer: '优惠券均有有效期，到期未使用将自动作废。请在有效期内使用，具体日期以券详情页为准。',
  },
  {
    key: 'sos',
    question: '紧急呼救如何使用？',
    answer: '在「我的」页面或订单详情中点击「SOS」按钮，长按 1.5 秒即可发出紧急呼救，陪诊师/客服会第一时间响应。',
  },
];

export default {
  name: 'SupportPage',
  data() {
    return {
      FAQS,
      openKeys: [], // 已展开的 FAQ key（多开）
      message: '',
      submitting: false,
    };
  },
  computed: {
    /** 留言校验：长度 10-500 */
    canSubmitMessage() {
      const len = String(this.message || '').trim().length;
      return len >= 10 && len <= 500 && !this.submitting;
    },
  },
  methods: {
    onToggleFaq(key) {
      const idx = this.openKeys.indexOf(key);
      if (idx >= 0) {
        this.openKeys.splice(idx, 1);
      } else {
        this.openKeys.push(key);
      }
    },

    onCallPhone() {
      // v1 mock：toast 提示，不真打；后续 plan 调 uni.makePhoneCall
      this._toast('拨打 400-100-1234（v1 mock）');
      if (typeof uni !== 'undefined' && typeof uni.makePhoneCall === 'function') {
        try {
          uni.makePhoneCall({ phoneNumber: '400-100-1234', fail: () => {} });
        } catch (_e) {
          // 静默：H5 环境通常不支持
        }
      }
    },

    /**
     * 留言：v1 mock —— 直接 toast + 清空。
     * 后续 plan 接 /support/messages POST。
     */
    async onSubmitMessage() {
      if (!this.canSubmitMessage) return;
      this.submitting = true;
      try {
        // v1 mock delay
        await new Promise((r) => setTimeout(r, 100));
        this._toast('留言成功，客服会尽快回复');
        this.message = '';
      } catch (_e) {
        this._toast('提交失败，请重试');
      } finally {
        this.submitting = false;
      }
    },

    _toast(title) {
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title, icon: 'none' });
      }
    },
  },
  onLoad(_query) {
    // 占位
  },
  mounted() {
    // 占位
  },
};
</script>

<style lang="scss" scoped>
.page-support {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 24px;
}

.page-support__phone-card,
.page-support__faq,
.page-support__message {
  background: #ffffff;
  margin: 12px;
  border-radius: 12px;
  padding: 16px;
}

.page-support__phone-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.page-support__phone-label {
  font-size: 13px;
  color: #909399;
}

.page-support__phone-value {
  font-size: 14px;
  color: #303133;
}

.page-support__section-title {
  display: block;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
}

.page-support__faq-item {
  border-bottom: 1px solid #f5f5f5;
  padding: 8px 0;
}

.page-support__faq-item:last-child {
  border-bottom: none;
}

.page-support__faq-question {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0;
}

.page-support__faq-q {
  flex: 1;
  font-size: 14px;
  color: #303133;
}

.page-support__faq-toggle {
  width: 24px;
  height: 24px;
  text-align: center;
  font-size: 18px;
  color: #1989fa;
  font-weight: 600;
}

.page-support__faq-answer {
  padding: 8px 0;
  font-size: 13px;
  color: #606266;
  line-height: 20px;
}

.page-support__textarea {
  width: 100%;
  min-height: 120px;
  font-size: 14px;
  color: #303133;
  padding: 8px;
  background: #f8f9fa;
  border-radius: 8px;
  box-sizing: border-box;
}

.page-support__counter {
  display: block;
  text-align: right;
  font-size: 11px;
  color: #909399;
  margin: 6px 0;
}
</style>
</content>
</invoke>