<!--
  src/pages/reviews/index/index.vue

  「我的评价」列表页 —— 患者对历史订单的评价（plan v1.1）
  （spec §4.6）

  页面流向：
    入口：profile「我的评价」menu / 订单详情「查看评价」按钮
       → 本页
       → onLoad+mounted 调 useReviewStore().loadList({ mine: true })
       → 渲染评价卡片列表（订单号 / 陪诊师 / 星级 / 摘要 / 时间）
       → 点击卡片 → /pages/order/detail?orderId=xxx
       → 「去评价」→ /pages/reviews/create?orderId=xxx&escortId=xxx（从订单列表来）

  模板要点：
    - 顶部 <u-navbar>「我的评价」+ 自动返回
    - 主体 ListView 循环评价卡片（u-card 风格）
    - 空状态：u-empty「暂无评价」
    - loading / error / empty / loaded 四态机
    - 「待评价」提示：可评价的订单列表（v1 mock 用 placeholder 行展示）

  数据来源：
    - 评价列表镜像自 useReviewStore().mine（store 暴露的 ref）
-->
<template>
  <view class="page-reviews-index" data-test="reviews-index-page">
    <u-navbar title="我的评价" :auto-back="true" />

    <!-- tab：已评价 / 待评价 -->
    <view class="page-reviews-index__tabs" data-test="tabs">
      <view
        v-for="t in TABS"
        :key="t.key"
        class="page-reviews-index__tab"
        :class="{ 'page-reviews-index__tab--active': currentTab === t.key }"
        :data-test="'tab-' + t.key"
        @click="onTabChange(t.key)"
      >{{ t.label }}</view>
    </view>

    <!-- loading -->
    <view v-if="loading && reviews.length === 0" class="page-reviews-index__loading" data-test="loading">
      <u-skeleton :rows="3" :title="true" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-reviews-index__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- empty -->
    <view v-else-if="reviews.length === 0" class="page-reviews-index__empty" data-test="empty-state">
      <u-empty
        :text="currentTab === 'reviewed' ? '暂无评价' : '没有待评价订单'"
        mode="list"
      />
    </view>

    <!-- list -->
    <view v-else class="page-reviews-index__list">
      <view
        v-for="r in reviews"
        :key="r.id"
        class="page-reviews-index__card"
        :data-test="'review-card-' + r.id"
        :data-review-id="r.id"
        :data-order-id="r.order_id"
        @click="onReviewClick(r)"
      >
        <view class="page-reviews-index__card-row">
          <text class="page-reviews-index__card-order">订单 #{{ r.order_id }}</text>
          <text
            class="page-reviews-index__card-stars"
            :data-test="'review-stars-' + r.id"
          >{{ '★'.repeat(Math.max(0, Math.min(5, r.rating || 0))) }}</text>
        </view>
        <text
          v-if="r.comment"
          class="page-reviews-index__card-comment"
          data-test="review-comment"
        >{{ r.comment }}</text>
        <view class="page-reviews-index__card-meta">
          <text class="page-reviews-index__card-date">{{ formatDate(r.created_at) }}</text>
          <text
            v-if="r.reply"
            class="page-reviews-index__card-reply"
            data-test="review-reply-badge"
          >客服已回复</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
// 我的评价列表 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / home）。
//
// 设计要点：
//   - 状态机：loading / error / loaded + empty / loaded + list
//   - 切换 tab 不重拉：复用已加载的数据
//   - onLoad+mounted：onLoad 读 ?tab=xxx，mounted 立即拉一次
//   - 占位：v1 不接"待评价"订单接口，仅展示已评价列表；
//     待评价 tab 以 u-empty 占位告知用户

import { useReviewStore } from '@/stores/review.js';

// tab 定义
const TABS = [
  { key: 'reviewed', label: '已评价' },
  { key: 'pending',  label: '待评价' },
];

export default {
  name: 'ReviewsIndexPage',
  data() {
    return {
      TABS,
      currentTab: 'reviewed',
      loading: false,
      loadError: false,
      _loaded: false,
    };
  },
  computed: {
    reviewStore() {
      return useReviewStore();
    },
    reviews() {
      // 待评价 tab 在 v1 用空列表占位（接口未接）
      if (this.currentTab === 'pending') return [];
      return Array.isArray(this.reviewStore.mine) ? this.reviewStore.mine : [];
    },
    loading() {
      return Boolean(this.reviewStore.loading);
    },
  },
  methods: {
    async fetchList() {
      this.loadError = false;
      try {
        await this.reviewStore.loadList({ page: 1, page_size: 20 });
      } catch (_e) {
        this.loadError = true;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this._loaded = true;
      }
    },

    /** u-tabs click → 切 tab（不再重拉，复用 store 缓存） */
    onTabChange(key) {
      if (!key) return;
      this.currentTab = key;
    },

    onRetry() {
      this.fetchList();
    },

    /** 卡片点击 → 跳订单详情 */
    onReviewClick(r) {
      if (!r || !r.order_id) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: `/pages/order/detail?orderId=${r.order_id}` });
    },

    /**
     * 时间格式化（ISO8601 → 「yyyy-MM-dd」）。
     * 容错：解析失败 → 空字符串。
     */
    formatDate(iso) {
      if (!iso) return '';
      const t = new Date(iso);
      if (Number.isNaN(t.getTime())) return '';
      const pad = (n) => String(n).padStart(2, '0');
      return `${t.getFullYear()}-${pad(t.getMonth() + 1)}-${pad(t.getDate())}`;
    },
  },
  onLoad(query) {
    if (query && query.tab === 'pending') {
      this.currentTab = 'pending';
    }
  },
  mounted() {
    if (!this._loaded) {
      this.fetchList();
    }
  },
};
</script>

<style lang="scss" scoped>
.page-reviews-index {
  min-height: 100vh;
  background: #f5f7fa;
}

.page-reviews-index__tabs {
  background: #fff;
  display: flex;
  border-bottom: 1px solid #f0f0f0;
}

.page-reviews-index__tab {
  flex: 1;
  text-align: center;
  padding: 14px 0;
  font-size: 14px;
  color: #606266;
  position: relative;
}

.page-reviews-index__tab--active {
  color: #1989fa;
  font-weight: 600;
  &::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 50%;
    transform: translateX(-50%);
    width: 28px;
    height: 3px;
    background: #1989fa;
    border-radius: 2px;
  }
}

.page-reviews-index__loading,
.page-reviews-index__empty,
.page-reviews-index__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-reviews-index__error .u-button { margin-top: 16px; width: 50%; }

.page-reviews-index__list {
  padding: 12px 16px;
}

.page-reviews-index__card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-reviews-index__card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.page-reviews-index__card-order {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.page-reviews-index__card-stars {
  font-size: 14px;
  color: #fa8c16;
  letter-spacing: 1px;
}

.page-reviews-index__card-comment {
  display: block;
  font-size: 13px;
  color: #606266;
  line-height: 20px;
  margin-bottom: 6px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.page-reviews-index__card-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #909399;
}

.page-reviews-index__card-reply {
  background: #e8f3ff;
  color: #1989fa;
  padding: 1px 6px;
  border-radius: 4px;
}
</style>
</content>
</invoke>