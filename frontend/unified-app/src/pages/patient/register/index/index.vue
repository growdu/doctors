<script setup lang="ts">
/**
 * patient/register/index/index.vue — alias path → auth/register。
 *
 * v1 嵌套路径；unified-app 保留路径兼容 v1 路由表 +，内 reLaunch 跳到主路由。
 */
import { onLoad } from '@dcloudio/uni-app';

onLoad(() => {
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/patient/auth/register' });
  }
});
</script>

<template>
  <view />
</template>