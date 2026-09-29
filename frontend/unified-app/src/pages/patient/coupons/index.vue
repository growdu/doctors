<script setup lang="ts">
/**
 * patient/coupons/index.vue — 优惠券页（v2 unified-app）。
 *
 * 入口：patient 域「个人中心」menu「我的优惠券」
 *   → onMounted 调 api/user.listMyCoupons()（已领取的）
 *   → 渲染可领取 + 已领取 Tab
 *
 * 设计要点：
 *   - 顶部 tab 切换：available（可领取） + mine（我的）
 *   - 优惠券卡：金额（分→元）+ 最低消费 + 有效期 + 「立即领取」/ 「已领取」
 */
import { ref, computed, onMounted } from 'vue';
import { listAvailableCoupons, listMyCoupons, claimCoupon } from '@/api/user';
import type { Coupon } from '@/api/user';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const activeTab = ref<'available' | 'mine'>('available');
const availableCoupons = ref<Coupon[]>([]);
const myCoupons = ref<Coupon[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const claimingId = ref<number | null>(null);

const currentList = computed<Coupon[]>(() => (activeTab.value === 'available' ? availableCoupons.value : myCoupons.value));

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const [avail, mine] = await Promise.all([listAvailableCoupons(), listMyCoupons()]);
    availableCoupons.value = avail.items;
    myCoupons.value = mine.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatAmount(cents: number): string {
  return (cents / 100).toFixed(2);
}

function formatMinAmount(cents: number): string {
  if (cents === 0) return '无门槛';
  return `满${(cents / 100).toFixed(2)}元可用`;
}

function formatDate(iso: string): string {
  return iso.split('T')[0] || iso;
}

async function onClaim(c: Coupon) {
  claimingId.value = c.id;
  try {
    await claimCoupon(c.id);
    myCoupons.value = [...myCoupons.value, { ...c, status: 'claimed' }];
  } catch (e) {
    // store / toast error
  } finally {
    claimingId.value = null;
  }
}

function switchTab(tab: 'available' | 'mine') {
  activeTab.value = tab;
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-coupons">
    <view class="coupons__tabs" data-testid="coupons-tabs">
      <view
        class="coupons__tab"
        :class="{ 'coupons__tab--active': activeTab === 'available' }"
        :data-testid="'coupons-tab-available'"
        @click="switchTab('available')"
      >
        可领取（{{ availableCoupons.length }}）
      </view>
      <view
        class="coupons__tab"
        :class="{ 'coupons__tab--active': activeTab === 'mine' }"
        :data-testid="'coupons-tab-mine'"
        @click="switchTab('mine')"
      >
        我的（{{ myCoupons.length }}）
      </view>
    </view>

    <view v-if="loading && currentList.length === 0" data-testid="coupons-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="coupons-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="coupons__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="currentList.length === 0" data-testid="coupons-empty">
      <UiEmpty
        :icon="activeTab === 'available' ? '🎁' : '📭'"
        :title="activeTab === 'available' ? '暂无可领取优惠券' : '还没有优惠券'"
        :description="activeTab === 'available' ? '去下单获取吧' : '快去领取你的第一张优惠券吧'"
      />
    </view>

    <view v-else data-testid="coupons-list">
      <view
        v-for="c in currentList"
        :key="c.id"
        class="coupons__item"
        :data-testid="`coupon-item-${c.id}`"
      >
        <view class="coupons__item-amount">
          <text class="coupons__item-currency">¥</text>
          <text class="coupons__item-value">{{ formatAmount(c.amount) }}</text>
        </view>
        <view class="coupons__item-body">
          <text class="coupons__item-title">{{ c.title }}</text>
          <text class="coupons__item-meta">{{ formatMinAmount(c.min_amount) }}</text>
          <text class="coupons__item-validity">{{ formatDate(c.valid_from) }} ~ {{ formatDate(c.valid_to) }}</text>
        </view>
        <view class="coupons__item-action">
          <UiButton
            v-if="activeTab === 'available'"
            type="primary"
            size="sm"
            :loading="claimingId === c.id"
            :data-testid="`claim-coupon-${c.id}`"
            @click="onClaim(c)"
          >
            立即领取
          </UiButton>
          <text v-else class="coupons__item-status">已领取</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.coupons__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
}

.coupons__tab {
  flex: 1;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  transition: all var(--ui-duration-fast);
}

.coupons__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.coupons__item {
  display: flex;
  align-items: center;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  margin-bottom: var(--ui-space-sm);
  box-shadow: var(--ui-shadow-sm);
  border-left: 4px solid var(--ui-color-error);
}

.coupons__item-amount {
  display: flex;
  align-items: baseline;
  color: var(--ui-color-error);
  flex-shrink: 0;
  padding-right: var(--ui-space-md);
  border-right: 1px dashed var(--ui-color-divider);
  margin-right: var(--ui-space-md);
}

.coupons__item-currency {
  font-size: var(--ui-font-md);
}

.coupons__item-value {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  margin-left: 2px;
}

.coupons__item-body {
  flex: 1;
}

.coupons__item-title {
  font-size: var(--ui-font-base);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.coupons__item-meta {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.coupons__item-validity {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: 2px;
}

.coupons__item-action {
  flex-shrink: 0;
}

.coupons__item-status {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-disabled);
}

.coupons__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>