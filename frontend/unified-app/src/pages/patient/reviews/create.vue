<script setup lang="ts">
/**
 * patient/reviews/create.vue — 创建评价（v2 unified-app）。
 *
 * 入口：order/detail（completed 状态后）「评价」/ 推送通知 → 创建评价
 *   → 5 星评分 + 标签多选 + 文字评价
 *   → 调 api/review.createReview({ order_id, rating, tags, content })
 *
 * 设计要点：
 *   - 评分：5 个星星按钮（点击切换）
 *   - 标签：7 个预置标签（态度好 / 守时 / 专业 / 细心 / 沟通好 / 经验丰富 / 准时完成）
 *   - 评价内容 ≤ 500 字（realText）
 */
import { ref, computed, onMounted } from 'vue';
import { createReview } from '@/api/review';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';

const orderId = ref<number | null>(null);
const rating = ref(0);
const tags = ref<string[]>([]);
const content = ref('');
const submitting = ref(false);
const error = ref<string | null>(null);

const AVAILABLE_TAGS = ['态度好', '守时', '专业', '细心', '沟通好', '经验丰富', '准时完成'];

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.orderId) orderId.value = Number(opts.orderId);
}

function onRating(n: number) {
  rating.value = n;
}

function onTagToggle(tag: string) {
  const idx = tags.value.indexOf(tag);
  if (idx >= 0) {
    tags.value.splice(idx, 1);
  } else {
    tags.value.push(tag);
  }
}

const canSubmit = computed(() => rating.value > 0 && content.value.trim().length >= 5);

async function onSubmit() {
  if (!canSubmit.value || !orderId.value) return;
  submitting.value = true;
  error.value = null;
  try {
    await createReview({
      order_id: orderId.value,
      rating: rating.value,
      tags: tags.value,
      content: content.value.trim(),
    });
    if (typeof uni !== 'undefined') {
      uni.showToast({ title: '评价已提交', icon: 'success' });
      uni.navigateBack({ delta: 1 });
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    submitting.value = false;
  }
}

onMounted(parseQuery);
</script>

<template>
  <view class="ui-page" data-testid="patient-review-create">
    <UiCard title="评价陪诊师">
      <view class="review-create__rating" data-testid="review-rating">
        <text class="review-create__label">评分</text>
        <view class="review-create__stars">
          <text
            v-for="n in 5"
            :key="n"
            class="review-create__star"
            :class="{ 'review-create__star--active': n <= rating }"
            :data-testid="`review-star-${n}`"
            @click="onRating(n)"
          >
            {{ n <= rating ? '★' : '☆' }}
          </text>
        </view>
        <text class="review-create__rating-value">{{ rating }} / 5</text>
      </view>

      <view class="review-create__tags" data-testid="review-tags">
        <text class="review-create__label">标签（多选）</text>
        <view class="review-create__tag-list">
          <text
            v-for="tag in AVAILABLE_TAGS"
            :key="tag"
            class="review-create__tag"
            :class="{ 'review-create__tag--active': tags.includes(tag) }"
            :data-testid="`review-tag-${tag}`"
            @click="onTagToggle(tag)"
          >
            {{ tag }}
          </text>
        </view>
      </view>

      <view class="review-create__content">
        <UiInput
          v-model="content"
          label="评价"
          placeholder="分享这次陪诊体验（5-500 字）"
          type="textarea"
          :maxlength="500"
          data-testid="review-content"
        />
      </view>

      <view v-if="error" class="review-create__error" data-testid="review-error">{{ error }}</view>
    </UiCard>

    <view class="review-create__actions">
      <UiButton
        type="primary"
        block
        :loading="submitting"
        :disabled="!canSubmit"
        data-testid="review-submit-btn"
        @click="onSubmit"
      >
        提交评价
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.review-create__rating {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
  padding: var(--ui-space-md) 0;
}

.review-create__label {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  flex-shrink: 0;
  width: 64px;
}

.review-create__stars {
  display: flex;
  gap: var(--ui-space-xs);
}

.review-create__star {
  font-size: 28px;
  color: var(--ui-color-border);
  cursor: pointer;
  transition: color var(--ui-duration-fast);
}

.review-create__star--active {
  color: var(--ui-color-warning);
}

.review-create__rating-value {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  margin-left: var(--ui-space-sm);
}

.review-create__tags {
  padding: var(--ui-space-md) 0;
  border-top: 1px solid var(--ui-color-divider);
}

.review-create__tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-sm);
}

.review-create__tag {
  font-size: var(--ui-font-sm);
  padding: var(--ui-space-xs) var(--ui-space-md);
  background: var(--ui-color-bg-hover);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-pill);
  cursor: pointer;
  transition: all var(--ui-duration-fast);
}

.review-create__tag--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
}

.review-create__content {
  border-top: 1px solid var(--ui-color-divider);
  padding-top: var(--ui-space-md);
}

.review-create__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-top: var(--ui-space-md);
}

.review-create__actions {
  margin-top: var(--ui-space-base);
}
</style>