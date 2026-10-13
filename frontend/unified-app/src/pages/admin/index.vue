<script setup lang="ts">
/**
 * admin/index.vue — alias path → dashboard/index (admin 域真首页)。
 *
 * DomainSwitcher 跳转 `/pages/admin/index`，本页面内 reLaunch 跳到 dashboard 真首页（拉 6 项指标）。
 */
import { onLoad } from '@dcloudio/uni-app';

onLoad(() => {
  if (typeof uni !== 'undefined') {
    uni.reLaunch({ url: '/pages/admin/dashboard/index' });
  }
});
</script>

<template>
  <view />
</template>