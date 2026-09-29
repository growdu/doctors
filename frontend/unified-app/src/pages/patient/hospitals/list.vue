<script setup lang="ts">
/**
 * patient/hospitals/list.vue — 医院列表页（v2 unified-app）。
 *
 * 入口：patient 域首页「选择医院」/「下单」流程
 *   → onMounted 调 api/user.listHospitals({ city, keyword })
 *   → 渲染医院卡片列表 + 「选择医院」按钮
 *
 * 设计要点：
 *   - 简单搜索栏（UiInput）：城市 / 关键字过滤
 *   - UiCard 医院卡：名称 / 城市 / 地址 / 等级 / 「查看详情」
 */
import { ref, computed, onMounted } from 'vue';
import { listHospitals } from '@/api/user';
import type { Hospital } from '@/api/user';
import UiInput from '@/components/shared/UiInput.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const hospitals = ref<Hospital[]>([]);
const total = ref(0);
const loading = ref(false);
const error = ref<string | null>(null);

const city = ref('');
const keyword = ref('');

const hasFilter = computed(() => city.value.trim() || keyword.value.trim());

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listHospitals({
      city: city.value.trim() || undefined,
      keyword: keyword.value.trim() || undefined,
    });
    hospitals.value = r.items;
    total.value = r.total;
  } catch (e) {
    error.value = (e as Error).message;
    hospitals.value = [];
  } finally {
    loading.value = false;
  }
}

let debounceTimer: number | null = null;
function onSearchChange() {
  if (debounceTimer !== null) clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    onLoad();
  }, 300) as unknown as number;
}

function onClear() {
  city.value = '';
  keyword.value = '';
  onLoad();
}

function onViewDetail(h: Hospital) {
  if (typeof uni !== 'undefined') {
    uni.navigateTo({ url: `/pages/patient/hospitals/detail?id=${h.id}` });
  }
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-hospital-list">
    <!-- 搜索栏 -->
    <UiCard shadow="none" :no-padding="true">
      <view class="hospital-list__filters">
        <UiInput v-model="city" label="城市" placeholder="如：北京" :maxlength="16" @update:modelValue="onSearchChange" data-testid="hospital-filter-city" />
        <UiInput v-model="keyword" label="关键字" placeholder="医院名 / 地址" :maxlength="32" clearable @update:modelValue="onSearchChange" data-testid="hospital-filter-keyword" />
        <view v-if="hasFilter" class="hospital-list__clear">
          <text @click="onClear" data-testid="hospital-clear-filter">清除筛选</text>
        </view>
      </view>
    </UiCard>

    <view class="hospital-list__count" data-testid="hospital-count">
      共 {{ total }} 家医院
    </view>

    <view v-if="loading && hospitals.length === 0" data-testid="hospital-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="hospital-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="hospital-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="hospitals.length === 0" data-testid="hospital-empty">
      <UiEmpty icon="🏥" title="暂无医院" description="试试调整筛选条件" />
    </view>

    <view v-else data-testid="hospital-list">
      <UiCard
        v-for="h in hospitals"
        :key="h.id"
        :title="h.name"
        :data-testid="`hospital-card-${h.id}`"
      >
        <view class="hospital-card__row">
          <text class="hospital-card__city">{{ h.city }}</text>
          <text v-if="h.level" class="hospital-card__level">{{ h.level }}</text>
        </view>
        <text class="hospital-card__address">{{ h.address }}</text>
        <text v-if="h.phone" class="hospital-card__phone">📞 {{ h.phone }}</text>
        <template #footer>
          <view class="hospital-card__actions">
            <text class="hospital-card__view" :data-testid="`view-hospital-${h.id}`" @click="onViewDetail(h)">
              查看详情 / 套餐 →
            </text>
          </view>
        </template>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.hospital-list__filters {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.hospital-list__clear {
  text-align: right;
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  padding: var(--ui-space-xs) 0;
}

.hospital-list__count {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  padding: var(--ui-space-sm) var(--ui-space-xs);
}

.hospital-list__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}

.hospital-card__row {
  display: flex;
  align-items: center;
  gap: var(--ui-space-sm);
  margin-bottom: var(--ui-space-xs);
}

.hospital-card__city {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.hospital-card__level {
  font-size: var(--ui-font-xs);
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  padding: 1px 6px;
  border-radius: var(--ui-radius-sm);
}

.hospital-card__address {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.hospital-card__phone {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.hospital-card__actions {
  text-align: right;
}

.hospital-card__view {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
}
</style>