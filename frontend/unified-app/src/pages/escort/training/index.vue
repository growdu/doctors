<script setup lang="ts">
/**
 * escort/training/index.vue — 陪诊师培训记录（v2 unified-app · escort 域）。
 *
 * 入口：escort home「培训记录」入口
 *   → listTrainings() 拉培训记录列表（只读）
 *   → 渲染卡片（标题 / 完成时间 / 证书链接）
 *
 * 设计要点：
 *   - 简化版：v2 后端暂未开放自助新增培训入口（admin 域审核员录入）
 *   - 仅展示，无操作按钮
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { listTrainings } from '@/api/escort';
import type { Training } from '@/api/escort';

const items = ref<Training[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

async function onLoad() {
  loading.value = true;
  error.value = null;
  try {
    const r = await listTrainings();
    items.value = r.items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatDate(iso: string): string {
  return iso.split('T')[0];
}

function onViewCert(url: string) {
  if (typeof uni !== 'undefined') uni.setClipboardData({ data: url });
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="escort-training-page">
    <view v-if="loading" data-testid="escort-training-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error" data-testid="escort-training-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="escort-training__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="items.length === 0" data-testid="escort-training-empty">
      <UiEmpty icon="📚" title="暂无培训记录" />
    </view>

    <view v-else data-testid="escort-training-list">
      <UiCard
        v-for="t in items"
        :key="t.id"
        :data-testid="`escort-training-card-${t.id}`"
      >
        <text class="escort-training__title">{{ t.title }}</text>
        <text class="escort-training__date">📅 完成于 {{ formatDate(t.completed_at) }}</text>
        <view v-if="t.cert_url" class="escort-training__cert-row">
          <text class="escort-training__cert-label">📎 证书：</text>
          <text
            class="escort-training__cert-url"
            :data-testid="`escort-training-cert-${t.id}`"
            @click="onViewCert(t.cert_url!)"
          >
            点击复制
          </text>
        </view>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.escort-training__title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
}

.escort-training__date {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.escort-training__cert-row {
  display: flex;
  align-items: center;
  margin-top: var(--ui-space-xs);
  padding-top: var(--ui-space-xs);
  border-top: 1px solid var(--ui-color-border);
}

.escort-training__cert-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.escort-training__cert-url {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-primary);
  text-decoration: underline;
  cursor: pointer;
}

.escort-training__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>