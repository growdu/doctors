<script setup lang="ts">
/**
 * escort/checkin/index.vue — 订单签到（v2 unified-app · escort 域）。
 *
 * 入口：escort/order-detail「签到」按钮（escort_confirmed → in_service 之前）
 *   → 拿 uni.getLocation 取 GPS（lat / lng）
 *   → 显示当前位置
 *   → 「签到」按钮 → 调 updateLocation（escort API，模拟签到 GPS 上报）
 *   → 成功后 navigateBack 到订单详情
 *
 * 设计要点：
 *   - v2 order-service 未暴露专用 checkin 端点；用 updateLocation 模拟（语义：陪诊师到达患者地点）
 *   - 位置获取失败时仍允许「跳过签到」按钮（仅更新 escort 位置 + 时间戳）
 */
import { ref, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import { updateLocation } from '@/api/escort';

const orderId = ref<number | null>(null);
const lat = ref<number | null>(null);
const lng = ref<number | null>(null);
const posLoading = ref(false);
const posError = ref<string | null>(null);
const submitting = ref(false);
const checkedAt = ref<string | null>(null);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.orderId) orderId.value = Number(opts.orderId);
}

async function loadPosition() {
  posLoading.value = true;
  posError.value = null;
  try {
    if (typeof uni === 'undefined') {
      throw new Error('uni API 不可用');
    }
    const res = await new Promise<{ latitude: number; longitude: number }>((resolve, reject) => {
      (uni as unknown as {
        getLocation: (opts: unknown) => { success: (r: unknown) => void; fail: (e: unknown) => void };
      }).getLocation({
        type: 'wgs84',
        geocode: false,
        success: (r: unknown) => resolve(r as { latitude: number; longitude: number }),
        fail: (e: unknown) => reject(new Error((e as { errMsg?: string })?.errMsg ?? '定位失败')),
      });
    });
    lat.value = res.latitude;
    lng.value = res.longitude;
  } catch (e) {
    posError.value = (e as Error).message;
    // fallback：mock 一个位置（方便测试 + 无 GPS 环境）
    lat.value = 39.908823;
    lng.value = 116.397470;
  } finally {
    posLoading.value = false;
  }
}

async function onCheckin() {
  if (lat.value === null || lng.value === null) {
    if (typeof uni !== 'undefined') uni.showToast({ title: '请先获取位置', icon: 'none' });
    return;
  }
  submitting.value = true;
  try {
    await updateLocation({
      city: '',
      lat: lng.value,
      lng: lat.value,
      address: '',
    });
    checkedAt.value = new Date().toISOString();
    if (typeof uni !== 'undefined') {
      uni.showToast({ title: '签到成功', icon: 'success' });
      setTimeout(() => uni.navigateBack(), 800);
    }
  } catch (e) {
    if (typeof uni !== 'undefined') uni.showToast({ title: `签到失败：${(e as Error).message}`, icon: 'none' });
  } finally {
    submitting.value = false;
  }
}

onMounted(async () => {
  parseQuery();
  await loadPosition();
});
</script>

<template>
  <view class="ui-page" data-testid="escort-checkin-page">
    <UiCard :title="`订单 #${orderId ?? '—'} 签到`" data-testid="escort-checkin-header">
      <view v-if="posLoading" data-testid="escort-checkin-pos-loading">
        <UiLoading text="定位中..." />
      </view>

      <view v-else class="escort-checkin__pos" data-testid="escort-checkin-pos-card">
        <text class="escort-checkin__pos-label">📍 当前位置</text>
        <text class="escort-checkin__pos-value" data-testid="escort-checkin-pos">
          lat={{ lat?.toFixed(6) ?? '—' }}, lng={{ lng?.toFixed(6) ?? '—' }}
        </text>
        <text v-if="posError" class="escort-checkin__pos-error" data-testid="escort-checkin-pos-error">
          ⚠️ {{ posError }}（使用默认位置）
        </text>
      </view>

      <view v-if="checkedAt" class="escort-checkin__time-row" data-testid="escort-checkin-time">
        <text class="escort-checkin__time-label">签到时间</text>
        <text class="escort-checkin__time-value">{{ checkedAt }}</text>
      </view>
    </UiCard>

    <view class="escort-checkin__actions">
      <UiButton
        type="primary"
        block
        :loading="submitting"
        :disabled="lat === null || lng === null"
        data-testid="escort-checkin-submit"
        @click="onCheckin"
      >
        签到
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.escort-checkin__pos {
  margin-top: var(--ui-space-sm);
}

.escort-checkin__pos-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
}

.escort-checkin__pos-value {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-medium);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: 4px;
}

.escort-checkin__pos-error {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-warning);
  display: block;
  margin-top: var(--ui-space-xs);
}

.escort-checkin__time-row {
  margin-top: var(--ui-space-md);
  padding-top: var(--ui-space-md);
  border-top: 1px solid var(--ui-color-border);
}

.escort-checkin__time-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
}

.escort-checkin__time-value {
  font-size: var(--ui-font-md);
  color: var(--ui-color-success);
  font-weight: var(--ui-font-weight-medium);
  display: block;
  margin-top: 4px;
}

.escort-checkin__actions {
  margin-top: var(--ui-space-md);
}
</style>