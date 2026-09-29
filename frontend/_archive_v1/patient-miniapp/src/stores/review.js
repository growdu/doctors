// src/stores/review.js
//
// 评价 store —— 患者端的评价提交 + 评价列表 + 评价详情 + loading/error 状态机
// (spec §4.6 + plan Task M2)
//
// 范围（本文件）：
//   - mine：当前用户的评价列表（响应式；list 查询结果）
//   - current：当前正在看的评价详情（响应式）
//   - loading / submitting / error：动作执行态
//   - lastSubmittedId：最近一次 submit 成功的评价 id
//
// 不在本文件范围：
//   - admin 回复（store 不暴露 reply action）
//   - 评分可视化（UI 组件 + store 配套）

import { defineStore } from 'pinia';
import { ref } from 'vue';

async function _apiSubmitReview(payload) {
  const mod = await import('@/api/review.js');
  return mod.submitReview(payload);
}
async function _apiListReviews(query) {
  const mod = await import('@/api/review.js');
  return mod.listReviews(query || {});
}
async function _apiGetReviewDetail(id) {
  const mod = await import('@/api/review.js');
  return mod.getReviewDetail(id);
}

export const useReviewStore = defineStore('review', () => {
  // ---- 响应式状态
  const mine = ref([]);
  const current = ref(null);
  const loading = ref(false);
  const submitting = ref(false);
  const error = ref(null);
  const lastSubmittedId = ref(null);

  // ---- actions

  /**
   * 提交评价（评价创建页 onSubmit）。
   *
   * @param {object} payload  { order_id, escort_id, rating, comment }
   * @returns {Promise<object>}
   */
  async function submit(payload) {
    submitting.value = true;
    error.value = null;
    try {
      const r = await _apiSubmitReview(payload || {});
      lastSubmittedId.value = r && r.id;
      // 本地缓存追加（用户提交后切回订单详情希望立即可见）
      mine.value = [r].concat(mine.value);
      return r;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      submitting.value = false;
    }
  }

  /**
   * 拉评价列表（按 escort / order / min_rating 过滤）。
   *
   * @param {object} [query]
   */
  async function loadList(query) {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiListReviews(query || {});
      const items = (r && (r.reviews || r.items || r.data)) || [];
      mine.value = Array.isArray(items) ? items : [];
      return mine.value;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 加载评价详情。
   *
   * @param {number|string} id
   */
  async function loadDetail(id) {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiGetReviewDetail(id);
      current.value = r || null;
      return current.value;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return {
    // state
    mine,
    current,
    loading,
    submitting,
    error,
    lastSubmittedId,
    // actions
    submit,
    loadList,
    loadDetail,
  };
});

export default { useReviewStore };