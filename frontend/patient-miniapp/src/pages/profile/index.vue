<!--
  src/pages/profile/index.vue

  个人中心 —— 患者端入口（plan Task M5）
  （spec §5 + plan v1 重构）

  页面流向：
    入口：tabBar me（uni-app 入口）
       → 本页
       → onLoad+mounted 调 useAuthStore().fetchMe() 拉当前用户 + useOrderStore().loadList({}) 拿订单数
       → 渲染用户信息 hero + 4 列订单状态入口 + 设置列表（钱包 / 优惠券 / 地址 / 设置）

  模板要点：
    - 顶部 <u-navbar>「我的」+ 自动返回（隐藏）
    - hero：头像 + 昵称 + 手机号（脱敏）
    - 4 列订单状态（待付款 / 待选陪诊 / 服务中 / 已完成）
    - 设置 list（钱包 / 优惠券 / 地址 / 设置 / 联系客服）
    - 退出登录（暂时禁用，提示后续 plan）

  数据来源：
    - 用户信息：useAuthStore().user（fetchMe 写入）
    - 我的订单数：useOrderStore().list（loadList 写入；按 status filter 分组）
    - 组件内维护本地 ref 是为了让 @vue/test-utils 能直接断言 DOM 渲染

  测试覆盖：src/pages/profile/index.test.js
-->
<template>
  <view class="page-profile" data-test="profile-page">
    <u-navbar title="我的" :auto-back="false" />

    <!-- hero -->
    <view class="page-profile__hero" data-test="hero">
      <view class="page-profile__avatar">
        <text class="page-profile__avatar-text">{{ avatarInitial }}</text>
      </view>
      <view class="page-profile__info">
        <text class="page-profile__name" data-test="nickname">{{ displayName }}</text>
        <text class="page-profile__phone" data-test="phone">{{ maskedPhoneText }}</text>
      </view>
    </view>

    <!-- 订单状态 4 列 -->
    <view class="page-profile__orders" data-test="orders-entry">
      <view class="page-profile__orders-header">
        <text class="page-profile__orders-title">我的订单</text>
        <text class="page-profile__orders-all" data-test="all-orders" @click="onAllOrders">全部 ›</text>
      </view>
      <view class="page-profile__orders-row">
        <view
          v-for="entry in ORDER_STATUS_TILES"
          :key="entry.key"
          class="page-profile__orders-tile"
          :data-test="'order-tile-' + entry.key"
          @click="onOrderTileClick(entry)"
        >
          <text class="page-profile__orders-tile-icon">{{ entry.icon }}</text>
          <text class="page-profile__orders-tile-label">{{ entry.label }}</text>
          <view
            v-if="entryCount(entry.key) > 0"
            class="page-profile__orders-tile-badge"
            :data-test="'order-tile-badge-' + entry.key"
          >{{ entryCount(entry.key) }}</view>
        </view>
      </view>
    </view>

    <!-- 设置 list -->
    <view class="page-profile__menu" data-test="menu">
      <view
        v-for="item in MENU_ITEMS"
        :key="item.key"
        class="page-profile__menu-item"
        :data-test="'menu-' + item.key"
        @click="onMenuClick(item)"
      >
        <text class="page-profile__menu-icon">{{ item.icon }}</text>
        <text class="page-profile__menu-label">{{ item.label }}</text>
        <text class="page-profile__menu-arrow">›</text>
      </view>
    </view>

    <!-- loading -->
    <view v-if="loading" class="page-profile__loading" data-test="loading">
      <u-skeleton :rows="4" :title="true" avatar />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-profile__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>
  </view>
</template>

<script>
// 个人中心页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / home）。
//
// 设计要点：
//   - 三态机：loading / error / loaded
//   - 用户信息：useAuthStore().fetchMe() → 写 user
//   - 我的订单：useOrderStore().loadList({}) → 按 status 分组计数
//   - onLoad+mounted：onLoad 先到（uni-app Page 钩子），mounted 立即拉

import { useAuthStore } from '@/stores/auth.js';
import { useOrderStore } from '@/stores/order.js';
import { maskPhone } from '@/utils/format.js';

// 4 个订单状态 tile —— label 与后端 status 1:1
const ORDER_STATUS_TILES = [
  { key: 'pendingPayment',           label: '待支付',         icon: '💰', status: 'pendingPayment' },
  { key: 'selectingEscort',          label: '待选陪诊',       icon: '🧑‍⚕️', status: 'selectingEscort' },
  { key: 'inService',                label: '服务中',         icon: '🏥', status: 'inService' },
  { key: 'completed',                label: '已完成',         icon: '✅', status: 'completed' },
];

// 设置 list —— 钱包 / 优惠券 / 地址 / 设置 / 联系客服
const MENU_ITEMS = [
  { key: 'wallet',    label: '我的钱包',     icon: '💳', url: '/pages/profile/wallet' },
  { key: 'coupons',   label: '优惠券',       icon: '🎟', url: '/pages/coupons/index' },
  { key: 'address',   label: '地址管理',     icon: '🏠', url: '/pages/address/list' },
  { key: 'settings',  label: '设置',         icon: '⚙️', url: '/pages/profile/settings' },
  { key: 'support',   label: '联系客服',     icon: '💬', url: '/pages/profile/support' },
];

export default {
  name: 'ProfilePage',
  data() {
    return {
      ORDER_STATUS_TILES,
      MENU_ITEMS,
      orders: [],
      loading: false,
      loadError: false,
    };
  },
  computed: {
    authStore() {
      return useAuthStore();
    },
    orderStore() {
      return useOrderStore();
    },
    // 用户昵称（兜底：登录后显示「我的」，未登录显示「未登录」）
    displayName() {
      const u = this.authStore.user;
      if (u && (u.nickname || u.name)) return u.nickname || u.name;
      return this.authStore.isLoggedIn ? '我的' : '未登录';
    },
    // 手机号脱敏展示
    maskedPhoneText() {
      const u = this.authStore.user;
      if (!u || !u.phone) return '';
      return maskPhone(String(u.phone));
    },
    // 头像占位字符：取昵称首字 / 手机号末位
    avatarInitial() {
      const u = this.authStore.user;
      if (u && (u.nickname || u.name)) {
        return (u.nickname || u.name).substring(0, 1);
      }
      if (u && u.phone) return String(u.phone).slice(-1);
      return '我';
    },
  },
  methods: {
    /**
     * 拉用户信息 + 我的订单（fetchProfile 是页面实际拉数据的入口）。
     */
    async fetchProfile() {
      this.loading = true;
      this.loadError = false;
      try {
        // 拉用户信息（已登录态下）
        if (this.authStore.isLoggedIn) {
          try {
            await this.authStore.fetchMe();
          } catch (_e) {
            // fetchMe 失败不阻断（用户信息可能从 loginByPhone 已缓存）
          }
        }
        // 拉我的订单
        const items = await this.orderStore.loadList({});
        this.orders = Array.isArray(items) ? items : [];
      } catch (_e) {
        this.loadError = true;
        this.orders = [];
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    onRetry() {
      this.fetchProfile();
    },

    /** 「全部订单」 */
    onAllOrders() {
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: '/pages/order/index' });
    },

    /** 订单状态 tile 点击 → 跳订单列表（带 status） */
    onOrderTileClick(entry) {
      if (!entry) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: `/pages/order/index?status=${entry.status}` });
    },

    /** 设置 menu 点击 */
    onMenuClick(item) {
      if (!item || !item.url) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: item.url });
    },

    /** 计算每个订单状态的订单数 */
    entryCount(statusKey) {
      const tile = this.ORDER_STATUS_TILES.find((t) => t.key === statusKey);
      if (!tile) return 0;
      return this.orders.filter((o) => o && o.status === tile.status).length;
    },
  },
  onLoad() {
    // 占位
  },
  mounted() {
    this.fetchProfile();
  },
};
</script>

<style lang="scss" scoped>
.page-profile {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 24px;
}

.page-profile__hero {
  padding: 24px 20px;
  background: linear-gradient(135deg, #1989fa 0%, #4eb7ff 100%);
  color: #fff;
  display: flex;
  align-items: center;
}

.page-profile__avatar {
  width: 64px;
  height: 64px;
  border-radius: 32px;
  background: rgba(255, 255, 255, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  flex-shrink: 0;
}

.page-profile__avatar-text {
  color: #fff;
  font-size: 28px;
  font-weight: 600;
}

.page-profile__info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.page-profile__name {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 6px;
}

.page-profile__phone {
  font-size: 13px;
  opacity: 0.85;
}

.page-profile__orders {
  background: #fff;
  margin: -16px 12px 12px;
  border-radius: 12px;
  padding: 14px 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-profile__orders-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 10px;
  border-bottom: 1px solid #f0f0f0;
}

.page-profile__orders-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.page-profile__orders-all {
  font-size: 13px;
  color: #1989fa;
}

.page-profile__orders-row {
  display: flex;
  padding-top: 12px;
}

.page-profile__orders-tile {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  padding: 4px 0;
}

.page-profile__orders-tile-icon {
  font-size: 24px;
  margin-bottom: 4px;
}

.page-profile__orders-tile-label {
  font-size: 12px;
  color: #606266;
}

.page-profile__orders-tile-badge {
  position: absolute;
  top: -2px;
  right: 16%;
  background: #ff4d4f;
  color: #fff;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 8px;
  min-width: 16px;
  text-align: center;
}

.page-profile__menu {
  background: #fff;
  margin: 0 12px;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-profile__menu-item {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.page-profile__menu-item:last-child {
  border-bottom: none;
}

.page-profile__menu-icon {
  font-size: 20px;
  margin-right: 12px;
}

.page-profile__menu-label {
  font-size: 15px;
  color: #303133;
  flex: 1;
}

.page-profile__menu-arrow {
  font-size: 18px;
  color: #c0c4cc;
}

.page-profile__loading,
.page-profile__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-profile__error .u-button { margin-top: 16px; width: 50%; }
</style>