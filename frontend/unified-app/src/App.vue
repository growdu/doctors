<script setup lang="ts">
/**
 * App.vue：unified-app 顶层入口。
 *
 * v2 行为：
 *   - onLaunch：从 localStorage 恢复 token + roles + active_role
 *   - 监听 active 切换，触发导航到对应域首页
 */
import { onLaunch } from '@dcloudio/uni-app';
import { useAuthStore } from '@/store/auth';

onLaunch(() => {
  const auth = useAuthStore();
  auth.bootstrap();
});
</script>

<style lang="scss">
/* 全局样式已迁移至 src/styles/global.scss（Phase 3.0.1 design tokens） */
@import '@/uni.scss';
</style>