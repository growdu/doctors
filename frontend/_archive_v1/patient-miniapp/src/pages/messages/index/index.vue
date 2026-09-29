<!--
  src/pages/messages/index/index.vue

  「站内信 — 按订单 tab」列表页 —— 患者端的订单相关消息（plan v1.1）
  （区别于 message/list.vue 系统通知）
  （spec §4.8）

  页面流向：
    入口：profile「消息中心」/ 订单详情「消息」
       → 本页
       → onLoad+mounted 调 useOrderStore().loadList() 拉订单列表 → 「按订单 tab」分组
       → 渲染 tab（按订单）+ 消息卡片：订单号 / 陪诊师 / 最新消息摘要 / 时间
       → 点击 → /pages/order/detail?orderId=xxx

  模板要点：
    - 顶部 <u-navbar>「订单消息」+ 自动返回
    - tab：按订单（v1 拉订单作为分组轴；每个订单 1 张消息卡片 + 最近一条摘要）
    - 空状态：u-empty「暂无订单消息」
    - loading / error / empty / loaded 四态机

  数据来源：
    - 订单列表镜像自 useOrderStore().list（store 暴露的 ref）
-->
<template>
  <view class="page-messages-index" data-test="messages-index-page">
    <u-navbar title="订单消息" :auto-back="true" />

    <!-- tab：全部分组（v1 仅一组，所以这里用伪 tab 占位 -->
    <view class="page-messages-index__tabs" data-test="tabs">
      <view
        v-for="t in TABS"
        :key="t.key"
        class="page-messages-index__tab"
        :class="{ 'page-messages-index__tab--active': currentTab === t.key }"
        :data-test="'tab-' + t.key"
        @click="onTabChange(t.key)"
      >{{ t.label }}</view>
    </view>

    <!-- loading -->
    <view v-if="loading && cards.length === 0" class="page-messages-index__loading" data-test="loading">
      <u-skeleton :rows="3" :title="true" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-messages-index__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- empty -->
    <view v-else-if="cards.length === 0" class="page-messages-index__empty" data-test="empty-state">
      <u-empty text="暂无订单消息" mode="message" />
    </view>

    <!-- list：每个订单 1 张消息卡片 -->
    <view v-else class="page-messages-index__list" data-test="messages-list">
      <view
        v-for="card in cards"
        :key="card.orderId"
        class="page-messages-index__card"
        :data-test="'msg-card-' + card.orderId"
        :data-order-id="card.orderId"
        @click="onCardClick(card)"
      >
        <view class="page-messages-index__card-row">
          <text class="page-messages-index__card-order">订单 #{{ card.orderId }}</text>
          <text
            v-if="card.unread > 0"
            class="page-messages-index__card-badge"
            :data-test="'unread-' + card.orderId"
          >{{ card.unread }} 条未读</text>
        </view>
        <text
          v-if="card.summary"
          class="page-messages-index__card-summary"
          data-test="msg-summary"
        >{{ card.summary }}</text>
        <view class="page-messages-index__card-meta">
          <text class="page-messages-index__card-time">{{ card.time }}</text>
          <text class="page-messages-index__card-status">{{ card.statusLabel }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
// 订单消息列表 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / home）。
//
// 设计要点：
//   - 状态机：loading / error / loaded + empty / loaded + list
//   - 数据源：v1 不接 /messages per order API；用 orderService.list() 拉订单作为分组轴。
//     每个订单 1 张消息卡片 + 「模拟最近一条摘要」（v1 mock）。
//   - onLoad+mounted 两段式生命周期

import { useOrderStore } from '@/stores/order.js';

// tab 定义（v1 仅「全部」一个分组；预留扩展）
const TABS = [
  { key: 'all',    label: '全部' },
  { key: 'unread', label: '未读' },
];

// 订单状态 → 友好文案
const STATUS_LABEL = {
  paid: '已支付',
  selectingEscort: '待选陪诊师',
  escortPendingAcceptance: '待陪诊师确认',
  accepted: '已接单',
  inService: '服务中',
  completed: '已完成',
  cancelled: '已取消',
};

export default {
  name: 'MessagesIndexPage',
  data() {
    return {
      TABS,
      currentTab: 'all',
      loading: false,
      loadError: false,
      _loaded: false,
    };
  },
  computed: {
    orderStore() {
      return useOrderStore();
    },
    /** 每个订单 1 张消息卡（已过滤掉不是 V1 范围内 status 的订单） */
    cards() {
      const list = Array.isArray(this.orderStore.list) ? this.orderStore.list : [];
      const mapped = list.map((o) => this._buildCard(o));
      if (this.currentTab === 'unread') {
        return mapped.filter((c) => c.unread > 0);
      }
      return mapped;
    },
    loading() {
      return Boolean(this.orderStore.loading);
    },
  },
  methods: {
    /**
     * 工具：从 order 派生 card：{ orderId / summary / unread / time / statusLabel }
     * v1 mock「summary」字段根据 status 推测（v1 不接 chat 摘要接口）
     */
    _buildCard(o) {
      const orderId = o && o.id;
      const status = (o && o.status) || '';
      const summary = (() => {
        if (!o) return '';
        switch (status) {
          case 'inService':          return '陪诊师已到达医院，请按时就诊';
          case 'accepted':           return '陪诊师已接单，请保持手机畅通';
          case 'escortPendingAcceptance': return '已通知陪诊师，等待确认';
          case 'selectingEscort':    return '请尽快选择陪诊师';
          case 'completed':          return '服务已完成，欢迎评价';
          default:                   return `订单状态：${STATUS_LABEL[status] || status || '未知'}`;
        }
      })();
      // v1 mock：未读条数 = 0 起步
      const unread = 0;
      const time = (() => {
        if (!o || !o.appointment_at) return '';
        const t = new Date(o.appointment_at);
        if (Number.isNaN(t.getTime())) return '';
        const pad = (n) => String(n).padStart(2, '0');
        return `${t.getMonth() + 1}-${pad(t.getDate())} ${pad(t.getHours())}:${pad(t.getMinutes())}`;
      })();
      return {
        orderId,
        status,
        statusLabel: STATUS_LABEL[status] || status || '-',
        summary,
        unread,
        time,
      };
    },

    async fetchList() {
      this.loading = true;
      this.loadError = false;
      try {
        // v1 简化：直接复用 order 列表，不传 status 过滤（取全部订单作为消息分组）
        await this.orderStore.loadList({});
      } catch (_e) {
        this.loadError = true;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
        this._loaded = true;
      }
    },

    onTabChange(key) {
      if (!key) return;
      this.currentTab = key;
    },

    onRetry() {
      this.fetchList();
    },

    onCardClick(card) {
      if (!card || !card.orderId) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: `/pages/order/detail?orderId=${card.orderId}` });
    },
  },
  onLoad(_query) {
    // 占位（admin-web 可能传 ?status=xxx 过滤；v1 仅展示全部）
  },
  mounted() {
    if (!this._loaded) {
      this.fetchList();
    }
  },
};
</script>

<style lang="scss" scoped>
.page-messages-index {
  min-height: 100vh;
  background: #f5f7fa;
}

.page-messages-index__tabs {
  background: #fff;
  display: flex;
  border-bottom: 1px solid #f0f0f0;
}

.page-messages-index__tab {
  flex: 1;
  text-align: center;
  padding: 14px 0;
  font-size: 14px;
  color: #606266;
  position: relative;
}

.page-messages-index__tab--active {
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

.page-messages-index__loading,
.page-messages-index__empty,
.page-messages-index__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-messages-index__error .u-button { margin-top: 16px; width: 50%; }

.page-messages-index__list {
  padding: 12px 16px;
}

.page-messages-index__card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-messages-index__card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.page-messages-index__card-order {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.page-messages-index__card-badge {
  font-size: 11px;
  background: #ff4d4f;
  color: #fff;
  padding: 2px 8px;
  border-radius: 10px;
}

.page-messages-index__card-summary {
  display: block;
  font-size: 13px;
  color: #606266;
  line-height: 20px;
  margin-bottom: 6px;
}

.page-messages-index__card-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #909399;
}
</style>
</content>
</invoke>