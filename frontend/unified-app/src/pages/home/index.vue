<script setup lang="ts">
/**
 * unified-app HomeShell（v2 spike demo 页面 — 原生 Vue 版）。
 *
 * Phase 2 spike：uview-plus 3.8.125 组件 u-index-list 有重复声明 bug，
 * 临时改用原生 HTML 元素；Phase 3 业务迁移时换掉 uview-plus 依赖或 fix 第三方包。
 *
 * 功能：
 *   - 显示当前用户（phone + active_role + roles[]）
 *   - 提供 SMS 登录入口
 *   - 列出可选角色 chips，点击切换 active
 *   - 提供 3 域入口的跳链接
 */
import { computed, ref } from 'vue';
import { useAuthStore } from '@/store/auth';
import type { Role } from '@/types/auth';

const auth = useAuthStore();
const phone = ref('13800138000');
const code = ref('');
const smsSent = ref(false);

const availableRoles = computed<Role[]>(() => Array.from(new Set(auth.roles as Role[])));

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
  smsSent.value = true;
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: '短信已发（查看 auth-service 日志拿验证码）', icon: 'none' });
  } else {
    alert('短信已发（查看 auth-service 日志拿验证码）');
  }
}

async function onLogin() {
  if (!phone.value || !code.value) {
    if (typeof uni !== 'undefined') uni.showToast({ title: 'phone + code 必填', icon: 'none' });
    return;
  }
  try {
    await auth.login(phone.value, code.value);
    if (typeof uni !== 'undefined') uni.showToast({ title: '登录成功', icon: 'success' });
  } catch (e) {
    const msg = (e as Error).message || '登录失败';
    if (typeof uni !== 'undefined') uni.showToast({ title: msg, icon: 'none' });
    else alert(msg);
  }
}

async function onSwitchRole(r: Role) {
  try {
    await auth.switchRole(r);
  } catch (e) {
    const msg = (e as Error).message || '切换失败';
    if (typeof uni !== 'undefined') uni.showToast({ title: msg, icon: 'none' });
    else alert(msg);
  }
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
      <view style="margin-bottom: 12px;">
        <label>手机号：</label>
        <input v-model="phone" placeholder="13800138000" style="border: 1px solid #ddd; padding: 4px 8px; width: 200px;" />
      </view>
      <view style="margin-bottom: 12px;">
        <label>验证码：</label>
        <input v-model="code" placeholder="6 位 mock code（看 auth-service 日志）" style="border: 1px solid #ddd; padding: 4px 8px; width: 200px;" />
      </view>
      <button @click="onSendSms" style="margin-right: 8px;">发验证码</button>
      <button @click="onLogin" style="background: #1677ff; color: #fff; padding: 4px 16px; border: none; border-radius: 4px;">登录</button>
      <view v-if="smsSent" style="margin-top: 8px; color: #999; font-size: 12px;">短信已发（SMS code 看 auth-service 日志）</view>
    </view>

    <!-- 已登录：用户卡 + 角色切换 + 域入口 -->
    <view v-else>
      <view class="ui-card">
        <view class="ui-title">{{ auth.user?.phone }}</view>
        <view class="ui-subtitle">激活角色：{{ auth.activeRole }}</view>
        <view class="ui-subtitle">全部角色：{{ auth.roles.join(' / ') }}</view>
      </view>

      <view class="ui-card">
        <view class="ui-title">切换激活角色</view>
        <view style="display: flex; gap: 8px; flex-wrap: wrap;">
          <button
            v-for="r in availableRoles"
            :key="r"
            :style="r === auth.activeRole ? 'background:#1677ff;color:#fff' : 'background:#eee'"
            @click="onSwitchRole(r)"
            style="padding: 6px 12px; border: none; border-radius: 4px; cursor: pointer;"
          >
            {{ domIcons[r] }} {{ r }}
          </button>
        </view>
      </view>

      <view class="ui-card">
        <view class="ui-title">域入口</view>
        <button
          v-for="(path, name) in { '🩺 患者域': 'patient', '🚑 陪诊师域': 'escort', '🛡️ 管理后台': 'admin' }"
          :key="name"
          @click="typeof uni !== 'undefined' ? uni.navigateTo({ url: `/pages/${path}/index` }) : null"
          style="display: block; width: 100%; padding: 10px; margin-bottom: 8px; border: 1px solid #ddd; background: #fff; border-radius: 4px;"
        >
          {{ name }}
        </button>
      </view>
    </view>
  </view>
</template>