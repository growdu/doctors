<script setup lang="ts">
/**
 * patient/settings/index.vue — 设置（v2 unified-app）。
 *
 * 入口：profile「设置」
 *   → 纯本地设置：无后端 API
 *   → 通知开关 / 主题 / 语言 / 关于 / 退出登录
 *
 * 设计要点：
 *   - localStorage 持久化通知 / 主题偏好
 *   - 退出登录按钮复用 profile 同款 UiModal 确认
 */
import { ref, onMounted } from 'vue';
import { useAuthStore } from '@/store/auth';
import { storage } from '@/utils/storage';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiModal from '@/components/shared/UiModal.vue';

const auth = useAuthStore();

const notifyEnabled = ref(true);
const theme = ref<'light' | 'dark' | 'auto'>('light');
const language = ref<'zh-CN' | 'en-US'>('zh-CN');

const showLogoutModal = ref(false);

const STORAGE_KEY = 'unified.settings';

interface StoredSettings {
  notifyEnabled: boolean;
  theme: 'light' | 'dark' | 'auto';
  language: 'zh-CN' | 'en-US';
}

function loadSettings() {
  const s = storage.getJSON<StoredSettings>(STORAGE_KEY);
  if (s) {
    notifyEnabled.value = s.notifyEnabled;
    theme.value = s.theme;
    language.value = s.language;
  }
}

function saveSettings() {
  const s: StoredSettings = {
    notifyEnabled: notifyEnabled.value,
    theme: theme.value,
    language: language.value,
  };
  storage.setItem(STORAGE_KEY, s);
}

function onNotifyToggle(e: any) {
  notifyEnabled.value = !!e.detail.value;
  saveSettings();
}

function onThemeChange(e: any) {
  theme.value = e.detail.value;
  saveSettings();
}

function onLanguageChange(e: any) {
  language.value = e.detail.value;
  saveSettings();
}

function onAbout() {
  if (typeof uni !== 'undefined') {
    uni.showModal({
      title: '关于 Doctors',
      content: '陪诊师平台 v2.0 (unified-app)\n© 2026',
      showCancel: false,
    });
  }
}

function onLogoutAsk() {
  showLogoutModal.value = true;
}

function onLogoutConfirm() {
  showLogoutModal.value = false;
  auth.onUnauthorized();
}

function onLogoutCancel() {
  showLogoutModal.value = false;
}

onMounted(loadSettings);
</script>

<template>
  <view class="ui-page" data-testid="patient-settings">
    <!-- 通知设置 -->
    <UiCard title="通知" shadow="none" :no-padding="true">
      <view class="settings__row" data-testid="settings-row-notify">
        <text class="settings__row-label">推送通知</text>
        <switch :checked="notifyEnabled" @change="onNotifyToggle" data-testid="settings-notify-switch" />
      </view>
    </UiCard>

    <!-- 外观 -->
    <UiCard title="外观" shadow="none" :no-padding="true">
      <view class="settings__row" data-testid="settings-row-theme">
        <text class="settings__row-label">主题</text>
        <picker
          mode="selector"
          :range="['light', 'dark', 'auto']"
          :value="['light', 'dark', 'auto'].indexOf(theme)"
          @change="onThemeChange"
          data-testid="settings-theme-picker"
        >
          <view class="settings__picker-value">
            {{ theme === 'light' ? '浅色' : theme === 'dark' ? '深色' : '跟随系统' }} ›
          </view>
        </picker>
      </view>
      <view class="settings__row" data-testid="settings-row-language">
        <text class="settings__row-label">语言</text>
        <picker
          mode="selector"
          :range="['zh-CN', 'en-US']"
          :value="['zh-CN', 'en-US'].indexOf(language)"
          @change="onLanguageChange"
          data-testid="settings-language-picker"
        >
          <view class="settings__picker-value">
            {{ language === 'zh-CN' ? '简体中文' : 'English' }} ›
          </view>
        </picker>
      </view>
    </UiCard>

    <!-- 关于 -->
    <UiCard title="关于" shadow="none" :no-padding="true">
      <view class="settings__row settings__row--clickable" data-testid="settings-about" @click="onAbout">
        <text class="settings__row-label">关于 Doctors</text>
        <text class="settings__row-arrow">›</text>
      </view>
      <view class="settings__row">
        <text class="settings__row-label">版本</text>
        <text class="settings__row-value">v2.0.0-sp2</text>
      </view>
    </UiCard>

    <!-- 退出登录 -->
    <view class="settings__logout">
      <UiButton
        type="danger"
        block
        data-testid="settings-logout-btn"
        @click="onLogoutAsk"
      >
        退出登录
      </UiButton>
    </view>

    <UiModal
      :visible="showLogoutModal"
      title="退出登录"
      content="确定要退出当前账号吗？"
      @update:visible="(v: boolean) => !v && onLogoutCancel()"
    >
      <template #footer>
        <view style="display: flex; gap: 12px;">
          <UiButton type="default" block @click="onLogoutCancel" data-testid="settings-logout-cancel">取消</UiButton>
          <UiButton type="danger" block @click="onLogoutConfirm" data-testid="settings-logout-confirm">退出</UiButton>
        </view>
      </template>
    </UiModal>
  </view>
</template>

<style scoped>
.settings__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--ui-space-md) var(--ui-space-base);
  border-bottom: 1px solid var(--ui-color-divider);
}

.settings__row:last-child {
  border-bottom: none;
}

.settings__row--clickable {
  cursor: pointer;
}

.settings__row--clickable:active {
  background: var(--ui-color-bg-hover);
}

.settings__row-label {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  flex: 1;
}

.settings__row-value {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.settings__row-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}

.settings__picker-value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-primary);
  padding: 4px 8px;
}

.settings__logout {
  margin-top: var(--ui-space-lg);
}
</style>