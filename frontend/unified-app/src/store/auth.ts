/**
 * auth store（Pinia setup syntax）：v2 多角色状态 + 角色切换。
 *
 * 用 setup syntax 而非 options syntax：TS 推导更准（this 类型完整）；
 * getter 是 computed + action 是普通 function，调用风格一致（auth.bootstrap()）。
 *
 * 状态：
 *   - token：当前 JWT
 *   - user：当前用户画像（id / phone / active_role / roles[]）
 *   - activeRole / roles：getter（computed）
 *
 * 行为：
 *   - bootstrap()：onLaunch 时从 localStorage 恢复
 *   - login()：SMS 登录后写入 token + user
 *   - switchRole()：调 /auth/switch-role 拿新 token + 更新 active
 *   - hasRole(role)：判断用户是否有该角色（前端路由守卫 + UI 显示用）
 *   - onUnauthorized()：401 回调清状态
 *
 * 对应 spec：2026-09-28-unified-app-v2.md §2.4
 */
import { computed, ref } from 'vue';
import { defineStore } from 'pinia';
import type { MeResponse, Role } from '@/types/auth';
import { fetchMe, loginByPhone, sendSmsCode, switchActiveRole } from '@/api/auth';
import { storage } from '@/utils/storage';

const STORAGE_KEY = 'unified.auth';

interface PersistedAuth {
  token: string;
  user: MeResponse;
}

function readPersisted(): PersistedAuth | null {
  return storage.getJSON<PersistedAuth>(STORAGE_KEY);
}

function writePersisted(data: PersistedAuth): void {
  storage.setItem(STORAGE_KEY, data);
}

function clearPersisted(): void {
  storage.removeItem(STORAGE_KEY);
}

function reLaunchHome() {
  if (typeof uni !== 'undefined') uni.reLaunch({ url: '/pages/home/index' });
}

export const useAuthStore = defineStore('auth', () => {
  // ── state ──────────────────────────────────────────────
  const token = ref('');
  const user = ref<MeResponse | null>(null);

  // ── getters ────────────────────────────────────────────
  const isAuthed = computed(() => !!token.value && !!user.value);
  const activeRole = computed<Role | ''>(() => (user.value?.active_role as Role) || '');
  const roles = computed<Role[]>(() => (user.value?.roles as Role[]) || []);

  // ── actions ────────────────────────────────────────────
  function bootstrap() {
    const data = readPersisted();
    if (data) {
      token.value = data.token;
      user.value = data.user;
    }
  }

  async function login(phone: string, code: string) {
    await sendSmsCode(phone);
    const r = await loginByPhone(phone, code);
    token.value = r.token;
    await refreshMe();
    persist();
  }

  async function refreshMe() {
    user.value = await fetchMe();
  }

  async function switchRole(active: Role) {
    const r = await switchActiveRole(active);
    token.value = r.token;
    await refreshMe();
    persist();
  }

  function hasRole(role: Role): boolean {
    return roles.value.includes(role);
  }

  function onUnauthorized() {
    token.value = '';
    user.value = null;
    clearPersisted();
    reLaunchHome();
  }

  function persist() {
    if (!user.value) return;
    writePersisted({ token: token.value, user: user.value });
  }

  return {
    token,
    user,
    isAuthed,
    activeRole,
    roles,
    bootstrap,
    login,
    refreshMe,
    switchRole,
    hasRole,
    onUnauthorized,
    persist,
  };
});