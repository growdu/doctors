<script setup lang="ts">
/**
 * admin/reviews/index.vue — 评价管理（v2 unified-app · admin 域 · v2.1 升级）。
 *
 * v2.1 升级：从"占位 toast"升级为"实时拉取评价 + 评分聚合 dashboard"。
 * 后端 listReviews 已可用，本页展示全量评价（按 rating 聚合）。
 *
 * 设计要点：
 *   - 4 张统计卡：总数 / 平均分 / 5 星 / 差评(≤3)
 *   - 列表：星级 + 标签 + 内容 + 创建时间
 *   - 区块级 catch：失败不阻塞整体渲染
 */
import { ref, computed, onMounted } from 'vue';
import { listReviews } from '@/api/review';
import type { Review } from '@/api/review';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const reviews = ref<Review[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listReviews({ page_size: 100 });
    reviews.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

const stats = computed(() => {
  const total = reviews.value.length;
  const sumRating = reviews.value.reduce((s, r) => s + r.rating, 0);
  const avg = total > 0 ? sumRating / total : 0;
  const fiveStar = reviews.value.filter((r) => r.rating === 5).length;
  const lowRating = reviews.value.filter((r) => r.rating <= 3).length;
  return { total, avg, fiveStar, lowRating };
});

function formatRating(r: number): string {
  return r.toFixed(1);
}

function renderStars(rating: number): string {
  return '★'.repeat(rating) + '☆'.repeat(5 - rating);
}

function formatDate(iso?: string): string {
  if (!iso) return '—';
  return iso.split('T')[0];
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-reviews-page">
    <view class="admin-reviews__header">
      <text class="admin-reviews__title">评价管理</text>
      <text class="admin-reviews__subtitle">实时统计 · 拉自 review-service</text>
    </view>

    <view v-if="loading && reviews.length === 0" data-testid="admin-reviews-loading">
      <UiLoading text="加载评价..." />
    </view>

    <view v-else-if="error && reviews.length === 0" data-testid="admin-reviews-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <UiButton type="primary" size="sm" @click="onLoad" data-testid="admin-reviews-retry">重试</UiButton>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <!-- 4 张统计卡 -->
      <view class="admin-reviews__stats" data-testid="admin-reviews-stats">
        <view class="admin-reviews__stat admin-reviews__stat--primary" data-testid="admin-reviews-stat-total">
          <text class="admin-reviews__stat-value">{{ stats.total }}</text>
          <text class="admin-reviews__stat-label">总评价</text>
        </view>
        <view class="admin-reviews__stat admin-reviews__stat--warning" data-testid="admin-reviews-stat-avg">
          <text class="admin-reviews__stat-value">{{ formatRating(stats.avg) }}</text>
          <text class="admin-reviews__stat-label">平均分</text>
        </view>
        <view class="admin-reviews__stat admin-reviews__stat--success" data-testid="admin-reviews-stat-five">
          <text class="admin-reviews__stat-value">{{ stats.fiveStar }}</text>
          <text class="admin-reviews__stat-label">5 星</text>
        </view>
        <view class="admin-reviews__stat admin-reviews__stat--error" data-testid="admin-reviews-stat-low">
          <text class="admin-reviews__stat-value">{{ stats.lowRating }}</text>
          <text class="admin-reviews__stat-label">差评 (≤3)</text>
        </view>
      </view>

      <!-- 评价列表 -->
      <view class="admin-reviews__section-title">⭐ 最近评价</view>
      <view v-if="reviews.length === 0" data-testid="admin-reviews-empty">
        <UiEmpty icon="⭐" title="暂无评价" />
      </view>
      <view v-else data-testid="admin-reviews-list">
        <UiCard
          v-for="r in reviews.slice(0, 20)"
          :key="r.id"
          :data-testid="`admin-review-${r.id}`"
        >
          <view class="admin-review__row">
            <text class="admin-review__stars" :data-testid="`admin-review-rating-${r.id}`">
              {{ renderStars(r.rating) }}
            </text>
            <text class="admin-review__date">{{ formatDate(r.created_at) }}</text>
          </view>
          <text v-if="r.tags.length > 0" class="admin-review__tags">
            <text v-for="t in r.tags" :key="t" class="admin-review__tag">#{{ t }}</text>
          </text>
          <text class="admin-review__content">{{ r.content }}</text>
          <view v-if="r.reply" class="admin-review__reply" :data-testid="`admin-review-reply-${r.id}`">
            <text class="admin-review__reply-label">陪诊师回复：</text>
            <text class="admin-review__reply-text">{{ r.reply }}</text>
          </view>
          <text class="admin-review__meta">订单 #{{ r.order_id }} · 陪诊师 #{{ r.escort_id }} · 患者 #{{ r.patient_id }}</text>
        </UiCard>
      </view>
    </template>
  </view>
</template>

<style scoped>
.admin-reviews__header {
  padding: var(--ui-space-base) 0 var(--ui-space-sm);
}

.admin-reviews__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-reviews__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-reviews__stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ui-space-md);
  margin-bottom: var(--ui-space-md);
}

.admin-reviews__stat {
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md);
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: var(--ui-shadow-sm);
}

.admin-reviews__stat-value {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-bold);
  display: block;
}

.admin-reviews__stat-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  margin-top: var(--ui-space-xs);
}

.admin-reviews__stat--primary .admin-reviews__stat-value { color: var(--ui-color-primary); }
.admin-reviews__stat--warning .admin-reviews__stat-value { color: var(--ui-color-warning); }
.admin-reviews__stat--success .admin-reviews__stat-value { color: var(--ui-color-success); }
.admin-reviews__stat--error .admin-reviews__stat-value { color: var(--ui-color-error); }

.admin-reviews__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-md) 0 var(--ui-space-sm);
}

.admin-review__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--ui-space-xs);
}

.admin-review__stars {
  font-size: var(--ui-font-md);
  color: var(--ui-color-warning);
  letter-spacing: 2px;
}

.admin-review__date {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
}

.admin-review__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ui-space-xs);
  margin-bottom: var(--ui-space-xs);
}

.admin-review__tag {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-primary);
  background: var(--ui-color-bg-hover);
  padding: 1px 6px;
  border-radius: var(--ui-radius-pill);
}

.admin-review__content {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
  display: block;
  margin-bottom: var(--ui-space-xs);
  line-height: 1.5;
}

.admin-review__reply {
  background: var(--ui-color-bg-hover);
  border-left: 3px solid var(--ui-color-primary);
  padding: var(--ui-space-xs) var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-xs);
}

.admin-review__reply-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-primary);
  font-weight: var(--ui-font-weight-medium);
}

.admin-review__reply-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: 2px;
}

.admin-review__meta {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: var(--ui-space-xs);
}
</style>