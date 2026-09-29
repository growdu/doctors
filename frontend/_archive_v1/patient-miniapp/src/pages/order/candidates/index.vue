<!--
  src/pages/order/candidates/index.vue

  candidates 页 —— 患者从候选陪诊师列表中选定 1 位陪诊师
  （spec `2026-09-24-order-matching-redesign.md` §1.2 + plan Task P5）

  页面流向：
    详情页（selectingEscort 状态） → uni.navigateTo(/pages/order/candidates?orderId=xxx)
                                  → 本页
                                  → confirm dialog → selectEscortBy(orderId, escortId)
                                  → uni.redirectTo(/pages/order/detail?orderId=xxx)

  模板要点：
    - 顶部 <u-navbar>「选择陪诊师」+ 自动返回
    - 主体 ListView 循环 EscortCandidateCard
    - 空状态：u-empty（暂无候选陪诊师）
    - loading 状态：u-skeleton 占位
    - 底部固定按钮「刷新列表」：再次调 loadCandidates（brief 要求）

  行为：
    - onLoad(query)：从 query 读 orderId
    - onMounted：orderId 存在时立即调 loadCandidates(orderId)（brief 要求）
    - 点击 EscortCandidateCard → confirm dialog → 调 selectEscortBy → 跳详情页
    - 选择失败 / 加载失败：uni.showToast「选择失败，请重试」/「加载失败，请重试」

  测试覆盖：src/pages/order/candidates/index.test.js（5 个 it）
-->
<template>
  <view class="page-candidates">
    <!-- 顶部导航 -->
    <u-navbar title="选择陪诊师" :auto-back="true" />

    <!-- 主体：loading 占位 -->
    <view v-if="loading && candidates.length === 0" class="page-candidates__skeleton">
      <u-skeleton
        :rows="3"
        :title="true"
        avatar
        avatar-shape="circle"
        avatar-size="40"
      />
      <u-skeleton
        :rows="3"
        :title="true"
        avatar
        avatar-shape="circle"
        avatar-size="40"
      />
    </view>

    <!-- 主体：空状态 -->
    <view v-else-if="candidates.length === 0" class="page-candidates__empty">
      <u-empty text="暂无候选陪诊师" mode="list" />
    </view>

    <!-- 主体：候选列表 -->
    <view v-else class="page-candidates__list">
      <EscortCandidateCard
        v-for="c in candidates"
        :key="c.escortId"
        :candidate="c"
        :loading="selectingEscortId === c.escortId"
        :disabled="selectingEscortId !== null"
        @select="onConfirmSelect"
      />
    </view>

    <!-- 底部固定：刷新列表 -->
    <view class="page-candidates__footer">
      <u-button
        type="primary"
        plain
        :loading="loading"
        data-test="refresh-btn"
        @click="onRefresh"
      >刷新列表</u-button>
    </view>
  </view>
</template>

<script>
// candidates 页 —— Vue 3 Options API（与项目既有页面风格统一）。
//
// 设计要点：
//   - 用 Options API 而非 <script setup>：uni-app 的 onLoad/onMounted 等生命周期钩子
//     在 Options API 中作为组件 method 直接声明，uni-app 编译器会自动识别（App.vue 同款）。
//   - 单次加载保护 `_loaded`：onLoad + onMounted 都有可能触发初次加载，用标志位去重。
//   - onRefresh / onConfirmSelect / doSelect 三个方法分别承担「重拉」「确认」「提交」职责，
//     便于测试单独验证。
//
// 数据来源：
//   - candidates 数组镜像自 useOrderStore().candidates（store 暴露的 ref）；
//     组件内维护本地 ref 是为了让 @vue/test-utils 能直接断言 DOM 渲染。

import { useOrderStore } from '@/stores/order.js';
import EscortCandidateCard from '@/components/EscortCandidateCard.vue';

export default {
  name: 'CandidatesPage',
  components: { EscortCandidateCard },
  data() {
    return {
      orderId: null,
      candidates: [],
      loading: false,
      selectingEscortId: null,
      // 单次加载去重：onLoad 与 onMounted 都会触发初次 fetch，加标志位避免重复。
      _loaded: false,
    };
  },
  computed: {
    orderStore() {
      return useOrderStore();
    },
  },
  methods: {
    /**
     * 拉候选陪诊师列表（fetchCandidates 是页面实际拉数据的入口）。
     * 与 onLoad/onMounted/onRefresh 三处共用。
     */
    async fetchCandidates() {
      if (!this.orderId) return;
      this.loading = true;
      try {
        const r = await this.orderStore.loadCandidates(this.orderId);
        this.candidates = (r && r.items) || [];
        this._loaded = true;
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error('[candidates] loadCandidates failed', e);
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    /** 刷新按钮：重置 _loaded 后重拉（强制刷新） */
    async onRefresh() {
      this._loaded = false;
      await this.fetchCandidates();
    },

    /**
     * EscortCandidateCard 点击回调 —— 弹 confirm dialog → 调 selectEscortBy → 跳详情。
     * @param {number|string} escortId
     */
    onConfirmSelect(escortId) {
      if (typeof uni === 'undefined' || typeof uni.showModal !== 'function') {
        // 测试 / SSR 兜底：直接执行提交
        this.doSelect(escortId);
        return;
      }
      uni.showModal({
        title: '确认选择',
        content: '选定后陪诊师需在 30 秒内确认，是否继续？',
        success: async (res) => {
          if (!res.confirm) return;
          await this.doSelect(escortId);
        },
      });
    },

    /**
     * 实际选定 escort：调 store.selectEscortBy → 跳详情页。
     * 失败 → toast 提示，并清 selectingEscortId 恢复可点击。
     * @param {number|string} escortId
     */
    async doSelect(escortId) {
      this.selectingEscortId = escortId;
      try {
        await this.orderStore.selectEscortBy(this.orderId, escortId);
        if (typeof uni !== 'undefined' && typeof uni.redirectTo === 'function') {
          uni.redirectTo({ url: `/pages/order/detail?orderId=${this.orderId}` });
        }
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error('[candidates] selectEscortBy failed', e);
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '选择失败，请重试', icon: 'none' });
        }
        this.selectingEscortId = null;
      }
    },
  },
  // ---- uni-app / Vue 生命周期
  // onLoad：uni-app Page 钩子，从 URL query 读 orderId
  // mounted：Vue 标准 Options API 钩子，组件挂载完成时触发（brief「onMounted」语义）
  // onShow：uni-app Page 钩子，从详情页返回本页面时刷新
  onLoad(query) {
    // 从 URL query 读 orderId（必填）
    this.orderId = query && query.orderId;
  },
  mounted() {
    // 页面挂载完成、orderId 已就绪 → 立即拉一次（brief「onMounted」语义）
    if (this.orderId && !this._loaded) {
      this.fetchCandidates();
    }
  },
  onShow() {
    // 从详情页返回本页面（如 escort 拒接重新选择）→ 自动刷新列表
    // 注意：首次进入由 mounted 处理；onShow 仅在「已加载过」时刷新（_loaded 仍为 true）
    if (this.orderId && this._loaded) {
      this.fetchCandidates();
    }
  },
};
</script>

<style lang="scss" scoped>
.page-candidates {
  min-height: 100vh;
  padding: 16px;
  padding-bottom: 96px; // 留出底部按钮空间
  box-sizing: border-box;
}

.page-candidates__list {
  margin-top: 8px;
}

.page-candidates__skeleton {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-candidates__empty {
  padding-top: 64px;
}

.page-candidates__footer {
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