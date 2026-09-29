<script setup lang="ts">
/**
 * escort/profile/index.vue — 陪诊师个人中心（v2 unified-app · escort 域）。
 *
 * 入口：escort home「个人中心」卡片
 *   → 展示当前 escort 用户的 authStore.user + 资料 + 退出登录
 *
 * 设计要点：
 *   - 完全基于 authStore.user（无额外 API 调用）
 *   - 简化版：仅展示用户基本信息 + 退出登录
 *   - 详细 escort profile（资质 / 在线时段 / 培训记录）待 Phase 3.3 增量
 */
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import { useAuthStore } from '@/store/auth';

const auth = useAuthStore();

function maskPhone(phone: string): string {
  if (phone.length !== 11) return phone;
  return `${phone.slice(0, 3)}****${phone.slice(7)}`;
}

function onLogout() {
  auth.onUnauthorized();
}
</script>

<template>
  <view class="ui-page" data-testid="escort-profile-page">
    <view v-if="auth.user" data-testid="escort-profile-content">
        <UiCard title="基本信息" data-testid="escort-profile-info-card">
          <view class="escort-profile__row">
            <text class="escort-profile__label">用户 ID</text>
            <text class="escort-profile__value">#{{ auth.user.id }}</text>
          </view>
          <view class="escort-profile__row">
            <text class="escort-profile__label">手机</text>
            <text class="escort-profile__value">{{ maskPhone(auth.user.phone) }}</text>
          </view>
          <view class="escort-profile__row">
            <text class="escort-profile__label">实名状态</text>
            <text
              class="escort-profile__value"
              :class="auth.user.real_name_verified ? 'escort-profile__verified' : 'escort-profile__unverified'"
              :data-testid="auth.user.real_name_verified ? 'escort-profile-verified' : 'escort-profile-unverified'"
            >
              {{ auth.user.real_name_verified ? '已实名' : '未实名' }}
            </text>
          </view>
        </UiCard>

        <UiCard title="角色信息" data-testid="escort-profile-roles-card">
          <view class="escort-profile__row">
            <text class="escort-profile__label">当前角色</text>
            <text class="escort-profile__value escort-profile__active-role" data-testid="escort-profile-active-role">
              {{ auth.user.active_role }}
            </text>
          </view>
          <view class="escort-profile__row escort-profile__row--top">
            <text class="escort-profile__label">所有角色</text>
            <view class="escort-profile__value">
              <view
                v-for="r in auth.user.roles"
                :key="r"
                class="escort-profile__role-chip"
                :class="{ 'escort-profile__role-chip--active': r === auth.user.active_role }"
                :data-testid="`escort-profile-role-${r}`"
              >
                {{ r }}
              </view>
            </view>
          </view>
        </UiCard>

        <view class="escort-profile__note" data-testid="escort-profile-note">
          <text class="escort-profile__note-text">
            💡 详细陪诊师资料（资质 / 在线时段 / 培训记录）请通过 escort-service 后台更新，或等后续 Phase 3.3 增量。
          </text>
        </view>

        <view class="escort-profile__actions" data-testid="escort-profile-actions">
          <UiButton type="danger" block data-testid="escort-profile-logout" @click="onLogout">
            退出登录
          </UiButton>
        </view>
      </view>

      <view v-else data-testid="escort-profile-empty">
        <UiCard>
          <text class="escort-profile__empty">请先登录</text>
        </UiCard>
      </view>
  </view>
</template>

<style scoped>
.escort-profile__row {
  display: flex;
  align-items: center;
  margin-bottom: var(--ui-space-sm);
}

.escort-profile__row--top {
  align-items: flex-start;
}

.escort-profile__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  width: 80px;
  flex-shrink: 0;
}

.escort-profile__value {
  flex: 1;
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
}

.escort-profile__verified { color: var(--ui-color-success); }
.escort-profile__unverified { color: var(--ui-color-text-disabled); }

.escort-profile__active-role {
  color: var(--ui-color-primary);
  font-weight: var(--ui-font-weight-medium);
}

.escort-profile__role-chip {
  display: inline-block;
  font-size: var(--ui-font-xs);
  padding: 2px var(--ui-space-sm);
  background: var(--ui-color-bg-hover);
  border-radius: var(--ui-radius-sm);
  color: var(--ui-color-text-secondary);
  margin: 2px 4px 2px 0;
}

.escort-profile__role-chip--active {
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-weight: var(--ui-font-weight-medium);
}

.escort-profile__note {
  background: var(--ui-color-bg-card);
  border-left: 3px solid var(--ui-color-info);
  padding: var(--ui-space-sm);
  border-radius: var(--ui-radius-sm);
  margin: var(--ui-space-md) 0;
}

.escort-profile__note-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.5;
}

.escort-profile__actions {
  margin-top: var(--ui-space-md);
}

.escort-profile__empty {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-secondary);
  display: block;
  text-align: center;
  padding: var(--ui-space-base) 0;
}
</style>