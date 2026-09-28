<script setup lang="ts">
/**
 * unified-app escort 域 demo 页面（v2 Phase 2 spike 占位）。
 *
 * 真实 escort 域业务（10 模块）待 Phase 3 业务迁移时填充。
 */
import RoleGuard from '@/components/shared/RoleGuard.vue';
import { useAuthStore } from '@/store/auth';

const auth = useAuthStore();
</script>

<template>
  <RoleGuard :required="['escort']" fallback-title="需要 escort 角色">
    <view class="ui-page">
      <view class="ui-card">
        <view class="ui-title">🚑 陪诊师域</view>
        <view class="ui-subtitle">激活角色：{{ auth.activeRole }}</view>
        <view class="ui-subtitle">
          Phase 3 业务迁移：feed 抢单池 / task 详情 / 我的订单 / 钱包 / 培训 / 消息 / 等 10 模块
        </view>
      </view>
    </view>
  </RoleGuard>
</template>