<script setup lang="ts">
/**
 * patient/reviews/index.vue — 我的评价列表（v2 unified-app）。
 *
 * 入口：profile / wallet 等附近场景；主要查看自己创建的评价
 *   → 调 api/review.listReviews({ page })
 *   → 渲染评价卡：评分（星）+ 内容 + 时间
 *
 * 简化版：当前仅展示 patient 视角下自己创建过的评价（不带 escort_id 过滤）
 * （escort 域也复用 listReviews，按 escort_id 过滤）
 */
import { ref, onMounted } from 'vue';
import { listReviews } from '@/api/review';
import type { Review } from '@/api/review';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const reviews = ref<Review[]>([]);
const total = ref(0);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listReviews();
    reviews.value = r.items;
    total.value = r.total;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function renderStars(n: number): string {
  return '★'.repeat(n) + '☆'.repeat(5 - n);
}

function formatDate(iso: string): string {
  return iso.replace('T', ' ').substring(0, 19);
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-review-list">
    <view v-if="loading" data-testid="review-list-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="review-list-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="review-list__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="reviews.length === 0" data-testid="review-list-empty">
      <UiEmpty icon="⭐" title="还没有评价" description="订单完成后可以评价陪诊师" />
    </view>

    <view v-else data-testid="review-list">
      <view class="review-list__count">共 {{ total }} 条评价</view>
      <UiCard
        v-for="r in reviews"
        :key="r.id"
        :data-testid="`review-card-${r.id}`"
      >
        <view class="review-list__header">
          <text class="review-list__stars" :data-testid="`review-stars-${r.id}`">{{ renderStars(r.rating) }}</text>
          <text class="review-list__order">订单 #{{ r.order_id }}</text>
        </view>
        <text class="review-list__content" data-testid="review-content">{{ r.content }}</text>
        <view v-if="r.tags.length" class="review-list__tags">
          <text v-for="tag in r.tags" :key="tag" class="review-list__tag">{{ tag }}</text>
        </view>
        <view v-if="r.reply" class="review-list__reply">
          <text class="review-list__reply-label">商家回复</text>
          <text class="review-list__reply-text">{{ r.reply }}</text>
        </view>
        <text class="review-list__time">{{ formatDate(r.created_at) }}</text>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.review-list__count {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  padding: var(--ui-space-sm) 0;
}

.review-list__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--ui-space-sm);
}

.review-list__stars {
  color: var(--ui-color-warning);
  font-size: var(--ui-font-md);
  letter-spacing: 2px;
}

.review-list__order {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-disabled);
}

.review-list__content {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  line-height: 1.6;
  display: block;
  margin-bottom: var(--ui-space-sm);
}

.review-list__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ui-space-xs);
  margin-bottom: var(--ui-space-sm);
}

.review-list__tag {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  background: var(--ui-color-bg-hover);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
}

.review-list__reply {
  padding: var(--ui-space-sm);
  background: var(--ui-color-bg-hover);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-sm);
}

.review-list__reply-label {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-bottom: 2px;
}

.review-list__reply-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
  display: block;
}

.review-list__time {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
}

.review-list__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>