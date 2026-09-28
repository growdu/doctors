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
@import '@/uni.scss';

/* 全局 box-sizing + safe-area */
page {
  box-sizing: border-box;
  padding-bottom: env(safe-area-inset-bottom);
}

/* 统一应用强调色 */
.ui-page { padding: 16px; }
.ui-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}
.ui-title { font-size: 18px; font-weight: 600; margin-bottom: 8px; }
.ui-subtitle { font-size: 14px; color: #666; }
</style>