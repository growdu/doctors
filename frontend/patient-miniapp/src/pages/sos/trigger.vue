<!--
  src/pages/sos/trigger.vue

  SOS 紧急呼救页 —— 长按 1.5s 触发按钮 + 位置上报（mock）+ 10s 倒计时确认
  （spec §4.5 + plan Task M6）

  页面流向：
    入口：订单详情「SOS」/ profile「紧急呼救」
       → 本页（?orderId=xxx 可选）
       → mounted：mock 拿位置（v1 固定北京协和医院经纬度）
       → 渲染：大圆按钮（红色）「长按 1.5 秒呼救」+ 进度环
       → 长按中：进度环从 0 → 100%；1.5s 完成 → 调 api.triggerSos
       → 触发后：进入「已触发」态（10s 倒计时确认）
         - 10s 内可点「我按错了」→ 调 api.cancelSos
         - 10s 后自动跳详情 / toast「已通知陪诊师与客服」
       → 失败 → toast 错误

  模板要点：
    - 顶部 <u-navbar>「紧急呼救」+ 自动返回
    - 大红圆按钮（200x200）+ 「长按 1.5 秒呼救」文案
    - 进度环：u-circle-progress（百分比）
    - 状态切换：idle → triggering → triggered（10s 倒计时）→ done
    - 「我按错了」按钮（triggered 状态 10s 内显示）

  行为：
    - onLoad(query)：从 query 读 orderId
    - mounted()：拿 mock 位置 + 重置状态
    - onPressStart / onPressEnd / onPressCancel：长按手势
    - onConfirm()：调 api.triggerSos
    - onCancel()：调 api.cancelSos

  数据来源：
    - api/sos.js 直接调

  测试覆盖：src/pages/sos/trigger.test.js
-->
<template>
  <view class="page-sos-trigger" data-test="sos-trigger-page">
    <u-navbar title="紧急呼救" :auto-back="true" />

    <!-- idle 状态：长按按钮 -->
    <view v-if="state === 'idle' || state === 'triggering'" class="page-sos-trigger__idle" data-test="idle-state">
      <text class="page-sos-trigger__title">长按 1.5 秒紧急呼救</text>
      <text class="page-sos-trigger__subtitle">长按过程可松开取消</text>

      <!-- v1.1 增量 —— 原因 chips（多选 0 个；默认无） -->
      <view class="page-sos-trigger__reasons" data-test="reasons-card">
        <text class="page-sos-trigger__reasons-title">紧急原因（可选）</text>
        <view class="page-sos-trigger__reasons-list">
          <view
            v-for="r in SOS_REASONS"
            :key="r.key"
            class="page-sos-trigger__reason"
            :class="{ 'page-sos-trigger__reason--active': selectedReasonKeys.includes(r.key) }"
            :data-test="'reason-' + r.key"
            :data-selected="selectedReasonKeys.includes(r.key)"
            @click="onReasonToggle(r.key)"
          >{{ r.label }}</view>
        </view>
      </view>

      <!-- v1.1 增量 —— 当前订单 + 位置展示 -->
      <view class="page-sos-trigger__context" data-test="context-card">
        <view v-if="orderId" class="page-sos-trigger__context-row">
          <text class="page-sos-trigger__context-label">关联订单</text>
          <text class="page-sos-trigger__context-value" data-test="order-context">#{{ orderId }}</text>
        </view>
        <view class="page-sos-trigger__context-row">
          <text class="page-sos-trigger__context-label">当前位置</text>
          <text class="page-sos-trigger__context-value" data-test="location-context">{{ location.address }}</text>
        </view>
      </view>

      <view class="page-sos-trigger__btn-wrap">
        <view
          class="page-sos-trigger__btn"
          :class="{ 'page-sos-trigger__btn--pressing': state === 'triggering' }"
          data-test="sos-btn"
          @touchstart="onPressStart"
          @touchend="onPressEnd"
          @touchcancel="onPressCancel"
          @mousedown="onPressStart"
          @mouseup="onPressEnd"
          @mouseleave="onPressCancel"
        >
          <text class="page-sos-trigger__btn-text">SOS</text>
        </view>
      </view>
      <text class="page-sos-trigger__progress-text" data-test="progress-text">
        {{ progressText }}
      </text>
    </view>

    <!-- triggered 状态：10s 倒计时确认 -->
    <view v-else-if="state === 'triggered'" class="page-sos-trigger__triggered" data-test="triggered-state">
      <text class="page-sos-trigger__alert-title">已通知陪诊师与客服</text>
      <text class="page-sos-trigger__alert-subtitle" data-test="countdown-text">
        {{ countdownSec }} 秒后可关闭
      </text>
      <u-button
        type="warning"
        plain
        size="large"
        :loading="cancelling"
        data-test="cancel-sos-btn"
        @click="onCancelSos"
      >我按错了（取消）</u-button>
    </view>

    <!-- done 状态 -->
    <view v-else class="page-sos-trigger__done" data-test="done-state">
      <text class="page-sos-trigger__done-title">已发送</text>
      <text class="page-sos-trigger__done-subtitle">陪诊师/客服会尽快联系您</text>
      <u-button type="primary" size="large" data-test="back-btn" @click="onBack">返回</u-button>
    </view>
  </view>
</template>

<script>
// sos/trigger 页 —— Vue 3 Options API。
//
// 关键设计：
//   - 长按检测：setInterval(50ms) tick 累加 progress；达到 1.5s → 触发
//   - 松开 / 离开 → clearInterval + 重置 progress（未触发）
//   - 触发后 10s 倒计时（cancel 窗口）；倒计时归零 → state=done
//   - mock 位置：v1 固定北京协和医院坐标；后续 plan 接入 uni.getLocation

import { triggerSos, cancelSos } from '@/api/sos.js';

const PRESS_DURATION_MS = 1500;
const PROGRESS_TICK_MS = 50;
const CANCEL_WINDOW_SEC = 10;
const MOCK_LAT = 39.9129;
const MOCK_LNG = 116.4148;

// v1.1: SOS 原因 chips（多选）
const SOS_REASONS = [
  { key: 'lost',        label: '找不到路' },
  { key: 'wait_long',   label: '等太久了' },
  { key: 'medical',     label: '突发不适' },
  { key: 'other',       label: '其他' },
];

export default {
  name: 'SosTriggerPage',
  data() {
    return {
      orderId: null,
      state: 'idle',         // idle / triggering / triggered / done
      progress: 0,           // 0-100
      _pressTimer: null,
      _countdownTimer: null,
      countdownSec: CANCEL_WINDOW_SEC,
      sosEventId: null,
      cancelling: false,
      // v1.1: 原因 chips 多选
      SOS_REASONS,
      selectedReasonKeys: [],
      location: {
        lat: MOCK_LAT,
        lng: MOCK_LNG,
        address: '北京市东城区帅府园 1 号（mock）',
      },
    };
  },
  computed: {
    progressText() {
      return `${Math.round(this.progress)}%`;
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     */
    onLoad(query) {
      this.orderId = (query && (query.orderId || query.order_id)) || null;
    },

    /**
     * Vue mounted 钩子（methods 内）。
     */
    mounted() {
      // v1 mock 位置：固定北京协和医院经纬度
      this.location = {
        lat: MOCK_LAT,
        lng: MOCK_LNG,
        address: '北京市东城区帅府园 1 号（mock）',
      };
    },

    /**
     * Vue beforeUnmount 钩子（methods 内）—— 清理 timers。
     */
    beforeUnmount() {
      this._clearPressTimer();
      this._clearCountdownTimer();
    },

    /**
     * 长按开始
     */
    onPressStart() {
      if (this.state !== 'idle') return;
      this.state = 'triggering';
      this.progress = 0;
      this._clearPressTimer();
      this._pressTimer = setInterval(() => {
        this.progress = Math.min(100, this.progress + (PROGRESS_TICK_MS / PRESS_DURATION_MS) * 100);
        if (this.progress >= 100) {
          this._clearPressTimer();
          this._onConfirm();
        }
      }, PROGRESS_TICK_MS);
    },

    /**
     * 长按结束 / 离开
     */
    onPressEnd() {
      if (this.state !== 'triggering') return;
      // 未达到 100% → 取消
      if (this.progress < 100) {
        this._clearPressTimer();
        this.progress = 0;
        this.state = 'idle';
      }
    },

    /**
     * 长按取消（touchcancel / mouseleave）
     */
    onPressCancel() {
      this.onPressEnd();
    },

    /**
     * 长按达到 1.5s → 调 api.triggerSos。
     * v1.1: 把 selectedReasonKeys 一并送上。
     */
    async _onConfirm() {
      try {
        const reasonText = (this.selectedReasonKeys || [])
          .map((k) => {
            const found = SOS_REASONS.find((r) => r.key === k);
            return found ? found.label : '';
          })
          .filter(Boolean)
          .join(',');
        const evt = await triggerSos({
          order_id: this.orderId,
          lat: this.location.lat,
          lng: this.location.lng,
          address: this.location.address,
          reason_keys: this.selectedReasonKeys || [],
          reason: reasonText,
        });
        this.sosEventId = evt && evt.id;
        this.state = 'triggered';
        this.countdownSec = CANCEL_WINDOW_SEC;
        this._startCountdown();
      } catch (e) {
        // 触发失败：回退 idle
        this.state = 'idle';
        this.progress = 0;
        const mod = await import('@/utils/format.js').catch(() => ({}));
        const msg = (e && e.message) || '触发失败，请重试';
        this._toast(msg);
      }
    },

    /** v1.1: 原因 toggle */
    onReasonToggle(key) {
      const idx = this.selectedReasonKeys.indexOf(key);
      if (idx >= 0) {
        this.selectedReasonKeys.splice(idx, 1);
      } else {
        this.selectedReasonKeys.push(key);
      }
    },

    /**
     * 10s 倒计时（cancel 窗口）
     */
    _startCountdown() {
      this._clearCountdownTimer();
      this._countdownTimer = setInterval(() => {
        this.countdownSec -= 1;
        if (this.countdownSec <= 0) {
          this._clearCountdownTimer();
          this.state = 'done';
        }
      }, 1000);
    },

    /**
     * 「我按错了」点击（cancel window 内有效）
     */
    async onCancelSos() {
      if (this.state !== 'triggered') return;
      this.cancelling = true;
      try {
        if (this.sosEventId) {
          await cancelSos(this.sosEventId, 'user_cancelled');
        }
        this._clearCountdownTimer();
        this.state = 'idle';
        this._toast('已取消');
      } catch (e) {
        this._toast((e && e.message) || '取消失败，请稍后再试');
      } finally {
        this.cancelling = false;
      }
    },

    /**
     * 「返回」点击（done 状态）
     */
    onBack() {
      if (typeof uni !== 'undefined' && typeof uni.navigateBack === 'function') {
        uni.navigateBack({ delta: 1 });
      }
    },

    _clearPressTimer() {
      if (this._pressTimer) {
        clearInterval(this._pressTimer);
        this._pressTimer = null;
      }
    },

    _clearCountdownTimer() {
      if (this._countdownTimer) {
        clearInterval(this._countdownTimer);
        this._countdownTimer = null;
      }
    },

    _toast(title) {
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title, icon: 'none' });
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.page-sos-trigger {
  min-height: 100vh;
  padding: 24px 16px;
  background: #f5f5f5;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.page-sos-trigger__idle,
.page-sos-trigger__triggered,
.page-sos-trigger__done {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  width: 100%;
}

.page-sos-trigger__title {
  font-size: 22px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-sos-trigger__subtitle {
  font-size: 13px;
  color: #8c8c8c;
}

.page-sos-trigger__reasons,
.page-sos-trigger__context {
  width: 100%;
  max-width: 480px;
  background: #ffffff;
  border-radius: 12px;
  padding: 12px 16px;
  box-sizing: border-box;
}

.page-sos-trigger__reasons-title {
  display: block;
  font-size: 13px;
  color: #606266;
  margin-bottom: 8px;
}

.page-sos-trigger__reasons-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.page-sos-trigger__reason {
  padding: 5px 12px;
  border-radius: 16px;
  background: #f4f4f5;
  color: #606266;
  font-size: 12px;
}

.page-sos-trigger__reason--active {
  background: #fff1f0;
  color: #ff4d4f;
  font-weight: 500;
}

.page-sos-trigger__context-row {
  display: flex;
  align-items: center;
  padding: 4px 0;
}

.page-sos-trigger__context-label {
  width: 80px;
  font-size: 12px;
  color: #909399;
  flex-shrink: 0;
}

.page-sos-trigger__context-value {
  font-size: 13px;
  color: #303133;
  flex: 1;
}

.page-sos-trigger__btn-wrap {
  margin-top: 32px;
  margin-bottom: 32px;
}

.page-sos-trigger__btn {
  width: 200px;
  height: 200px;
  border-radius: 50%;
  background: #ff4d4f;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  user-select: none;
  transition: transform 0.1s ease;
}

.page-sos-trigger__btn--pressing {
  transform: scale(0.95);
  background: #d4380d;
}

.page-sos-trigger__btn-text {
  font-size: 48px;
  font-weight: 700;
}

.page-sos-trigger__progress-text {
  font-size: 14px;
  color: #8c8c8c;
  font-variant-numeric: tabular-nums;
}

.page-sos-trigger__alert-title {
  font-size: 22px;
  font-weight: 600;
  color: #ff4d4f;
}

.page-sos-trigger__alert-subtitle {
  font-size: 14px;
  color: #595959;
}

.page-sos-trigger__done-title {
  font-size: 24px;
  font-weight: 700;
  color: #1f1f1f;
}

.page-sos-trigger__done-subtitle {
  font-size: 14px;
  color: #595959;
  margin-bottom: 16px;
}
</style>