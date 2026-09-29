<script setup lang="ts">
/**
 * RoleGuard 组件：unified-app 路由级守卫。
 *
 * 用法（pages.json 注册路由时）：
 *   <RoleGuard :required="['patient']">
 *     <patient-page />
 *   </RoleGuard>
 *
 * v2 行为：
 *   - activeRole ∈ required：渲染默认插槽
 *   - else：渲染 fallback 插槽（默认"403 无权限"提示 + 切换角色按钮）
 *
 * 对应 spec：2026-09-28-unified-app-v2.md §2.5
 */
import { computed } from 'vue';
import { useAuthStore } from '@/store/auth';
import type { Role } from '@/types/auth';

const props = defineProps<{
  required: Role[];
  fallbackTitle?: string;
}>();

const auth = useAuthStore();
const active = computed<Role | ''>(() => auth.activeRole);
const allowed = computed(() => props.required.includes(active.value as Role));
</script>

<template>
  <view v-if="allowed">
    <slot />
  </view>
  <view v-else class="ui-page">
    <view class="ui-card">
      <view class="ui-title">403 · 无访问权限</view>
      <view class="ui-subtitle">
        当前角色：{{ active || '未登录' }}；本页面要求：{{ required.join(' / ') }}
      </view>
      <view style="margin-top: 12px;">
        <u-button type="primary" size="small" @click="auth.switchRole(required[0])">
          切换到 {{ required[0] }}
        </u-button>
      </view>
    </view>
  </view>
</template>
