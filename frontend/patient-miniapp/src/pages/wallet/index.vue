<!--
  src/pages/wallet/index.vue

  钱包首页 —— 余额 + 冻结 + 流水（Pinia wallet store）
  （spec §4.6 + plan Task M2）

  页面流向：
    入口：profile「我的钱包」
       → 本页
       → mounted 调 store.loadWallet() + store.loadTransactions()
       → 渲染：余额 + 冻结金额 + 流水列表
       → 下拉刷新 / 触底加载更多（v1 mock：仅展示首页 20 条）

  模板要点：
    - 顶部 <u-navbar>「我的钱包」+ 自动返回
    - 余额卡：大字 ¥xxx.xx（蓝色）+ 冻结金额 chip
    - 流水 section：u-list（type 颜色映射：recharge=绿 / payment=红 / refund=蓝 / withdraw=灰）
    - loading / empty / error 三态机

  行为：
    - onLoad(query)：query.redirect（登录前跳转场景）
    - mounted()：并发拉 wallet + transactions

  数据来源：
    - useWalletStore().wallet / .transactions / .pagination / .loading / .loadingTx

  测试覆盖：src/pages/wallet/index.test.js
-->
<template>
  <view class="page-wallet" data-test="wallet-page">
    <u-navbar title="我的钱包" :auto-back="true" />

    <!-- 余额卡 -->
    <view v-if="loading && !wallet" class="page-wallet__balance-loading" data-test="balance-loading">
      <u-skeleton :rows="3" :title="true" />
    </view>
    <view v-else-if="loadError" class="page-wallet__balance-error" data-test="balance-error">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>
    <view v-else-if="wallet" class="page-wallet__balance-card" data-test="balance-card">
      <text class="page-wallet__balance-label">可用余额（元）</text>
      <text class="page-wallet__balance-value" data-test="balance-value">¥{{ formatAmount(wallet.balance) }}</text>
      <view class="page-wallet__balance-frozen" data-test="balance-frozen">
        <text class="page-wallet__balance-frozen-label">冻结金额</text>
        <text class="page-wallet__balance-frozen-value">¥{{ formatAmount(wallet.frozen) }}</text>
      </view>
    </view>

    <!-- 流水 -->
    <view class="page-wallet__transactions" data-test="transactions">
      <text class="page-wallet__transactions-title">流水</text>
      <view v-if="loadingTx && transactions.length === 0" class="page-wallet__tx-loading" data-test="tx-loading">
        <u-skeleton :rows="3" />
      </view>
      <view v-else-if="transactions.length === 0" class="page-wallet__tx-empty" data-test="tx-empty">
        <u-empty text="暂无流水" mode="list" />
      </view>
      <view v-else class="page-wallet__tx-list">
        <view
          v-for="tx in transactions"
          :key="tx.id"
          class="page-wallet__tx-item"
          :data-test="'tx-' + tx.id"
        >
          <view class="page-wallet__tx-main">
            <text class="page-wallet__tx-desc">{{ tx.description || tx.type }}</text>
            <text class="page-wallet__tx-time">{{ formatTime(tx.created_at) }}</text>
          </view>
          <text
            class="page-wallet__tx-amount"
            :class="'page-wallet__tx-amount--' + (tx.amount >= 0 ? 'positive' : 'negative')"
            :data-test="'tx-amount-' + tx.id"
          >{{ tx.amount >= 0 ? '+' : '' }}{{ formatAmount(tx.amount) }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
// wallet/index 页 —— Vue 3 Options API。
//
// 关键设计：
//   - onLoad/mounted 放在 methods 内（uni-app 支持 + 测试可通过 vm 调用）
//   - 并发拉 wallet + transactions（Promise.all）
//   - 流水金额正负着色：recharge/refund → 绿 / payment/withdraw → 红

import { reactive } from 'vue';
import { useWalletStore } from '@/stores/wallet.js';

const TX_TYPE_COLORS = {
  recharge: '#52c41a',
  payment: '#ff4d4f',
  refund: '#1677ff',
  withdraw: '#8c8c8c',
};

export default {
  name: 'WalletIndexPage',
  data() {
    return {
      loadError: false,
    };
  },
  computed: {
    walletStore() {
      return useWalletStore();
    },
    wallet() {
      return this.walletStore.wallet;
    },
    transactions() {
      return this.walletStore.transactions;
    },
    loading() {
      return this.walletStore.loading;
    },
    loadingTx() {
      return this.walletStore.loadingTx;
    },
    txTypeColors() {
      return TX_TYPE_COLORS;
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     */
    onLoad(_query) {
      // 预留：未来支持 redirect（如登录前跳转）
    },

    /**
     * Vue mounted 钩子（async + 在 methods 内）。
     */
    async mounted() {
      await this.fetchAll();
    },

    /**
     * Vue unmounted 钩子（methods 内）。
     */
    unmounted() {
      try { this.walletStore.clearAll(); } catch (_e) {}
    },

    /**
     * 并发拉 wallet + transactions。
     */
    async fetchAll() {
      this.loadError = false;
      try {
        await Promise.all([
          this.walletStore.loadWallet(),
          this.walletStore.loadTransactions({ page: 1, page_size: 20 }),
        ]);
      } catch (_e) {
        this.loadError = true;
      }
    },

    /**
     * 「重试」按钮
     */
    async onRetry() {
      await this.fetchAll();
    },

    /**
     * 金额格式化（元 → 「xxx.xx」）。
     */
    formatAmount(v) {
      if (v == null || Number.isNaN(Number(v))) return '0.00';
      return Number(v).toFixed(2);
    },

    /**
     * 时间格式化（ISO8601 → 「yyyy-MM-dd HH:mm」）。
     */
    formatTime(iso) {
      if (!iso) return '';
      const t = new Date(iso);
      if (Number.isNaN(t.getTime())) return '';
      const pad = (n) => String(n).padStart(2, '0');
      return `${t.getFullYear()}-${pad(t.getMonth() + 1)}-${pad(t.getDate())} ${pad(t.getHours())}:${pad(t.getMinutes())}`;
    },
  },
};
</script>

<style lang="scss" scoped>
.page-wallet {
  min-height: 100vh;
  padding: 16px;
  background: #f5f5f5;
  box-sizing: border-box;
}

.page-wallet__balance-card {
  background: linear-gradient(135deg, #1677ff, #4096ff);
  padding: 24px 16px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
  color: #ffffff;
}

.page-wallet__balance-label {
  font-size: 13px;
  opacity: 0.85;
}

.page-wallet__balance-value {
  font-size: 36px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.page-wallet__balance-frozen {
  display: flex;
  justify-content: space-between;
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.2);
}

.page-wallet__balance-frozen-label {
  font-size: 12px;
  opacity: 0.85;
}

.page-wallet__balance-frozen-value {
  font-size: 14px;
  font-weight: 600;
}

.page-wallet__transactions {
  background: #ffffff;
  border-radius: 8px;
  padding: 16px;
}

.page-wallet__transactions-title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #1f1f1f;
  margin-bottom: 12px;
}

.page-wallet__tx-list {
  display: flex;
  flex-direction: column;
}

.page-wallet__tx-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.page-wallet__tx-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-wallet__tx-desc {
  font-size: 14px;
  color: #1f1f1f;
}

.page-wallet__tx-time {
  font-size: 12px;
  color: #8c8c8c;
}

.page-wallet__tx-amount {
  font-size: 16px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.page-wallet__tx-amount--positive {
  color: #52c41a;
}

.page-wallet__tx-amount--negative {
  color: #ff4d4f;
}
</style>