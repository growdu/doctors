<!--
  src/pages/order/create.vue

  订单创建页 —— 医院 / 服务包 / 时间 / 地址 / 优惠券选择（plan Task M8）
  （spec §4.1 + plan v1 重构）

  页面流向：
    入口：医院详情「立即下单」/ 首页快捷入口
       → 本页
       → onLoad 读 ?hospitalId=xxx&packageId=xxx
       → mounted 调：
           useHospitalStore().loadDetail(hospitalId) 拉医院详情（含服务包占位）
           useAddressStore().loadList() 拉地址列表
           useCouponStore().loadMine() 拉我的券
       → 渲染：医院 / 服务包 / 就诊时间 / 联系人 / 地址 / 优惠券 / 提交

  模板要点：
    - 顶部 <u-navbar>「创建订单」+ 自动返回
    - 5 个 section（uView Plus u-card）：
        1. 医院 + 服务包（下拉选择）
        2. 就诊时间（picker → 「yyyy-MM-dd HH:mm」）
        3. 联系人（姓名 / 手机号）
        4. 地址（下拉选择默认地址 + 「管理」入口）
        5. 优惠券（下拉选择我的券 + 不使用）
    - 底部「提交订单」+ 金额汇总
    - loading / error 三态机

  数据来源：
    - 医院：useHospitalStore().detail
    - 地址：useAddressStore().list + defaultId
    - 优惠券：useCouponStore().mine (status=unused)

  测试覆盖：src/pages/order/create.test.js
-->
<template>
  <view class="page-order-create" data-test="order-create-page">
    <u-navbar title="创建订单" :auto-back="true" />

    <!-- loading -->
    <view v-if="loading" class="page-order-create__loading" data-test="loading">
      <u-skeleton :rows="6" :title="true" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-order-create__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <template v-else>
      <!-- 1. 医院 + 服务包 -->
      <view class="page-order-create__section" data-test="hospital-section">
        <text class="page-order-create__section-title">就诊医院</text>
        <view class="page-order-create__section-row">
          <text class="page-order-create__section-value" data-test="hospital-name">
            {{ hospitalName }}
          </text>
        </view>
        <view class="page-order-create__section-row">
          <text class="page-order-create__section-label">服务包</text>
          <view class="page-order-create__package-list" data-test="package-list">
            <view
              v-for="pkg in packages"
              :key="pkg.id"
              class="page-order-create__package"
              :class="{ 'page-order-create__package--active': selectedPackageId === pkg.id }"
              :data-test="'package-' + pkg.id"
              :data-package-id="pkg.id"
              @click="onPackageSelect(pkg)"
            >
              <text class="page-order-create__package-name">{{ pkg.name }}</text>
              <text class="page-order-create__package-price">¥{{ pkg.price }}</text>
            </view>
            <view v-if="packages.length === 0" class="page-order-create__package-empty" data-test="package-empty">
              <text class="page-order-create__package-empty-text">暂未上线服务包，请手动选择套餐</text>
            </view>
          </view>
        </view>
      </view>

      <!-- 2. 就诊时间 -->
      <view class="page-order-create__section" data-test="time-section">
        <text class="page-order-create__section-title">就诊时间</text>
        <picker
          mode="multiSelector"
          :value="timePickerIndex"
          :range="timePickerRange"
          data-test="time-picker"
          @change="onTimePickerChange"
        >
          <view class="page-order-create__picker">
            <text data-test="time-text">{{ formattedAppointmentAt || '请选择就诊时间' }}</text>
          </view>
        </picker>
      </view>

      <!-- 3. 联系人 -->
      <view class="page-order-create__section" data-test="contact-section">
        <text class="page-order-create__section-title">联系人</text>
        <view class="page-order-create__field">
          <text class="page-order-create__field-label">姓名</text>
          <input
            v-model="contactName"
            class="page-order-create__field-input"
            data-test="input-name"
            placeholder="请输入姓名"
          />
        </view>
        <view class="page-order-create__field">
          <text class="page-order-create__field-label">手机号</text>
          <input
            v-model="contactPhone"
            class="page-order-create__field-input"
            data-test="input-phone"
            placeholder="请输入手机号"
          />
        </view>
      </view>

      <!-- 4. 地址 -->
      <view class="page-order-create__section" data-test="address-section">
        <view class="page-order-create__section-row">
          <text class="page-order-create__section-title">就诊地址</text>
          <text class="page-order-create__section-link" data-test="manage-address" @click="onManageAddress">管理</text>
        </view>
        <view v-if="addresses.length === 0" class="page-order-create__empty" data-test="address-empty">
          <text>暂无地址，请先</text>
          <text class="page-order-create__empty-link" @click="onManageAddress">添加地址</text>
        </view>
        <view v-else class="page-order-create__address-list" data-test="address-list">
          <view
            v-for="a in addresses"
            :key="a.id"
            class="page-order-create__address"
            :class="{ 'page-order-create__address--active': selectedAddressId === a.id }"
            :data-test="'address-' + a.id"
            :data-address-id="a.id"
            @click="onAddressSelect(a)"
          >
            <text class="page-order-create__address-name">{{ a.recipient }} {{ a.phone }}</text>
            <text class="page-order-create__address-detail">{{ a.detail }}</text>
            <text
              v-if="a.id === defaultAddressId"
              class="page-order-create__address-default"
            >默认</text>
          </view>
        </view>
      </view>

      <!-- 5. 优惠券 -->
      <view class="page-order-create__section" data-test="coupon-section">
        <text class="page-order-create__section-title">优惠券</text>
        <view class="page-order-create__coupon-list" data-test="coupon-list">
          <view
            class="page-order-create__coupon"
            :class="{ 'page-order-create__coupon--active': selectedUserCouponId === null }"
            data-test="coupon-none"
            @click="onCouponSelect(null)"
          >
            <text>不使用优惠券</text>
          </view>
          <view
            v-for="uc in usableCoupons"
            :key="uc.id"
            class="page-order-create__coupon"
            :class="{ 'page-order-create__coupon--active': selectedUserCouponId === uc.id }"
            :data-test="'coupon-' + uc.id"
            @click="onCouponSelect(uc)"
          >
            <text class="page-order-create__coupon-name">{{ uc.coupon && uc.coupon.name || '-' }}</text>
            <text class="page-order-create__coupon-value">
              <template v-if="uc.coupon && uc.coupon.type === 'amount'">减 ¥{{ uc.coupon.value }}</template>
              <template v-else-if="uc.coupon">{{ uc.coupon.value }}</template>
            </text>
          </view>
          <view v-if="usableCoupons.length === 0" class="page-order-create__empty" data-test="coupon-empty">
            <text>暂无可用优惠券</text>
          </view>
        </view>
      </view>

      <!-- 底部 -->
      <view class="page-order-create__bottom">
        <view class="page-order-create__amount">
          <text class="page-order-create__amount-label">合计</text>
          <text class="page-order-create__amount-value" data-test="total-amount">¥{{ totalAmount.toFixed(2) }}</text>
          <text
            v-if="discountAmount > 0"
            class="page-order-create__amount-discount"
            data-test="discount-amount"
          >已优惠 ¥{{ discountAmount.toFixed(2) }}</text>
        </view>
        <u-button
          type="primary"
          :disabled="!canSubmit"
          data-test="submit-btn"
          @click="onSubmit"
        >{{ submitting ? '提交中' : '提交订单' }}</u-button>
      </view>
    </template>
  </view>
</template>

<script>
// 订单创建页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / home）。
//
// 设计要点：
//   - 三态机：loading / error / loaded
//   - 数据来源：
//       hospital  + service packages  → useHospitalStore().loadDetail(hospitalId)
//       addresses                  → useAddressStore().loadList()
//       coupons                    → useCouponStore().loadMine()
//   - 时间选择器：v1 用占位（date picker 不便 mock 单测），提交时直接读 appointmentAt
//   - 提交：v1 仅做 store.add（无后端 createOrder API；后续 plan 接 order-service 的 POST /orders）

import { useHospitalStore } from '@/stores/hospital.js';
import { useAddressStore } from '@/stores/address.js';
import { useCouponStore } from '@/stores/coupon.js';

// 时间 picker 占位（v1 不接 picker 多列；用户文本填 appointmentAt）
const TIME_PICKER_RANGE = [
  ['今天', '明天', '后天', '大后天'],
  ['09:00', '10:00', '11:00', '14:00', '15:00', '16:00'],
];

export default {
  name: 'OrderCreatePage',
  data() {
    return {
      hospitalId: null,
      packageId: null,
      appointmentAt: '',
      contactName: '',
      contactPhone: '',
      selectedPackageId: null,
      selectedAddressId: null,
      selectedUserCouponId: null,
      hospital: null,
      packages: [],
      addresses: [],
      defaultAddressId: null,
      usableCoupons: [],
      loading: false,
      loadError: false,
      submitting: false,
      TIME_PICKER_RANGE,
      timePickerIndex: [0, 0],
    };
  },
  computed: {
    hospitalStore() {
      return useHospitalStore();
    },
    addressStore() {
      return useAddressStore();
    },
    couponStore() {
      return useCouponStore();
    },
    hospitalName() {
      return (this.hospital && this.hospital.name) || '请选择医院';
    },
    formattedAppointmentAt() {
      return this.appointmentAt || '';
    },
    totalAmount() {
      const pkg = this.packages.find((p) => p.id === this.selectedPackageId);
      const base = pkg ? Number(pkg.price) : 0;
      return Math.max(0, base - this.discountAmount);
    },
    discountAmount() {
      const uc = this.usableCoupons.find((u) => u.id === this.selectedUserCouponId);
      if (!uc || !uc.coupon) return 0;
      const base = this.packages.find((p) => p.id === this.selectedPackageId);
      const price = base ? Number(base.price) : 0;
      if (uc.coupon.type === 'amount' && price >= (uc.coupon.threshold || 0)) {
        return Math.min(price, Number(uc.coupon.value));
      }
      return 0;
    },
    canSubmit() {
      return Boolean(
        this.hospitalId &&
        this.selectedPackageId &&
        this.appointmentAt &&
        this.contactName &&
        this.contactPhone &&
        this.selectedAddressId &&
        !this.submitting,
      );
    },
  },
  methods: {
    async fetchAll() {
      this.loading = true;
      this.loadError = false;
      try {
        // 拉医院详情（含服务包占位）
        if (this.hospitalId) {
          const h = await this.hospitalStore.loadDetail(this.hospitalId);
          this.hospital = h || null;
          this.packages = (h && h.packages) || [];
        }
        // 拉地址
        const addrs = await this.addressStore.loadList();
        this.addresses = Array.isArray(addrs) ? addrs : [];
        this.defaultAddressId = this.addressStore.defaultId;
        this.selectedAddressId = this.defaultAddressId
          || (this.addresses[0] && this.addresses[0].id)
          || null;
        // 拉我的券（仅 unused）
        const ucList = await this.couponStore.loadMine();
        const list = Array.isArray(ucList) ? ucList : [];
        this.usableCoupons = list.filter((uc) => uc && uc.status === 'unused');

        // 应用 onLoad 传入的 packageId
        if (this.packageId) {
          this.selectedPackageId = this.packageId;
        } else if (this.packages.length > 0) {
          this.selectedPackageId = this.packages[0].id;
        }
      } catch (_e) {
        this.loadError = true;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    onRetry() {
      this.fetchAll();
    },

    onPackageSelect(pkg) {
      if (!pkg || !pkg.id) return;
      this.selectedPackageId = pkg.id;
    },

    onAddressSelect(a) {
      if (!a || !a.id) return;
      this.selectedAddressId = a.id;
    },

    onCouponSelect(uc) {
      this.selectedUserCouponId = uc ? uc.id : null;
    },

    /** 时间 picker 多列（占位：v1 仅写字符串，不实际解析多列） */
    onTimePickerChange(e) {
      const val = e && e.detail && e.detail.value;
      if (Array.isArray(val) && val.length === 2) {
        const dayLabel = TIME_PICKER_RANGE[0][val[0]] || '';
        const hourLabel = TIME_PICKER_RANGE[1][val[1]] || '';
        this.appointmentAt = `${dayLabel} ${hourLabel}`;
      }
    },

    /** 跳地址管理 */
    onManageAddress() {
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: '/pages/address/list' });
    },

    /** 提交订单（v1 占位：调 store + toast；后续接 POST /api/v1/orders） */
    async onSubmit() {
      if (!this.canSubmit) return;
      this.submitting = true;
      try {
        // v1 占位：成功后 toast + 跳订单列表
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '订单已提交', icon: 'success' });
        }
        // 提交成功后 → 我的订单 tab（spec §4.1 流程）
        if (typeof uni !== 'undefined' && typeof uni.redirectTo === 'function') {
          uni.redirectTo({ url: '/pages/order/index' });
        }
      } catch (_e) {
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '提交失败', icon: 'none' });
        }
      } finally {
        this.submitting = false;
      }
    },
  },
  onLoad(query) {
    if (query) {
      if (query.hospitalId) this.hospitalId = query.hospitalId;
      if (query.packageId) this.packageId = query.packageId;
    }
  },
  mounted() {
    if (this.hospitalId) {
      this.fetchAll();
    } else {
      this.loadError = true;
    }
  },
};
</script>

<style lang="scss" scoped>
.page-order-create {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 96px;
}

.page-order-create__loading,
.page-order-create__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-order-create__error .u-button { margin-top: 16px; width: 50%; }

.page-order-create__section {
  background: #fff;
  margin: 12px;
  border-radius: 12px;
  padding: 14px 16px;
}

.page-order-create__section-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.page-order-create__section-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  display: block;
  margin-bottom: 12px;
}

.page-order-create__section-label {
  font-size: 13px;
  color: #909399;
  margin-right: 12px;
}

.page-order-create__section-value {
  font-size: 14px;
  color: #303133;
  flex: 1;
}

.page-order-create__section-link {
  font-size: 13px;
  color: #1989fa;
}

.page-order-create__package-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 4px;
}

.page-order-create__package {
  background: #f4f4f5;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid transparent;
  display: flex;
  flex-direction: column;
  min-width: 120px;
}

.page-order-create__package--active {
  background: #e8f3ff;
  border-color: #1989fa;
}

.page-order-create__package-name {
  font-size: 13px;
  color: #303133;
}

.page-order-create__package-price {
  font-size: 14px;
  font-weight: 600;
  color: #ff4d4f;
  margin-top: 4px;
}

.page-order-create__package-empty {
  padding: 6px 0;
}

.page-order-create__package-empty-text {
  font-size: 13px;
  color: #909399;
}

.page-order-create__picker {
  padding: 8px 0;
}

.page-order-create__field {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.page-order-create__field-label {
  font-size: 14px;
  color: #909399;
  width: 70px;
  flex-shrink: 0;
}

.page-order-create__field-input {
  flex: 1;
  font-size: 14px;
  color: #303133;
}

.page-order-create__empty {
  padding: 12px 0;
  font-size: 13px;
  color: #909399;
  display: flex;
  align-items: center;
}

.page-order-create__empty-link {
  color: #1989fa;
  margin-left: 4px;
}

.page-order-create__address-list,
.page-order-create__coupon-list {
  display: flex;
  flex-direction: column;
}

.page-order-create__address {
  padding: 12px 0;
  border-bottom: 1px dashed #f0f0f0;
  display: flex;
  flex-direction: column;
  position: relative;
}

.page-order-create__address:last-child {
  border-bottom: none;
}

.page-order-create__address--active {
  background: #f8fbff;
}

.page-order-create__address-name {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
  margin-bottom: 4px;
}

.page-order-create__address-detail {
  font-size: 13px;
  color: #606266;
}

.page-order-create__address-default {
  position: absolute;
  top: 12px;
  right: 0;
  font-size: 11px;
  background: #1989fa;
  color: #fff;
  padding: 2px 6px;
  border-radius: 4px;
}

.page-order-create__coupon {
  padding: 10px 12px;
  border-radius: 8px;
  background: #f4f4f5;
  margin-bottom: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid transparent;
}

.page-order-create__coupon--active {
  background: #e8f3ff;
  border-color: #1989fa;
}

.page-order-create__coupon-name {
  font-size: 14px;
  color: #303133;
}

.page-order-create__coupon-value {
  font-size: 13px;
  color: #ff4d4f;
  font-weight: 600;
}

.page-order-create__bottom {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: #fff;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-order-create__amount {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.page-order-create__amount-label {
  font-size: 12px;
  color: #909399;
}

.page-order-create__amount-value {
  font-size: 18px;
  font-weight: 600;
  color: #ff4d4f;
  font-variant-numeric: tabular-nums;
}

.page-order-create__amount-discount {
  font-size: 11px;
  color: #19be6b;
  margin-top: 2px;
}

.page-order-create__bottom .u-button {
  width: 40%;
  flex-shrink: 0;
}
</style>