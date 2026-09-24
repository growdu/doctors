<!--
  src/pages/order/index.vue

  订单列表页 —— 患者端「我的订单」入口（plan Task P7）
  （依据 spec `2026-09-24-order-matching-redesign.md` + plan Task P7）

  页面流向：
    入口：uni-app home 入口 / admin-web 跳转（?status=xxx 过滤） / DashboardPage
    → 本页
    → onLoad 读 query.status → mounted 调 useOrderStore().loadList({ status })
    → 渲染 OrderListItem 卡片列表
    → 点击卡片 → /pages/order/detail?orderId=xxx
    → 点击「选陪诊师」按钮 → /pages/order/candidates/index?orderId=xxx
    → 点击「取消订单」按钮 → confirm → store.cancel(orderId, reason) → reload list

  模板要点：
    - 顶部 <u-navbar>「我的订单」+ 自动返回
    - 状态筛选 tab（uView Plus u-tabs）：8 项（全部 / 已支付 / 待选陪诊师 / 待陪诊师确认 /
      已接单 / 服务中 / 已完成 / 已取消）；切换 tab 重新 loadList
    - 主体 ListView 循环 OrderListItem 卡片（u-card 风格）：
        头部：订单号 + StatusBadge
        中部：医院 + 就诊时间
        底部：金额 + 操作按钮（详情 / 选陪诊师 / 取消）
        点击卡片跳 detail 页
    - 空状态：uView Plus u-empty
    - loading 状态：u-skeleton
    - 错误态：u-empty + 重试按钮 + toast

  数据来源：
    - 列表镜像自 useOrderStore().list（store 暴露的 ref）；
      组件内维护本地 ref 是为了让 @vue/test-utils 能直接断言 DOM 渲染。
    - 状态 tab key 由本组件常量 `STATUS_TABS` 定义，落到 store.loadList({ status })。

  测试覆盖：src/pages/order/index.test.js（5 个 it）
-->
<template>
  <view class="page-order-list">
    <!-- 顶部导航 -->
    <u-navbar title="我的订单" :auto-back="true" />

    <!-- 状态筛选 tabs -->
    <view class="page-order-list__tabs" data-test="status-tabs">
      <u-tabs
        :list="statusTabs"
        :current="currentTabIndex"
        line-color="#1677ff"
        :active-style="{ color: '#1677ff', fontWeight: 600 }"
        inactive-color="#606266"
        @click="onTabChange"
      />
    </view>

    <!-- loading 占位（首次加载 / 切 tab 重载） -->
    <view
      v-if="loading && orders.length === 0"
      class="page-order-list__skeleton"
      data-test="skeleton"
    >
      <u-skeleton
        :rows="2"
        :title="true"
        avatar
        avatar-shape="square"
        avatar-size="48"
      />
      <u-skeleton
        :rows="2"
        :title="true"
        avatar
        avatar-shape="square"
        avatar-size="48"
      />
    </view>

    <!-- 错误态 -->
    <view
      v-else-if="loadError"
      class="page-order-list__error"
      data-test="error-state"
    >
      <u-empty text="加载失败" mode="data" />
      <u-button
        type="primary"
        plain
        data-test="retry-btn"
        @click="onRetry"
      >重试</u-button>
    </view>

    <!-- 空状态 -->
    <view
      v-else-if="orders.length === 0"
      class="page-order-list__empty"
      data-test="empty-state"
    >
      <u-empty text="暂无订单" mode="list" />
    </view>

    <!-- 列表 -->
    <view v-else class="page-order-list__list">
      <OrderListItem
        v-for="order in orders"
        :key="order.id"
        :order="order"
        :status-actions="statusActions"
        @view="onView"
        @select="onSelect"
        @cancel="onCancel"
      />
    </view>
  </view>
</template>

<script>
// 订单列表页 —— Vue 3 Options API（与项目既有页面风格统一 —— detail / candidates）。
//
// 设计要点：
//   - 三态机：loading / error / loaded + empty / loaded + list
//     loading=true 且 orders 为空 → u-skeleton 占位
//     error：u-empty + 重试按钮 + 失败 toast
//     orders.length === 0 → u-empty「暂无订单」
//     orders.length > 0 → 渲染 OrderListItem 列表
//   - tab 切换：直接重调 loadList；_loaded 标志 + 切换后强制刷新
//   - onLoad → mounted：onLoad 先到（uni-app Page 钩子），orderId / status 已写入 →
//     mounted 立即拉取（与 detail / candidates 同款两段式生命周期）
//
// 数据来源：
//   - 列表镜像自 useOrderStore().list（store 暴露的 ref）；
//     组件内维护本地 ref 是为了让 @vue/test-utils 能直接断言 DOM 渲染。

import { useOrderStore } from '@/stores/order.js';
import OrderListItem from '@/components/OrderListItem.vue';

// ---- 状态 tab 定义：label / 状态过滤 key（与 backend 状态机 1:1）
// key 为空表示「全部」—— 不传 status
const STATUS_TABS = [
  { key: '',                            label: '全部' },
  { key: 'paid',                        label: '已支付' },
  { key: 'selectingEscort',             label: '待选陪诊师' },
  { key: 'escortPendingAcceptance',     label: '待陪诊师确认' },
  { key: 'accepted',                    label: '已接单' },
  { key: 'inService',                   label: '服务中' },
  { key: 'completed',                   label: '已完成' },
  { key: 'cancelled',                   label: '已取消' },
];

// ---- 默认操作矩阵：每个状态应展示的按钮 key 列表
// view: 跳详情；select: 跳 candidates 选陪诊师；cancel: 弹 confirm 后取消
// （订单处于 escortPendingAcceptance / inService / completed / cancelled 时不应再取消）
const DEFAULT_STATUS_ACTIONS = {
  paid: ['view', 'select'],
  selectingEscort: ['view', 'select'],
  escortPendingAcceptance: ['view'],
  // 后端用 'escortConfirmed' 作为已接单标识，但 OrderStatusProgress 显示为 'accepted'。
  // status filter 用 'accepted'，但允许 'escortConfirmed' 走同一组动作。
  accepted: ['view', 'cancel'],
  escortConfirmed: ['view', 'cancel'],
  inService: ['view'],
  completed: ['view'],
  cancelled: ['view'],
  canceled: ['view'],
};

export default {
  name: 'OrderListPage',
  components: { OrderListItem },
  data() {
    return {
      statusTabs: STATUS_TABS,
      currentTabIndex: 0,
      currentStatus: '',
      orders: [],
      loading: false,
      loadError: false,
      statusActions: DEFAULT_STATUS_ACTIONS,
      // 单次加载去重：onLoad 与 mounted 都可能触发初次加载，避免重复
      _loaded: false,
    };
  },
  computed: {
    orderStore() {
      return useOrderStore();
    },
  },
  methods: {
    /**
     * 拉订单列表（fetchList 是页面实际拉数据的入口）。
     * onLoad + mounted + onTabChange + onRetry + 取消成功后刷新 共用。
     * @param {string} [status] 状态过滤 key（空字符串 = 全部）
     */
    async fetchList(status) {
      this.loading = true;
      this.loadError = false;
      try {
        const query = status ? { status } : {};
        const items = await this.orderStore.loadList(query);
        this.orders = Array.isArray(items) ? items : [];
        this._loaded = true;
      } catch (err) {
        // eslint-disable-next-line no-console
        console.error('[order/list] loadList failed', err);
        this.loadError = true;
        this.orders = [];
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    /** u-tabs click → 切 tab + 重拉（reset _loaded 强制刷新） */
    onTabChange(item) {
      const idx = (item && typeof item.index === 'number') ? item.index : 0;
      const tab = this.statusTabs[idx] || this.statusTabs[0];
      this.currentTabIndex = idx;
      this.currentStatus = (tab && tab.key) || '';
      this._loaded = false;
      this.fetchList(this.currentStatus);
    },

    /** 重试按钮 → 重新 fetch（保持当前 tab） */
    onRetry() {
      this.fetchList(this.currentStatus);
    },

    /** 跳详情页（view 事件 / 卡片中部 click） */
    onView(orderId) {
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({
        url: `/pages/order/detail?orderId=${orderId}`,
      });
    },

    /** 跳 candidates 选陪诊师（select 事件 / 「选陪诊师」按钮） */
    onSelect(orderId) {
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({
        url: `/pages/order/candidates/index?orderId=${orderId}`,
      });
    },

    /**
     * 取消订单（cancel 事件）—— 弹 confirm → 调 store.cancel → 刷新列表。
     * @param {number|string} orderId
     */
    onCancel(orderId) {
      if (typeof uni === 'undefined' || typeof uni.showModal !== 'function') {
        // 测试 / SSR 兜底：直接执行取消
        this.doCancel(orderId);
        return;
      }
      uni.showModal({
        title: '取消订单',
        content: '确定要取消该订单吗？取消后不可恢复',
        success: async (res) => {
          if (!res.confirm) return;
          await this.doCancel(orderId);
        },
      });
    },

    /**
     * 实际执行取消：调 store.cancel → 成功 toast + reload list。
     * @param {number|string} orderId
     */
    async doCancel(orderId) {
      try {
        await this.orderStore.cancel(orderId, '用户主动取消');
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '订单已取消', icon: 'success' });
        }
        // 取消成功后刷新当前 tab 列表
        await this.fetchList(this.currentStatus);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error('[order/list] cancel failed', e);
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '取消失败，请重试', icon: 'none' });
        }
      }
    },
  },
  // ---- uni-app / Vue 生命周期
  // onLoad：uni-app Page 钩子，从 URL query 读 status（admin-web 跳转 / DashboardPage 传入）
  // mounted：Vue 标准 Options API 钩子，组件挂载完成时触发 → 立即拉一次
  onLoad(query) {
    const status = query && query.status;
    if (status) {
      const idx = this.statusTabs.findIndex((t) => t.key === status);
      if (idx >= 0) {
        this.currentTabIndex = idx;
        this.currentStatus = status;
      }
    }
  },
  mounted() {
    // 页面挂载完成 → 立即拉一次（brief「mounted 调 loadList」语义）
    if (!this._loaded) {
      this.fetchList(this.currentStatus);
    }
  },
};
</script>

<style lang="scss" scoped>
.page-order-list {
  min-height: 100vh;
  background: #f5f7fa;
}

.page-order-list__tabs {
  background: #ffffff;
  border-bottom: 1px solid #f0f0f0;
}

.page-order-list__skeleton {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-order-list__error {
  margin-top: 64px;
  display: flex;
  flex-direction: column;
  align-items: center;

  .u-button {
    margin-top: 16px;
    width: 50%;
  }
}

.page-order-list__empty {
  padding-top: 64px;
}

.page-order-list__list {
  padding: 12px 16px;
}
</style>