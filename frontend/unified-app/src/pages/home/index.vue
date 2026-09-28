<script setup lang="ts">
/**
 * unified-app HomeShell（v2 spike demo 页面）。
 *
 * 功能：
 *   - 显示当前用户（phone + active_role + roles[]）
 *   - 提供 SMS 登录入口（输入 phone 拿 mock 验证码 → 填 code 登录）
 *   - 列出可选角色 chips，点击切换 active（调 /auth/switch-role）
 *   - 提供 3 域入口（patient / escort / admin）的跳转链接
 *
 * 对应 spec：2026-09-28-unified-app-v2.md §2.4
 */
import { computed, ref } from 'vue';
import { useAuthStore } from '@/store/auth';
import type { Role } from '@/types/auth';

const auth = useAuthStore();
const phone = ref('13800138000');
const code = ref('');
const smsSent = ref(false);

const availableRoles = computed<Role[]>(() => {
  const set = new Set<Role>(auth.roles as Role[]);
  return Array.from(set);
});

const domIcons: Record<Role, string> = {
  patient: '🩺',
  escort: '🚑',
  super_admin: '🛡️',
  order_admin: '📋',
  refund_admin: '💰',
  cs: '🎧',
  audit_admin: '🔍',
  viewer: '👁️',
};

async function onSendSms() {
  // dev spike：直接调 /sms/send；实际后端会返回 ttl，code 从 log 读
  await auth.$reset(); // 清旧状态
  // 这里简化：调 login 时 SMS 已下发
  smsSent.value = true;
  uni.showToast({ title: '短信已发（查看 auth-service 日志拿验证码）', icon: 'none' });
}

async function onLogin() {
  if (!phone.value || !code.value) {
    uni.showToast({ title: 'phone + code 必填', icon: 'none' });
    return;
  }
  try {
    await auth.login(phone.value, code.value);
    uni.showToast({ title: '登录成功', icon: 'success' });
  } catch (e) {
    uni.showToast({ title: (e as Error).message || '登录失败', icon: 'none' });
  }
}

async function onSwitchRole(r: Role) {
  try {
    await auth.switchRole(r);
    uni.showToast({ title: `已切换到 ${r}`, icon: 'success' });
  } catch (e) {
    uni.showToast({ title: (e as Error).message || '切换失败', icon: 'none' });
  }
}

function goPage(path: string) {
  uni.navigateTo({ url: `/pages/${path}/index` });
}
</script>

<template>
  <view class="ui-page">
    <view class="ui-card">
      <view class="ui-title">unified-app v2 spike</view>
      <view class="ui-subtitle">3 端合并 + 多角色 + 角色切换 demo</view>
    </view>

    <!-- 未登录：SMS 登录 -->
    <view v-if="!auth.isAuthed" class="ui-card">
      <view class="ui-title">SMS 登录</view>
      <u-form>
        <u-form-item label="手机号">
          <u-input v-model="phone" placeholder="13800138000" />
        </u-form-item>
        <u-form-item label="验证码">
          <u-input v-model="code" placeholder="6 位 mock code（看 auth-service 日志）" />
        </u-form-item>
      </u-form>
      <view style="display: flex; gap: 8px;">
        <u-button @click="onSendSms">发验证码</u-button>
        <u-button type="primary" @click="onLogin">登录</u-button>
      </view>
    </view>

    <!-- 已登录：显示用户 + 角色切换 + 域入口 -->
    <view v-else>
      <view class="ui-card">
        <view class="ui-title">{{ auth.user?.phone }}</view>
        <view class="ui-subtitle">激活角色：{{ auth.activeRole }}</view>
        <view class="ui-subtitle">全部角色：{{ auth.roles.join(' / ') }}</view>
      </view>

      <view class="ui-card">
        <view class="ui-title">切换激活角色</view>
        <view style="display: flex; gap: 8px; flex-wrap: wrap;">
          <u-tag
            v-for="r in availableRoles"
            :key="r"
            :text="domIcons[r] + ' ' + r"
            :type="r === auth.activeRole ? 'primary' : 'info'"
            @click="onSwitchRole(r)"
          />
        </view>
      </view>

      <view class="ui-card">
        <view class="ui-title">域入口</view>
        <u-button block @click="goPage('patient')">🩺 患者域（patient）</u-button>
        <view style="height: 8px;" />
        <u-button block @click="goPage('escort')">🚑 陪诊师域（escort）</u-button>
        <view style="height: 8px;" />
        <u-button block @click="goPage('admin')">🛡️ 管理后台（admin）</u-button>
      </view>
    </view>
  </view>
</template>