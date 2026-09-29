<!--
  src/pages/reviews/create.vue

  评价创建页 —— 已完成订单评价（plan Task M7）
  （spec §4.6 + plan v1 重构）

  页面流向：
    入口：订单详情页「评价」按钮（completed 状态）→ /pages/reviews/create?orderId=xxx&escortId=xxx
                                → 本页
                                → onLoad+mounted 读 query.orderId / query.escortId
                                → 用户选星级（1..5）+ 填评论 → 调 useReviewStore().submit()
                                → 成功后 → uni.redirectTo → /pages/order/detail

  模板要点：
    - 顶部 <u-navbar>「评价」+ 自动返回
    - 5 颗星选择（uView Plus u-rate）
    - 评论输入（textarea）
    - 「提交」按钮（未选星级时禁用）
    - 提交 loading（u-loading 占位）

  数据来源：
    - 入参：onLoad query.orderId / query.escortId
    - 提交：useReviewStore().submit({ order_id, escort_id, rating, comment })

  测试覆盖：src/pages/reviews/create.test.js
-->
<template>
  <view class="page-review-create" data-test="review-create-page">
    <u-navbar title="评价" :auto-back="true" />

    <view class="page-review-create__body" data-test="rating-card">
      <text class="page-review-create__label">请评分</text>
      <view class="page-review-create__stars">
        <view
          v-for="n in 5"
          :key="n"
          class="page-review-create__star"
          :class="{ 'page-review-create__star--active': rating >= n }"
          :data-test="'star-' + n"
          @click="onStarClick(n)"
        >★</view>
      </view>
      <text class="page-review-create__rating-text" data-test="rating-text">
        {{ rating ? `${rating} 星` : '未评分' }}
      </text>
    </view>

    <!-- v1.1: 标签 chips（多选） -->
    <view class="page-review-create__tags" data-test="tags-card">
      <text class="page-review-create__label">标签（可多选）</text>
      <view class="page-review-create__tag-list">
        <view
          v-for="t in REVIEW_TAGS"
          :key="t.key"
          class="page-review-create__tag"
          :class="{ 'page-review-create__tag--active': selectedTagKeys.includes(t.key) }"
          :data-test="'tag-' + t.key"
          :data-selected="selectedTagKeys.includes(t.key)"
          @click="onTagToggle(t.key)"
        >{{ t.label }}</view>
      </view>
    </view>

    <view class="page-review-create__comment">
      <text class="page-review-create__label">评价内容</text>
      <textarea
        v-model="comment"
        class="page-review-create__textarea"
        data-test="comment-input"
        placeholder="请输入评价内容（最多 500 字）"
        :maxlength="500"
      />
      <!-- v1.1: 字符计数 -->
      <text
        class="page-review-create__counter"
        data-test="char-counter"
      >{{ comment.length }} / 500</text>
    </view>

    <view class="page-review-create__bottom">
      <u-button
        type="primary"
        :disabled="!canSubmit"
        :data-test="'submit-btn'"
        @click="onSubmit"
      >{{ submitting ? '提交中' : '提交' }}</u-button>
    </view>

    <!-- loading -->
    <view v-if="submitting" class="page-review-create__loading" data-test="loading">
      <u-loading mode="circle" size="32" />
    </view>
  </view>
</template>

<script>
// 评价创建页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index）。
//
// 设计要点：
//   - rating 1..5：点星直接设置；未评分时按钮禁用
//   - comment：textarea；maxlength=500（后端兜底）
//   - 提交成功 → uni.redirectTo 跳回订单详情

import { useReviewStore } from '@/stores/review.js';

// v1.1 增量 —— 评价标签（多选）
const REVIEW_TAGS = [
  { key: 'professional', label: '专业' },
  { key: 'patient',      label: '耐心' },
  { key: 'punctual',     label: '准时' },
  { key: 'considerate',  label: '周到' },
  { key: 'friendly',     label: '态度好' },
];

export default {
  name: 'ReviewCreatePage',
  data() {
    return {
      orderId: null,
      escortId: null,
      rating: 0,    // 1..5
      comment: '',
      selectedTagKeys: [], // v1.1 多选 tag
      REVIEW_TAGS,
      submitting: false,
    };
  },
  computed: {
    reviewStore() {
      return useReviewStore();
    },
    canSubmit() {
      return this.rating >= 1 && this.rating <= 5 && !this.submitting;
    },
  },
  methods: {
    onStarClick(n) {
      this.rating = n;
    },

    /** v1.1: tag 多选 toggle */
    onTagToggle(key) {
      const idx = this.selectedTagKeys.indexOf(key);
      if (idx >= 0) {
        this.selectedTagKeys.splice(idx, 1);
      } else {
        this.selectedTagKeys.push(key);
      }
    },

    async onSubmit() {
      if (!this.canSubmit) return;
      if (!this.orderId || !this.escortId) {
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '订单信息缺失', icon: 'none' });
        }
        return;
      }
      this.submitting = true;
      try {
        // v1.1: 拼接 tags 到 comment 前面（v1 后端 review api 不支持 tags 字段，先合并到 comment）
        const tagPrefix = (this.selectedTagKeys || [])
          .map((k) => {
            const found = REVIEW_TAGS.find((t) => t.key === k);
            return found ? `#${found.label}` : '';
          })
          .filter(Boolean)
          .join(' ');
        const finalComment = tagPrefix
          ? `${tagPrefix} ${this.comment || ''}`.trim()
          : (this.comment || '');
        await this.reviewStore.submit({
          order_id: this.orderId,
          escort_id: this.escortId,
          rating: this.rating,
          comment: finalComment,
        });
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '评价成功', icon: 'success' });
        }
        // 跳回订单详情
        if (typeof uni !== 'undefined' && typeof uni.redirectTo === 'function') {
          uni.redirectTo({ url: `/pages/order/detail?orderId=${this.orderId}` });
        }
      } catch (e) {
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({
            title: (e && e.message) || '提交失败，请重试',
            icon: 'none',
          });
        }
      } finally {
        this.submitting = false;
      }
    },
  },
  onLoad(query) {
    if (query) {
      if (query.orderId) this.orderId = query.orderId;
      if (query.escortId) this.escortId = query.escortId;
    }
  },
  mounted() {
    // 占位：mounted 不主动拉数据（仅渲染 + 等待用户填 + 提交）
  },
};
</script>

<style lang="scss" scoped>
.page-review-create {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 96px;
}

.page-review-create__body,
.page-review-create__tags,
.page-review-create__comment {
  background: #fff;
  margin: 12px;
  border-radius: 12px;
  padding: 16px;
}

.page-review-create__tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 6px;
}

.page-review-create__tag {
  padding: 6px 12px;
  border-radius: 16px;
  background: #f4f4f5;
  color: #606266;
  font-size: 13px;
}

.page-review-create__tag--active {
  background: #e8f3ff;
  color: #1989fa;
  font-weight: 500;
}

.page-review-create__counter {
  display: block;
  text-align: right;
  font-size: 11px;
  color: #909399;
  margin-top: 6px;
}

.page-review-create__label {
  display: block;
  font-size: 14px;
  color: #303133;
  font-weight: 500;
  margin-bottom: 12px;
}

.page-review-create__stars {
  display: flex;
  align-items: center;
  padding: 6px 0;
}

.page-review-create__star {
  font-size: 36px;
  color: #dcdfe6;
  margin-right: 12px;
}

.page-review-create__star--active {
  color: #fa8c16;
}

.page-review-create__rating-text {
  display: block;
  font-size: 13px;
  color: #909399;
  margin-top: 8px;
}

.page-review-create__textarea {
  width: 100%;
  min-height: 140px;
  font-size: 14px;
  color: #303133;
  padding: 8px;
  background: #f8f9fa;
  border-radius: 8px;
  box-sizing: border-box;
}

.page-review-create__bottom {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: #fff;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  z-index: 10;
  .u-button { width: 100%; }
}

.page-review-create__loading {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.5);
  z-index: 50;
}
</style>