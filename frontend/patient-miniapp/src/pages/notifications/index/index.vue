<!--
  src/pages/notifications/index/index.vue

  通知中心 —— 按 type 分类（plan v1.1）
  （spec §4.8 + plan 增量）

  页面流向：
    入口：profile「通知中心」/ 设置页「推送」
       → 本页
       → 渲染 tab（按 type 分类：订单 / 系统 / 营销）
       → 每 tab 下：通知列表（标题 + 摘要 + 时间 + 未读红点）
       → 点击通知 → 跳详情或外链

  模板要点：
    - 顶部 <u-navbar>「通知中心」+ 自动返回
    - tab（全部 / 订单 / 系统 / 营销）
    - 列表：每条 title / type badge / time / 未读红点
    - loading / error / empty / loaded 四态机

  数据来源：
    - v1 mock：本地 NOTIFICATIONS 常量（按 type 分类）
    - 后续 plan 接 /notifications GET
-->
<template>
  <view class="page-notifications" data-test="notifications-page">
    <u-navbar title="通知中心" :auto-back="true" />

    <!-- tab：按 type 分类 -->
    <view class="page-notifications__tabs" data-test="tabs">
      <view
        v-for="t in TABS"
        :key="t.key"
        class="page-notifications__tab"
        :class="{ 'page-notifications__tab--active': currentTab === t.key }"
        :data-test="'tab-' + t.key"
        @click="onTabChange(t.key)"
      >
        {{ t.label }}
        <text
          v-if="unreadCountByType(t.key) > 0"
          class="page-notifications__tab-badge"
          :data-test="'badge-' + t.key"
        >{{ unreadCountByType(t.key) }}</text>
      </view>
    </view>

    <!-- loading -->
    <view v-if="loading" class="page-notifications__loading" data-test="loading">
      <u-skeleton :rows="3" :title="true" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-notifications__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- empty -->
    <view v-else-if="filteredItems.length === 0" class="page-notifications__empty" data-test="empty-state">
      <u-empty text="暂无通知" mode="message" />
    </view>

    <!-- 列表 -->
    <view v-else class="page-notifications__list" data-test="list">
      <view
        v-for="n in filteredItems"
        :key="n.id"
        class="page-notifications__item"
        :class="{ 'page-notifications__item--unread': !n.read_at }"
        :data-test="'notification-' + n.id"
        :data-notif-id="n.id"
        :data-unread="!n.read_at"
        @click="onItemClick(n)"
      >
        <view class="page-notifications__item-row">
          <text class="page-notifications__item-title">{{ n.title }}</text>
          <text
            v-if="!n.read_at"
            class="page-notifications__item-dot"
            data-test="unread-dot"
          >●</text>
        </view>
        <text class="page-notifications__item-summary">{{ n.summary }}</text>
        <view class="page-notifications__item-meta">
          <text class="page-notifications__item-type">{{ typeLabel(n.type) }}</text>
          <text class="page-notifications__item-time">{{ formatTime(n.created_at) }}</text>
        </view>
      </view>
    </view>

    <!-- 底部「全部已读」 -->
    <view v-if="hasUnread" class="page-notifications__bottom">
      <u-button
        type="primary"
        plain
        data-test="mark-all-btn"
        @click="onMarkAllRead"
      >全部已读</u-button>
    </view>
  </view>
</template>

<script>
// 通知中心页 —— Vue 3 Options API（与项目既有页面风格统一 —— message/list）。
//
// 关键设计：
//   - tab（全部 / 订单 / 系统 / 营销）切换不重拉（v1 本地数据）
//   - 未读 badge 显示在 tab 上 + 列表条目右侧红点
//   - 「全部已读」点击 → 把数据标记为 read（写入本地 state）

// v1 mock 数据（按 type 分类）
const NOTIFICATIONS = [
  {
    id: 1, type: 'order', title: '陪诊师已接单', summary: '订单 #7 已由陪诊师张三接单，请保持手机畅通。',
    read_at: null, created_at: '2026-09-26T10:30:00+08:00',
  },
  {
    id: 2, type: 'order', title: '订单已支付', summary: '订单 #6 支付成功，等待匹配陪诊师。',
    read_at: null, created_at: '2026-09-25T15:00:00+08:00',
  },
  {
    id: 3, type: 'system', title: '实名认证提醒', summary: '请尽快完成实名认证以解锁全部功能。',
    read_at: null, created_at: '2026-09-24T09:00:00+08:00',
  },
  {
    id: 4, type: 'system', title: '隐私政策更新', summary: '我们更新了隐私政策，请查阅最新版本。',
    read_at: '2026-09-23T10:00:00+08:00', created_at: '2026-09-23T09:00:00+08:00',
  },
  {
    id: 5, type: 'marketing', title: '国庆陪诊优惠', summary: '国庆期间陪诊服务满 300 减 50，领取优惠券。',
    read_at: null, created_at: '2026-09-22T08:00:00+08:00',
  },
];

const TABS = [
  { key: 'all',       label: '全部' },
  { key: 'order',     label: '订单' },
  { key: 'system',    label: '系统' },
  { key: 'marketing', label: '营销' },
];

const TYPE_LABEL = {
  order: '订单',
  system: '系统',
  marketing: '营销',
};

export default {
  name: 'NotificationsPage',
  data() {
    return {
      TABS,
      TYPE_LABEL,
      currentTab: 'all',
      notifications: NOTIFICATIONS.map((n) => ({ ...n })),
      loading: false,
      loadError: false,
      _loaded: false,
    };
  },
  computed: {
    /** 当前 tab 下的通知列表 */
    filteredItems() {
      if (this.currentTab === 'all') return this.notifications;
      return this.notifications.filter((n) => n.type === this.currentTab);
    },
    hasUnread() {
      return this.notifications.some((n) => !n.read_at);
    },
  },
  methods: {
    onTabChange(key) {
      if (!key) return;
      this.currentTab = key;
    },

    onRetry() {
      this._load();
    },

    /**
     * v1 mock：异步模拟一次 fetch，本地直接展开 NOTIFICATIONS。
     * 后续 plan 接 /notifications GET。
     */
    _load() {
      this.loading = true;
      this.loadError = false;
      try {
        // v1 mock：本地常量即可——但保留异步语义以贴合真实接入
        this.notifications = NOTIFICATIONS.map((n) => ({ ...n }));
        this._loaded = true;
      } catch (_e) {
        this.loadError = true;
      } finally {
        this.loading = false;
      }
    },

    /** 每个 type 的未读数 — 用于 tab badge */
    unreadCountByType(key) {
      if (key === 'all') return this.notifications.filter((n) => !n.read_at).length;
      return this.notifications.filter((n) => n.type === key && !n.read_at).length;
    },

    typeLabel(t) {
      return TYPE_LABEL[t] || t || '-';
    },

    formatTime(iso) {
      if (!iso) return '';
      const t = new Date(iso);
      if (Number.isNaN(t.getTime())) return '';
      const pad = (n) => String(n).padStart(2, '0');
      return `${pad(t.getMonth() + 1)}-${pad(t.getDate())} ${pad(t.getHours())}:${pad(t.getMinutes())}`;
    },

    /** 点击条目 → markRead 副作用 */
    onItemClick(n) {
      if (!n || n.read_at) return;
      const idx = this.notifications.findIndex((x) => x.id === n.id);
      if (idx >= 0) {
        this.notifications.splice(idx, 1, Object.assign({}, n, {
          read_at: new Date().toISOString(),
        }));
      }
    },

    /** 全部已读 */
    onMarkAllRead() {
      const now = new Date().toISOString();
      this.notifications = this.notifications.map((n) =>
        n.read_at ? n : Object.assign({}, n, { read_at: now }),
      );
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title: '全部已读', icon: 'success' });
      }
    },
  },
  onLoad(_query) {
    // 占位
  },
  mounted() {
    if (!this._loaded) {
      this._load();
    }
  },
};
</script>

<style lang="scss" scoped>
.page-notifications {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 88px;
}

.page-notifications__tabs {
  background: #fff;
  display: flex;
  border-bottom: 1px solid #f0f0f0;
}

.page-notifications__tab {
  flex: 1;
  text-align: center;
  padding: 14px 0;
  font-size: 14px;
  color: #606266;
  position: relative;
}

.page-notifications__tab--active {
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

.page-notifications__tab-badge {
  display: inline-block;
  margin-left: 4px;
  padding: 0 6px;
  background: #ff4d4f;
  color: #fff;
  border-radius: 8px;
  font-size: 11px;
  line-height: 16px;
  min-width: 16px;
  text-align: center;
  vertical-align: middle;
}

.page-notifications__loading,
.page-notifications__empty,
.page-notifications__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-notifications__error .u-button { margin-top: 16px; width: 50%; }

.page-notifications__list {
  padding: 12px 16px;
}

.page-notifications__item {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-notifications__item--unread {
  background: #e6f4ff;
}

.page-notifications__item-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.page-notifications__item-title {
  font-size: 15px;
  font-weight: 500;
  color: #303133;
  flex: 1;
}

.page-notifications__item-dot {
  font-size: 12px;
  color: #ff4d4f;
}

.page-notifications__item-summary {
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

.page-notifications__item-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  color: #909399;
}

.page-notifications__item-type {
  background: #f0f0f0;
  color: #606266;
  padding: 1px 6px;
  border-radius: 4px;
}

.page-notifications__bottom {
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
</style>
</content>
</invoke>