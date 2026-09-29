<script setup lang="ts">
/**
 * admin/users/index.vue — 用户管理（v2 unified-app · admin 域）。
 *
 * 入口：admin dashboard「用户」相关入口
 *   → 调 listUsers() 拉用户列表
 *   → 行操作：仅查看（v2 admin 未暴露 ban/unban 端点，跳 user-detail 占位）
 *
 * 设计要点：
 *   - 简化版：纯列表展示 + 角色筛选（patient / escort / admin）
 *   - 复用 patient/list 的列表卡风格
 */
import { ref, onMounted } from 'vue';
import { listUsers } from '@/api/admin';
import type { AdminUser } from '@/api/admin';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const items = ref<AdminUser[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const roleFilter = ref<string | 'all'>('all');

const ROLE_LABEL: Record<string, string> = {
  patient: '患者',
  escort: '陪诊师',
  admin: '管理员',
  super_admin: '超管',
  audit_admin: '审核',
  order_admin: '订单',
  refund_admin: '退款',
  finance_admin: '财务',
  cs: '客服',
  viewer: '只读',
};

function roleLabel(r: string): string {
  return ROLE_LABEL[r] ?? r;
}

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listUsers(
      roleFilter.value === 'all' ? {} : { role: roleFilter.value },
    );
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onSwitchRole(role: string | 'all') {
  roleFilter.value = role;
  await onLoad();
}

const ROLE_TABS: Array<{ key: string; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'patient', label: '患者' },
  { key: 'escort', label: '陪诊师' },
  { key: 'super_admin', label: '超管' },
];

function formatDate(iso: string): string {
  return iso.split('T')[0] + ' ' + (iso.split('T')[1]?.substring(0, 5) || '');
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="admin-users-page">
    <!-- role 过滤 tab -->
    <view class="admin-users__tabs" data-testid="admin-users-tabs">
      <view
        v-for="t in ROLE_TABS"
        :key="t.key"
        class="admin-users__tab"
        :class="{ 'admin-users__tab--active': roleFilter === t.key }"
        :data-testid="`admin-users-tab-${t.key}`"
        @click="onSwitchRole(t.key)"
      >
        {{ t.label }}
      </view>
    </view>

    <view v-if="loading" data-testid="admin-users-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="admin-users-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="admin-users__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="admin-users-empty">
      <UiEmpty icon="👥" title="暂无用户" />
    </view>

    <view v-else data-testid="admin-users-list">
      <UiCard
        v-for="u in items"
        :key="u.id"
        :data-testid="`admin-users-card-${u.id}`"
      >
        <view class="admin-users__row">
          <text class="admin-users__id">#{{ u.id }}</text>
          <text
            v-if="u.real_name_verified"
            class="admin-users__badge admin-users__badge--verified"
            :data-testid="`admin-users-verified-${u.id}`"
          >
            已实名
          </text>
          <text
            v-else
            class="admin-users__badge admin-users__badge--unverified"
            :data-testid="`admin-users-unverified-${u.id}`"
          >
            未实名
          </text>
        </view>
        <text class="admin-users__name">{{ u.nickname }}</text>
        <text class="admin-users__phone">📱 {{ u.phone }}</text>
        <text class="admin-users__role">当前角色：{{ roleLabel(u.active_role) }}</text>
        <text class="admin-users__roles">所有角色：{{ u.roles.map(roleLabel).join('、') }}</text>
        <text class="admin-users__time">📅 注册于 {{ formatDate(u.created_at) }}</text>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.admin-users__tabs {
  display: flex;
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
  padding: 4px;
  overflow-x: auto;
}

.admin-users__tab {
  flex: 1;
  min-width: 64px;
  text-align: center;
  padding: var(--ui-space-sm);
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  border-radius: var(--ui-radius-sm);
  cursor: pointer;
}

.admin-users__tab--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.admin-users__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-sm);
}

.admin-users__id {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
}

.admin-users__badge {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
  font-weight: var(--ui-font-weight-medium);
}

.admin-users__badge--verified {
  background: var(--ui-color-success);
  color: var(--ui-color-text-inverse);
}

.admin-users__badge--unverified {
  background: var(--ui-color-text-disabled);
  color: var(--ui-color-text-inverse);
}

.admin-users__name {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.admin-users__phone,
.admin-users__role,
.admin-users__roles,
.admin-users__time {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.admin-users__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>