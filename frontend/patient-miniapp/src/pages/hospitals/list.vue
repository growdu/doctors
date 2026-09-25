<!--
  src/pages/hospitals/list.vue

  医院列表页 —— 患者端入口（plan Task M4）
  （spec §4.3 + plan v1 重构）

  页面流向：
    入口：首页「查看更多」/ 个人中心 / 快捷入口
       → 本页
       → onLoad+mounted 调 useHospitalStore().loadList({ keyword, level })
       → 渲染医院卡片列表（u-card 风格）
       → 点击卡片 → /pages/hospitals/detail?id=xxx

  模板要点：
    - 顶部 <u-navbar>「选择医院」+ 自动返回
    - 搜索框（uView Plus u-search 占位）
    - 等级 tab（一行滚动 chips）
    - 主体 ListView 循环医院卡片
    - 空状态：u-empty「暂无可用医院」

  数据来源：
    - 列表镜像自 useHospitalStore().list（store 暴露的 ref）

  测试覆盖：src/pages/hospitals/list.test.js
-->
<template>
  <view class="page-hospitals-list" data-test="hospitals-list-page">
    <u-navbar title="选择医院" :auto-back="true" />

    <!-- 搜索框 -->
    <view class="page-hospitals-list__search" data-test="search-bar">
      <u-search
        v-model="keyword"
        placeholder="搜索医院名称"
        :show-action="false"
        @search="onSearch"
        @clear="onClear"
      />
    </view>

    <!-- 等级筛选 chips -->
    <scroll-view
      class="page-hospitals-list__filters"
      data-test="level-filters"
      scroll-x
    >
      <view
        v-for="lv in LEVEL_FILTERS"
        :key="lv.key"
        class="page-hospitals-list__chip"
        :class="{ 'page-hospitals-list__chip--active': currentLevel === lv.key }"
        :data-test="'level-chip-' + lv.key"
        @click="onLevelChange(lv.key)"
      >{{ lv.label }}</view>
    </scroll-view>

    <!-- loading -->
    <view v-if="loading && hospitals.length === 0" class="page-hospitals-list__loading" data-test="loading">
      <u-skeleton :rows="3" :title="true" avatar avatar-shape="square" avatar-size="48" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-hospitals-list__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- empty -->
    <view v-else-if="hospitals.length === 0" class="page-hospitals-list__empty" data-test="empty-state">
      <u-empty text="暂无可用医院" mode="list" />
    </view>

    <!-- list -->
    <view v-else class="page-hospitals-list__list">
      <view
        v-for="h in hospitals"
        :key="h.id"
        class="page-hospitals-list__card"
        :data-test="'hospital-card-' + h.id"
        :data-hospital-id="h.id"
        @click="onHospitalClick(h)"
      >
        <view class="page-hospitals-list__card-row">
          <text class="page-hospitals-list__card-name">{{ h.name }}</text>
          <text
            v-if="h.level"
            class="page-hospitals-list__card-level"
          >{{ h.level }}</text>
        </view>
        <text v-if="h.address" class="page-hospitals-list__card-addr">{{ h.address }}</text>
        <view v-if="h.phone" class="page-hospitals-list__card-phone">
          <text class="page-hospitals-list__card-phone-label">电话</text>
          <text class="page-hospitals-list__card-phone-value">{{ h.phone }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
// 医院列表页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / home）。
//
// 设计要点：
//   - 三态机：loading / error / loaded + empty / loaded + list
//   - 搜索 + 等级 tab 切换 → 重置 _loaded + 重新拉
//   - onLoad+mounted 两段式生命周期

import { useHospitalStore } from '@/stores/hospital.js';

// 等级筛选（与后端 hospital.level 1:1；key='' 表示全部）
const LEVEL_FILTERS = [
  { key: '',         label: '全部' },
  { key: '三甲',     label: '三甲' },
  { key: '三乙',     label: '三乙' },
  { key: '二甲',     label: '二甲' },
];

export default {
  name: 'HospitalsListPage',
  data() {
    return {
      LEVEL_FILTERS,
      keyword: '',
      currentLevel: '',
      hospitals: [],
      loading: false,
      loadError: false,
      _loaded: false,
    };
  },
  computed: {
    hospitalStore() {
      return useHospitalStore();
    },
  },
  methods: {
    /**
     * 拉医院列表（fetchList 是页面实际拉数据的入口）。
     * @param {object} [extra] 覆盖 keyword / level
     */
    async fetchList(extra) {
      this.loading = true;
      this.loadError = false;
      try {
        const query = Object.assign({
          page: 1,
          limit: 20,
          keyword: this.keyword || '',
          level: this.currentLevel || '',
        }, extra || {});
        const r = await this.hospitalStore.loadList(query);
        const items = (r && r.items) || [];
        this.hospitals = items;
        this._loaded = true;
      } catch (_e) {
        this.loadError = true;
        this.hospitals = [];
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    /** 搜索框 search → 立即拉 */
    onSearch() {
      this._loaded = false;
      this.fetchList();
    },

    /** 搜索框 clear → 清 keyword 后拉 */
    onClear() {
      this.keyword = '';
      this._loaded = false;
      this.fetchList();
    },

    /** 等级 chip 切换 */
    onLevelChange(key) {
      this.currentLevel = key;
      this._loaded = false;
      this.fetchList();
    },

    onRetry() {
      this.fetchList();
    },

    /** 卡片点击 → 跳详情 */
    onHospitalClick(h) {
      if (!h || !h.id) return;
      if (typeof uni === 'undefined' || typeof uni.navigateTo !== 'function') return;
      uni.navigateTo({ url: `/pages/hospitals/detail?id=${h.id}` });
    },
  },
  onLoad(query) {
    // 预留：可能从外部传入 ?keyword=xxx / ?level=xxx（admin-web 跳转 / DashboardPage）
    if (query && query.keyword) this.keyword = query.keyword;
    if (query && query.level) this.currentLevel = query.level;
  },
  mounted() {
    if (!this._loaded) {
      this.fetchList();
    }
  },
};
</script>

<style lang="scss" scoped>
.page-hospitals-list {
  min-height: 100vh;
  background: #f5f7fa;
}

.page-hospitals-list__search {
  background: #fff;
  padding: 12px 16px 4px;
  border-bottom: 1px solid #f0f0f0;
}

.page-hospitals-list__filters {
  background: #fff;
  white-space: nowrap;
  padding: 8px 12px 12px;
  border-bottom: 1px solid #f0f0f0;
}

.page-hospitals-list__chip {
  display: inline-block;
  padding: 6px 14px;
  margin-right: 8px;
  border-radius: 16px;
  background: #f4f4f5;
  color: #606266;
  font-size: 13px;
}

.page-hospitals-list__chip--active {
  background: #e8f3ff;
  color: #1989fa;
  font-weight: 500;
}

.page-hospitals-list__loading,
.page-hospitals-list__empty,
.page-hospitals-list__error {
  padding: 16px;
}
.page-hospitals-list__error {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 32px;
  .u-button { margin-top: 16px; width: 50%; }
}

.page-hospitals-list__list {
  padding: 12px 16px;
}

.page-hospitals-list__card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-hospitals-list__card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.page-hospitals-list__card-name {
  font-size: 15px;
  font-weight: 500;
  color: #303133;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.page-hospitals-list__card-level {
  font-size: 12px;
  color: #1989fa;
  background: #e8f3ff;
  padding: 2px 8px;
  border-radius: 4px;
  flex-shrink: 0;
  margin-left: 8px;
}

.page-hospitals-list__card-addr {
  display: block;
  font-size: 13px;
  color: #909399;
  margin-bottom: 4px;
}

.page-hospitals-list__card-phone {
  display: flex;
  align-items: center;
  padding-top: 4px;
  border-top: 1px dashed #f0f0f0;
}

.page-hospitals-list__card-phone-label {
  font-size: 12px;
  color: #909399;
  margin-right: 8px;
}

.page-hospitals-list__card-phone-value {
  font-size: 13px;
  color: #1989fa;
}
</style>