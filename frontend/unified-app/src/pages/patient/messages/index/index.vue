<script setup lang="ts">
/**
 * patient/messages/index/index.vue — alias path → messages/index。
 *
 * v1 嵌套路径（message/list 顶层 + messages/index/index 聚合）；），），unified-app 保留路径，
 * 内 reLaunch 跳到主聚合路由。
 */
import { onLoad } from '@dcloudio/uni-app';

onLoad(() => {
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/patient/messages/index' });
  }
});
</script>

<template>
  <view />
</template>