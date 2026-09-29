<script setup lang="ts">
/**
 * App.vue：unified-app 顶层入口（Phase 3.0.6）。
 *
 * v2 行为：
 *   - onLaunch：从 localStorage 恢复 token + roles + active_role
 *   - 注入 :root CSS 变量（来自 src/styles/tokens.ts）
 *     - 替代 SCSS @import：uni-app sass-loader 在 scoped style 里解析相对路径不可靠
 *     - CSS 变量是 runtime 方案，组件 var(--ui-color-primary) 直接可用
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
 * 第二个 script 块：在 setup 外注入 :root CSS 变量
 * （uni-app Vue SFC 限制：style 块不能放运行时字符串拼接）
 */
import { generateCssVarsBlock } from '@/styles/tokens';

if (typeof document !== 'undefined') {
  const style = document.createElement('style');
  style.setAttribute('data-ui-tokens', '');
  style.textContent = generateCssVarsBlock();
  document.head.appendChild(style);
}
</script>

<style lang="css">
/* 工具类（CSS var 直接引用，uni.scss 已 inline） */
</style>