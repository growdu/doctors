<script setup lang="ts">
/**
 * admin/settings/index.vue — 系统设置（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「设置」相关入口
 *   → v2 admin-service 未暴露系统设置端点
 *   → 本页为占位说明：仅展示环境信息 + 版本号
 *
 * 设计要点：
 *   - 不暴露运行时配置（避免越权）
 *   - 仅展示只读信息 + 维护入口（重新登录 / 清缓存）
 */
import { ref, onMounted } from 'vue';
import { useAuthStore } from '@/store/auth';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';

const auth = useAuthStore();

const version = ref('v2.0.0');
const envName = ref('dev');

function onRelaunch() {
  if (typeof uni !== 'undefined') uni.reLaunch({ url: '/pages/home/index' });
}

onMounted(() => {
  // 占位：仅读 user/token 存在性
});
</script>

<template>
  <view class="ui-page" data-testid="admin-settings-page">
    <UiCard title="系统信息" data-testid="admin-settings-system-card">
      <view class="admin-settings__row">
        <text class="admin-settings__label">版本</text>
        <text class="admin-settings__value" data-testid="admin-settings-version">{{ version }}</text>
      </view>
      <view class="admin-settings__row">
        <text class="admin-settings__label">环境</text>
        <text class="admin-settings__value" data-testid="admin-settings-env">{{ envName }}</text>
      </view>
      <view class="admin-settings__row">
        <text class="admin-settings__label">登录状态</text>
        <text
          class="admin-settings__value"
          :class="auth.isAuthed ? 'admin-settings__online' : 'admin-settings__offline'"
          :data-testid="auth.isAuthed ? 'admin-settings-online' : 'admin-settings-offline'"
        >
          {{ auth.isAuthed ? '已登录' : '未登录' }}
        </text>
      </view>
      <view class="admin-settings__row">
        <text class="admin-settings__label">当前角色</text>
        <text class="admin-settings__value">{{ auth.activeRole || '—' }}</text>
      </view>
    </UiCard>

    <view class="admin-settings__note" data-testid="admin-settings-note">
      <text class="admin-settings__note-text">
        💡 v2 admin-service 暂未暴露系统配置端点。本期仅展示只读环境信息。
      </text>
    </view>

    <view class="admin-settings__actions" data-testid="admin-settings-actions">
      <UiButton type="default" block data-testid="admin-settings-relaunch" @click="onRelaunch">
        返回首页
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.admin-settings__row {
  display: flex;
  align-items: center;
  margin-bottom: var(--ui-space-sm);
}

.admin-settings__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  width: 80px;
  flex-shrink: 0;
}

.admin-settings__value {
  flex: 1;
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
}

.admin-settings__online { color: var(--ui-color-success); }
.admin-settings__offline { color: var(--ui-color-text-disabled); }

.admin-settings__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-info);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin: var(--ui-space-md) 0;
}

.admin-settings__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.admin-settings__actions {
  margin-top: var(--ui-space-base);
}
</style>