<!--
  src/pages/coupons/index.vue

  优惠券中心 —— 领券 / 用券 / 我的券 tab（plan Task M6）
  （spec §4.5 + plan v1 重构）

  页面流向：
    入口：首页「优惠券」快捷入口 / 个人中心 menu
       → 本页
       → onLoad+mounted 调 useCouponStore().loadTemplates({}) + loadMine()
       → 渲染 tab（领券 / 我的券）
         - 「领券」tab：可领取券模板列表 + 「立即领取」按钮
         - 「我的券」tab：我的券列表 + status filter（全部 / 未使用 / 已使用 / 已过期）
       → 点击「立即领取」→ claim(id) → 切到「我的券」tab + toast

  模板要点：
    - 顶部 <u-navbar>「优惠券」+ 自动返回
    - 2 个 tab（领券 / 我的券）
    - 我的券 tab 内嵌套 4 个 status 子 tab
    - 券卡片：左色块（金额）+ 右侧模板名/门槛/状态
    - loading / error / empty / loaded 四态机

  数据来源：
    - templates：useCouponStore().templates
    - mine：useCouponStore().mine

  测试覆盖：src/pages/coupons/index.test.js
-->
<template>
  <view class="page-coupons" data-test="coupons-page">
    <u-navbar title="优惠券" :auto-back="true" />

    <!-- 顶部 tab -->
    <view class="page-coupons__tabs" data-test="top-tabs">
      <view
        v-for="t in TOP_TABS"
        :key="t.key"
        class="page-coupons__tab"
        :class="{ 'page-coupons__tab--active': currentTab === t.key }"
        :data-test="'top-tab-' + t.key"
        @click="onTabChange(t.key)"
      >{{ t.label }}</view>
    </view>

    <!-- ============= 领券 tab ============= -->
    <template v-if="currentTab === 'claim'">
      <!-- loading -->
      <view v-if="loading && templates.length === 0" class="page-coupons__loading" data-test="loading">
        <u-skeleton :rows="3" :title="true" />
      </view>

      <!-- error -->
      <view v-else-if="loadError" class="page-coupons__error" data-test="error-state">
        <u-empty text="加载失败" mode="data" />
        <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
      </view>

      <!-- empty -->
      <view v-else-if="templates.length === 0" class="page-coupons__empty" data-test="empty-state">
        <u-empty text="暂无可领取的券" mode="list" />
      </view>

      <!-- list -->
      <view v-else class="page-coupons__list" data-test="template-list">
        <view
          v-for="c in templates"
          :key="c.id"
          class="page-coupons__card"
          :data-test="'coupon-card-' + c.id"
        >
          <view class="page-coupons__card-left">
            <text class="page-coupons__card-amount" v-if="c.type === 'amount'">
              <text class="page-coupons__card-amount-symbol">¥</text>{{ c.value }}
            </text>
            <text class="page-coupons__card-amount" v-else>{{ c.value }}</text>
            <text class="page-coupons__card-amount-unit" v-if="c.threshold">满{{ c.threshold }}可用</text>
          </view>
          <view class="page-coupons__card-right">
            <text class="page-coupons__card-name">{{ c.name }}</text>
            <text class="page-coupons__card-period">有效期至 {{ formatDate(c.valid_until) }}</text>
            <u-button
              type="primary"
              size="small"
              :data-test="'claim-btn-' + c.id"
              :disabled="claiming === c.id"
              @click="onClaim(c)"
            >{{ claiming === c.id ? '领取中' : '立即领取' }}</u-button>
          </view>
        </view>
      </view>
    </template>

    <!-- ============= 我的券 tab ============= -->
    <template v-else-if="currentTab === 'mine'">
      <!-- 子 status tab -->
      <view class="page-coupons__subtabs" data-test="status-tabs">
        <view
          v-for="st in STATUS_TABS"
          :key="st.key"
          class="page-coupons__subtab"
          :class="{ 'page-coupons__subtab--active': currentStatus === st.key }"
          :data-test="'status-tab-' + st.key"
          @click="onStatusTabChange(st.key)"
        >{{ st.label }}</view>
      </view>

      <view v-if="loadingMine && mine.length === 0" class="page-coupons__loading" data-test="loading-mine">
        <u-skeleton :rows="3" :title="true" />
      </view>

      <view v-else-if="filteredMine.length === 0" class="page-coupons__empty" data-test="empty-mine">
        <u-empty text="暂无该状态的券" mode="list" />
      </view>

      <view v-else class="page-coupons__list" data-test="mine-list">
        <view
          v-for="uc in filteredMine"
          :key="uc.id"
          class="page-coupons__mine-card"
          :class="{ 'page-coupons__mine-card--disabled': uc.status !== 'unused' }"
          :data-test="'mine-card-' + uc.id"
          :data-status="uc.status"
        >
          <view class="page-coupons__card-left">
            <text class="page-coupons__card-amount" v-if="uc.coupon && uc.coupon.type === 'amount'">
              <text class="page-coupons__card-amount-symbol">¥</text>{{ uc.coupon.value }}
            </text>
            <text class="page-coupons__card-amount" v-else-if="uc.coupon">{{ uc.coupon.value }}</text>
            <text class="page-coupons__card-amount-unit" v-if="uc.coupon && uc.coupon.threshold">满{{ uc.coupon.threshold }}可用</text>
          </view>
          <view class="page-coupons__card-right">
            <text class="page-coupons__card-name">{{ uc.coupon && uc.coupon.name || '-' }}</text>
            <text class="page-coupons__card-period">有效期至 {{ formatDate(uc.expires_at) }}</text>
            <text
              class="page-coupons__mine-status"
              :data-test="'mine-status-' + uc.status"
            >{{ statusLabel(uc.status) }}</text>
          </view>
        </view>
      </view>
    </template>
  </view>
</template>

<script>
// 优惠券中心页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / home）。
//
// 设计要点：
//   - 三态机：loading / error / loaded + empty / loaded + list
//   - 2 个主 tab（claim / mine）：切换 tab 仅切显隐，不重拉（已拉过数据缓存到 store）
//   - 我的券内嵌套 4 个 status 子 tab（all / unused / used / expired）
//   - 领取：调 claim(id) → 切到「我的券」tab + toast 成功
//   - onLoad+mounted 两段式生命周期

import { useCouponStore, COUPON_STATUS_UNUSED, COUPON_STATUS_USED, COUPON_STATUS_EXPIRED } from '@/stores/coupon.js';
import { formatDateTime } from '@/utils/format.js';

// 顶部 tab
const TOP_TABS = [
  { key: 'claim', label: '领券' },
  { key: 'mine',  label: '我的券' },
];

// 我的券 status 子 tab
const STATUS_TABS = [
  { key: 'all',     label: '全部' },
  { key: COUPON_STATUS_UNUSED,  label: '未使用' },
  { key: COUPON_STATUS_USED,    label: '已使用' },
  { key: COUPON_STATUS_EXPIRED, label: '已过期' },
];

// status → 友好文案
const STATUS_LABEL = {
  [COUPON_STATUS_UNUSED]: '未使用',
  [COUPON_STATUS_USED]: '已使用',
  [COUPON_STATUS_EXPIRED]: '已过期',
};

export default {
  name: 'CouponsPage',
  data() {
    return {
      TOP_TABS,
      STATUS_TABS,
      currentTab: 'claim',
      currentStatus: 'all',
      templates: [],
      mine: [],
      loading: false,
      loadingMine: false,
      loadError: false,
      claiming: null, // 当前正在领取的 coupon.id
    };
  },
  computed: {
    couponStore() {
      return useCouponStore();
    },
    filteredMine() {
      if (this.currentStatus === 'all') return this.mine;
      return this.mine.filter((uc) => uc && uc.status === this.currentStatus);
    },
  },
  methods: {
    async fetchTemplates() {
      this.loading = true;
      this.loadError = false;
      try {
        const r = await this.couponStore.loadTemplates({});
        this.templates = Array.isArray(r) ? r : [];
      } catch (_e) {
        this.loadError = true;
        this.templates = [];
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    async fetchMine() {
      this.loadingMine = true;
      try {
        const r = await this.couponStore.loadMine();
        this.mine = Array.isArray(r) ? r : [];
      } catch (_e) {
        this.mine = [];
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loadingMine = false;
      }
    },

    onRetry() {
      this.fetchTemplates();
    },

    onTabChange(key) {
      this.currentTab = key;
    },

    onStatusTabChange(key) {
      this.currentStatus = key;
    },

    /**
     * 领取券 → 切到「我的券」tab + toast
     */
    async onClaim(c) {
      if (!c || !c.id) return;
      this.claiming = c.id;
      try {
        await this.couponStore.claim(c.id);
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '领取成功', icon: 'success' });
        }
        // 切到「我的券」tab 让用户看到
        this.currentTab = 'mine';
        // 重新拉一次以同步
        await this.fetchMine();
      } catch (_e) {
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '领取失败，请重试', icon: 'none' });
        }
      } finally {
        this.claiming = null;
      }
    },

    formatDate(input) {
      return formatDateTime(input);
    },

    statusLabel(status) {
      return STATUS_LABEL[status] || status || '-';
    },
  },
  onLoad() {
    // 占位
  },
  mounted() {
    this.fetchTemplates();
    this.fetchMine();
  },
};
</script>

<style lang="scss" scoped>
.page-coupons {
  min-height: 100vh;
  background: #f5f7fa;
}

.page-coupons__tabs {
  background: #fff;
  display: flex;
  border-bottom: 1px solid #f0f0f0;
}

.page-coupons__tab {
  flex: 1;
  text-align: center;
  padding: 14px 0;
  font-size: 14px;
  color: #606266;
  position: relative;
}

.page-coupons__tab--active {
  color: #1989fa;
  font-weight: 600;
  &::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 50%;
    transform: translateX(-50%);
    width: 28px;
    height: 3px;
    background: #1989fa;
    border-radius: 2px;
  }
}

.page-coupons__subtabs {
  background: #fff;
  display: flex;
  padding: 8px 12px;
  border-bottom: 1px solid #f0f0f0;
}

.page-coupons__subtab {
  padding: 6px 14px;
  margin-right: 8px;
  border-radius: 16px;
  background: #f4f4f5;
  color: #606266;
  font-size: 13px;
}

.page-coupons__subtab--active {
  background: #e8f3ff;
  color: #1989fa;
  font-weight: 500;
}

.page-coupons__loading,
.page-coupons__empty,
.page-coupons__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-coupons__error .u-button { margin-top: 16px; width: 50%; }

.page-coupons__list {
  padding: 12px 16px;
}

.page-coupons__card,
.page-coupons__mine-card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-coupons__mine-card--disabled {
  opacity: 0.55;
}

.page-coupons__card-left {
  width: 110px;
  border-right: 1px dashed #f0f0f0;
  padding-right: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
}

.page-coupons__card-amount {
  font-size: 28px;
  font-weight: 600;
  color: #ff4d4f;
  font-variant-numeric: tabular-nums;
}

.page-coupons__card-amount-symbol {
  font-size: 16px;
  margin-right: 2px;
}

.page-coupons__card-amount-unit {
  font-size: 11px;
  color: #909399;
  margin-top: 4px;
}

.page-coupons__card-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding-left: 12px;
}

.page-coupons__card-name {
  font-size: 15px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 6px;
}

.page-coupons__card-period {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
}

.page-coupons__mine-status {
  font-size: 12px;
  font-weight: 500;
}

.page-coupons__mine-status {
  &[data-status="unused"] {
    color: #1989fa;
  }
  &[data-status="used"] {
    color: #909399;
  }
  &[data-status="expired"] {
    color: #ff4d4f;
  }
}
</style>