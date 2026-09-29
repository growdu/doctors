<script setup lang="ts">
/**
 * patient/hospitals/detail.vue — 医院详情页（v2 unified-app）。
 *
 * 入口：hospitals/list 点击「查看详情」→ 本页（?id=xxx）
 *   → 拉医院资料 + 该医院的套餐列表
 *   → 渲染医院头部 + 套餐卡片网格
 *
 * 设计要点：
 *   - 复用 UiCard / UiEmpty / UiLoading
 *   - 套餐卡片含价格 + 时长 + 「选择下单」
 */
import { ref, onMounted } from 'vue';
import { getHospital, listPackagesByHospital } from '@/api/user';
import type { Hospital, Package } from '@/api/user';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const hospitalId = ref<number | null>(null);
const hospital = ref<Hospital | null>(null);
const packages = ref<Package[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const current = pages[pages.length - 1];
  const opts = current?.options;
  if (opts?.id) {
    hospitalId.value = Number(opts.id);
  }
}

async function onLoad() {
  parseQuery();
  if (!hospitalId.value) {
    error.value = '缺少医院 ID';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    const [h, pkgs] = await Promise.all([
      getHospital(hospitalId.value),
      listPackagesByHospital(hospitalId.value),
    ]);
    hospital.value = h;
    packages.value = pkgs.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatPrice(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDuration(min: number): string {
  if (min < 60) return `${min} 分钟`;
  const h = Math.floor(min / 60);
  const m = min % 60;
  return m === 0 ? `${h} 小时` : `${h} 小时 ${m} 分`;
}

function onBook(pkg: Package) {
  // Phase 3.1 后接入 patient/order/create
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: `/pages/patient/order/create?hospitalId=${hospitalId.value}&packageId=${pkg.id}` });
  }
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-hospital-detail">
    <view v-if="loading" data-testid="hospital-detail-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="hospital-detail-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="hospital-detail__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else-if="hospital">
      <!-- 医院头部 -->
      <UiCard :title="hospital.name" data-testid="hospital-detail-header">
        <view class="hospital-detail__info">
          <view class="hospital-detail__row">
            <text class="hospital-detail__label">城市</text>
            <text class="hospital-detail__value">{{ hospital.city }}</text>
          </view>
          <view v-if="hospital.level" class="hospital-detail__row">
            <text class="hospital-detail__label">等级</text>
            <text class="hospital-detail__value hospital-detail__value--level">{{ hospital.level }}</text>
          </view>
          <view class="hospital-detail__row">
            <text class="hospital-detail__label">地址</text>
            <text class="hospital-detail__value">{{ hospital.address }}</text>
          </view>
          <view v-if="hospital.phone" class="hospital-detail__row">
            <text class="hospital-detail__label">电话</text>
            <text class="hospital-detail__value">{{ hospital.phone }}</text>
          </view>
        </view>
      </UiCard>

      <!-- 套餐列表 -->
      <view class="hospital-detail__packages-title" data-testid="hospital-packages-title">
        套餐列表（{{ packages.length }}）
      </view>

      <view v-if="packages.length === 0" data-testid="hospital-packages-empty">
        <UiEmpty icon="📦" title="暂无套餐" description="该医院暂未发布套餐" />
      </view>

      <view v-else data-testid="hospital-packages-list">
        <UiCard
          v-for="p in packages"
          :key="p.id"
          :title="p.title"
          :data-testid="`package-card-${p.id}`"
        >
          <text class="hospital-detail__pkg-desc">{{ p.description }}</text>
          <view class="hospital-detail__pkg-meta">
            <text class="hospital-detail__pkg-price">{{ formatPrice(p.price) }}</text>
            <text class="hospital-detail__pkg-duration">{{ formatDuration(p.duration_minutes) }}</text>
          </view>
          <template #footer>
            <UiButton
              type="primary"
              size="sm"
              :data-testid="`book-package-${p.id}`"
              @click="onBook(p)"
            >
              选择下单
            </UiButton>
          </template>
        </UiCard>
      </view>
    </template>
  </view>
</template>

<style scoped>
.hospital-detail__info {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.hospital-detail__row {
  display: flex;
  align-items: flex-start;
  gap: var(--ui-space-md);
}

.hospital-detail__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  flex-shrink: 0;
  width: 48px;
}

.hospital-detail__value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
}

.hospital-detail__value--level {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  padding: 1px 8px;
  border-radius: var(--ui-radius-sm);
  font-size: var(--ui-font-xs);
}

.hospital-detail__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}

.hospital-detail__packages-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-base) 0 var(--ui-space-sm);
}

.hospital-detail__pkg-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
  margin-bottom: var(--ui-space-sm);
}

.hospital-detail__pkg-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.hospital-detail__pkg-price {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.hospital-detail__pkg-duration {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}
</style>