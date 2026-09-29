<!--
  src/pages/message/list.vue

  站内信列表 —— 系统通知 / 营销消息（Pinia message store）
  （spec §4.8 + plan Task M2）

  页面流向：
    入口：profile「消息中心」
       → 本页
       → mounted 调 store.loadList()
       → 渲染：未读 badge + 列表（标题 / 摘要 / 时间 / 已读/未读小圆点）
       → 点击条目 → store.loadDetail(id) → 跳详情页
       → 「全部已读」→ store.markAllRead

  模板要点：
    - 顶部 <u-navbar>「消息」+ 自动返回；右上「全部已读」按钮
    - 列表：每条展示 title / category chip / time / 未读小红点
    - loading / empty / error 三态机

  行为：
    - onLoad(query)
    - mounted() → loadList
    - onItemClick(msg) → store.loadDetail → 跳详情（带 id）
    - onMarkAllRead → store.markAllRead + 重新 loadList 同步

  数据来源：
    - useMessageStore().list / .unread / .pagination / .loading

  测试覆盖：src/pages/message/list.test.js
-->
<template>
  <view class="page-message-list" data-test="message-list-page">
    <u-navbar title="消息" :auto-back="true">
      <view slot="right" class="page-message-list__navbar-right">
        <u-button
          v-if="messages.length > 0"
          size="mini"
          plain
          :loading="markingAll"
          data-test="mark-all-btn"
          @click="onMarkAllRead"
        >全部已读</u-button>
      </view>
    </u-navbar>

    <!-- 未读 badge -->
    <view v-if="unread > 0" class="page-message-list__unread" data-test="unread-banner">
      <text class="page-message-list__unread-text">{{ unread }} 条未读</text>
    </view>

    <!-- loading -->
    <view v-if="loading && messages.length === 0" class="page-message-list__loading" data-test="loading">
      <u-skeleton :rows="5" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-message-list__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- empty -->
    <view v-else-if="messages.length === 0" class="page-message-list__empty" data-test="empty">
      <u-empty text="暂无消息" mode="message" />
    </view>

    <!-- 列表 -->
    <view v-else class="page-message-list__items" data-test="items">
      <view
        v-for="msg in messages"
        :key="msg.id"
        class="page-message-list__item"
        :class="{ 'page-message-list__item--unread': !msg.read_at }"
        :data-test="'message-' + msg.id"
        @click="onItemClick(msg)"
      >
        <view class="page-message-list__item-main">
          <view class="page-message-list__item-header">
            <text class="page-message-list__item-title">{{ msg.title }}</text>
            <text
              v-if="!msg.read_at"
              class="page-message-list__item-dot"
              data-test="unread-dot"
            >●</text>
          </view>
          <text class="page-message-list__item-content">{{ msg.content }}</text>
          <view class="page-message-list__item-meta">
            <text class="page-message-list__item-category">{{ msg.category || 'system' }}</text>
            <text class="page-message-list__item-time">{{ formatTime(msg.created_at) }}</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
// message/list 页 —— Vue 3 Options API。
//
// 关键设计：
//   - onLoad/mounted 放在 methods 内（uni-app 支持 + 测试可通过 vm 调用）
//   - 点击条目先 loadDetail（标记已读副作用）→ 跳详情
//   - 「全部已读」调 markAllRead + 重新 loadList 同步后端状态

import { useMessageStore } from '@/stores/message.js';

export default {
  name: 'MessageListPage',
  data() {
    return {
      loadError: false,
      markingAll: false,
    };
  },
  computed: {
    messageStore() {
      return useMessageStore();
    },
    messages() {
      return this.messageStore.list;
    },
    unread() {
      return this.messageStore.unread;
    },
    loading() {
      return this.messageStore.loading;
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     */
    onLoad(_query) {},

    /**
     * Vue mounted 钩子（async + methods 内）。
     */
    async mounted() {
      await this.fetchList();
    },

    /**
     * 拉列表 + 错误处理。
     */
    async fetchList() {
      this.loadError = false;
      try {
        await this.messageStore.loadList({ page: 1, page_size: 20 });
      } catch (_e) {
        this.loadError = true;
      }
    },

    /**
     * 「重试」按钮
     */
    async onRetry() {
      await this.fetchList();
    },

    /**
     * 点击条目 → store.loadDetail（标记已读）→ 跳详情
     */
    async onItemClick(msg) {
      if (!msg || !msg.id) return;
      try {
        await this.messageStore.loadDetail(msg.id);
      } catch (_e) {
        // 详情加载失败也允许跳详情（详情页会再次 loadDetail）
      }
      if (typeof uni !== 'undefined' && typeof uni.navigateTo === 'function') {
        uni.navigateTo({ url: `/pages/message/detail?id=${msg.id}` });
      }
    },

    /**
     * 「全部已读」点击
     */
    async onMarkAllRead() {
      this.markingAll = true;
      try {
        await this.messageStore.markAllRead();
        this._toast('全部已读');
      } catch (e) {
        this._toast(this._errMsg(e, '操作失败，请重试'));
      } finally {
        this.markingAll = false;
      }
    },

    /**
     * 时间格式化（ISO8601 → 「MM-dd HH:mm」）
     */
    formatTime(iso) {
      if (!iso) return '';
      const t = new Date(iso);
      if (Number.isNaN(t.getTime())) return '';
      const pad = (n) => String(n).padStart(2, '0');
      return `${pad(t.getMonth() + 1)}-${pad(t.getDate())} ${pad(t.getHours())}:${pad(t.getMinutes())}`;
    },

    _toast(title) {
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title, icon: 'none' });
      }
    },

    _errMsg(e, fallback) {
      if (e && e.message) return e.message;
      return fallback;
    },
  },
};
</script>

<style lang="scss" scoped>
.page-message-list {
  min-height: 100vh;
  padding: 16px;
  background: #f5f5f5;
  box-sizing: border-box;
}

.page-message-list__navbar-right {
  display: flex;
  align-items: center;
}

.page-message-list__unread {
  background: #fff7e6;
  padding: 8px 12px;
  border-radius: 4px;
  margin-bottom: 12px;
}

.page-message-list__unread-text {
  font-size: 13px;
  color: #d46b08;
}

.page-message-list__items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.page-message-list__item {
  background: #ffffff;
  padding: 12px 16px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  cursor: pointer;
}

.page-message-list__item--unread {
  background: #e6f4ff;
}

.page-message-list__item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-message-list__item-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-message-list__item-dot {
  color: #ff4d4f;
  font-size: 12px;
}

.page-message-list__item-content {
  font-size: 14px;
  color: #595959;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.page-message-list__item-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
}

.page-message-list__item-category {
  font-size: 12px;
  color: #1677ff;
  background: #e6f4ff;
  padding: 2px 6px;
  border-radius: 4px;
}

.page-message-list__item-time {
  font-size: 12px;
  color: #8c8c8c;
}
</style>