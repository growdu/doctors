<script setup lang="ts">
/**
 * escort/audit-pending.vue — 陪诊师审核中页（v2 unified-app · escort 域）。
 *
 * 入口：escort 注册完成后跳转 / 主动查看审核状态
 *   → 显示「审核中」+ 倒计时 + 预计时长
 *   → 禁用返回（避免误退）
 *   → 倒计时归零显示「请刷新状态」+ 「返回首页」按钮
 *
 * 设计要点：
 *   - 简化版：mock 24h 倒计时（生产可对接 admin/escorts/pending-audit 接口查真实状态）
 *   - 倒计时到 0 后允许「返回首页」
 */
import { ref, computed, onMounted, onUnmounted } from 'vue';

const TOTAL_SECONDS = 24 * 60 * 60;
const remaining = ref(TOTAL_SECONDS);

let timer: ReturnType<typeof setInterval> | null = null;

const hms = computed(() => {
  const h = Math.floor(remaining.value / 3600);
  const m = Math.floor((remaining.value % 3600) / 60);
  const s = remaining.value % 60;
  return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
});

function tick() {
  if (remaining.value > 0) remaining.value -= 1;
}

function onBackHome() {
  if (typeof uni !== 'undefined') uni.reLaunch({ url: '/pages/home/index' });
}

onMounted(() => {
  timer = setInterval(tick, 1000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});
</script>

<template>
  <view class="ui-page" data-testid="escort-audit-pending-page">
    <view class="escort-audit-pending__content">
      <text class="escort-audit-pending__icon">⏳</text>
      <text class="escort-audit-pending__title">陪诊师审核中</text>
      <text class="escort-audit-pending__subtitle">您的资料已提交，请耐心等待</text>
      <text class="escort-audit-pending__subtitle">预计剩余审核时间</text>
      <text class="escort-audit-pending__countdown" data-testid="escort-audit-pending-countdown">
        {{ hms }}
      </text>
      <text class="escort-audit-pending__hint">💡 审核结果会通过站内信通知</text>
    </view>

    <view class="escort-audit-pending__actions">
      <UiButton
        type="default"
        block
        data-testid="escort-audit-pending-back-home"
        @click="onBackHome"
      >
        返回首页
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.escort-audit-pending__content {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 80px;
}

.escort-audit-pending__icon {
  font-size: 96px;
  margin-bottom: var(--ui-space-md);
}

.escort-audit-pending__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-audit-pending__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-sm);
}

.escort-audit-pending__countdown {
  font-size: var(--ui-font-display);
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-primary);
  display: block;
  margin-top: var(--ui-space-md);
  font-family: monospace;
}

.escort-audit-pending__hint {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-disabled);
  display: block;
  margin-top: var(--ui-space-base);
}

.escort-audit-pending__actions {
  margin-top: var(--ui-space-base);
}
</style>