// src/stores/hospital.js
//
// 医院 store —— 患者端的医院列表 / 详情 + loading/error 状态机
// (spec §4.3 + plan Task M2)
//
// 范围（本文件）：
//   - list：医院列表（响应式）
//   - detail：当前医院详情（响应式）
//   - loading / error：动作执行态（响应式；UI 渲染三态机用）
//   - pagination：分页元数据（total / page / limit）
//   - lastQuery：上一次查询条件（重试 / 切 tab 用）
//
// 不在本文件范围：
//   - 服务包拉取（detail.vue 直接调 useHospitalStore().loadDetail(id) 拿详情里的 departments / 套餐占位）
//   - 地址 / 收藏（plan 后续）

import { defineStore } from 'pinia';
import { ref } from 'vue';

async function _apiGetHospitals(query) {
  const mod = await import('@/api/hospital.js');
  return mod.getHospitals(query || {});
}
async function _apiGetHospitalDetail(id) {
  const mod = await import('@/api/hospital.js');
  return mod.getHospitalDetail(id);
}

export const useHospitalStore = defineStore('hospital', () => {
  // ---- 响应式状态
  const list = ref([]);
  const detail = ref(null);
  const pagination = ref({ total: 0, page: 1, limit: 20 });
  const lastQuery = ref({});
  const loading = ref(false);
  const loadingDetail = ref(false);
  const error = ref(null);

  // ---- actions

  /**
   * 加载医院列表（医院列表页 / 首页推荐用）。
   *
   * @param {object} [query]
   * @param {number} [query.page]
   * @param {number} [query.limit]
   * @param {number} [query.city_id]
   * @param {string} [query.level]
   * @param {string} [query.keyword]
   * @returns {Promise<{items: object[], total: number, page: number, limit: number}>}
   */
  async function loadList(query) {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiGetHospitals(query || {});
      const items = (r && (r.items || r.data)) || [];
      list.value = Array.isArray(items) ? items : [];
      pagination.value = {
        total: r && typeof r.total === 'number' ? r.total : list.value.length,
        page: r && typeof r.page === 'number' ? r.page : (query && query.page) || 1,
        limit: r && typeof r.limit === 'number' ? r.limit : (query && query.limit) || 20,
      };
      lastQuery.value = query || {};
      return { items: list.value, ...pagination.value };
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 加载医院详情。
   *
   * @param {number|string} id
   * @returns {Promise<object>}
   */
  async function loadDetail(id) {
    loadingDetail.value = true;
    error.value = null;
    try {
      const r = await _apiGetHospitalDetail(id);
      detail.value = r || null;
      return detail.value;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loadingDetail.value = false;
    }
  }

  /**
   * 工具：清空当前详情（页面卸载时调）。
   */
  function clearDetail() {
    detail.value = null;
  }

  return {
    // state
    list,
    detail,
    pagination,
    lastQuery,
    loading,
    loadingDetail,
    error,
    // actions
    loadList,
    loadDetail,
    clearDetail,
  };
});

export default { useHospitalStore };