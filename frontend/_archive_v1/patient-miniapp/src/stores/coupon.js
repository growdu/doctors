// src/stores/coupon.js
//
// 优惠券 store —— 患者端的券模板列表 / 领取 / 我的券 + loading/error 状态机
// (spec §4.5 + plan Task M2)
//
// 范围（本文件）：
//   - templates：可用券模板列表（响应式）—— 「领券」tab
//   - mine：我的券（响应式）—— 「我的券」tab
//   - loading / error：动作执行态
//   - lastClaimedId：最近一次 claimCoupon 成功的 user_coupon id
//
// 不在本文件范围：
//   - 订单核销（v1 不传 order_id，v2 接订单后再扩）
//   - 库存倒计时（前端只展示 stock，由后端兜底）

import { defineStore } from 'pinia';
import { ref } from 'vue';

async function _apiGetCoupons(query) {
  const mod = await import('@/api/coupon.js');
  return mod.getCoupons(query || {});
}
async function _apiGetMyCoupons() {
  const mod = await import('@/api/coupon.js');
  return mod.getMyCoupons();
}
async function _apiClaimCoupon(id) {
  const mod = await import('@/api/coupon.js');
  return mod.claimCoupon(id);
}
async function _apiUseMyCoupon(userCouponId) {
  const mod = await import('@/api/coupon.js');
  return mod.useMyCoupon(userCouponId);
}

// 我的券 status 优先级（按 spec §4.5：unused 优先）
export const COUPON_STATUS_UNUSED = 'unused';
export const COUPON_STATUS_USED = 'used';
export const COUPON_STATUS_EXPIRED = 'expired';

/** 我的券 status → 友好文案 */
export const COUPON_STATUS_LABEL = {
  [COUPON_STATUS_UNUSED]: '未使用',
  [COUPON_STATUS_USED]: '已使用',
  [COUPON_STATUS_EXPIRED]: '已过期',
};

export const useCouponStore = defineStore('coupon', () => {
  // ---- 响应式状态
  const templates = ref([]);
  const mine = ref([]);
  const loading = ref(false);
  const loadingMine = ref(false);
  const error = ref(null);
  const lastClaimedId = ref(null);

  // ---- actions

  /**
   * 加载可用券模板列表（「领券」tab）。
   *
   * @param {object} [query]
   */
  async function loadTemplates(query) {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiGetCoupons(query || {});
      const items = (r && (r.items || r.data)) || (Array.isArray(r) ? r : []);
      templates.value = items;
      return items;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 加载我的券（「我的券」tab）。
   */
  async function loadMine() {
    loadingMine.value = true;
    error.value = null;
    try {
      const r = await _apiGetMyCoupons();
      const items = (r && (r.items || r.data)) || (Array.isArray(r) ? r : []);
      mine.value = items;
      return items;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loadingMine.value = false;
    }
  }

  /**
   * 领取优惠券；成功后 reload 我的券。
   *
   * @param {number|string} id
   * @returns {Promise<object>}
   */
  async function claim(id) {
    const uc = await _apiClaimCoupon(id);
    lastClaimedId.value = uc && uc.id;
    // 重新拉我的券以同步（claim 成功后通常立即会切到「我的」tab）
    await loadMine();
    return uc;
  }

  /**
   * 核销我的券（v1 不传 order）。
   *
   * @param {number|string} userCouponId
   */
  async function useCoupon(userCouponId) {
    await _apiUseMyCoupon(userCouponId);
    // 本地缓存同步：把对应 user_coupon.status 标 used
    mine.value = mine.value.map((uc) =>
      uc && uc.id === userCouponId
        ? Object.assign({}, uc, { status: COUPON_STATUS_USED, used_at: new Date().toISOString() })
        : uc,
    );
  }

  /**
   * 工具：按 status 过滤我的券。
   *
   * @param {string} status
   * @returns {object[]}
   */
  function filterByStatus(status) {
    if (!status) return mine.value;
    return mine.value.filter((uc) => uc && uc.status === status);
  }

  /**
   * 工具：统计各状态数量。
   *
   * @returns {{unused: number, used: number, expired: number, total: number}}
   */
  function statusCount() {
    const out = { unused: 0, used: 0, expired: 0, total: mine.value.length };
    for (const uc of mine.value) {
      if (uc && out[uc.status] !== undefined) out[uc.status] += 1;
    }
    return out;
  }

  return {
    // state
    templates,
    mine,
    loading,
    loadingMine,
    error,
    lastClaimedId,
    // actions
    loadTemplates,
    loadMine,
    claim,
    useCoupon,
    filterByStatus,
    statusCount,
  };
});

export default { useCouponStore, COUPON_STATUS_UNUSED, COUPON_STATUS_USED, COUPON_STATUS_EXPIRED, COUPON_STATUS_LABEL };