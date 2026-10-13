<script setup lang="ts">
/**
 * patient/order/candidates/index.vue — 陪诊师候选列表（v2.1 选人模式）。
 *
 * §48.S v2.1 改进：
 *   - 旧：只是列出候选，无操作
 *   - 新：选择候选调 orderService.SelectEscort(orderId, escortId)
 *     → 后端写 selected_escort_id + status='selecting_escort'
 *     → 30s 倒计时启动（escort_pending_expire_at = NOW()+30s）
 *
 * 入口：
 *   - order/create 选择「手动选陪诊师」展开
 *   - URL: /pages/patient/order/candidates/index?orderId=N
 */
import { ref, onMounted } from 'vue';
import { listCandidates } from '@/api/match';
import type { MatchCandidate } from '@/api/match';
import { selectEscort } from '@/api/orders';
import UiCard from '@/components/shared/UiCard.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiButton from '@/components/shared/UiButton.vue';

const orderId = ref<number | null>(null);
const candidates = ref<MatchCandidate[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const picking = ref(false);

function parseQuery() {
  // §48.N: h5 dev mode getCurrentPages() 不带 options；优先读 window.location.search
  const search = typeof window !== 'undefined' ? window.location.search : '';
  const m = search.match(/[?&](?:orderId|id)=(\d+)/);
  if (m) { orderId.value = Number(m[1]); return; }
  if (typeof uni !== 'undefined') {
    const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
    const opts = pages[pages.length - 1]?.options;
    if (opts?.orderId) orderId.value = Number(opts.orderId);
  }
}

async function onLoad() {
  parseQuery();
  if (!orderId.value) {
    error.value = '缺少订单 ID';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    const r = await listCandidates({ order_id: orderId.value, limit: 20 });
    candidates.value = r.items.sort((a, b) => b.score - a.score);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function onPick(c: MatchCandidate) {
  if (!orderId.value) return;
  picking.value = true;
  try {
    // §48.S: 真实调 SelectEscort → 后端写 selected_escort_id + 启动 30s 倒计时
    await selectEscort(orderId.value, c.escort_id);
    // 跳到订单详情（status 已是 selecting_escort，等待 escort accept）
    if (typeof uni !== 'undefined') {
      uni.redirectTo({ url: `/pages/patient/order/detail?id=${orderId.value}` });
    } else if (typeof window !== 'undefined') {
      window.location.href = `/pages/patient/order/detail?id=${orderId.value}`;
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    picking.value = false;
  }
}

onMounted(onLoad);
</script>

<template>
  <view class="ui-page" data-testid="patient-order-candidates">
    <view class="candidates__header">
      <text class="candidates__title">👨‍⚕️ 候选陪诊师</text>
      <text v-if="orderId" class="candidates__subtitle">为订单 #{{ orderId }} 选择陪诊师（30s 倒计时）</text>
    </view>

    <view v-if="loading" data-testid="candidates-loading">
      <UiLoading text="加载候选陪诊师..." />
    </view>

    <view v-else-if="error" data-testid="candidates-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="candidates__retry" @click="onLoad">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <view v-else-if="candidates.length === 0" data-testid="candidates-empty">
      <UiEmpty icon="👨‍⚕️" title="暂无候选陪诊师" description="试试自动分配，或稍后再试" />
    </view>

    <view v-else data-testid="candidates-list">
      <view class="candidates__count">共 {{ candidates.length }} 位候选陪诊师</view>
      <UiCard
        v-for="c in candidates"
        :key="c.escort_id"
        :data-testid="`candidate-${c.escort_id}`"
      >
        <view class="candidates__row">
          <text class="candidates__score" :data-testid="`candidate-score-${c.escort_id}`">{{ c.score }}</text>
          <view class="candidates__info">
            <text class="candidates__name">陪诊师 #{{ c.escort_id }}</text>
            <view class="candidates__meta">
              <text class="candidates__meta-item">📍 {{ c.distance_m }}m</text>
              <text class="candidates__meta-item">⏱️ ETA {{ c.eta_s }}s</text>
              <text :class="['candidates__status', c.available ? 'candidates__status--avail' : 'candidates__status--busy']">
                {{ c.available ? '🟢 可接单' : '🔴 忙碌' }}
              </text>
            </view>
          </view>
        </view>
        <template #footer>
          <UiButton
            type="primary"
            size="sm"
            :disabled="!c.available || picking"
            :loading="picking"
            :data-testid="`pick-candidate-${c.escort_id}`"
            @click="onPick(c)"
          >
            选择此陪诊师
          </UiButton>
        </template>
      </UiCard>
    </view>
  </view>
</template>

<style scoped>
.candidates__header {
  padding: var(--ui-space-base) 0;
}
.candidates__title {
  font-size: var(--ui-font-xl);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  display: block;
}
.candidates__subtitle {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: var(--ui-space-xs);
}

.candidates__count {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  padding: var(--ui-space-sm) 0;
}

.candidates__row {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.candidates__score {
  font-size: 32px;
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-primary);
  flex-shrink: 0;
  width: 56px;
  text-align: center;
}

.candidates__info {
  flex: 1;
  min-width: 0;
}

.candidates__name {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
  margin-bottom: var(--ui-space-xs);
}

.candidates__meta {
  display: flex;
  gap: var(--ui-space-md);
  flex-wrap: wrap;
}

.candidates__meta-item {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
}

.candidates__status {
  font-size: var(--ui-font-xs);
  padding: 2px 8px;
  border-radius: var(--ui-radius-sm);
}

.candidates__status--avail {
  background: rgba(82, 196, 26, 0.1);
  color: var(--ui-color-success);
}
.candidates__status--busy {
  background: rgba(255, 77, 79, 0.1);
  color: var(--ui-color-error);
}

.candidates__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>
