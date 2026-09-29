<script setup lang="ts">
/**
 * escort/splash.vue — escort 域启动页（v2 unified-app · escort 域）。
 *
 * 入口：escort 域第一入口（路由 guards）
 *   → onLoad 时 bootstrap authStore（恢复 token + 用户）
 *   → isAuthed: 跳 /pages/escort/invitations
 *   → else: 跳 /pages/escort/login
 *
 * 设计要点：
 *   - 极简版：单 logo + 文案 + 进度条
 *   - 不做额外 API 调用（authStore.bootstrap 已存在）
 *   - 跳转后 Splash 自身卸载
 */
import { onMounted } from 'vue';
import { useAuthStore } from '@/store/auth';

const auth = useAuthStore();

function navigateTo(path: string) {
  if (typeof uni !== 'undefined') uni.reLaunch({ url: path });
}

onMounted(async () => {
  auth.bootstrap();
  // 模拟 brief 加载延迟（让 splash 可见）
  setTimeout(() => {
    if (auth.isAuthed) {
      navigateTo('/pages/escort/invitations/index');
    } else {
      navigateTo('/pages/escort/login');
    }
  }, 500);
});
</script>

<template>
  <view class="ui-page" data-testid="escort-splash-page">
    <view class="escort-splash__content">
      <text class="escort-splash__logo">🚑</text>
      <text class="escort-splash__title">escort-app v2</text>
      <text class="escort-splash__subtitle">陪诊师工作台</text>
      <text class="escort-splash__loading">加载中...</text>
    </view>
  </view>
</template>

<style scoped>
.escort-splash__content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
}

.escort-splash__logo {
  font-size: 96px;
  margin-bottom: var(--ui-space-md);
}

.escort-splash__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-splash__subtitle {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-splash__loading {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: var(--ui-space-base);
}
</style>