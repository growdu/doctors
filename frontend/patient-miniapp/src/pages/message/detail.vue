<!--
  src/pages/message/detail.vue

  站内信详情 —— from / to / content / timestamp
  （spec §4.8 + plan Task M2）

  页面流向：
    入口：message/list 点击条目
       → 本页（?id=xxx）
       → mounted 调 store.loadDetail(id)
       → 渲染：分类 chip + 标题 + from（发送方） + to（接收方） + 时间 + 内容
       → 若有 link → 「查看详情」按钮（跳 link 指定的 url）

  模板要点：
    - 顶部 <u-navbar>「消息详情」+ 自动返回
    - 卡片：
        - 顶部 chip（category）+ 标题（大字）
        - 「发件人」字段（from）
        - 「收件人」字段（to = 当前用户）
        - 「时间」字段（完整 ISO8601 → yyyy-MM-dd HH:mm:ss）
        - 正文 content（多行文本）
        - 「查看详情」按钮（若有 link）

  行为：
    - onLoad(query)：从 query 读 id
    - mounted() → loadDetail(id)
    - 「查看详情」→ uni.navigateTo(link)

  数据来源：
    - useMessageStore().current / .loadingDetail / .error

  测试覆盖：src/pages/message/detail.test.js
-->
<template>
  <view class="page-message-detail" data-test="message-detail-page">
    <u-navbar title="消息详情" :auto-back="true" />

    <!-- loading -->
    <view v-if="loading" class="page-message-detail__loading" data-test="loading">
      <u-skeleton :rows="5" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-message-detail__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- 详情卡片 -->
    <view v-else-if="msg" class="page-message-detail__card" data-test="detail-card">
      <view class="page-message-detail__chip" :data-test="'category-chip'">
        <text class="page-message-detail__chip-text">{{ msg.category || 'system' }}</text>
      </view>

      <text class="page-message-detail__title" data-test="title">{{ msg.title }}</text>

      <view class="page-message-detail__field" data-test="from-field">
        <text class="page-message-detail__field-label">发件人</text>
        <text class="page-message-detail__field-value" data-test="from-value">{{ msg.from || '系统通知' }}</text>
      </view>

      <view class="page-message-detail__field" data-test="to-field">
        <text class="page-message-detail__field-label">收件人</text>
        <text class="page-message-detail__field-value" data-test="to-value">{{ msg.to || '我' }}</text>
      </view>

      <view class="page-message-detail__field" data-test="time-field">
        <text class="page-message-detail__field-label">时间</text>
        <text class="page-message-detail__field-value" data-test="time-value">{{ formatTime(msg.created_at) }}</text>
      </view>

      <view class="page-message-detail__content" data-test="content">
        <text class="page-message-detail__content-text">{{ msg.content }}</text>
      </view>

      <view v-if="msg.link" class="page-message-detail__action">
        <u-button
          type="primary"
          size="large"
          plain
          data-test="link-btn"
          @click="onOpenLink"
        >查看详情</u-button>
      </view>
    </view>

    <view v-else class="page-message-detail__missing" data-test="missing">
      <u-empty text="消息不存在" mode="data" />
    </view>
  </view>
</template>

<script>
// message/detail 页 —— Vue 3 Options API。
//
// 关键设计：
//   - onLoad/mounted 放在 methods 内（uni-app 支持 + 测试可通过 vm 调用）
//   - store.loadDetail 内已含 markRead 副作用；本页面无需再调 markRead

import { useMessageStore } from '@/stores/message.js';

export default {
  name: 'MessageDetailPage',
  data() {
    return {
      id: null,
      loadError: false,
    };
  },
  computed: {
    messageStore() {
      return useMessageStore();
    },
    msg() {
      return this.messageStore.current;
    },
    loading() {
      return this.messageStore.loadingDetail;
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     */
    onLoad(query) {
      this.id = (query && (query.id || query.messageId)) || null;
    },

    /**
     * Vue mounted 钩子（async + methods 内）。
     */
    async mounted() {
      if (this.id) {
        await this.fetchDetail();
      }
    },

    /**
     * 拉详情（含 markRead 副作用）
     */
    async fetchDetail() {
      this.loadError = false;
      try {
        await this.messageStore.loadDetail(this.id);
        if (!this.messageStore.current) {
          this.loadError = true;
        }
      } catch (_e) {
        this.loadError = true;
      }
    },

    /**
     * 「重试」按钮
     */
    async onRetry() {
      await this.fetchDetail();
    },

    /**
     * 「查看详情」点击（msg.link 跳转）
     */
    onOpenLink() {
      if (!this.msg || !this.msg.link) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: this.msg.link });
    },

    /**
     * 时间格式化（ISO8601 → 「yyyy-MM-dd HH:mm:ss」）
     */
    formatTime(iso) {
      if (!iso) return '';
      const t = new Date(iso);
      if (Number.isNaN(t.getTime())) return '';
      const pad = (n) => String(n).padStart(2, '0');
      return `${t.getFullYear()}-${pad(t.getMonth() + 1)}-${pad(t.getDate())} ${pad(t.getHours())}:${pad(t.getMinutes())}:${pad(t.getSeconds())}`;
    },
  },
};
</script>

<style lang="scss" scoped>
.page-message-detail {
  min-height: 100vh;
  padding: 16px;
  background: #f5f5f5;
  box-sizing: border-box;
}

.page-message-detail__card {
  background: #ffffff;
  padding: 24px 16px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-message-detail__chip {
  display: inline-block;
  align-self: flex-start;
  background: #e6f4ff;
  padding: 2px 8px;
  border-radius: 4px;
}

.page-message-detail__chip-text {
  font-size: 12px;
  color: #1677ff;
}

.page-message-detail__title {
  font-size: 20px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-message-detail__field {
  display: flex;
  align-items: center;
  gap: 8px;
  border-top: 1px solid #f0f0f0;
  padding-top: 8px;
}

.page-message-detail__field-label {
  width: 60px;
  font-size: 13px;
  color: #8c8c8c;
}

.page-message-detail__field-value {
  flex: 1;
  font-size: 14px;
  color: #1f1f1f;
}

.page-message-detail__content {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
}

.page-message-detail__content-text {
  font-size: 15px;
  line-height: 1.7;
  color: #1f1f1f;
  white-space: pre-wrap;
}

.page-message-detail__action {
  margin-top: 12px;
}
</style>