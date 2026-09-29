<script setup lang="ts">
/**
 * admin/hospitals/index.vue — 医院字典管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「医院」相关入口
 *   → listHospitals() 拉医院列表（user-service，无权限限制）
 *   → city 关键词搜索（简化版：客户端筛选）
 *   → 点击医院卡显示套餐列表（listPackagesByHospital）
 *
 * 设计要点：
 *   - 复用 patient/hospitals 风格 + admin 操作列
 *   - 选医院后展开套餐列表（折叠展开，无 modal）
 */
import { ref, onMounted } from 'vue';
import { listHospitals, listPackagesByHospital } from '@/api/user';
import type { Hospital, Package } from '@/api/user';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const items = ref<Hospital[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const keyword = ref('');

const expanded = ref<number | null>(null);
const packagesByHospital = ref<Record<number, Package[]>>({});
const packagesLoading = ref<number | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listHospitals();
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onToggle(h: Hospital) {
  if (expanded.value === h.id) {
    expanded.value = null;
    return;
  }
  expanded.value = h.id;
  if (packagesByHospital.value[h.id]) return;
  packagesLoading.value = h.id;
  try {
    const r = await listPackagesByHospital(h.id);
    packagesByHospital.value = { ...packagesByHospital.value, [h.id]: r.items };
  } catch {
    packagesByHospital.value = { ...packagesByHospital.value, [h.id]: [] };
  } finally {
    packagesLoading.value = null;
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function filtered(): Hospital[] {
  const k = keyword.value.trim().toLowerCase();
  if (!k) return items.value;
  return items.value.filter((h) => h.name.toLowerCase().includes(k) || h.city.toLowerCase().includes(k));
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-hospitals-page">
    <view class="admin-hospitals__search">
      <UiInput
        v-model="keyword"
        placeholder="按名称 / 城市 搜索"
        clearable
        data-testid="admin-hospitals-search"
      />
    </view>

    <view v-if="loading" data-testid="admin-hospitals-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="admin-hospitals-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-hospitals__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="filtered().length === 0" data-testid="admin-hospitals-empty">
      <UiEmpty icon="🏥" title="暂无医院" />
    </view>

    <view v-else data-testid="admin-hospitals-list">
      <UiCard
        v-for="h in filtered()"
        :key="h.id"
        :data-testid="`admin-hospitals-card-${h.id}`"
      >
        <view class="admin-hospitals__row" @click="onToggle(h)">
          <view class="admin-hospitals__info">
            <text class="admin-hospitals__name">{{ h.name }}</text>
            <text class="admin-hospitals__city">📍 {{ h.city }} · {{ h.level }}</text>
            <text class="admin-hospitals__address">{{ h.address }}</text>
            <text v-if="h.phone" class="admin-hospitals__phone">📞 {{ h.phone }}</text>
          </view>
          <text
            class="admin-hospitals__toggle"
            :class="{ 'admin-hospitals__toggle--open': expanded === h.id }"
            :data-testid="`admin-hospitals-toggle-${h.id}`"
          >{{ expanded === h.id ? '收起' : '套餐 ▾' }}</text>
        </view>

        <!-- 套餐展开区 -->
        <view
          v-if="expanded === h.id"
          class="admin-hospitals__packages"
          :data-testid="`admin-hospitals-packages-${h.id}`"
        >
          <view v-if="packagesLoading === h.id" data-testid="admin-hospitals-packages-loading">
            <UiLoading text="加载套餐中..." />
          </view>
          <view
            v-else-if="(packagesByHospital[h.id] || []).length === 0"
            data-testid="admin-hospitals-packages-empty"
          >
            <UiEmpty icon="📦" title="暂无套餐" />
          </view>
          <view v-else data-testid="admin-hospitals-packages-list">
            <view
              v-for="p in packagesByHospital[h.id]"
              :key="p.id"
              class="admin-hospitals__package"
              :data-testid="`admin-hospitals-package-${p.id}`"
            >
              <view class="admin-hospitals__package-info">
                <text class="admin-hospitals__package-title">{{ p.title }}</text>
                <text class="admin-hospitals__package-desc">{{ p.description }}</text>
                <text class="admin-hospitals__package-duration">⏱ {{ p.duration_minutes }} 分钟</text>
              </view>
              <text class="admin-hospitals__package-price">{{ formatYuan(p.price) }}</text>
            </view>
          </view>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-hospitals__search {
  margin-bottom: var(--ui-space-md);
}

.admin-hospitals__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  cursor: pointer;
}

.admin-hospitals__info {
  flex: 1;
}

.admin-hospitals__name {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-hospitals__city,
.admin-hospitals__address,
.admin-hospitals__phone {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-hospitals__toggle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  padding: var(--ui-space-xs) var(--ui-space-sm);
  white-space: nowrap;
}

.admin-hospitals__toggle--open {
  color: var(--ui-color-text-secondary);
}

.admin-hospitals__packages {
  margin-top: var(--ui-space-md);
  padding-top: var(--ui-space-md);
  border-top: 1px solid var(--ui-color-border);
}

.admin-hospitals__package {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: var(--ui-space-sm) 0;
  border-bottom: 1px solid var(--ui-color-border-light, var(--ui-color-border));
}

.admin-hospitals__package:last-child {
  border-bottom: none;
}

.admin-hospitals__package-info {
  flex: 1;
}

.admin-hospitals__package-title {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-hospitals__package-desc,
.admin-hospitals__package-duration {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-hospitals__package-price {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}

.admin-hospitals__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>