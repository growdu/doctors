<template>
  <view class="page-home">
    <view class="page-home__hero">
      <text class="page-home__title">patient-miniapp</text>
      <text class="page-home__subtitle">陪诊患者端 · 骨架已就绪</text>
    </view>

    <view class="page-home__panel">
      <text class="page-home__panel-title">初始化清单（Task 1~3）</text>
      <view class="page-home__check" v-for="item in checklist" :key="item.label">
        <text class="page-home__check-mark">✓</text>
        <text class="page-home__check-text">{{ item.label }}</text>
      </view>
    </view>

    <view class="page-home__panel">
      <text class="page-home__panel-title">后续计划</text>
      <text class="page-home__hint">页面 / Pinia store / 业务 API 在后续 plan 落地（Task 4~13）。</text>
    </view>
  </view>
</template>

<script setup>
// Task 3 占位：仅展示骨架已就绪。
// 真正的首页（搜索医院 / 我的订单入口 / 客服）见 Task 13 / 后续 plan。
import { ref } from 'vue';

const checklist = ref([
  { label: 'uni-app + Vue 3 工程骨架' },
  { label: 'pages.json / manifest.json（含 app-plus 原生打包）' },
  { label: 'uView Plus + Pinia 已注册' },
  { label: '全局请求/响应拦截器（X-Trace-Id / Authorization / 11001）' },
]);
</script>

<style lang="scss" scoped>
.page-home {
  padding: 24px 16px;
}

.page-home__hero {
  padding: 32px 16px;
  background: linear-gradient(135deg, #1989fa 0%, #4eb7ff 100%);
  border-radius: 12px;
  margin-bottom: 24px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.page-home__title {
  color: #fff;
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 8px;
}

.page-home__subtitle {
  color: rgba(255, 255, 255, 0.85);
  font-size: 14px;
}

.page-home__panel {
  background: #fff;
  border-radius: 12px;
  padding: 20px 16px;
  margin-bottom: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-home__panel-title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #1989fa;
  margin-bottom: 16px;
}

.page-home__check {
  display: flex;
  align-items: center;
  padding: 8px 0;
}

.page-home__check-mark {
  color: #67c23a;
  font-weight: 700;
  margin-right: 8px;
}

.page-home__check-text {
  color: #333;
  font-size: 14px;
}

.page-home__hint {
  display: block;
  color: #909399;
  font-size: 13px;
  line-height: 20px;
}
</style>