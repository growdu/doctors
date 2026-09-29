<script setup lang="ts">
/**
 * patient/login/index/index.vue — alias path → auth/login。
 *
 * v1 patient-miniapp 用此嵌套路径（深度 3），），），与 auth/login 功能完全相同。
 * unified-app 重构后保留此路径以兼容 v1 路由表，），），页面内 reLaunch 跳到主路由。
 */
import { onLoad } from '@dcloudio/uni-app';

onLoad(() => {
  // 重定向到统一登录页（patient 域）
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/patient/auth/login' });
  }
});
</script>

<template>
  <view />
</template>