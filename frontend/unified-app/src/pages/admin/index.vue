<script setup lang="ts">
/**
 * unified-app admin 域 demo 页面（v2 Phase 2 spike 占位）。
 *
 * 真实 admin 域业务（51 个页面：dashboard / orders / escorts / refunds / 等）待 Phase 3 迁移。
 * 当前仅占位：验证 RoleGuard admin-only + 显示域标识。
 */
import RoleGuard from '@/components/shared/RoleGuard.vue';
import { useAuthStore } from '@/store/auth';

const auth = useAuthStore();
</script>

<template>
  <RoleGuard :required="['super_admin', 'refund_admin', 'order_admin', 'audit_admin', 'cs', 'viewer']">
    <view class="ui-page">
      <view class="ui-card">
        <view class="ui-title">🛡️ 管理后台</view>
        <view class="ui-subtitle">激活角色：{{ auth.activeRole }}</view>
        <view class="ui-subtitle">
          Phase 3 业务迁移：dashboard 12 模块 + orders / escorts / refunds / wallets / messages / reviews / sos / couriers / packages 等 51 个页面
        </view>
      </view>
    </view>
  </RoleGuard>
</template>