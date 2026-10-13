<script setup lang="ts">
/**
 * admin/coupons/index.vue — 优惠券管理（v2 unified-app · admin 域）。
 *
 * v2.1 升级：从"占位 toast + 跳转 patient 视角"升级为"实时拉取优惠券 + 状态聚合 dashboard"。
 * 后端 listAvailableCoupons / listMyCoupons 已可用，本页用 user-service 数据呈现 admin 视角。
 *
 * 设计要点：
 *   - 4 张统计卡：总券 / 可领取 / 已领取 / 已使用
 *   - 列表：每张券的标题 / 面额 / 最低消费 / 有效期 / 状态徽章
 *   - 加载失败降级为空态 + 「重试」按钮
 */
import { ref, computed, onMounted } from 'vue';
import { listAvailableCoupons, listMyCoupons } from '@/api/user';
import type { Coupon } from '@/api/user';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const available = ref<Coupon[]>([]);
const claimed = ref<Coupon[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    // 任一失败都不阻塞；但用 Promise.all 让两个并发
    const results = await Promise.allSettled([
      listAvailableCoupons(),
      listMyCoupons(),
    ]);
    available.value = results[0].status === 'fulfilled' ? results[0].value.items : [];
    claimed.value = results[1].status === 'fulfilled' ? results[1].value.items : [];
    // 两个都失败时进入 error 态
    if (results.every((r) => r.status === 'rejected')) {
      error.value = (results[0] as PromiseRejectedResult).reason?.message ?? '加载失败';
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

const stats = computed(() => {
  const used = claimed.value.filter((c) => c.status === 'used').length;
  return {
    total: available.value.length + claimed.value.length,
    available: available.value.length,
    claimed: claimed.value.length - used,
    used,
  };
});

const STATUS_LABEL: Record<Coupon['status'], string> = {
  available: '可领取',
  claimed: '已领取',
  used: '已使用',
  expired: '已过期',
};

function formatDate(iso?: string): string {
  if (!iso) return '—';
  return iso.split('T')[0];
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-coupons-page">
    <view class="admin-coupons__header">
      <text class="admin-coupons__title">优惠券管理</text>
      <text class="admin-coupons__subtitle">实时统计 · 拉自 user-service</text>
    </view>

    <view v-if="loading && available.length === 0 && claimed.length === 0" data-testid="admin-coupons-loading">
      <UiLoading text="加载优惠券..." />
    </view>

    <view v-else-if="error && available.length === 0 && claimed.length === 0" data-testid="admin-coupons-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <UiButton type="primary" size="sm" @click="onLoad" data-testid="admin-coupons-retry">重试</UiButton>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <!-- 4 张统计卡 -->
      <view class="admin-coupons__stats" data-testid="admin-coupons-stats">
        <view class="admin-coupons__stat admin-coupons__stat--primary" data-testid="admin-coupons-stat-total">
          <text class="admin-coupons__stat-value">{{ stats.total }}</text>
          <text class="admin-coupons__stat-label">总券数</text>
        </view>
        <view class="admin-coupons__stat admin-coupons__stat--success" data-testid="admin-coupons-stat-available">
          <text class="admin-coupons__stat-value">{{ stats.available }}</text>
          <text class="admin-coupons__stat-label">可领取</text>
        </view>
        <view class="admin-coupons__stat admin-coupons__stat--warning" data-testid="admin-coupons-stat-claimed">
          <text class="admin-coupons__stat-value">{{ stats.claimed }}</text>
          <text class="admin-coupons__stat-label">已领取</text>
        </view>
        <view class="admin-coupons__stat admin-coupons__stat--disabled" data-testid="admin-coupons-stat-used">
          <text class="admin-coupons__stat-value">{{ stats.used }}</text>
          <text class="admin-coupons__stat-label">已使用</text>
        </view>
      </view>

      <!-- 可领取券列表 -->
      <view class="admin-coupons__section-title" data-testid="admin-coupons-available-title">
        🎟️ 可领取券（{{ available.length }}）
      </view>
      <view v-if="available.length === 0" data-testid="admin-coupons-available-empty">
        <UiEmpty icon="🎟️" title="暂无可领取券" />
      </view>
      <view v-else data-testid="admin-coupons-available-list">
        <UiCard v-for="c in available" :key="c.id" :data-testid="`admin-coupon-${c.id}`">
          <view class="admin-coupon__row">
            <text class="admin-coupon__title">{{ c.title }}</text>
            <text class="admin-coupon__amount">{{ formatYuan(c.amount) }}</text>
          </view>
          <view class="admin-coupon__meta">
            <text class="admin-coupon__meta-item">满 {{ formatYuan(c.min_amount) }} 可用</text>
            <text class="admin-coupon__meta-item">{{ formatDate(c.valid_from) }} ~ {{ formatDate(c.valid_to) }}</text>
            <text class="admin-coupon__status admin-coupon__status--available">
              {{ STATUS_LABEL[c.status] }}
            </text>
          </view>
        </UiCard>
      </view>

      <!-- 已领取券列表 -->
      <view v-if="claimed.length > 0" class="admin-coupons__section-title" data-testid="admin-coupons-claimed-title">
        👤 用户已领取（{{ claimed.length }}）
      </view>
      <view v-if="claimed.length > 0" data-testid="admin-coupons-claimed-list">
        <UiCard v-for="c in claimed" :key="c.id" :data-testid="`admin-my-coupon-${c.id}`">
          <view class="admin-coupon__row">
            <text class="admin-coupon__title">{{ c.title }}</text>
            <text class="admin-coupon__amount">{{ formatYuan(c.amount) }}</text>
          </view>
          <view class="admin-coupon__meta">
            <text class="admin-coupon__meta-item">满 {{ formatYuan(c.min_amount) }} 可用</text>
            <text class="admin-coupon__meta-item">{{ formatDate(c.valid_from) }} ~ {{ formatDate(c.valid_to) }}</text>
            <text
              class="admin-coupon__status"
              :class="{
                'admin-coupon__status--claimed': c.status === 'claimed',
                'admin-coupon__status--used': c.status === 'used',
                'admin-coupon__status--expired': c.status === 'expired',
              }"
            >
              {{ STATUS_LABEL[c.status] }}
            </text>
          </view>
        </UiCard>
      </view>
    </template>
  </view>
</template>

<style scoped>
.admin-coupons__header {
  padding: var(--ui-space-base) 0 var(--ui-space-sm);
}

.admin-coupons__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-coupons__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-coupons__stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ui-space-md);
  margin-bottom: var(--ui-space-md);
}

.admin-coupons__stat {
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: var(--ui-shadow-sm);
}

.admin-coupons__stat-value {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-bold);
  display: block;
}

.admin-coupons__stat-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  margin-top: var(--ui-space-xs);
}

.admin-coupons__stat--primary .admin-coupons__stat-value { color: var(--ui-color-primary); }
.admin-coupons__stat--success .admin-coupons__stat-value { color: var(--ui-color-success); }
.admin-coupons__stat--warning .admin-coupons__stat-value { color: var(--ui-color-warning); }
.admin-coupons__stat--disabled .admin-coupons__stat-value { color: var(--ui-color-text-disabled); }

.admin-coupons__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.admin-coupon__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--ui-space-xs);
}

.admin-coupon__title {
  font-size: var(--ui-font-base);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  flex: 1;
}

.admin-coupon__amount {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.admin-coupon__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ui-space-sm);
  align-items: center;
}

.admin-coupon__meta-item {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
}

.admin-coupon__status {
  font-size: var(--ui-font-xs);
  padding: 1px 6px;
  border-radius: var(--ui-radius-pill);
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
}

.admin-coupon__status--available { background: var(--ui-color-success); }
.admin-coupon__status--claimed { background: var(--ui-color-warning); }
.admin-coupon__status--used { background: var(--ui-color-text-disabled); }
.admin-coupon__status--expired { background: var(--ui-color-text-disabled); opacity: 0.6; }
</style>