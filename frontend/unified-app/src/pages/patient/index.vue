<script setup lang="ts">
/**
 * patient/index.vue — alias path → index/index (tabBar 主页)。
 *
 * v1 patient-miniapp 用此嵌套路径（深度 2），unified-app 重构后保留此路径以兼容 v1 路由表，
 * 页面内 reLaunch 跳到主路由（tabBar 真首页）。
 */
import { onLoad } from '@dcloudio/uni-app';

onLoad(() => {
  // 重定向到 patient 域 tabBar 首页（含推荐医院 + 套餐瀑布流 + 我的进行中订单 widget）
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/patient/index/index' });
  }
});
</script>

<template>
  <view />
</template>