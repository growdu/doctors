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

    <view class="page-review-create__comment">
      <text class="page-review-create__label">评价内容</text>
      <textarea
        v-model="comment"
        class="page-review-create__textarea"
        data-test="comment-input"
        placeholder="请输入评价内容（最多 500 字）"
        :maxlength="500"
      />
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

export default {
  name: 'ReviewCreatePage',
  data() {
    return {
      orderId: null,
      escortId: null,
      rating: 0,    // 1..5
      comment: '',
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
        await this.reviewStore.submit({
          order_id: this.orderId,
          escort_id: this.escortId,
          rating: this.rating,
          comment: this.comment || '',
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
.page-review-create__comment {
  background: #fff;
  margin: 12px;
  border-radius: 12px;
  padding: 16px;
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