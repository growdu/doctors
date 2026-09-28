<script setup lang="ts">
/**
 * unified-app patient 域 demo 页面（v2 Phase 2 spike 占位）。
 *
 * 真实 patient 域业务（48 个模块）待 Phase 3 业务迁移时填充；
 * 当前仅占位：验证 RoleGuard patient-only + 显示域标识。
 */
import RoleGuard from '@/components/shared/RoleGuard.vue';
import { useAuthStore } from '@/store/auth';

const auth = useAuthStore();
</script>

<template>
  <RoleGuard :required="['patient']" fallback-title="需要 patient 角色">
    <view class="ui-page">
      <view class="ui-card">
        <view class="ui-title">🩺 患者域</view>
        <view class="ui-subtitle">激活角色：{{ auth.activeRole }}</view>
        <view class="ui-subtitle">
          Phase 3 业务迁移：hospital / package / order / payment / refund / address / coupon / review / virtualnumber / message / sos / 等 14 模块
        </view>
      </view>
    </view>
  </RoleGuard>
</template>