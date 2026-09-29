<script setup lang="ts">
/**
 * patient/index/index.vue — patient 域首页（v2 unified-app）。
 *
 * 入口：登录后或 HomeShell 域切换
 *   → 顶部欢迎语 + 快捷入口（4 网格）
 *   → 推荐医院（顶部 3 个）
 *   → 推荐套餐（瀑布流）
 *
 * 设计要点：
 *   - 简化版：4 入口 + 医院列表 + 套餐列表
 *   - 真实接口：listHospitals + listPackagesByHospital
 */
import { ref, onMounted } from 'vue';
import { useAuthStore } from '@/store/auth';
import { listHospitals, listPackagesByHospital } from '@/api/user';
import type { Hospital, Package } from '@/api/user';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';

const auth = useAuthStore();
const hospitals = ref<Hospital[]>([]);
const packages = ref<Package[]>([]);
const loading = ref(false);

async function onLoad() {
  loading.value = true;
  try {
    const r = await listHospitals();
    hospitals.value = r.items.slice(0, 3);
    // 取第一个医院的套餐作展示
    if (hospitals.value[0]) {
      const pkgs = await listPackagesByHospital(hospitals.value[0].id);
      packages.value = pkgs.items.slice(0, 3);
    }
  } catch (e) {
    // ignore
  } finally {
    loading.value = false;
  }
}

interface QuickLink {
  icon: string;
  label: string;
  path: string;
}

const QUICK_LINKS: QuickLink[] = [
  { icon: '🏥', label: '选择医院', path: '/pages/patient/hospitals/list' },
  { icon: '📋', label: '我的订单', path: '/pages/patient/order/list' },
  { icon: '🎁', label: '优惠券', path: '/pages/patient/coupons/index' },
  { icon: '💬', label: '消息', path: '/pages/patient/message/list' },
];

function onQuickLink(l: QuickLink) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: l.path });
}

function onHospital(h: Hospital) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/patient/hospitals/detail?id=${h.id}` });
}

function formatPrice(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-home">
    <UiCard shadow="sm" data-testid="patient-home-welcome">
      <view class="home__welcome">
        <text class="home__welcome-text">您好，{{ auth.user?.phone || '游客' }} 👋</text>
        <text class="home__welcome-subtitle">选择一个医院，开启陪诊服务</text>
      </view>
    </UiCard>

    <!-- 快捷入口 4 列 -->
    <view class="home__quick-links" data-testid="patient-home-quick-links">
      <view
        v-for="l in QUICK_LINKS"
        :key="l.path"
        class="home__quick-link"
        :data-testid="`home-quick-${l.label}`"
        @click="onQuickLink(l)"
      >
        <text class="home__quick-icon">{{ l.icon }}</text>
        <text class="home__quick-label">{{ l.label }}</text>
      </view>
    </view>

    <!-- 推荐医院 -->
    <view class="home__section-title" data-testid="home-hospitals-title">推荐医院</view>
    <view v-if="hospitals.length === 0" data-testid="home-hospitals-empty">
      <UiEmpty icon="🏥" title="暂无医院" />
    </view>
    <view v-else data-testid="home-hospitals-list">
      <UiCard
        v-for="h in hospitals"
        :key="h.id"
        :title="h.name"
        :data-testid="`home-hospital-${h.id}`"
        @click="onHospital(h)"
      >
        <text class="home__hospital-city">{{ h.city }}{{ h.level ? ' · ' + h.level : '' }}</text>
        <text class="home__hospital-address">{{ h.address }}</text>
      </UiCard>
    </view>

    <!-- 推荐套餐 -->
    <view v-if="packages.length > 0" class="home__section-title" data-testid="home-packages-title">推荐套餐</view>
    <view v-if="packages.length > 0" data-testid="home-packages-list">
      <UiCard
        v-for="p in packages"
        :key="p.id"
        :title="p.title"
        :data-testid="`home-package-${p.id}`"
      >
        <text class="home__package-desc">{{ p.description }}</text>
        <text class="home__package-price">{{ formatPrice(p.price) }}</text>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.home__welcome {
  padding: var(--ui-space-sm) 0;
}

.home__welcome-text {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.home__welcome-subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.home__quick-links {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--ui-space-md);
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  margin-bottom: var(--ui-space-md);
}

.home__quick-link {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--ui-space-sm);
  cursor: pointer;
}

.home__quick-icon {
  font-size: 28px;
  margin-bottom: var(--ui-space-xs);
}

.home__quick-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
}

.home__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.home__hospital-city {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
}

.home__hospital-address {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.home__package-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-bottom: var(--ui-space-sm);
}

.home__package-price {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
  display: block;
}
</style>