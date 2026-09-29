<script setup lang="ts">
/**
 * admin/packages/index.vue — 套餐字典管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「套餐」相关入口
 *   → 先选医院 → listHospitals()
 *   → 选中医院调 listPackagesByHospital(hospitalId) 拉套餐
 *   → 简化版：只读列表（v2 admin 未暴露套餐 CRUD 端点）
 *
 * 设计要点：
 *   - 顶部：医院选择器（下拉替代 select，简单 list 滚动）
 *   - 中部：选中医院后展示套餐列表
 */
import { ref, onMounted } from 'vue';
import { listHospitals, listPackagesByHospital } from '@/api/user';
import type { Hospital, Package } from '@/api/user';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const hospitals = ref<Hospital[]>([]);
const hospitalsLoading = ref(false);

const selectedHospitalId = ref<number | null>(null);
const packages = ref<Package[]>([]);
const packagesLoading = ref(false);
const packagesError = ref<string | null>(null);

async function onLoadHospitals() {
  hospitalsLoading.value = true;
  try {
    const r = await listHospitals();
    hospitals.value = r.items;
  } catch (e) {
    // ignore
  } finally {
    hospitalsLoading.value = false;
  }
}

async function onSelectHospital(h: Hospital) {
  selectedHospitalId.value = h.id;
  packagesLoading.value = true;
  packagesError.value = null;
  try {
    const r = await listPackagesByHospital(h.id);
    packages.value = r.items;
  } catch (e) {
    packagesError.value = (e as Error).message;
    packages.value = [];
  } finally {
    packagesLoading.value = false;
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

onMounted(onLoadHospitals);
</script>

<template>
  <view class="ui-page" data-testid="admin-packages-page">
    <view class="admin-packages__header">
      <text class="admin-packages__title">套餐管理</text>
      <text class="admin-packages__subtitle">按医院查看套餐</text>
    </view>

    <view class="admin-packages__sections" data-testid="admin-packages-sections">
      <!-- 医院选择 -->
      <UiCard title="选择医院">
        <view v-if="hospitalsLoading" data-testid="admin-packages-hospitals-loading">
          <UiLoading text="加载医院中..." />
        </view>
        <view v-else-if="hospitals.length === 0" data-testid="admin-packages-hospitals-empty">
          <UiEmpty icon="🏥" title="暂无医院" />
        </view>
        <view v-else data-testid="admin-packages-hospitals-list" class="admin-packages__hospital-list">
          <view
            v-for="h in hospitals"
            :key="h.id"
            class="admin-packages__hospital"
            :class="{ 'admin-packages__hospital--active': selectedHospitalId === h.id }"
            :data-testid="`admin-packages-hospital-${h.id}`"
            @click="onSelectHospital(h)"
          >
            <text class="admin-packages__hospital-name">{{ h.name }}</text>
            <text class="admin-packages__hospital-city">{{ h.city }} · {{ h.level }}</text>
          </view>
        </view>
      </UiCard>

      <!-- 套餐列表 -->
      <view v-if="selectedHospitalId" data-testid="admin-packages-packages-section">
        <UiCard title="套餐列表">
          <view v-if="packagesLoading" data-testid="admin-packages-packages-loading">
            <UiLoading text="加载套餐中..." />
          </view>

          <view
            v-else-if="packagesError"
            data-testid="admin-packages-packages-error"
          >
            <UiEmpty icon="⚠️" title="加载失败" :description="packagesError" />
          </view>

          <view
            v-else-if="packages.length === 0"
            data-testid="admin-packages-packages-empty"
          >
            <UiEmpty icon="📦" title="暂无套餐" />
          </view>

          <view v-else data-testid="admin-packages-packages-list">
            <view
              v-for="p in packages"
              :key="p.id"
              class="admin-packages__package"
              :data-testid="`admin-packages-package-${p.id}`"
            >
              <view class="admin-packages__package-info">
                <text class="admin-packages__package-title">{{ p.title }}</text>
                <text class="admin-packages__package-desc">{{ p.description }}</text>
                <text class="admin-packages__package-duration">⏱ {{ p.duration_minutes }} 分钟</text>
              </view>
              <text class="admin-packages__package-price">{{ formatYuan(p.price) }}</text>
            </view>
          </view>
        </UiCard>
      </view>
    </view>
  </view>
</template>

<style scoped>
.admin-packages__header {
  padding: var(--ui-space-base) 0;
}

.admin-packages__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-packages__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-packages__sections {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.admin-packages__hospital-list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
  max-height: 300px;
  overflow-y: auto;
}

.admin-packages__hospital {
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  background: var(--ui-color-bg-hover);
  cursor: pointer;
}

.admin-packages__hospital--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
}

.admin-packages__hospital--active .admin-packages__hospital-city {
  color: var(--ui-color-text-inverse);
  opacity: 0.85;
}

.admin-packages__hospital-name {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  display: block;
}

.admin-packages__hospital-city {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-packages__package {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: var(--ui-space-sm) 0;
  border-bottom: 1px solid var(--ui-color-border);
}

.admin-packages__package:last-child {
  border-bottom: none;
}

.admin-packages__package-info {
  flex: 1;
}

.admin-packages__package-title {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-packages__package-desc,
.admin-packages__package-duration {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-packages__package-price {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-error);
}
</style>