<script setup lang="ts">
/**
 * App.vue：unified-app 顶层入口（Phase 3.0.6）。
 *
 * v2 行为：
 *   - onLaunch：从 localStorage / uni storage 恢复 token + roles + active_role
 *   - 注入 :root CSS 变量（来自 src/styles/tokens.ts）
 *     - 仅 h5 端执行：mp-weixin / app-plus 无 `document`
 *     - 移动端用 uni.scss 预编译走 tokens（Phase 3.0.6 已 inline）
 */
import { onLaunch } from '@dcloudio/uni-app';
import { useAuthStore } from '@/store/auth';

onLaunch(() => {
  const auth = useAuthStore();
  auth.bootstrap();
});
</script>

<script lang="ts">
/**
 * 第二个 script 块：h5 端注入 :root CSS 变量
 * （uni-app Vue SFC 限制：style 块不能放运行时字符串拼接）
 *
 * 端兼容：
 *   - h5        : document.head.appendChild 注入
 *   - mp-weixin : ❌ 无 document，跳过（CSS 变量由 uni.scss 预编译注入）
 *   - app-plus  : ❌ 无 document，跳过（同上）
 */
import { generateCssVarsBlock } from '@/styles/tokens';

// #ifdef H5
if (typeof document !== 'undefined') {
  const style = document.createElement('style');
  style.setAttribute('data-ui-tokens', '');
  style.textContent = generateCssVarsBlock();
  document.head.appendChild(style);
}
// #endif
</script>

<style lang="css">
/* 工具类（CSS var 直接引用，uni.scss 已 inline） */
</style>