<script setup lang="ts">
/**
 * escort/wallet/withdraw.vue — 提现申请（v2 unified-app · escort 域）。
 *
 * 入口：escort/wallet「提现」按钮
 *   → 拉钱包余额（getEscortWallet）
 *   → 3 字段表单：金额（分）+ 收款账号 + 账户名
 *   → 校验：> 0 且 ≤ 余额
 *   → 调 requestWithdrawal → 跳回 wallet
 *
 * 设计要点：
 *   - 顶部显示余额（参照 Flutter 版设计）
 *   - 金额校验前端 + 后端双重
 *   - 提现成功后显示 toast + 跳回
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { getEscortWallet, requestWithdrawal } from '@/api/wallet';

const balance = ref(0);
const walletLoading = ref(false);

const amountYuan = ref('');
const account = ref('');
const accountName = ref('');
const submitting = ref(false);
const errorMsg = ref<string | null>(null);

async function loadWallet() {
  walletLoading.value = true;
  try {
    const w = await getEscortWallet();
    balance.value = w.balance;
  } catch {
    // ignore; 余额读不到时仍允许提交（后端校验）
  } finally {
    walletLoading.value = false;
  }
}

async function onSubmit() {
  errorMsg.value = null;
  const yuan = Number(amountYuan.value.trim());
  if (!yuan || yuan <= 0) {
    errorMsg.value = '请输入有效金额';
    return;
  }
  const cents = Math.round(yuan * 100);
  if (balance.value > 0 && cents > balance.value) {
    errorMsg.value = `金额超过余额（¥${(balance.value / 100).toFixed(2)}）`;
    return;
  }
  if (!account.value.trim()) {
    errorMsg.value = '请输入收款账号';
    return;
  }
  if (!accountName.value.trim()) {
    errorMsg.value = '请输入账户名';
    return;
  }
  submitting.value = true;
  try {
    await requestWithdrawal({
      amount: cents,
      account: account.value.trim(),
      account_name: accountName.value.trim(),
    });
    if (typeof uni !== 'undefined') {
      uni.showToast({ title: '提现申请已提交', icon: 'success' });
      setTimeout(() => uni.navigateBack(), 800);
    }
  } catch (e) {
    errorMsg.value = `提现失败：${(e as Error).message}`;
  } finally {
    submitting.value = false;
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function isValid(): boolean {
  const yuan = Number(amountYuan.value.trim());
  if (!yuan || yuan <= 0) return false;
  if (balance.value > 0 && Math.round(yuan * 100) > balance.value) return false;
  if (!account.value.trim()) return false;
  if (!accountName.value.trim()) return false;
  return true;
}

onMounted(loadWallet);
</script>

<template>
  <view class="ui-page" data-testid="escort-withdraw-page">
    <!-- 余额显示 -->
    <view v-if="walletLoading" data-testid="escort-withdraw-wallet-loading">
      <UiLoading text="加载余额中..." />
    </view>

    <view v-else class="escort-withdraw__balance" data-testid="escort-withdraw-balance-card">
      <UiCard title="可提现余额">
        <text class="escort-withdraw__balance-value" data-testid="escort-withdraw-balance">
          {{ formatYuan(balance) }}
        </text>
      </UiCard>
    </view>

    <!-- 表单 -->
    <view class="escort-withdraw__form" data-testid="escort-withdraw-form">
      <UiInput
        v-model="amountYuan"
        label="提现金额（元）"
        placeholder="请输入金额"
        type="number"
        data-testid="escort-withdraw-amount"
      />
      <UiInput
        v-model="account"
        label="收款账号"
        placeholder="银行卡号 / 支付宝号"
        data-testid="escort-withdraw-account"
      />
      <UiInput
        v-model="accountName"
        label="账户名"
        placeholder="持卡人姓名 / 支付宝实名"
        data-testid="escort-withdraw-account-name"
      />

      <view v-if="errorMsg" class="escort-withdraw__error" data-testid="escort-withdraw-error">
        <text class="escort-withdraw__error-text">⚠️ {{ errorMsg }}</text>
      </view>

      <view class="escort-withdraw__actions">
        <UiButton
          type="primary"
          block
          :loading="submitting"
          :disabled="!isValid()"
          data-testid="escort-withdraw-submit"
          @click="onSubmit"
        >
          申请提现
        </UiButton>
      </view>
    </view>
  </view>
</template>

<style scoped>
.escort-withdraw__balance {
  margin-bottom: var(--ui-space-md);
}

.escort-withdraw__balance-value {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-success);
  display: block;
  text-align: center;
  padding: var(--ui-space-sm) 0;
}

.escort-withdraw__form {
  margin-top: var(--ui-space-sm);
}

.escort-withdraw__error {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-error);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-md);
}

.escort-withdraw__error-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
}

.escort-withdraw__actions {
  margin-top: var(--ui-space-md);
}
</style>