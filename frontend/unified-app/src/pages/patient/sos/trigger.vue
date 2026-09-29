<script setup lang="ts">
/**
 * patient/sos/trigger.vue — SOS 一键紧急联系（v2 unified-app）。
 *
 * 入口：patient 域底部 Tab SOS 按钮 / 服务中订单「紧急联系」入口
 *   → 必填：原因
 *   → 可选：关联订单 ID + 当前位置
 *   → 调 api/sos.raiseSos({ order_id?, reason, location? })
 *
 * 设计要点：
 *   - 危险视觉：红色头部 + 大按钮
 *   - 30s 防重复触发（前端兜底；后端有频率去重）
 */
import { ref, onMounted } from 'vue';
import { raiseSos } from '@/api/sos';
import UiInput from '@/components/shared/UiInput.vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';

const orderId = ref<number | null>(null);
const reason = ref('');
const locationText = ref('');
const submitting = ref(false);
const successId = ref<number | null>(null);
const error = ref<string | null>(null);

const lastTriggerTime = ref(0);
const COOLDOWN_MS = 30_000;

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.orderId) orderId.value = Number(opts.orderId);
}

function onPickLocation() {
  // uni-app h5 / mp 平台调用 wx.chooseLocation / uni.chooseLocation
  if (typeof uni === 'undefined') return;
  const chooseLocation = (uni as unknown as { chooseLocation?: (opts: object) => void }).chooseLocation;
  if (typeof chooseLocation === 'function') {
    chooseLocation({
      success: (res: { address?: string; latitude?: number; longitude?: number }) => {
        locationText.value = res.address || '';
      },
      fail: () => {
        // 用户取消选点
      },
    });
  }
}

async function onSubmit() {
  if (!reason.value.trim()) {
    error.value = '请描述紧急情况';
    return;
  }
  const now = Date.now();
  if (now - lastTriggerTime.value < COOLDOWN_MS) {
    error.value = `30 秒内不可重复触发，请稍候 ${Math.ceil((COOLDOWN_MS - (now - lastTriggerTime.value)) / 1000)} 秒`;
    return;
  }
  submitting.value = true;
  error.value = null;
  try {
    const r = await raiseSos({
      order_id: orderId.value || undefined,
      reason: reason.value.trim(),
      location: locationText.value
        ? { lat: 0, lng: 0, address: locationText.value }
        : undefined,
    });
    successId.value = r.id;
    lastTriggerTime.value = now;
    reason.value = '';
    locationText.value = '';
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    submitting.value = false;
  }
}

function onDismissSuccess() {
  successId.value = null;
  if (typeof uni !== 'undefined') uni.navigateBack({ delta: 1 });
}

onMounted(parseQuery);
</script>

<template>
  <view class="ui-page" data-testid="patient-sos-trigger">
    <view v-if="successId" class="sos-success" data-testid="sos-success">
      <UiEmpty icon="🚨" title="已发起 SOS" :description="`工单 #${successId} 已提交，客服会尽快联系您`">
        <template #action>
          <UiButton type="primary" size="sm" data-testid="sos-success-back-btn" @click="onDismissSuccess">返回</UiButton>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <!-- 危险视觉头部 -->
      <view class="sos-trigger__warning" data-testid="sos-warning-banner">
        <text class="sos-trigger__warning-icon">🚨</text>
        <text class="sos-trigger__warning-text">
          请只在真正紧急时使用。频繁误报可能影响您的信用。
        </text>
      </view>

      <UiCard title="紧急联系">
        <view v-if="orderId" class="sos-trigger__field">
          <text class="sos-trigger__label">关联订单</text>
          <text class="sos-trigger__value" data-testid="sos-order-id">订单 #{{ orderId }}</text>
        </view>

        <UiInput
          v-model="reason"
          label="情况描述"
          placeholder="请简要描述紧急情况（如：陪诊师失联、患者身体不适）"
          type="textarea"
          :maxlength="200"
          data-testid="sos-reason"
        />

        <view class="sos-trigger__field">
          <text class="sos-trigger__label">位置（可选）</text>
          <view class="sos-trigger__location">
            <text class="sos-trigger__location-text">{{ locationText || '未选择' }}</text>
            <UiButton type="default" size="sm" data-testid="sos-pick-location-btn" @click="onPickLocation">
              {{ locationText ? '重新选择' : '选择位置' }}
            </UiButton>
          </view>
        </view>

        <view v-if="error" class="sos-trigger__error" data-testid="sos-error">{{ error }}</view>
      </UiCard>

      <view class="sos-trigger__actions">
        <UiButton
          type="danger"
          block
          :loading="submitting"
          :disabled="!reason.trim()"
          data-testid="sos-submit-btn"
          @click="onSubmit"
        >
          🚨 立即发起 SOS
        </UiButton>
      </view>
    </template>
  </view>
</template>

<style scoped>
.sos-trigger__warning {
  display: flex;
  align-items: center;
  gap: var(--ui-space-sm);
  background: var(--ui-color-error);
  color: var(--ui-color-text-inverse);
  padding: var(--ui-space-md);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
}

.sos-trigger__warning-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.sos-trigger__warning-text {
  font-size: var(--ui-font-sm);
  flex: 1;
  ;
}

.sos-trigger__field {
  padding: var(--ui-space-md) 0;
  border-bottom: 1px solid var(--ui-color-divider);
}

.sos-trigger__field:last-of-type {
  border-bottom: none;
}

.sos-trigger__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-bottom: var(--ui-space-xs);
}

.sos-trigger__value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
}

.sos-trigger__location {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
}

.sos-trigger__location-text {
  flex: 1;
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
}

.sos-trigger__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-top: var(--ui-space-md);
}

.sos-trigger__actions {
  margin-top: var(--ui-space-base);
}
</style>