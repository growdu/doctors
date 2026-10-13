<script setup lang="ts">
/**
 * patient/index/index.vue — patient 域首页（v2 unified-app · tabBar 主页）。
 *
 * 入口：登录后或 HomeShell 域切换（tabBar "首页"）
 *   → 顶部欢迎语
 *   → 「进行中订单」 widget（pending_escort / escort_confirmed / in_service，未登录态自动隐藏）
 *   → 快捷入口（4 网格）
 *   → 推荐医院（顶部 3 个）
 *   → 推荐套餐（瀑布流）
 *
 * 设计要点：
 *   - 真实接口：listOrders（role=patient）+ listHospitals + listPackagesByHospital
 *   - 「进行中订单」 widget：让用户打开 app 第一眼看到自己订单进展（v2 SaaS 体验提升）
 *   - 失败回退：每个 section 独立 catch，单个失败不阻塞整体渲染
 */
import { ref, computed, onMounted } from 'vue';
import { useAuthStore } from '@/store/auth';
import { listHospitals, listPackagesByHospital } from '@/api/user';
import { listOrders } from '@/api/orders';
import type { Hospital, Package } from '@/api/user';
import type { Order, OrderStatus } from '@/api/orders';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const auth = useAuthStore();
const hospitals = ref<Hospital[]>([]);
const packages = ref<Package[]>([]);
const activeOrders = ref<Order[]>([]);
const hospitalsLoading = ref(false);
const ordersLoading = ref(false);

const ACTIVE_STATUSES: OrderStatus[] = ['pending_escort', 'escort_confirmed', 'in_service'];

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending_escort: '待接单',
  escort_pending_acceptance: '陪诊师待确认',
  escort_confirmed: '待服务',
  in_service: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

async function loadHospitals() {
  hospitalsLoading.value = true;
  try {
    const r = await listHospitals();
    hospitals.value = r.items.slice(0, 3);
    // 取第一个医院的套餐作展示
    if (hospitals.value[0]) {
      const pkgs = await listPackagesByHospital(hospitals.value[0].id);
      packages.value = pkgs.items.slice(0, 3);
    }
  } catch {
    // ignore — 区块级回退
  } finally {
    hospitalsLoading.value = false;
  }
}

async function loadActiveOrders() {
  if (!auth.isAuthed) return;
  ordersLoading.value = true;
  try {
    // 拉全部 patient 订单，按状态本地过滤；后端无 in[] 过滤，前端分页过滤
    const r = await listOrders({ role: 'patient', page_size: 50 });
    activeOrders.value = r.items.filter((o) => ACTIVE_STATUSES.includes(o.status));
  } catch {
    // ignore — 区块级回退
  } finally {
    ordersLoading.value = false;
  }
}

const hasActiveOrders = computed(() => activeOrders.value.length > 0);

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

function onOrder(o: Order) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: `/pages/patient/order/detail?id=${o.id}` });
}

function onViewAllOrders() {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: '/pages/patient/order/list' });
}

function formatPrice(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function formatDate(iso?: string): string {
  if (!iso) return '—';
  const t = iso.split('T');
  return t[0] + ' ' + (t[1]?.substring(0, 5) || '');
}

onMounted(() => {
  loadActiveOrders();
  loadHospitals();
});
</script>

<template>
  <view class="ui-page" data-testid="patient-home">
    <UiCard shadow="sm" data-testid="patient-home-welcome">
      <view class="home__welcome">
        <text class="home__welcome-text">您好，{{ auth.user?.phone || '游客' }} 👋</text>
        <text class="home__welcome-subtitle">选择一个医院，开启陪诊服务</text>
      </view>
    </UiCard>

    <!-- 进行中订单 widget：登录态 + 有订单时才显示 -->
    <view
      v-if="auth.isAuthed && (hasActiveOrders || ordersLoading)"
      data-testid="home-active-orders"
    >
      <view class="home__section-title home__section-title--row">
        <text>📋 进行中订单</text>
        <text class="home__section-link" @click="onViewAllOrders" data-testid="home-view-all-orders">
          查看全部 →
        </text>
      </view>
      <view v-if="ordersLoading && activeOrders.length === 0" data-testid="home-active-orders-loading">
        <UiLoading text="加载订单..." />
      </view>
      <view v-else-if="hasActiveOrders" data-testid="home-active-orders-list">
        <UiCard
          v-for="o in activeOrders"
          :key="o.id"
          :data-testid="`home-active-order-${o.id}`"
          @click="onOrder(o)"
        >
          <view class="home-order__row">
            <text class="home-order__id">#{{ o.order_no || o.id }}</text>
            <text class="home-order__status" :data-testid="`home-order-status-${o.status}`">
              {{ STATUS_LABEL[o.status] }}
            </text>
          </view>
          <view class="home-order__meta">
            <text class="home-order__time">📅 {{ formatDate(o.service_start_at) }}</text>
            <text class="home-order__amount">{{ formatPrice(o.amount) }}</text>
          </view>
        </UiCard>
      </view>
    </view>

    <!-- 快捷入口 4 列 -->
    <view class="home__section-title" data-testid="home-quick-title">快捷入口</view>
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
    <view v-if="hospitalsLoading && hospitals.length === 0" data-testid="home-hospitals-loading">
      <UiLoading text="加载医院..." />
    </view>
    <view v-else-if="hospitals.length === 0" data-testid="home-hospitals-empty">
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

.home__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.home__section-title--row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.home__section-link {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  font-weight: var(--ui-font-weight-regular);
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

.home-order__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--ui-space-xs);
}

.home-order__id {
  font-size: var(--ui-font-base);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.home-order__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-pill);
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
}

.home-order__meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.home-order__time {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.home-order__amount {
  font-size: var(--ui-font-base);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}
</style>