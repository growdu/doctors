<script setup lang="ts">
/**
 * admin/audit/index.vue — 审计管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「审计」相关入口
 *   → 简化版：聚合入口，跳到已有审计流（陪诊师审核 + 退款审批 + 工单）
 *
 * 设计要点：
 *   - v2 admin-service 未暴露全局 audit 端点；
 *   - 本页作为入口聚合，跳到 escorts/pending-audit / refunds/list / work-orders 等具体页
 *   - 不再做额外 API 调用（避免空跑）
 */
import { onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';

interface AuditEntry {
  key: string;
  icon: string;
  title: string;
  desc: string;
  path: string;
}

const ENTRIES: AuditEntry[] = [
  {
    key: 'escorts',
    icon: '👨‍⚕️',
    title: '陪诊师审核',
    desc: '查看待审陪诊师 + 审批历史',
    path: '/pages/admin/escorts/pending-audit',
  },
  {
    key: 'refunds',
    icon: '💸',
    title: '退款审批',
    desc: '查看退款工单 + 审批记录',
    path: '/pages/admin/refunds/list',
  },
  {
    key: 'work-orders',
    icon: '🎫',
    title: '客服工单',
    desc: '查看工单处理历史',
    path: '/pages/admin/work-orders/index',
  },
];

function onNavigate(path: string) {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: path });
}

onMounted(() => {
  // 占位：审计聚合入口不发起 API 调用
});
</script>

<template>
  <view class="ui-page" data-testid="admin-audit-page">
    <view class="admin-audit__header">
      <text class="admin-audit__title">审计管理</text>
      <text class="admin-audit__subtitle">审核 / 审批 / 处理历史</text>
    </view>

    <view class="admin-audit__note" data-testid="admin-audit-note">
      <text class="admin-audit__note-text">
        💡 v2 审计数据来源于具体审批操作的 timeline（陪诊师审核 / 退款审批 / 工单处理），请从下方入口查看。
      </text>
    </view>

    <view class="admin-audit__list" data-testid="admin-audit-list">
      <UiCard
        v-for="e in ENTRIES"
        :key="e.key"
        :data-testid="`admin-audit-entry-${e.key}`"
        @click="onNavigate(e.path)"
      >
        <view class="admin-audit__entry">
          <text class="admin-audit__entry-icon">{{ e.icon }}</text>
          <view class="admin-audit__entry-info">
            <text class="admin-audit__entry-title">{{ e.title }}</text>
            <text class="admin-audit__entry-desc">{{ e.desc }}</text>
          </view>
          <text class="admin-audit__entry-arrow">→</text>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-audit__header {
  padding: var(--ui-space-base) 0;
}

.admin-audit__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-audit__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.admin-audit__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-primary);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin-bottom: var(--ui-space-md);
}

.admin-audit__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.admin-audit__list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
}

.admin-audit__entry {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.admin-audit__entry-icon {
  font-size: 32px;
}

.admin-audit__entry-info {
  flex: 1;
}

.admin-audit__entry-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-audit__entry-desc {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-audit__entry-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-disabled);
}
</style>