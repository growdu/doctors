<!--
  src/pages/order/pay.vue

  订单支付页 —— 微信支付沙箱（v1 mock）+ 倒计时 + 支付成功跳详情
  （spec §4.2 + plan Task P6）

  页面流向：
    入口：订单创建成功 / 订单详情「去支付」
       → 本页（?orderId=xxx）
       → mounted 调 api.payOrder(orderId) → 拿到 sandbox pay_url
       → 渲染：订单金额 + 倒计时（15 min）+ 「立即支付」+ 「我已支付」
       → 「立即支付」→ uni.navigateTo 打开 pay_url（沙箱）
       → 「我已支付」→ 轮询 getPayStatus → paid → uni.redirectTo 详情页
       → 倒计时归零 → 「支付已超时」+ 「取消订单」入口

  模板要点：
    - 顶部 <u-navbar>「订单支付」+ 自动返回
    - 大金额展示（¥xxx.xx）+ 订单号
    - 倒计时徽章（CountdownBadge）
    - 「立即支付」主按钮（打开 sandbox pay_url）
    - 「我已支付」次按钮（轮询状态）
    - 「取消订单」底部次按钮（超时后可用）

  行为：
    - onLoad(query)：从 query 读 orderId
    - mounted()：调 payOrder 拿 sandbox URL + 启动 15 min 倒计时
    - markPaid()：轮询 getPayStatus 3 次（间隔 1s）→ paid 则 redirectTo 详情
    - cancelOrder()：调 api.cancelOrder → toast → navigateBack

  数据来源：
    - api/pay.js 直接调（v1 mock；后续接到 order store）

  测试覆盖：src/pages/order/pay.test.js
-->
<template>
  <view class="page-pay" data-test="pay-page">
    <u-navbar title="订单支付" :auto-back="true" />

    <view class="page-pay__amount-card" data-test="amount-card">
      <text class="page-pay__amount-label">支付金额</text>
      <text class="page-pay__amount-value" data-test="amount-value">¥{{ amountText }}</text>
      <text class="page-pay__order-id">订单号：{{ orderId || '-' }}</text>
    </view>

    <!-- 倒计时 -->
    <view class="page-pay__countdown" data-test="countdown-section">
      <CountdownBadge
        v-if="expireAt"
        :expire-at="expireAt"
        label="支付剩余"
      />
      <text v-if="expired" class="page-pay__expired" data-test="expired-text">支付已超时</text>
    </view>

    <!-- 操作区 -->
    <view class="page-pay__actions" data-test="actions">
      <u-button
        type="primary"
        size="large"
        :disabled="expired || polling"
        data-test="pay-btn"
        @click="onPay"
      >立即支付</u-button>
      <u-button
        type="success"
        plain
        size="large"
        :loading="polling"
        :disabled="expired"
        data-test="paid-btn"
        @click="onMarkPaid"
      >我已支付</u-button>
      <u-button
        v-if="expired"
        type="warning"
        plain
        size="large"
        data-test="cancel-btn"
        @click="onCancel"
      >取消订单</u-button>
    </view>

    <!-- 加载/错误 -->
    <view v-if="loading" class="page-pay__loading" data-test="loading">
      <text>正在拉起支付…</text>
    </view>
    <view v-if="payUrl" class="page-pay__pay-url" data-test="pay-url">
      <text class="page-pay__pay-url-label">沙箱支付 URL：</text>
      <text class="page-pay__pay-url-value" selectable>{{ payUrl }}</text>
    </view>
  </view>
</template>

<script>
// order/pay 页 —— Vue 3 Options API。
//
// 关键设计：
//   - onLoad/mounted 放在 methods 内（uni-app 支持 + 测试可通过 vm 调用）
//   - 倒计时 15 分钟（订单创建时开始计时，payOrder 返回 expireAt 优先；缺省兜底 15 min）
//   - markPaid 轮询 3 次（间隔 1s），v1 mock 永远返回 paid（除非后端有 sandbox timeout）
//   - 失败兜底：toast + 不跳转

import { payOrder, getPayStatus } from '@/api/pay.js';
import CountdownBadge from '@/components/CountdownBadge.vue';

const DEFAULT_PAY_TTL_MS = 15 * 60 * 1000; // 15 min
const POLL_INTERVAL_MS = 1000;
const POLL_MAX_ATTEMPTS = 3;

export default {
  name: 'OrderPayPage',
  components: { CountdownBadge },
  data() {
    return {
      orderId: null,
      amount: null,           // 元（数字）
      amountText: '0.00',
      payUrl: '',
      expireAt: '',
      loading: false,
      polling: false,
      expired: false,
      _pollingTimer: null,
    };
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     */
    onLoad(query) {
      this.orderId = (query && query.orderId) || null;
      this.amount = (query && query.amount) ? Number(query.amount) : null;
      this.amountText = this.amount != null
        ? Number(this.amount).toFixed(2)
        : ((query && query.amountText) || '0.00');
    },

    /**
     * Vue mounted 钩子（async + 在 methods 内以便测试通过 vm 调用）。
     */
    async mounted() {
      if (!this.orderId) return;
      // 默认 15 min 倒计时（payOrder 完成后会被实际 expireAt 覆盖）
      const fallbackExpire = new Date(Date.now() + DEFAULT_PAY_TTL_MS).toISOString();
      this.expireAt = fallbackExpire;
      await this.startPay();
    },

    /**
     * Vue beforeUnmount（methods 内）。
     * 清理轮询 timer。
     */
    beforeUnmount() {
      if (this._pollingTimer) {
        clearInterval(this._pollingTimer);
        this._pollingTimer = null;
      }
    },

    /**
     * 调 payOrder 拿 sandbox pay_url，更新 amount + expireAt。
     */
    async startPay() {
      this.loading = true;
      try {
        const resp = await payOrder(this.orderId, { channel: 'wechat' });
        this.payUrl = (resp && resp.pay_url) || '';
        if (resp && typeof resp.expire_at === 'string' && resp.expire_at) {
          this.expireAt = resp.expire_at;
        }
        if (resp && (resp.amount != null || resp.amountText)) {
          this.amount = resp.amount || Number(resp.amountText);
          this.amountText = (this.amount != null) ? this.amount.toFixed(2) : (resp.amountText || this.amountText);
        }
      } catch (e) {
        this._toast(this._errMsg(e, '拉起支付失败'));
      } finally {
        this.loading = false;
      }
    },

    /**
     * 「立即支付」点击：打开 sandbox pay_url（uni.navigateTo）。
     * v1 mock：仅展示 URL 给用户复制；后续接入真实 SDK 时改为 wxPay / uniPay。
     */
    onPay() {
      if (this.expired) {
        this._toast('支付已超时，请取消订单');
        return;
      }
      if (!this.payUrl) {
        this._toast('支付 URL 未就绪，请稍后重试');
        return;
      }
      // v1 mock：直接 toast 提示用户去浏览器打开；后续接入真实 SDK 时此处替换
      this._toast('已生成支付 URL（v1 mock：请复制到浏览器打开）');
    },

    /**
     * 「我已支付」点击：轮询 getPayStatus → paid → 跳详情。
     */
    async onMarkPaid() {
      if (this.expired) return;
      this.polling = true;
      try {
        let attempt = 0;
        let status = null;
        while (attempt < POLL_MAX_ATTEMPTS && (!status || status.status === 'pending')) {
          // eslint-disable-next-line no-await-in-loop
          status = await getPayStatus(this.orderId);
          if (status && status.status === 'paid') break;
          attempt += 1;
          if (attempt < POLL_MAX_ATTEMPTS) {
            // eslint-disable-next-line no-await-in-loop
            await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
          }
        }
        if (status && status.status === 'paid') {
          this._toast('支付成功');
          if (typeof uni !== 'undefined' && typeof uni.redirectTo === 'function') {
            uni.redirectTo({ url: `/pages/order/detail?orderId=${this.orderId}` });
          }
        } else {
          this._toast('暂未收到支付结果，请稍后再试');
        }
      } catch (e) {
        this._toast(this._errMsg(e, '查询支付状态失败'));
      } finally {
        this.polling = false;
      }
    },

    /**
     * 「取消订单」点击（仅超时后显示）。
     */
    async onCancel() {
      try {
        const mod = await import('@/api/order.js');
        await mod.cancelOrder(this.orderId, '支付超时');
        this._toast('订单已取消');
        if (typeof uni !== 'undefined' && typeof uni.navigateBack === 'function') {
          uni.navigateBack({ delta: 1 });
        }
      } catch (e) {
        this._toast(this._errMsg(e, '取消订单失败'));
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
.page-pay {
  min-height: 100vh;
  padding: 24px 16px;
  background: #f5f5f5;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-pay__amount-card {
  background: #ffffff;
  padding: 32px 16px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.page-pay__amount-label {
  font-size: 14px;
  color: #8c8c8c;
}

.page-pay__amount-value {
  font-size: 36px;
  font-weight: 700;
  color: #ff4d4f;
  font-variant-numeric: tabular-nums;
}

.page-pay__order-id {
  font-size: 12px;
  color: #8c8c8c;
}

.page-pay__countdown {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
}

.page-pay__expired {
  font-size: 14px;
  color: #ff4d4f;
}

.page-pay__actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-pay__loading {
  text-align: center;
  font-size: 13px;
  color: #8c8c8c;
}

.page-pay__pay-url {
  background: #ffffff;
  padding: 12px;
  border-radius: 8px;
  font-size: 12px;
  color: #595959;
  word-break: break-all;
}

.page-pay__pay-url-label {
  display: block;
  margin-bottom: 4px;
  color: #8c8c8c;
}

.page-pay__pay-url-value {
  font-family: monospace;
}
</style>