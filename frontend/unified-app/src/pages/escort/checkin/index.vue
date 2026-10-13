<script setup lang="ts">
/**
 * escort/checkin/index.vue — 订单签到（v2 unified-app · escort 域 · v2.1 升级）。
 *
 * v2.1 升级：3 步骤状态机（定位 → 确认 → 完成）+ 距离医院估算 + 备注字段。
 * v2 order-service 未暴露专用 /checkin 端点；用 updateLocation（escort-service）模拟签到。
 *
 * 设计要点：
 *   - 3 步状态：locating / ready / done（每步独立显示，UX 更清晰）
 *   - 距离估算：硬编码医院经纬度作参考（v2 不强求精确；v3 接 order.address 后计算真实距离）
 *   - 备注字段：陪诊师可填「已到达患者门口」/「提前 5 分钟」等
 *   - 区块级 catch：位置失败时降级到 mock + 「使用默认位置」提示
 */
import { ref, computed, onMounted } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiLoading from '@/components/shared/UiLoading.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import { updateLocation } from '@/api/escort';

const orderId = ref<number | null>(null);
const lat = ref<number | null>(null);
const lng = ref<number | null>(null);
const posLoading = ref(false);
const posError = ref<string | null>(null);
const submitting = ref(false);
const checkedAt = ref<string | null>(null);
const remark = ref('');

type Step = 'locating' | 'ready' | 'done';
const step = ref<Step>('locating');

/** v2 mock 医院位置（北京协和医院）—— 距离估算用 */
const HOSPITAL_LAT = 39.9135;
const HOSPITAL_LNG = 116.4188;

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
    step.value = 'ready';
  } catch (e) {
    posError.value = (e as Error).message;
    // fallback：mock 一个位置（方便测试 + 无 GPS 环境）
    lat.value = 39.908823;
    lng.value = 116.397470;
    step.value = 'ready';
  } finally {
    posLoading.value = false;
  }
}

/** Haversine 公式估算距离（km） */
const distanceKm = computed<number | null>(() => {
  if (lat.value === null || lng.value === null) return null;
  const R = 6371;
  const dLat = ((HOSPITAL_LAT - lat.value) * Math.PI) / 180;
  const dLng = ((HOSPITAL_LNG - lng.value) * Math.PI) / 180;
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos((lat.value * Math.PI) / 180) *
      Math.cos((HOSPITAL_LAT * Math.PI) / 180) *
      Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(a));
});

const STEP_LABEL: Record<Step, string> = {
  locating: '① 获取位置',
  ready: '② 确认签到',
  done: '③ 已完成',
};

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
      address: remark.value || '',
    });
    checkedAt.value = new Date().toISOString();
    step.value = 'done';
    if (typeof uni !== 'undefined') {
      uni.showToast({ title: '签到成功', icon: 'success' });
    }
  } catch (e) {
    if (typeof uni !== 'undefined') uni.showToast({ title: `签到失败：${(e as Error).message}`, icon: 'none' });
  } finally {
    submitting.value = false;
  }
}

function onDone() {
  if (typeof uni !== 'undefined') {
    uni.navigateBack();
  }
}

onMounted(async () => {
  parseQuery();
  await loadPosition();
});
</script>

<template>
  <view class="ui-page" data-testid="escort-checkin-page">
    <!-- 步骤状态机 -->
    <view class="escort-checkin__steps" data-testid="escort-checkin-steps">
      <text
        v-for="s in (['locating', 'ready', 'done'] as Step[])"
        :key="s"
        class="escort-checkin__step"
        :class="{
          'escort-checkin__step--active': step === s,
          'escort-checkin__step--done': step === 'done' || (s === 'locating' && step !== 'locating'),
        }"
        :data-testid="`escort-checkin-step-${s}`"
      >
        {{ STEP_LABEL[s] }}
      </text>
    </view>

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
        <view v-if="distanceKm !== null" class="escort-checkin__distance" data-testid="escort-checkin-distance">
          <text class="escort-checkin__distance-label">距离医院（参考）：</text>
          <text class="escort-checkin__distance-value">{{ distanceKm.toFixed(2) }} km</text>
        </view>
      </view>

      <!-- 备注字段（仅在 ready / done 显示） -->
      <view v-if="step !== 'locating'" class="escort-checkin__remark" data-testid="escort-checkin-remark-field">
        <text class="escort-checkin__remark-label">备注（可选）</text>
        <textarea
          v-model="remark"
          class="escort-checkin__remark-input"
          placeholder="如：已到达患者门口 / 提前 5 分钟"
          :disabled="step === 'done'"
          data-testid="escort-checkin-remark-input"
        />
      </view>

      <!-- 完成态：显示签到时间 + 返回按钮 -->
      <view v-if="step === 'done'" class="escort-checkin__done" data-testid="escort-checkin-done">
        <view class="escort-checkin__time-row">
          <text class="escort-checkin__time-label">✅ 签到成功</text>
          <text class="escort-checkin__time-value" data-testid="escort-checkin-time">{{ checkedAt }}</text>
        </view>
        <view v-if="remark" class="escort-checkin__remark-display">
          <text class="escort-checkin__remark-label">备注：</text>
          <text class="escort-checkin__remark-text">{{ remark }}</text>
        </view>
      </view>
    </UiCard>

    <!-- 操作区 -->
    <view v-if="step === 'ready'" class="escort-checkin__actions">
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
    <view v-else-if="step === 'done'" class="escort-checkin__actions">
      <UiButton type="primary" block data-testid="escort-checkin-back" @click="onDone">
        返回订单详情
      </UiButton>
    </view>
    <view v-else class="escort-checkin__actions">
      <UiEmpty icon="📡" title="等待定位..." />
    </view>
  </view>
</template>

<style scoped>
.escort-checkin__steps {
  display: flex;
  justify-content: space-between;
  gap: var(--ui-space-sm);
  margin-bottom: var(--ui-space-md);
}

.escort-checkin__step {
  flex: 1;
  text-align: center;
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  padding: var(--ui-space-xs);
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  border: 1px solid var(--ui-color-border);
}

.escort-checkin__step--active {
  color: var(--ui-color-text-inverse);
  background: var(--ui-color-primary);
  border-color: var(--ui-color-primary);
  font-weight: var(--ui-font-weight-medium);
}

.escort-checkin__step--done {
  color: var(--ui-color-success);
  border-color: var(--ui-color-success);
}

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

.escort-checkin__distance {
  display: flex;
  align-items: center;
  margin-top: var(--ui-space-sm);
  padding-top: var(--ui-space-sm);
  border-top: 1px dashed var(--ui-color-divider);
}

.escort-checkin__distance-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.escort-checkin__distance-value {
  font-size: var(--ui-font-base);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-primary);
  margin-left: var(--ui-space-xs);
}

.escort-checkin__remark {
  margin-top: var(--ui-space-md);
  padding-top: var(--ui-space-md);
  border-top: 1px solid var(--ui-color-border);
}

.escort-checkin__remark-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-bottom: var(--ui-space-xs);
}

.escort-checkin__remark-input {
  width: 100%;
  min-height: 60px;
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
  background: var(--ui-color-bg-base);
  border: 1px solid var(--ui-color-border);
  border-radius: var(--ui-radius-sm);
  padding: var(--ui-space-sm);
  box-sizing: border-box;
}

.escort-checkin__time-row {
  margin-top: var(--ui-space-md);
  padding-top: var(--ui-space-md);
  border-top: 1px solid var(--ui-color-border);
}

.escort-checkin__time-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-success);
  font-weight: var(--ui-font-weight-semibold);
  display: block;
}

.escort-checkin__time-value {
  font-size: var(--ui-font-md);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
  display: block;
  margin-top: 4px;
}

.escort-checkin__remark-display {
  margin-top: var(--ui-space-sm);
}

.escort-checkin__remark-text {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
  display: block;
  margin-top: 2px;
}

.escort-checkin__done {
  margin-top: var(--ui-space-md);
}

.escort-checkin__actions {
  margin-top: var(--ui-space-md);
}
</style>