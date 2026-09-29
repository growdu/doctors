<!--
  src/pages/order/detail/index.vue

  订单详情页 —— 单一订单的状态机视图（spec §4.2 + plan Task P6）
  （依据 spec `2026-09-24-order-matching-redesign.md` §1.2 增量 v1.1 选人模式）

  页面流向：
    列表 / candidates 选择后跳转 → /pages/order/detail?orderId=xxx
                                    → 本页
                                    → onLoad+mounted 调 useOrderStore().loadOrder
                                    → 渲染 OrderStatusProgress + 状态卡片 + 基础信息

  模板要点：
    - 顶部 <u-navbar>「订单详情」+ 自动返回
    - 中部：OrderStatusProgress 6 节点进度条
    - 中下：状态卡片（status-specific UI）：
        selectingEscort            → 「去选陪诊师」按钮
        escortPendingAcceptance    → CountdownBadge + 「已选陪诊师 #42」
        accepted / inService       → 「陪诊师已确认」 + 陪诊师 info
        completed                  → 「已完成」 + 评价按钮
    - 拒接回退 Alert：
        escortRejectReason 非空 → 渲染文案 + 「重新选择陪诊师」链接
    - 底部：订单金额 / 订单号 / 医院 / 就诊时间

  数据来源：
    - 详情镜像自 useOrderStore().current（store 暴露的 ref）
    - 组件内维护本地 ref 是为了让 @vue/test-utils 能直接断言

  测试覆盖：src/pages/order/detail/index.test.js（6 个 it）
-->
<template>
  <view class="page-order-detail">
    <!-- 顶部导航 -->
    <u-navbar title="订单详情" :auto-back="true" />

    <!-- 加载中 -->
    <view v-if="loading && !order" class="page-order-detail__loading">
      <u-skeleton :rows="4" :title="true" />
    </view>

    <!-- 错误态 -->
    <view v-else-if="loadError" class="page-order-detail__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button
        type="primary"
        plain
        data-test="retry-btn"
        @click="onRetry"
      >重试</u-button>
    </view>

    <!-- 正常态 -->
    <template v-else-if="order">
      <!-- 进度条 -->
      <view class="page-order-detail__progress">
        <OrderStatusProgress :current-status="progressStatus" />
      </view>

      <!-- 状态卡片 -->
      <view class="page-order-detail__status-card" data-test="status-card">
        <!-- selectingEscort：去选陪诊师 -->
        <template v-if="isSelectingEscort">
          <text class="page-order-detail__status-tip">请选择陪诊师开始服务</text>
          <u-button
            type="primary"
            data-test="goto-candidates-btn"
            @click="onGoCandidates"
          >去选陪诊师</u-button>
        </template>

        <!-- escortPendingAcceptance：30s 倒计时 + 已选 escort -->
        <template v-else-if="isEscortPendingAcceptance">
          <CountdownBadge
            v-if="order.escortPendingExpireAt"
            :expire-at="order.escortPendingExpireAt"
            label="陪诊师确认剩余"
          />
          <text class="page-order-detail__status-text">已选陪诊师 #{{ order.selectedEscortId }}</text>
        </template>

        <!-- accepted / inService：陪诊师已确认 -->
        <template v-else-if="isAccepted || isInService">
          <text class="page-order-detail__status-text">陪诊师已确认</text>
          <text
            v-if="escortDisplayName"
            class="page-order-detail__status-sub"
          >{{ escortDisplayName }}</text>
        </template>

        <!-- completed：已完成 + 评价按钮 -->
        <template v-else-if="isCompleted">
          <text class="page-order-detail__status-text">已完成</text>
          <u-button
            type="warning"
            plain
            data-test="review-btn"
            @click="onReview"
          >评价</u-button>
        </template>
      </view>

      <!-- 拒接回退 Alert -->
      <view
        v-if="rejectReasonText"
        class="page-order-detail__reject-alert"
        data-test="reject-alert"
      >
        <view class="page-order-detail__reject-icon">!</view>
        <view class="page-order-detail__reject-content">
          <text class="page-order-detail__reject-title">本次选择失败</text>
          <text class="page-order-detail__reject-text">{{ rejectReasonText }}</text>
        </view>
        <u-button
          type="primary"
          size="small"
          data-test="reselect-btn"
          @click="onGoCandidates"
        >重新选择陪诊师</u-button>
      </view>

      <!-- 基础信息 -->
      <view class="page-order-detail__info">
        <view class="page-order-detail__info-row">
          <text class="page-order-detail__info-label">订单金额</text>
          <text
            class="page-order-detail__info-value"
            data-test="amount"
          >{{ formattedAmount }}</text>
        </view>
        <view class="page-order-detail__info-row">
          <text class="page-order-detail__info-label">订单号</text>
          <text class="page-order-detail__info-value">{{ order.id }}</text>
        </view>
        <view
          v-if="order.hospitalId"
          class="page-order-detail__info-row"
        >
          <text class="page-order-detail__info-label">医院</text>
          <text class="page-order-detail__info-value">{{ order.hospitalId }}</text>
        </view>
        <view
          v-if="order.appointmentAt"
          class="page-order-detail__info-row"
        >
          <text class="page-order-detail__info-label">就诊时间</text>
          <text class="page-order-detail__info-value">{{ formattedAppointmentAt }}</text>
        </view>
      </view>
    </template>
  </view>
</template>

<script>
// 订单详情页 —— Vue 3 Options API（与项目既有页面风格统一 —— EscortCandidateCard / candidates）。
//
// 设计要点：
//   - 三态机：loading / error / loaded
//     loading=true 且 order 未就绪：u-skeleton 占位
//     error：u-empty + 重试按钮
//     loaded：完整渲染
//   - state-specific UI：直接用 computed isXxx 判断当前订单状态，模板分支极简（避免嵌套 v-if 链）
//   - 拒接回退：depends on order.escortRejectReason；值映射 escort_declined / lock_expired

import { useOrderStore } from '@/stores/order.js';
import OrderStatusProgress from '@/components/OrderStatusProgress.vue';
import CountdownBadge from '@/components/CountdownBadge.vue';
import { formatMoney, formatDateTime } from '@/utils/format.js';

const REJECT_REASON_ZH = {
  escort_declined: '陪诊师主动拒接',
  lock_expired: '陪诊师超时未确认',
};

export default {
  name: 'OrderDetailPage',
  components: { OrderStatusProgress, CountdownBadge },
  data() {
    return {
      orderId: null,
      loading: false,
      loadError: false,
      // 本地镜像：便于模板 v-if / 单测断言
      order: null,
    };
  },
  computed: {
    orderStore() {
      return useOrderStore();
    },
    isSelectingEscort() {
      return this.order && this.order.status === 'selectingEscort';
    },
    isEscortPendingAcceptance() {
      return this.order && this.order.status === 'escortPendingAcceptance';
    },
    // store 内常量是 'escortConfirmed'；brief 用 'accepted'（更语义化）。本组件按 brief 语义判断。
    isAccepted() {
      return this.order && (
        this.order.status === 'accepted' ||
        this.order.status === 'escortConfirmed'
      );
    },
    isInService() {
      return this.order && this.order.status === 'inService';
    },
    isCompleted() {
      return this.order && this.order.status === 'completed';
    },
    // 给 OrderStatusProgress 用的状态字符串：escortConfirmed → accepted 映射
    progressStatus() {
      if (!this.order) return '';
      return this.order.status === 'escortConfirmed' ? 'accepted' : this.order.status;
    },
    formattedAmount() {
      if (this.order && typeof this.order.amount === 'number') {
        try {
          return formatMoney(this.order.amount);
        } catch (_e) {
          return String(this.order.amount);
        }
      }
      return '-';
    },
    formattedAppointmentAt() {
      return formatDateTime(this.order && this.order.appointmentAt);
    },
    rejectReasonText() {
      const reason = this.order && this.order.escortRejectReason;
      if (!reason) return '';
      return REJECT_REASON_ZH[reason] || reason;
    },
    // 陪诊师展示名（order.escort 可能存在；缺则用 #selectedEscortId 兜底）
    escortDisplayName() {
      const escort = this.order && this.order.escort;
      if (escort && (escort.name || escort.nickname)) {
        return escort.name || escort.nickname;
      }
      if (this.order && this.order.selectedEscortId) {
        return `陪诊师 #${this.order.selectedEscortId}`;
      }
      return '';
    },
  },
  methods: {
    /** 拉取订单详情（fetchOrder 统一进入；onMounted + onRetry 共用） */
    async fetchOrder() {
      if (!this.orderId) return;
      this.loading = true;
      this.loadError = false;
      try {
        const detail = await this.orderStore.loadOrder(this.orderId);
        this.order = detail;
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error('[order/detail] loadOrder failed', e);
        this.loadError = true;
        this.order = null;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },
    /** 重试按钮 */
    onRetry() {
      this.fetchOrder();
    },
    /** 「去选陪诊师」/「重新选择陪诊师」均跳 candidates */
    onGoCandidates() {
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({
        url: `/pages/order/candidates/index?orderId=${this.orderId}`,
      });
    },
    /** 评价按钮（brief 仅要求 emit/placeholder，本组件以 toast 兜底） */
    onReview() {
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title: '评价功能开发中', icon: 'none' });
      }
    },
  },
  // ---- uni-app / Vue 生命周期
  onLoad(query) {
    this.orderId = (query && query.orderId) || null;
  },
  mounted() {
    this.fetchOrder();
  },
  onUnload() {
    // 离开详情页 → 停止 store 端的轮询（如有），避免后台 timer 续跑
    try {
      this.orderStore.stopPolling();
    } catch (_e) {
      // store 未起轮询时静默忽略
    }
  },
};
</script>

<style lang="scss" scoped>
.page-order-detail {
  min-height: 100vh;
  padding: 16px 16px 32px 16px;
  box-sizing: border-box;
  background: #f5f7fa;
}

.page-order-detail__loading {
  margin-top: 16px;
}

.page-order-detail__error {
  margin-top: 64px;
  display: flex;
  flex-direction: column;
  align-items: center;

  .u-button {
    margin-top: 16px;
    width: 50%;
  }
}

.page-order-detail__progress {
  margin-bottom: 16px;
}

.page-order-detail__status-card {
  background: #ffffff;
  border-radius: 12px;
  padding: 20px 16px;
  margin-bottom: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
}

.page-order-detail__status-tip {
  font-size: 14px;
  color: #303133;
}

.page-order-detail__status-text {
  font-size: 16px;
  font-weight: 600;
  color: #1677ff;
}

.page-order-detail__status-sub {
  font-size: 13px;
  color: #606266;
  margin-top: -4px;
}

.page-order-detail__reject-alert {
  background: #fff7e6;
  border: 1px solid #ffd591;
  border-radius: 12px;
  padding: 12px 16px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-order-detail__reject-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #fa8c16;
  color: #ffffff;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.page-order-detail__reject-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.page-order-detail__reject-title {
  font-size: 14px;
  font-weight: 600;
  color: #d46b08;
}

.page-order-detail__reject-text {
  font-size: 13px;
  color: #874d00;
}

.page-order-detail__info {
  background: #ffffff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-order-detail__info-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}

.page-order-detail__info-label {
  font-size: 14px;
  color: #909399;
}

.page-order-detail__info-value {
  font-size: 14px;
  color: #303133;
  font-variant-numeric: tabular-nums;
}
</style>
