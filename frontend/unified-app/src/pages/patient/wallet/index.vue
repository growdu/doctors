<script setup lang="ts">
/**
 * patient/wallet/index.vue — 患者钱包（v2 unified-app）。
 *
 * 入口：profile「我的钱包」
 *   → 调 api/wallet.getUserWallet() 拿余额
 *   → 调 api/wallet.listTransactions() 拿流水列表
 *   → 渲染余额 + 4 个统计 + 流水列表
 *
 * 设计要点：
 *   - 余额大字号显示（蓝）
 *   - 4 类型流水颜色映射（income / withdraw / refund / freeze）
 *   - 「充值」按钮占位（v2 暂未接）
 */
import { ref } from 'vue';
import { getUserWallet, listTransactions } from '@/api/wallet';
import type { Wallet, Transaction, TransactionType } from '@/api/wallet';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const wallet = ref<Wallet | null>(null);
const transactions = ref<Transaction[]>([]);
const total = ref(0);
const loading = ref(false);
const error = ref<string | null>(null);

const TYPE_LABEL: Record<TransactionType, string> = {
  income: '收入',
  withdraw: '提现',
  refund: '退款',
  freeze: '冻结',
  unfreeze: '解冻',
};

const TYPE_SIGN: Record<TransactionType, string> = {
  income: '+',
  withdraw: '-',
  refund: '+',
  freeze: '−',
  unfreeze: '+',
};

const TYPE_CLASS: Record<TransactionType, string> = {
  income: 'wallet__tx--income',
  withdraw: 'wallet__tx--withdraw',
  refund: 'wallet__tx--refund',
  freeze: 'wallet__tx--freeze',
  unfreeze: 'wallet__tx--unfreeze',
};

const STATUS_LABEL: Record<string, string> = {
  pending: '处理中',
  success: '成功',
  failed: '失败',
};

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const [w, t] = await Promise.all([getUserWallet(), listTransactions()]);
    wallet.value = w;
    transactions.value = t.items;
    total.value = t.total;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatAmount(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso: string): string {
  return iso.replace('T', ' ').substring(0, 19);
}

function onRecharge() {
  // Phase 3.1 后续接：充值流程
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: '充值功能即将上线', icon: 'none' });
  }
}
</script>

<template>
  <view class="ui-page" data-testid="patient-wallet">
    <view v-if="loading && !wallet" data-testid="wallet-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error && !wallet" data-testid="wallet-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="wallet__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else-if="wallet">
      <!-- 余额卡 -->
      <UiCard data-testid="wallet-balance-card">
        <view class="wallet__balance">
          <text class="wallet__balance-label">可用余额（元）</text>
          <text class="wallet__balance-value" data-testid="wallet-balance">{{ formatAmount(wallet.balance) }}</text>
          <view class="wallet__stats">
            <view class="wallet__stat">
              <text class="wallet__stat-label">冻结</text>
              <text class="wallet__stat-value" data-testid="wallet-frozen">{{ formatAmount(wallet.frozen) }}</text>
            </view>
            <view class="wallet__stat">
              <text class="wallet__stat-label">累计收入</text>
              <text class="wallet__stat-value">{{ formatAmount(wallet.total_earned) }}</text>
            </view>
            <view class="wallet__stat">
              <text class="wallet__stat-label">累计提现</text>
              <text class="wallet__stat-value">{{ formatAmount(wallet.total_withdrawn) }}</text>
            </view>
          </view>
          <view class="wallet__actions">
            <UiButton type="primary" size="sm" data-testid="wallet-recharge-btn" @click="onRecharge">
              充值
            </UiButton>
          </view>
        </view>
      </UiCard>

      <!-- 流水列表 -->
      <view class="wallet__tx-header" data-testid="wallet-tx-header">流水记录（{{ total }}）</view>

      <view v-if="transactions.length === 0" data-testid="wallet-tx-empty">
        <UiEmpty icon="📊" title="暂无流水" description="下单完成后会有收入记录" />
      </view>

      <view v-else data-testid="wallet-tx-list">
        <view
          v-for="t in transactions"
          :key="t.id"
          class="wallet__tx-item"
          :data-testid="`wallet-tx-${t.id}`"
        >
          <view class="wallet__tx-body">
            <text class="wallet__tx-type">{{ TYPE_LABEL[t.type] }}</text>
            <text class="wallet__tx-remark">{{ t.remark || '-' }}</text>
            <text class="wallet__tx-time">{{ formatDate(t.created_at) }}</text>
          </view>
          <view class="wallet__tx-amount" :class="TYPE_CLASS[t.type]">
            <text class="wallet__tx-amount-value">
              {{ TYPE_SIGN[t.type] }}{{ formatAmount(t.amount) }}
            </text>
            <text class="wallet__tx-status">{{ STATUS_LABEL[t.status] }}</text>
          </view>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped>
.wallet__balance {
  text-align: center;
  padding: var(--ui-space-base) 0;
}

.wallet__balance-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-bottom: var(--ui-space-xs);
}

.wallet__balance-value {
  font-size: 36px;
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-primary);
  display: block;
  margin-bottom: var(--ui-space-md);
}

.wallet__stats {
  display: flex;
  justify-content: space-around;
  padding: var(--ui-space-md) 0;
  border-top: 1px solid var(--ui-color-divider);
}

.wallet__stat {
  text-align: center;
}

.wallet__stat-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
}

.wallet__stat-value {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
  margin-top: 2px;
  display: block;
}

.wallet__actions {
  margin-top: var(--ui-space-md);
  text-align: center;
}

.wallet__tx-header {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-base) 0 var(--ui-space-sm);
}

.wallet__tx-item {
  display: flex;
  align-items: center;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  margin-bottom: var(--ui-space-sm);
  box-shadow: var(--ui-shadow-sm);
}

.wallet__tx-body {
  flex: 1;
  min-width: 0;
}

.wallet__tx-type {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
  display: block;
}

.wallet__tx-remark {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wallet__tx-time {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: 2px;
}

.wallet__tx-amount {
  flex-shrink: 0;
  text-align: right;
}

.wallet__tx-amount-value {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  display: block;
}

.wallet__tx-status {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.wallet__tx--income,
 .wallet__tx--refund,
 .wallet__tx--unfreeze {
  color: var(--ui-color-success);
}

.wallet__tx--withdraw,
 .wallet__tx--freeze {
  color: var(--ui-color-text-primary);
}

.wallet__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>