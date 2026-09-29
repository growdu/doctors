<script setup lang="ts">
/**
 * patient/orders/index/index.vue — alias path → order/list。
 *
 * v1 嵌套路径（order/list 顶层 + orders/index/index alias）；
 * unified-app 保留路径 + 内 reLaunch 跳到主路由。
 */
import { onLoad } from '@dcloudio/uni-app';

onLoad(() => {
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/patient/order/list' });
  }
});
</script>

<template>
  <view />
</template>