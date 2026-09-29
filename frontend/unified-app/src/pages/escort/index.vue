<script setup lang="ts">
/**
 * escort/index.vue — 陪诊师域工作台首页（v2 unified-app · escort 域）。
 *
 * 入口：escort 域首页（HomeShell 切换角色后）
 *   → 4 入口卡片：抢单池 / 我的任务 / 个人中心 / 设置
 *
 * 设计要点：
 *   - 极简聚合入口，不重复渲染列表（列表在各 page 单独实现）
 *   - RoleGuard 限制 escort 角色
 */
import { onMounted } from 'vue';
import RoleGuard from '@/components/shared/RoleGuard.vue';
import UiCard from '@/components/shared/UiCard.vue';
import { useAuthStore } from '@/store/auth';

const auth = useAuthStore();

interface Entry {
  key: string;
  icon: string;
  title: string;
  desc: string;
  path: string;
}

const ENTRIES: Entry[] = [
  {
    key: 'invitations',
    icon: '🎯',
    title: '抢单池',
    desc: '查看可接候选订单',
    path: '/pages/escort/invitations/index',
  },
  {
    key: 'orders',
    icon: '📋',
    title: '我的任务',
    desc: '查看已接订单',
    path: '/pages/escort/orders/index',
  },
  {
    key: 'profile',
    icon: '👤',
    title: '个人中心',
    desc: '资料 + 在线时段',
    path: '/pages/escort/profile/index',
  },
];

function onNavigate(path: string) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: path });
}

onMounted(() => {
  // 占位：聚合入口页不发起 API 调用
});
</script>

<template>
  <RoleGuard :required="['escort']" fallback-title="需要 escort 角色">
    <view class="ui-page" data-testid="escort-home">
      <view class="escort-home__header">
        <text class="escort-home__title">🚑 陪诊师工作台</text>
        <text class="escort-home__subtitle">当前角色：{{ auth.activeRole }}</text>
      </view>

      <view class="escort-home__list" data-testid="escort-home-entries">
        <UiCard
          v-for="e in ENTRIES"
          :key="e.key"
          :data-testid="`escort-home-entry-${e.key}`"
          @click="() => onNavigate(e.path)"
        >
          <view class="escort-home__entry">
            <text class="escort-home__entry-icon">{{ e.icon }}</text>
            <view class="escort-home__entry-info">
              <text class="escort-home__entry-title">{{ e.title }}</text>
              <text class="escort-home__entry-desc">{{ e.desc }}</text>
            </view>
            <text class="escort-home__entry-arrow">→</text>
          </view>
        </UiCard>
      </view>
    </view>
  </RoleGuard>
</template>

<style scoped>
.escort-home__header {
  padding: var(--ui-space-base) 0;
}

.escort-home__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-home__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-home__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-md);
}

.escort-home__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.escort-home__entry-icon {
  font-size: 32px;
}

.escort-home__entry-info {
  flex: 1;
}

.escort-home__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-home__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-home__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}
</style>