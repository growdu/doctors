<!--
  src/pages/package/detail.vue

  服务包详情页 —— 医院 + 描述 + 价格 + 立即下单
  （spec §4.3 + plan v1）

  页面流向：
    入口：医院详情「服务包」点击 / 订单创建服务包选择
       → 本页（?hospitalId=xxx&packageId=xxx）
       → mounted 调 useHospitalStore().loadDetail(hospitalId)
       → 渲染：医院名 / 服务包名 / 价格 / 描述
       → 「立即下单」→ uni.navigateTo(/pages/order/create?hospitalId=xxx&packageId=xxx)

  模板要点：
    - 顶部 <u-navbar>「服务包详情」+ 自动返回
    - 顶部 Banner：医院名（chip 风格）
    - 服务包主信息卡：服务包名 + 价格 + 描述
    - 底部固定「立即下单」按钮

  行为：
    - onLoad(query)：从 query 读 hospitalId + packageId
    - loadHospital()：调 store.loadDetail
    - 「立即下单」→ navigateTo 订单创建页（带 hospitalId + packageId）

  数据来源：
    - hospital store：useHospitalStore().detail（含 packages 数组）
    - 注：服务包信息挂在医院详情的 packages 字段上（v1 后端字段约定）

  测试覆盖：src/pages/package/detail.test.js
-->
<template>
  <view class="page-package-detail" data-test="package-detail-page">
    <u-navbar title="服务包详情" :auto-back="true" />

    <!-- loading -->
    <view v-if="loading" class="page-package-detail__loading" data-test="loading">
      <u-skeleton :rows="4" :title="true" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-package-detail__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="loadHospital">重试</u-button>
    </view>

    <!-- 主内容 -->
    <template v-else-if="hospital">
      <view class="page-package-detail__banner" data-test="hospital-banner">
        <text class="page-package-detail__hospital-name">{{ hospital.name }}</text>
        <text class="page-package-detail__hospital-level">{{ hospital.level || '' }}</text>
      </view>

      <view v-if="pkg" class="page-package-detail__card" data-test="package-card">
        <text class="page-package-detail__pkg-name" data-test="package-name">{{ pkg.name }}</text>
        <text class="page-package-detail__pkg-price" data-test="package-price">¥{{ pkg.price }}</text>
        <view class="page-package-detail__pkg-desc" data-test="package-desc">
          <text class="page-package-detail__pkg-desc-text">{{ pkg.description || '暂无服务说明' }}</text>
        </view>
        <view v-if="pkg.durationHours" class="page-package-detail__pkg-meta" data-test="package-meta">
          <text class="page-package-detail__pkg-meta-text">服务时长：{{ pkg.durationHours }} 小时</text>
        </view>
      </view>

      <view v-else class="page-package-detail__missing" data-test="missing-package">
        <u-empty text="服务包不存在" mode="data" />
      </view>
    </template>

    <!-- 底部按钮 -->
    <view v-if="hospital && pkg" class="page-package-detail__footer">
      <u-button
        type="primary"
        size="large"
        data-test="order-btn"
        @click="onOrder"
      >立即下单</u-button>
    </view>
  </view>
</template>

<script>
// package/detail 页 —— Vue 3 Options API。
//
// 关键设计：
//   - onLoad 放在 methods 内（uni-app 支持 + 测试可通过 vm.onLoad 直接调用）
//   - 服务包来自 hospital.detail.packages（v1 后端字段约定）
//   - 「立即下单」带 hospitalId + packageId 跳订单创建页

import { useHospitalStore } from '@/stores/hospital.js';

export default {
  name: 'PackageDetailPage',
  data() {
    return {
      hospitalId: null,
      packageId: null,
      loading: false,
      loadError: false,
      // 单次加载去重
      _loaded: false,
    };
  },
  computed: {
    hospitalStore() {
      return useHospitalStore();
    },
    hospital() {
      return this.hospitalStore.detail;
    },
    packages() {
      const d = this.hospital;
      if (!d) return [];
      return Array.isArray(d.packages) ? d.packages : [];
    },
    pkg() {
      const list = this.packages;
      if (!list.length) return null;
      // 优先按 packageId 匹配；否则取第一个（仅一个服务包时）
      if (this.packageId != null) {
        const found = list.find((p) => String(p.id) === String(this.packageId));
        if (found) return found;
      }
      return list[0];
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     * @param {object} query
     */
    onLoad(query) {
      this.hospitalId = (query && (query.hospitalId || query.hospital_id)) || null;
      this.packageId = (query && (query.packageId || query.package_id)) || null;
    },

    /**
     * 拉医院详情（含服务包）。
     */
    async loadHospital() {
      if (!this.hospitalId) {
        this.loadError = true;
        return;
      }
      this.loading = true;
      this.loadError = false;
      try {
        await this.hospitalStore.loadDetail(this.hospitalId);
        this._loaded = true;
      } catch (_e) {
        this.loadError = true;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    /**
     * 「立即下单」点击 → 订单创建页（带 hospitalId + packageId）
     */
    onOrder() {
      if (!this.hospital || !this.pkg) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      const url = `/pages/order/create?hospitalId=${this.hospitalId}&packageId=${this.pkg.id}`;
      uni.navigateTo({ url });
    },

    /**
     * Vue mounted 钩子（在 methods 内以便测试通过 vm 调用）。
     * async —— 让 await w.vm.mounted() 等到 loadHospital 完成。
     */
    async mounted() {
      if (!this.hospitalId) {
        // 缺医院 id → 直接置 error（不需要调 store）
        this.loadError = true;
        return;
      }
      if (!this._loaded) {
        await this.loadHospital();
      }
    },

    /**
     * Vue unmounted 钩子（页面卸载时清空详情避免脏读）。
     */
    unmounted() {
      try { this.hospitalStore.clearDetail(); } catch (_e) {}
    },
  },
};
</script>

<style lang="scss" scoped>
.page-package-detail {
  min-height: 100vh;
  padding: 16px;
  padding-bottom: 96px; // 留底部按钮空间
  box-sizing: border-box;
  background: #f5f5f5;
}

.page-package-detail__loading {
  margin-top: 32px;
}

.page-package-detail__error {
  margin-top: 32px;
}

.page-package-detail__banner {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #ffffff;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 12px;
}

.page-package-detail__hospital-name {
  font-size: 18px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-package-detail__hospital-level {
  font-size: 12px;
  color: #1677ff;
  background: #e6f4ff;
  padding: 2px 8px;
  border-radius: 4px;
}

.page-package-detail__card {
  background: #ffffff;
  padding: 24px 16px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-package-detail__pkg-name {
  font-size: 20px;
  font-weight: 600;
  color: #1f1f1f;
}

.page-package-detail__pkg-price {
  font-size: 28px;
  font-weight: 700;
  color: #ff4d4f;
}

.page-package-detail__pkg-desc {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
}

.page-package-detail__pkg-desc-text {
  font-size: 14px;
  line-height: 1.6;
  color: #595959;
}

.page-package-detail__pkg-meta-text {
  font-size: 13px;
  color: #8c8c8c;
}

.page-package-detail__missing {
  margin-top: 64px;
}

.page-package-detail__footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 12px 16px;
  background: #ffffff;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
  z-index: 10;
}
</style>