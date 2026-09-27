<!--
  src/pages/settings/index/index.vue

  设置页 —— 4 类设置 tab：基础 / 支付 / 短信 / 推送 + 退出登录（plan v1.1）
  （spec §5.4）

  页面流向：
    入口：profile「设置」
       → 本页
       → 渲染 4 个 tab + 每 tab 下分组设置项
       → 修改本地设置：toggle 写入本地 state
       → 「退出登录」→ auth.logout() → 跳 /pages/login/index

  模板要点：
    - 顶部 <u-navbar>「设置」+ 自动返回
    - 4 个 tab（基本 / 支付 / 通知 / 推送）
    - 每 tab 下 list item 卡片（含 toggle / link / value）
    - 底部「退出登录」按钮（红色）

  数据来源：
    - 本地 component state（v1 不持久化）
-->
<template>
  <view class="page-settings" data-test="settings-page">
    <u-navbar title="设置" :auto-back="true" />

    <!-- 4 类 tab -->
    <view class="page-settings__tabs" data-test="tabs">
      <view
        v-for="t in TABS"
        :key="t.key"
        class="page-settings__tab"
        :class="{ 'page-settings__tab--active': currentTab === t.key }"
        :data-test="'tab-' + t.key"
        @click="onTabChange(t.key)"
      >{{ t.label }}</view>
    </view>

    <!-- 设置项分组 -->
    <view class="page-settings__group" data-test="settings-list">
      <view
        v-for="item in currentTabItems"
        :key="item.key"
        class="page-settings__item"
        :data-test="'item-' + item.key"
      >
        <text class="page-settings__item-label">{{ item.label }}</text>
        <view class="page-settings__item-control">
          <!-- toggle -->
          <view
            v-if="item.type === 'toggle'"
            class="page-settings__toggle"
            :class="{ 'page-settings__toggle--on': settings[item.key] }"
            :data-test="'toggle-' + item.key"
            :data-on="settings[item.key]"
            @click="onToggle(item.key)"
          >
            <view class="page-settings__toggle-knob"></view>
          </view>
          <!-- value（readonly） -->
          <text
            v-else-if="item.type === 'value'"
            class="page-settings__value"
          >{{ item.value }}</text>
          <!-- link -->
          <text
            v-else-if="item.type === 'link'"
            class="page-settings__link"
            :data-test="'link-' + item.key"
            @click="item.onClick && item.onClick()"
          >{{ item.label2 || '查看' }}</text>
        </view>
      </view>
    </view>

    <!-- 底部退出登录（仅在前 3 个 tab 显示；推送 tab 不显示） -->
    <view v-if="currentTab !== 'push'" class="page-settings__bottom" data-test="logout-row">
      <u-button
        type="error"
        plain
        data-test="logout-btn"
        @click="onLogout"
      >退出登录</u-button>
    </view>
  </view>
</template>

<script>
// 设置页 —— Vue 3 Options API（与项目既有页面风格统一 —— coupons/index tabs）。
//
// 关键设计：
//   - 4 类 tab 定义常量化：
//       basic（基础）/ pay（支付）/ sms（短信）/ push（推送）
//   - 每个 tab 下的设置项也常量化（item.type = toggle / value / link）
//   - 本地 settings 对象记录 toggle 状态；v1 不持久化（plan v2 走 uni.storage）
//   - 退出登录：调 auth.logout() → store 内置 reLaunch login 兜底

import { useAuthStore } from '@/stores/auth.js';

const TABS = [
  { key: 'basic', label: '基础' },
  { key: 'pay',   label: '支付' },
  { key: 'sms',   label: '短信' },
  { key: 'push',  label: '推送' },
];

// 每个 tab 下的设置项（type: toggle / value / link）
const ITEMS = {
  basic: [
    { key: 'language',   label: '语言',          type: 'value', value: '简体中文' },
    { key: 'timezone',   label: '时区',          type: 'value', value: 'GMT+8' },
    { key: 'darkMode',   label: '深色模式',      type: 'toggle' },
    { key: 'biometric',  label: '指纹/面容登录', type: 'toggle' },
  ],
  pay: [
    { key: 'defaultPayMethod', label: '默认支付方式', type: 'value', value: '微信支付' },
    { key: 'autoPay',          label: '自动续费',     type: 'toggle' },
    { key: 'invoiceEmail',     label: '发票邮箱',     type: 'link',  label2: '设置' },
  ],
  sms: [
    { key: 'orderUpdateSms', label: '订单状态短信', type: 'toggle' },
    { key: 'sosSms',         label: 'SOS 紧急短信', type: 'toggle' },
    { key: 'marketingSms',   label: '营销短信',     type: 'toggle' },
  ],
  push: [
    { key: 'orderPush',     label: '订单推送',     type: 'toggle' },
    { key: 'messagePush',   label: '消息推送',     type: 'toggle' },
    { key: 'systemPush',    label: '系统通知推送', type: 'toggle' },
    { key: 'quietHours',    label: '免打扰时段',   type: 'value', value: '23:00 - 08:00' },
  ],
};

export default {
  name: 'SettingsPage',
  data() {
    return {
      TABS,
      ITEMS,
      currentTab: 'basic',
      settings: {
        // v1 默认值
        darkMode: false,
        biometric: false,
        autoPay: false,
        orderUpdateSms: true,
        sosSms: true,
        marketingSms: false,
        orderPush: true,
        messagePush: true,
        systemPush: true,
      },
      loggingOut: false,
    };
  },
  computed: {
    authStore() {
      return useAuthStore();
    },
    currentTabItems() {
      return this.ITEMS[this.currentTab] || [];
    },
  },
  methods: {
    onTabChange(key) {
      if (!key) return;
      this.currentTab = key;
    },

    onToggle(key) {
      // 部分 toggle 显式控制关闭/开启
      this.settings = Object.assign({}, this.settings, {
        [key]: !this.settings[key],
      });
    },

    /**
     * 退出登录：调 store.logout()（store 内部已 reLaunch 到 /pages/auth/login；
     * 这里再保险地 reLaunch 到 v2 login 页 /pages/login/index）。
     */
    async onLogout() {
      if (this.loggingOut) return;
      this.loggingOut = true;
      try {
        await this.authStore.logout();
      } catch (_e) {
        // 静默：仍走 reLaunch
      } finally {
        this.loggingOut = false;
        if (typeof uni !== 'undefined' && typeof uni.reLaunch === 'function') {
          uni.reLaunch({ url: '/pages/login/index' });
        }
      }
    },
  },
  onLoad(query) {
    if (query && query.tab && this.TABS.find((t) => t.key === query.tab)) {
      this.currentTab = query.tab;
    }
  },
  mounted() {
    // 占位
  },
};
</script>

<style lang="scss" scoped>
.page-settings {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 96px;
}

.page-settings__tabs {
  background: #fff;
  display: flex;
  border-bottom: 1px solid #f0f0f0;
}

.page-settings__tab {
  flex: 1;
  text-align: center;
  padding: 14px 0;
  font-size: 14px;
  color: #606266;
  position: relative;
}

.page-settings__tab--active {
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

.page-settings__group {
  margin: 12px;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
}

.page-settings__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid #f5f5f5;
}

.page-settings__item:last-child {
  border-bottom: none;
}

.page-settings__item-label {
  font-size: 14px;
  color: #303133;
}

.page-settings__item-control {
  display: flex;
  align-items: center;
}

.page-settings__toggle {
  width: 44px;
  height: 24px;
  background: #e0e0e0;
  border-radius: 12px;
  position: relative;
  transition: background-color 0.2s;
}

.page-settings__toggle--on {
  background: #1989fa;
}

.page-settings__toggle-knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  background: #fff;
  border-radius: 50%;
  transition: transform 0.2s;
}

.page-settings__toggle--on .page-settings__toggle-knob {
  transform: translateX(20px);
}

.page-settings__value,
.page-settings__link {
  font-size: 13px;
  color: #909399;
}

.page-settings__link {
  color: #1989fa;
}

.page-settings__bottom {
  padding: 24px 16px;
}
</style>
</content>
</invoke>