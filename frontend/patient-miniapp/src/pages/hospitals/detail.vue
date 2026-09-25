<!--
  src/pages/hospitals/detail.vue

  医院详情页 —— 单医院视图 + 服务包（plan Task M4）
  （spec §4.3 + plan v1 重构）

  页面流向：
    入口：医院列表卡片点击 → /pages/hospitals/detail?id=xxx
                       → 本页
                       → onLoad+mounted 调 useHospitalStore().loadDetail(id)
                       → 渲染 hero + 基础信息 + 科室标签 + 套餐占位 + 「立即下单」

  模板要点：
    - 顶部 <u-navbar>「医院详情」+ 自动返回
    - hero：医院名 + 等级 + 地址
    - 科室 chips（来自 hospital.departments）
    - 套餐卡片（占位：来自医院数据的 packages 字段，若无则空）
    - 「立即下单」按钮 → /pages/order/create?hospitalId=xxx

  数据来源：
    - 详情镜像自 useHospitalStore().detail（store 暴露的 ref）

  测试覆盖：src/pages/hospitals/detail.test.js
-->
<template>
  <view class="page-hospital-detail" data-test="hospital-detail-page">
    <u-navbar title="医院详情" :auto-back="true" />

    <!-- loading -->
    <view v-if="loading && !hospital" class="page-hospital-detail__loading" data-test="loading">
      <u-skeleton :rows="5" :title="true" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-hospital-detail__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- empty -->
    <view v-else-if="!hospital" class="page-hospital-detail__empty" data-test="empty-state">
      <u-empty text="医院不存在" mode="data" />
    </view>

    <!-- loaded -->
    <template v-else>
      <!-- hero -->
      <view class="page-hospital-detail__hero" data-test="hero">
        <text class="page-hospital-detail__hero-name">{{ hospital.name }}</text>
        <text
          v-if="hospital.level"
          class="page-hospital-detail__hero-level"
          data-test="level"
        >{{ hospital.level }}</text>
        <text
          v-if="hospital.address"
          class="page-hospital-detail__hero-addr"
        >📍 {{ hospital.address }}</text>
      </view>

      <!-- 基础信息 -->
      <view class="page-hospital-detail__info" data-test="info-card">
        <view v-if="hospital.phone" class="page-hospital-detail__info-row">
          <text class="page-hospital-detail__info-label">联系电话</text>
          <text class="page-hospital-detail__info-value">{{ hospital.phone }}</text>
        </view>
        <view v-if="hospital.description" class="page-hospital-detail__info-row">
          <text class="page-hospital-detail__info-label">简介</text>
          <text class="page-hospital-detail__info-value">{{ hospital.description }}</text>
        </view>
        <view
          v-if="hospital.departments && hospital.departments.length"
          class="page-hospital-detail__info-row"
          data-test="departments-row"
        >
          <text class="page-hospital-detail__info-label">科室</text>
          <view class="page-hospital-detail__chips">
            <text
              v-for="d in hospital.departments"
              :key="d"
              class="page-hospital-detail__chip"
            >{{ d }}</text>
          </view>
        </view>
      </view>

      <!-- 服务包（占位） -->
      <view class="page-hospital-detail__packages" data-test="packages">
        <text class="page-hospital-detail__packages-title">推荐服务包</text>
        <view v-if="packages.length === 0" class="page-hospital-detail__packages-empty" data-test="packages-empty">
          <u-empty text="暂未上线服务包" mode="list" />
        </view>
        <view
          v-for="pkg in packages"
          v-else
          :key="pkg.id"
          class="page-hospital-detail__package-card"
          :data-test="'package-' + pkg.id"
          @click="onPackageClick(pkg)"
        >
          <view class="page-hospital-detail__package-row">
            <text class="page-hospital-detail__package-name">{{ pkg.name }}</text>
            <text class="page-hospital-detail__package-price">¥{{ pkg.price }}</text>
          </view>
          <text v-if="pkg.description" class="page-hospital-detail__package-desc">{{ pkg.description }}</text>
        </view>
      </view>

      <!-- 底部「立即下单」 -->
      <view class="page-hospital-detail__bottom">
        <u-button
          type="primary"
          data-test="create-order-btn"
          @click="onCreateOrder"
        >立即下单</u-button>
      </view>
    </template>
  </view>
</template>

<script>
// 医院详情页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/detail）。
//
// 设计要点：
//   - 三态机：loading / error / loaded + empty / loaded + data
//   - 服务包：当前 v1 hospital.detail 不含 packages（后端 hospital module 仅基础字段），
//     此处模板按「packages 为空」兜底渲染；后续 plan 接 services/package 模块后扩
//   - onLoad+mounted 两段式生命周期：读 ?id=xxx → mounted 立即拉详情

import { useHospitalStore } from '@/stores/hospital.js';

export default {
  name: 'HospitalDetailPage',
  data() {
    return {
      hospitalId: null,
      hospital: null,
      loading: false,
      loadError: false,
      packages: [],
    };
  },
  computed: {
    hospitalStore() {
      return useHospitalStore();
    },
  },
  methods: {
    async fetchDetail() {
      this.loading = true;
      this.loadError = false;
      try {
        const r = await this.hospitalStore.loadDetail(this.hospitalId);
        this.hospital = r || null;
        // 服务包：当前 v1 后端未返；保 packages 为空数组（模板按空态渲染）
        this.packages = (r && r.packages) || [];
      } catch (_e) {
        this.loadError = true;
        this.hospital = null;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    onRetry() {
      if (this.hospitalId) this.fetchDetail();
    },

    /** 服务包卡片 → 跳订单创建页（带 hospitalId + packageId） */
    onPackageClick(pkg) {
      if (!pkg || !pkg.id) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({
        url: `/pages/order/create?hospitalId=${this.hospitalId}&packageId=${pkg.id}`,
      });
    },

    /** 「立即下单」 → 跳订单创建页（带 hospitalId） */
    onCreateOrder() {
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({
        url: `/pages/order/create?hospitalId=${this.hospitalId}`,
      });
    },
  },
  onLoad(query) {
    if (query && query.id) {
      this.hospitalId = query.id;
    }
  },
  mounted() {
    if (this.hospitalId) {
      this.fetchDetail();
    } else {
      this.loadError = true;
    }
  },
};
</script>

<style lang="scss" scoped>
.page-hospital-detail {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 88px;
}

.page-hospital-detail__loading,
.page-hospital-detail__empty,
.page-hospital-detail__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-hospital-detail__error .u-button { margin-top: 16px; width: 50%; }

.page-hospital-detail__hero {
  background: linear-gradient(135deg, #1989fa 0%, #4eb7ff 100%);
  color: #fff;
  padding: 24px 20px;
  display: flex;
  flex-direction: column;
}

.page-hospital-detail__hero-name {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 6px;
}

.page-hospital-detail__hero-level {
  font-size: 12px;
  background: rgba(255, 255, 255, 0.25);
  color: #fff;
  padding: 2px 8px;
  border-radius: 4px;
  align-self: flex-start;
  margin-bottom: 8px;
}

.page-hospital-detail__hero-addr {
  font-size: 13px;
  opacity: 0.9;
}

.page-hospital-detail__info {
  background: #fff;
  margin: 12px;
  border-radius: 12px;
  padding: 16px;
}

.page-hospital-detail__info-row {
  display: flex;
  padding: 8px 0;
  align-items: flex-start;
}

.page-hospital-detail__info-label {
  font-size: 13px;
  color: #909399;
  width: 70px;
  flex-shrink: 0;
}

.page-hospital-detail__info-value {
  font-size: 14px;
  color: #303133;
  flex: 1;
}

.page-hospital-detail__chips {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.page-hospital-detail__chip {
  font-size: 12px;
  background: #f4f4f5;
  color: #606266;
  padding: 4px 10px;
  border-radius: 4px;
}

.page-hospital-detail__packages {
  margin: 12px;
}

.page-hospital-detail__packages-title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  padding: 4px 0 12px;
}

.page-hospital-detail__packages-empty {
  background: #fff;
  border-radius: 12px;
  padding: 16px;
}

.page-hospital-detail__package-card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-hospital-detail__package-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.page-hospital-detail__package-name {
  font-size: 15px;
  font-weight: 500;
  color: #303133;
}

.page-hospital-detail__package-price {
  font-size: 16px;
  font-weight: 600;
  color: #ff4d4f;
}

.page-hospital-detail__package-desc {
  font-size: 13px;
  color: #909399;
  display: block;
}

.page-hospital-detail__bottom {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: #fff;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  z-index: 10;
  .u-button { width: 100%; }
}
</style>