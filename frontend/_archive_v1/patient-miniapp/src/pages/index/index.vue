<!--
  src/pages/index/index.vue

  首页 —— 患者端入口（plan Task M3）
  （spec §4 + plan v1 重构）

  页面流向：
    入口：tabBar home（uni-app 入口）
       → 本页
       → onLoad+mounted 调 useHospitalStore().loadList({ limit: 5 }) 拉推荐医院
       → 顶部 hero（已登录 / 未登录）
       → 4 个快捷入口（创建订单 / 我的订单 / 优惠券 / 地址）
       → 公告 banner（写死占位）
       → 推荐医院（hospitals 列表前 N 条 + 「查看更多」）

  模板要点：
    - 顶部 <u-navbar>（fixed=false，让 hero 紧贴顶部）
    - hero：蓝紫渐变 banner，显示用户昵称 / 「请登录」
    - 4 列快捷入口（grid 风格）
    - 公告 banner（uView Plus u-notice 风格占位）
    - 推荐医院：u-card 风格卡片 + 「查看更多」按钮
    - loading / error / empty / loaded 四态机

  数据来源：
    - 推荐医院：useHospitalStore().list（store 暴露的 ref）
    - 组件内维护本地 ref 是为了让 @vue/test-utils 能直接断言 DOM 渲染

  测试覆盖：src/pages/index/index.test.js
-->
<template>
  <view class="page-home" data-test="home-page">
    <!-- 顶部 hero -->
    <view class="page-home__hero" data-test="hero">
      <text class="page-home__hero-title">{{ greeting }}</text>
      <text class="page-home__hero-subtitle">专业陪诊 · 安心就诊</text>
    </view>

    <!-- 快捷入口 -->
    <view class="page-home__quick" data-test="quick-actions">
      <view
        v-for="entry in QUICK_ENTRIES"
        :key="entry.key"
        class="page-home__quick-item"
        :data-test="'quick-' + entry.key"
        @click="onQuick(entry)"
      >
        <view class="page-home__quick-icon" :class="['page-home__quick-icon--' + entry.color]">
          {{ entry.icon }}
        </view>
        <text class="page-home__quick-label">{{ entry.label }}</text>
      </view>
    </view>

    <!-- 公告 banner -->
    <view class="page-home__notice" data-test="notice">
      <text class="page-home__notice-tag">公告</text>
      <text class="page-home__notice-text">{{ notice }}</text>
    </view>

    <!-- 推荐医院 loading -->
    <view v-if="loading" class="page-home__loading" data-test="loading">
      <u-skeleton :rows="2" :title="true" avatar avatar-shape="square" avatar-size="56" />
    </view>

    <!-- 推荐医院 error -->
    <view v-else-if="loadError" class="page-home__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- 推荐医院 empty -->
    <view v-else-if="recommended.length === 0" class="page-home__empty" data-test="empty-state">
      <u-empty text="暂无可用医院" mode="list" />
    </view>

    <!-- 推荐医院列表 -->
    <view v-else class="page-home__list" data-test="recommended">
      <view class="page-home__list-header">
        <text class="page-home__list-title">推荐医院</text>
        <text class="page-home__list-more" data-test="view-more" @click="onViewMore">查看更多 ›</text>
      </view>
      <view
        v-for="h in recommended"
        :key="h.id"
        class="page-home__hospital-card"
        :data-test="'hospital-card-' + h.id"
        :data-hospital-id="h.id"
        @click="onHospitalClick(h)"
      >
        <view class="page-home__hospital-row">
          <text class="page-home__hospital-name">{{ h.name }}</text>
          <text class="page-home__hospital-level">{{ h.level || '' }}</text>
        </view>
        <view v-if="h.address" class="page-home__hospital-row">
          <text class="page-home__hospital-addr">{{ h.address }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
// 首页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / detail）。
//
// 设计要点：
//   - 三态机：loading / error / loaded + empty / loaded + list
//   - 推荐医院：useHospitalStore().loadList({ limit: 5 })；本地 ref 镜像便于断言
//   - 快捷入口 → uni.navigateTo 对应路由（订单创建 / 我的订单 / 优惠券 / 地址）
//   - onLoad+mounted：onLoad 先到（uni-app Page 钩子，可能带 ?city_id=xxx）；
//     mounted 立即拉一次（与 order/index 同款两段式生命周期）

import { useHospitalStore } from '@/stores/hospital.js';

// ---- 快捷入口配置（图标用 emoji 占位 + 颜色后缀；生产由 uView Plus u-icon 替换）
const QUICK_ENTRIES = [
  { key: 'createOrder', label: '创建订单', icon: '➕', color: 'blue',   url: '/pages/order/create' },
  { key: 'myOrders',    label: '我的订单', icon: '📋', color: 'green',  url: '/pages/order/index' },
  { key: 'coupons',     label: '优惠券',   icon: '🎟', color: 'orange', url: '/pages/coupons/index' },
  { key: 'address',     label: '地址管理', icon: '🏠', color: 'purple', url: '/pages/address/list' },
];

export default {
  name: 'HomePage',
  data() {
    return {
      QUICK_ENTRIES,
      recommended: [],
      loading: false,
      loadError: false,
      notice: '陪诊服务升级：本周新增 12 家三甲医院，欢迎体验',
    };
  },
  computed: {
    hospitalStore() {
      return useHospitalStore();
    },
    // 已登录 → 显示昵称 / 未登录 → 「请登录」
    greeting() {
      // 占位：未接入 auth store.isLoggedIn 判断前，先用静态文案；后续 plan 接 auth
      return '你好，欢迎使用陪诊';
    },
  },
  methods: {
    /**
     * 拉推荐医院（fetchRecommended 是页面实际拉数据的入口）。
     * onLoad + mounted + onRetry 共用。
     */
    async fetchRecommended() {
      this.loading = true;
      this.loadError = false;
      try {
        const r = await this.hospitalStore.loadList({ limit: 5, page: 1 });
        const items = (r && r.items) || [];
        this.recommended = items.slice(0, 5);
      } catch (_e) {
        this.loadError = true;
        this.recommended = [];
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    onRetry() {
      this.fetchRecommended();
    },

    /**
     * 快捷入口 → 跳对应路由。
     * @param {object} entry
     */
    onQuick(entry) {
      if (!entry || !entry.url) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: entry.url });
    },

    /**
     * 医院卡片点击 → 跳医院详情。
     * @param {object} h
     */
    onHospitalClick(h) {
      if (!h || !h.id) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: `/pages/hospitals/detail?id=${h.id}` });
    },

    /** 「查看更多」 → 跳医院列表页 */
    onViewMore() {
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: '/pages/hospitals/list' });
    },
  },
  onLoad() {
    // 暂未读 query（如 ?city_id=xxx）；占位以保持两段式生命周期风格
  },
  mounted() {
    this.fetchRecommended();
  },
};
</script>

<style lang="scss" scoped>
.page-home {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 24px;
}

.page-home__hero {
  padding: 32px 20px 24px;
  background: linear-gradient(135deg, #1989fa 0%, #4eb7ff 100%);
  color: #fff;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.page-home__hero-title {
  font-size: 22px;
  font-weight: 600;
  margin-bottom: 8px;
}

.page-home__hero-subtitle {
  font-size: 14px;
  opacity: 0.85;
}

.page-home__quick {
  background: #fff;
  margin: -16px 12px 12px;
  border-radius: 12px;
  padding: 16px 0;
  display: flex;
  justify-content: space-around;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-home__quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.page-home__quick-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 22px;
  color: #fff;
  margin-bottom: 6px;
}
.page-home__quick-icon--blue   { background: #1989fa; }
.page-home__quick-icon--green  { background: #19be6b; }
.page-home__quick-icon--orange { background: #ff9900; }
.page-home__quick-icon--purple { background: #8a2be2; }

.page-home__quick-label {
  font-size: 13px;
  color: #303133;
}

.page-home__notice {
  background: #fff7e6;
  margin: 0 12px 12px;
  border-radius: 8px;
  padding: 10px 12px;
  display: flex;
  align-items: center;
}

.page-home__notice-tag {
  background: #ff9900;
  color: #fff;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  margin-right: 8px;
  flex-shrink: 0;
}

.page-home__notice-text {
  font-size: 13px;
  color: #5b4206;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.page-home__loading,
.page-home__empty,
.page-home__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-home__error .u-button { margin-top: 16px; width: 50%; }

.page-home__list {
  padding: 0 12px;
}

.page-home__list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 4px 8px;
}

.page-home__list-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.page-home__list-more {
  font-size: 13px;
  color: #1989fa;
}

.page-home__hospital-card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-home__hospital-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.page-home__hospital-name {
  font-size: 15px;
  font-weight: 500;
  color: #303133;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.page-home__hospital-level {
  font-size: 12px;
  color: #1989fa;
  background: #e8f3ff;
  padding: 2px 6px;
  border-radius: 4px;
  flex-shrink: 0;
  margin-left: 8px;
}

.page-home__hospital-addr {
  font-size: 13px;
  color: #909399;
}
</style>