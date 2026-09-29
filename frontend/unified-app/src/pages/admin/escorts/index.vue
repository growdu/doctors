<script setup lang="ts">
/**
 * admin/escorts/index.vue — 陪诊师管理首页（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「在线陪诊师」卡片
 *   → 4 入口：待审核队列 / 已通过 / 已拒绝 / 详情
 *
 * 设计要点：
 *   - 简化版：核心入口是待审核；其余入口占位（v2 后续接 list API）
 *   - 跳 pending-audit 是主流程
 */
import { onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';

interface Entry {
  key: string;
  icon: string;
  title: string;
  desc: string;
  path: string;
}

const ENTRIES: Entry[] = [
  {
    key: 'pending-audit',
    icon: '📋',
    title: '待审核队列',
    desc: '查看待审陪诊师并执行通过/拒绝',
    path: '/pages/admin/escorts/pending-audit',
  },
];

function onNavigate(path: string) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: path });
}

onMounted(() => {
  // 占位：escorts 主页是路由聚合入口，无需额外加载
});
</script>

<template>
  <view class="ui-page" data-testid="admin-escorts-index">
    <view class="admin-escorts-index__header">
      <text class="admin-escorts-index__title">陪诊师管理</text>
      <text class="admin-escorts-index__subtitle">审核 · 详情 · 列表</text>
    </view>

    <view class="admin-escorts-index__list" data-testid="admin-escorts-entries">
      <UiCard
        v-for="e in ENTRIES"
        :key="e.key"
        :data-testid="`admin-escorts-entry-${e.key}`"
        @click="onNavigate(e.path)"
      >
        <view class="admin-escorts-index__entry">
          <text class="admin-escorts-index__entry-icon">{{ e.icon }}</text>
          <view class="admin-escorts-index__entry-info">
            <text class="admin-escorts-index__entry-title">{{ e.title }}</text>
            <text class="admin-escorts-index__entry-desc">{{ e.desc }}</text>
          </view>
          <text class="admin-escorts-index__entry-arrow">→</text>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-escorts-index__header {
  padding: var(--ui-space-base) 0;
}

.admin-escorts-index__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-escorts-index__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-escorts-index__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.admin-escorts-index__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.admin-escorts-index__entry-icon {
  font-size: 32px;
}

.admin-escorts-index__entry-info {
  flex: 1;
}

.admin-escorts-index__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-escorts-index__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-escorts-index__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}
</style>