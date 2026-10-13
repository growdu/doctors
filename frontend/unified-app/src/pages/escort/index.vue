<script setup lang="ts">
/**
 * escort/index.vue — 陪诊师域工作台首页（v2 unified-app · escort 域）。
 *
 * 入口：escort 域第一入口（HomeShell 切换角色后）
 *   → 顶部余额卡（可用余额 + 冻结 + 累计收入）
 *   → 6 入口卡片：抢单池 / 我的任务 / 钱包 / 可用时段 / 培训 / 个人中心
 *
 * 设计要点：
 *   - 极简聚合入口；具体列表在各 page 单独实现
 *   - 钱包余额用区块级 catch，渲染失败时显示「—」+ 「前往查看」兜底
 *   - RoleGuard 限制 escort 角色
 */
import { ref, onMounted } from 'vue';
import RoleGuard from '@/components/shared/RoleGuard.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { useAuthStore } from '@/store/auth';
import { getEscortWallet } from '@/api/wallet';
import type { Wallet } from '@/api/wallet';

const auth = useAuthStore();

const wallet = ref<Wallet | null>(null);
const walletLoading = ref(false);

interface Entry {
  key: string;
  icon: string;
  title: string;
  desc: string;
  path: string;
}

const ENTRIES: Entry[] = [
  {
    key: 'invitations',
    icon: '🎯',
    title: '抢单池',
    desc: '查看可接候选订单',
    path: '/pages/escort/invitations/index',
  },
  {
    key: 'orders',
    icon: '📋',
    title: '我的任务',
    desc: '查看已接订单',
    path: '/pages/escort/orders/index',
  },
  {
    key: 'wallet',
    icon: '💰',
    title: '我的钱包',
    desc: '余额 · 流水 · 提现',
    path: '/pages/escort/wallet/index',
  },
  {
    key: 'availability',
    icon: '📅',
    title: '可接时段',
    desc: '设置可接单时段',
    path: '/pages/escort/availability/index',
  },
  {
    key: 'training',
    icon: '🎓',
    title: '培训记录',
    desc: '已完成的培训',
    path: '/pages/escort/training/index',
  },
  {
    key: 'profile',
    icon: '👤',
    title: '个人中心',
    desc: '资料 · 设置 · 退出',
    path: '/pages/escort/profile/index',
  },
];

async function loadWallet() {
  walletLoading.value = true;
  try {
    wallet.value = await getEscortWallet();
  } catch {
    // 区块级回退：不阻塞入口列表
  } finally {
    walletLoading.value = false;
  }
}

function onNavigate(path: string) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: path });
}

function formatYuan(cents?: number): string {
  if (cents === undefined || cents === null) return '—';
  return `¥${(cents / 100).toFixed(2)}`;
}

onMounted(() => {
  loadWallet();
});
</script>

<template>
  <RoleGuard :required="['escort']" fallback-title="需要 escort 角色">
    <view class="ui-page" data-testid="escort-home">
      <view class="escort-home__header">
        <text class="escort-home__title">🚑 陪诊师工作台</text>
        <text class="escort-home__subtitle">当前角色：{{ auth.activeRole }}</text>
      </view>

      <!-- 钱包余额卡（区块级 catch，失败时降级为空态） -->
      <view data-testid="escort-home-wallet-card">
        <view v-if="walletLoading && !wallet" data-testid="escort-home-wallet-loading">
          <UiLoading text="加载余额..." />
        </view>
        <UiCard v-else-if="wallet" shadow="sm">
          <view class="escort-wallet__row">
            <text class="escort-wallet__label">可用余额</text>
            <text class="escort-wallet__amount" data-testid="escort-home-wallet-balance">
              {{ formatYuan(wallet.balance) }}
            </text>
          </view>
          <view class="escort-wallet__row">
            <text class="escort-wallet__label">冻结金额</text>
            <text class="escort-wallet__secondary">{{ formatYuan(wallet.frozen) }}</text>
          </view>
          <view class="escort-wallet__row">
            <text class="escort-wallet__label">累计收入</text>
            <text class="escort-wallet__secondary">{{ formatYuan(wallet.total_earned) }}</text>
          </view>
        </UiCard>
        <UiEmpty v-else icon="💰" title="暂未获取余额" description="点击钱包查看详情">
          <template #action>
            <text
              class="escort-home__link"
              @click="onNavigate('/pages/escort/wallet/index')"
              data-testid="escort-home-wallet-fallback-link"
            >
              前往钱包 →
            </text>
          </template>
        </UiEmpty>
      </view>

      <!-- 入口卡片列表 -->
      <view class="escort-home__section-title">功能入口</view>
      <view class="escort-home__list" data-testid="escort-home-entries">
        <UiCard
          v-for="e in ENTRIES"
          :key="e.key"
          :data-testid="`escort-home-entry-${e.key}`"
          @click="() => onNavigate(e.path)"
        >
          <view class="escort-home__entry">
            <text class="escort-home__entry-icon">{{ e.icon }}</text>
            <view class="escort-home__entry-info">
              <text class="escort-home__entry-title">{{ e.title }}</text>
              <text class="escort-home__entry-desc">{{ e.desc }}</text>
            </view>
            <text class="escort-home__entry-arrow">→</text>
          </view>
        </UiCard>
      </view>
    </view>
  </RoleGuard>
</template>

<style scoped>
.escort-home__header {
  padding: var(--ui-space-base) 0;
}

.escort-home__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-home__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-home__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.escort-home__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.escort-home__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.escort-home__entry-icon {
  font-size: 32px;
}

.escort-home__entry-info {
  flex: 1;
}

.escort-home__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-home__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-home__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}

.escort-wallet__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--ui-space-xs) 0;
}

.escort-wallet__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.escort-wallet__amount {
  font-size: var(--ui-font-xxl);
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-primary);
}

.escort-wallet__secondary {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
}

.escort-home__link {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>