<script setup lang="ts">
/**
 * escort/wallet/index.vue — 陪诊师钱包首页（v2 unified-app · escort 域）。
 *
 * 入口：escort home 或聚合菜单「钱包」入口
 *   → 顶部余额卡：可用余额 / 冻结 / 累计收入 / 累计提现（getEscortWallet）
 *   → 中部流水 tab：全部 / 收入 / 提现 / 退款（listTransactions 按 type 过滤）
 *   → 「提现」按钮 → 跳 wallet/withdraw
 *
 * 设计要点：
 *   - 余额高亮（颜色：success 绿）
 *   - 流水按 type 标签渲染（图标 + 文案 + 金额颜色：收入 + 绿，提现 - 红）
 *   - 复用 patient/wallet 的视觉语言（如果存在）
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { getEscortWallet, listTransactions } from '@/api/wallet';
import type { Wallet, Transaction, TransactionType } from '@/api/wallet';

const TYPE_TABS: Array<{ key: TransactionType | 'all'; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'income', label: '收入' },
  { key: 'withdraw', label: '提现' },
  { key: 'refund', label: '退款' },
];

const TYPE_LABEL: Record<TransactionType, string> = {
  income: '收入',
  withdraw: '提现',
  refund: '退款',
  freeze: '冻结',
  unfreeze: '解冻',
};

const TYPE_ICON: Record<TransactionType, string> = {
  income: '💵',
  withdraw: '💸',
  refund: '↩️',
  freeze: '🧊',
  unfreeze: '🔥',
};

const wallet = ref<Wallet | null>(null);
const walletLoading = ref(false);
const walletError = ref<string | null>(null);

const items = ref<Transaction[]>([]);
const txLoading = ref(false);
const txError = ref<string | null>(null);
const activeTab = ref<TransactionType | 'all'>('all');

async function loadWallet() {
  walletLoading.value = true;
  walletError.value = null;
  try {
    wallet.value = await getEscortWallet();
  } catch (e) {
    walletError.value = (e as Error).message;
  } finally {
    walletLoading.value = false;
  }
}

async function loadTransactions() {
  txLoading.value = true;
  txError.value = null;
  try {
    const query: { type?: TransactionType } = {};
    if (activeTab.value !== 'all') query.type = activeTab.value;
    const r = await listTransactions(query);
    items.value = r.items;
  } catch (e) {
    txError.value = (e as Error).message;
  } finally {
    txLoading.value = false;
  }
}

async function onSwitchTab(t: TransactionType | 'all') {
  activeTab.value = t;
  await loadTransactions();
}

function onWithdraw() {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: '/pages/escort/wallet/withdraw' });
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

function txSign(type: TransactionType): string {
  return type === 'income' ? '+' : '-';
}

function txAmountColor(type: TransactionType): string {
  return type === 'income' ? 'escort-wallet__amount--income' : 'escort-wallet__amount--out';
}

onMounted(async () => {
  await Promise.all([loadWallet(), loadTransactions()]);
});
</script>

<template>
  <view class="ui-page" data-testid="escort-wallet-page">
    <!-- 余额卡 -->
    <view v-if="walletLoading && !wallet" data-testid="escort-wallet-loading">
      <UiLoading text="加载钱包中..." />
    </view>

    <view v-else-if="walletError && !wallet" data-testid="escort-wallet-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="walletError">
        <template #action>
          <text class="escort-wallet__retry" @click="loadWallet">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="wallet" class="escort-wallet__card" data-testid="escort-wallet-card">
      <UiCard title="💰 我的钱包">
        <view class="escort-wallet__balance-row">
          <text class="escort-wallet__balance-label">可用余额</text>
          <text class="escort-wallet__balance-value" data-testid="escort-wallet-balance">
            {{ formatYuan(wallet.balance) }}
          </text>
        </view>
        <view class="escort-wallet__sub-row">
          <view class="escort-wallet__sub-cell">
            <text class="escort-wallet__sub-label">冻结金额</text>
            <text class="escort-wallet__sub-value" data-testid="escort-wallet-frozen">
              {{ formatYuan(wallet.frozen) }}
            </text>
          </view>
          <view class="escort-wallet__sub-cell">
            <text class="escort-wallet__sub-label">累计收入</text>
            <text class="escort-wallet__sub-value escort-wallet__sub-value--income" data-testid="escort-wallet-earned">
              {{ formatYuan(wallet.total_earned) }}
            </text>
          </view>
          <view class="escort-wallet__sub-cell">
            <text class="escort-wallet__sub-label">累计提现</text>
            <text class="escort-wallet__sub-value" data-testid="escort-wallet-withdrawn">
              {{ formatYuan(wallet.total_withdrawn) }}
            </text>
          </view>
        </view>
        <view class="escort-wallet__withdraw-action">
          <UiButton
            type="primary"
            block
            data-testid="escort-wallet-withdraw-btn"
            @click="onWithdraw"
          >
            提现
          </UiButton>
        </view>
      </UiCard>
    </view>

    <!-- 流水 -->
    <view class="escort-wallet__tx-section">
      <view class="escort-wallet__tx-header">
        <text class="escort-wallet__tx-title">📋 流水</text>
      </view>

      <view class="escort-wallet__tabs" data-testid="escort-wallet-tabs">
        <view
          v-for="t in TYPE_TABS"
          :key="t.key"
          class="escort-wallet__tab"
          :class="{ 'escort-wallet__tab--active': activeTab === t.key }"
          :data-testid="`escort-wallet-tab-${t.key}`"
          @click="onSwitchTab(t.key)"
        >
          {{ t.label }}
        </view>
      </view>

      <view v-if="txLoading" data-testid="escort-wallet-tx-loading">
        <UiLoading text="加载流水中..." />
      </view>

      <view v-else-if="txError" data-testid="escort-wallet-tx-error">
        <UiEmpty icon="⚠️" title="加载失败" :description="txError">
          <template #action>
            <text class="escort-wallet__retry" @click="loadTransactions">点击重试</text>
          </template>
        </UiEmpty>
      </view>

      <view v-else-if="items.length === 0" data-testid="escort-wallet-tx-empty">
        <UiEmpty icon="📋" title="暂无流水" />
      </view>

      <view v-else data-testid="escort-wallet-tx-list">
        <view
          v-for="tx in items"
          :key="tx.id"
          class="escort-wallet__tx-item"
          :data-testid="`escort-wallet-tx-${tx.id}`"
        >
          <view class="escort-wallet__tx-info">
            <text class="escort-wallet__tx-icon">{{ TYPE_ICON[tx.type] }}</text>
            <view class="escort-wallet__tx-text">
              <text class="escort-wallet__tx-type">{{ TYPE_LABEL[tx.type] }}</text>
              <text class="escort-wallet__tx-time">📅 {{ formatDate(tx.created_at) }}</text>
              <text v-if="tx.order_id" class="escort-wallet__tx-order">订单 #{{ tx.order_id }}</text>
              <text v-if="tx.remark" class="escort-wallet__tx-remark">{{ tx.remark }}</text>
            </view>
          </view>
          <text
            class="escort-wallet__amount"
            :class="txAmountColor(tx.type)"
            :data-testid="`escort-wallet-tx-amount-${tx.id}`"
          >
            {{ txSign(tx.type) }}{{ formatYuan(tx.amount) }}
          </text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.escort-wallet__card {
  margin-bottom: var(--ui-space-md);
}

.escort-wallet__balance-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: var(--ui-space-md);
}

.escort-wallet__balance-label {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-secondary);
}

.escort-wallet__balance-value {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-success);
}

.escort-wallet__sub-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--ui-space-md);
  padding: var(--ui-space-sm) 0;
  border-top: 1px solid var(--ui-color-border);
}

.escort-wallet__sub-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.escort-wallet__sub-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
}

.escort-wallet__sub-value {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  margin-top: 4px;
}

.escort-wallet__sub-value--income { color: var(--ui-color-success); }

.escort-wallet__withdraw-action {
  margin-top: var(--ui-space-sm);
}

.escort-wallet__tx-section {
  margin-top: var(--ui-space-md);
}

.escort-wallet__tx-header {
  padding: var(--ui-space-sm) 0;
}

.escort-wallet__tx-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.escort-wallet__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.escort-wallet__tab {
  flex: 1;
  min-width: 64px;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.escort-wallet__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.escort-wallet__tx-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-sm) var(--ui-space-md);
  margin-bottom: var(--ui-space-sm);
}

.escort-wallet__tx-info {
  display: flex;
  align-items: center;
  gap: var(--ui-space-sm);
  flex: 1;
}

.escort-wallet__tx-icon {
  font-size: 24px;
}

.escort-wallet__tx-type {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-wallet__tx-time,
.escort-wallet__tx-order,
.escort-wallet__tx-remark {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-wallet__amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
}

.escort-wallet__amount--income { color: var(--ui-color-success); }
.escort-wallet__amount--out { color: var(--ui-color-error); }

.escort-wallet__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>