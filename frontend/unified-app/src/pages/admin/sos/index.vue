<script setup lang="ts">
/**
 * admin/sos/index.vue — SOS 紧急工单管理（v2 unified-app · admin 域 · v2.1 升级）。
 *
 * v2.1 升级：从"占位 toast"升级为"实时拉取 SOS + 状态聚合 dashboard"。
 * 后端 listSos 已可用，本页展示全量 SOS 工单（按 status 聚合）。
 *
 * 设计要点：
 *   - 4 张状态卡：pending / in_progress / resolved / closed
 *   - 列表：触发人 / 关联订单 / 触发原因 / 触发时间 / 状态徽章
 *   - 区块级 catch：失败不阻塞整体渲染
 */
import { ref, computed, onMounted } from 'vue';
import { listSos } from '@/api/sos';
import type { Sos, SosStatus } from '@/api/sos';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const sosList = ref<Sos[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listSos({ page_size: 100 });
    sosList.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

const stats = computed(() => {
  const byStatus: Record<SosStatus, number> = {
    pending: 0, in_progress: 0, resolved: 0, closed: 0,
  };
  for (const s of sosList.value) {
    byStatus[s.status] = (byStatus[s.status] ?? 0) + 1;
  }
  return { total: sosList.value.length, byStatus };
});

const STATUS_LABEL: Record<SosStatus, string> = {
  pending: '待处理',
  in_progress: '处理中',
  resolved: '已解决',
  closed: '已关闭',
};

const STATUS_CLASS: Record<SosStatus, string> = {
  pending: 'admin-sos__status--pending',
  in_progress: 'admin-sos__status--progress',
  resolved: 'admin-sos__status--resolved',
  closed: 'admin-sos__status--closed',
};

function formatDate(iso?: string): string {
  if (!iso) return '—';
  const t = iso.split('T');
  return t[0] + ' ' + (t[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-sos-page">
    <view class="admin-sos__header">
      <text class="admin-sos__title">SOS 管理</text>
      <text class="admin-sos__subtitle">实时统计 · 拉自 sos-service</text>
    </view>

    <view v-if="loading && sosList.length === 0" data-testid="admin-sos-loading">
      <UiLoading text="加载 SOS..." />
    </view>

    <view v-else-if="error && sosList.length === 0" data-testid="admin-sos-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <UiButton type="primary" size="sm" @click="onLoad" data-testid="admin-sos-retry">重试</UiButton>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <!-- 4 张状态卡 -->
      <view class="admin-sos__stats" data-testid="admin-sos-stats">
        <view
          v-for="(count, status) in stats.byStatus"
          :key="status"
          class="admin-sos__stat"
          :data-testid="`admin-sos-stat-${status}`"
        >
          <text class="admin-sos__stat-value">{{ count }}</text>
          <text class="admin-sos__stat-label">{{ STATUS_LABEL[status] }}</text>
        </view>
      </view>

      <!-- SOS 列表 -->
      <view class="admin-sos__section-title">🚨 最近 SOS</view>
      <view v-if="sosList.length === 0" data-testid="admin-sos-empty">
        <UiEmpty icon="🛟" title="暂无 SOS 工单" />
      </view>
      <view v-else data-testid="admin-sos-list">
        <UiCard
          v-for="s in sosList.slice(0, 20)"
          :key="s.id"
          :data-testid="`admin-sos-${s.id}`"
        >
          <view class="admin-sos-card__row">
            <text class="admin-sos-card__id">SOS #{{ s.id }}</text>
            <text
              class="admin-sos-card__status"
              :class="STATUS_CLASS[s.status]"
              :data-testid="`admin-sos-status-${s.id}`"
            >
              {{ STATUS_LABEL[s.status] }}
            </text>
          </view>
          <text class="admin-sos-card__reason">{{ s.reason }}</text>
          <text class="admin-sos-card__meta">
            触发人 #{{ s.trigger_user_id }}
            <text v-if="s.order_id"> · 订单 #{{ s.order_id }}</text>
            · {{ formatDate(s.created_at) }}
          </text>
          <text v-if="s.location" class="admin-sos-card__location">📍 {{ s.location.address }}</text>
          <text v-if="s.contact_phone_masked" class="admin-sos-card__phone">📱 {{ s.contact_phone_masked }}</text>
        </UiCard>
      </view>
    </template>
  </view>
</template>

<style scoped>
.admin-sos__header {
  padding: var(--ui-space-base) 0 var(--ui-space-sm);
}

.admin-sos__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-sos__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-sos__stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ui-space-md);
  margin-bottom: var(--ui-space-md);
}

.admin-sos__stat {
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: var(--ui-shadow-sm);
}

.admin-sos__stat-value {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-error);
  display: block;
}

.admin-sos__stat-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  margin-top: var(--ui-space-xs);
}

.admin-sos__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.admin-sos-card__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--ui-space-xs);
}

.admin-sos-card__id {
  font-size: var(--ui-font-base);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.admin-sos-card__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-pill);
  background: var(--ui-color-warning);
  color: var(--ui-color-text-inverse);
}

.admin-sos-card__status--pending { background: var(--ui-color-error); }
.admin-sos-card__status--progress { background: var(--ui-color-warning); }
.admin-sos-card__status--resolved { background: var(--ui-color-success); }
.admin-sos-card__status--closed { background: var(--ui-color-text-disabled); }

.admin-sos-card__reason {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
  display: block;
  margin-bottom: var(--ui-space-xs);
  line-height: 1.5;
}

.admin-sos-card__meta {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-bottom: 2px;
}

.admin-sos-card__location,
.admin-sos-card__phone {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
}
</style>