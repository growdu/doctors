// src/stores/virtualnumber.js
//
// 虚拟号 store —— 患者端的虚拟号分配 + 详情 + loading/error 状态机
// (spec §4.7 + plan Task M2)
//
// 范围（本文件）：
//   - current：当前订单的虚拟号（响应式）
//   - byOrderId：orderId → 虚拟号 的本地缓存（响应式；详情页快速回显用）
//   - loading / error：动作执行态
//
// 不在本文件范围：
//   - 释放 / 重新分配（v1 前端只读；分配由 order-service 内部触发）

import { defineStore } from 'pinia';
import { ref } from 'vue';

async function _apiAllocateVirtualNumber(payload) {
  const mod = await import('@/api/virtualnumber.js');
  return mod.allocateVirtualNumber(payload);
}
async function _apiGetVirtualNumber(id) {
  const mod = await import('@/api/virtualnumber.js');
  return mod.getVirtualNumber(id);
}

/** 虚拟号 status 常量 */
export const VIRTUAL_NUMBER_STATUS_ACTIVE = 'active';
export const VIRTUAL_NUMBER_STATUS_RELEASED = 'released';
export const VIRTUAL_NUMBER_STATUS_EXPIRED = 'expired';

/** 虚拟号 status → 友好文案 */
export const VIRTUAL_NUMBER_STATUS_LABEL = {
  [VIRTUAL_NUMBER_STATUS_ACTIVE]: '使用中',
  [VIRTUAL_NUMBER_STATUS_RELEASED]: '已释放',
  [VIRTUAL_NUMBER_STATUS_EXPIRED]: '已过期',
};

export const useVirtualNumberStore = defineStore('virtualNumber', () => {
  // ---- 响应式状态
  const current = ref(null);
  const byOrderId = ref({}); // { [orderId]: VirtualNumber }
  const loading = ref(false);
  const error = ref(null);

  // ---- actions

  /**
   * 分配虚拟号（v1 主要由 order-service 触发；前端暴露供调试用）。
   *
   * @param {object} payload
   * @returns {Promise<object>}
   */
  async function allocate(payload) {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiAllocateVirtualNumber(payload || {});
      current.value = r || null;
      if (r && r.order_id) {
        byOrderId.value = Object.assign({}, byOrderId.value, {
          [r.order_id]: r,
        });
      }
      return r;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 加载虚拟号详情。
   *
   * @param {number|string} id
   */
  async function loadDetail(id) {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiGetVirtualNumber(id);
      current.value = r || null;
      if (r && r.order_id) {
        byOrderId.value = Object.assign({}, byOrderId.value, {
          [r.order_id]: r,
        });
      }
      return current.value;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 工具：按 orderId 拿缓存中的虚拟号（无 → null）。
   *
   * @param {number|string} orderId
   * @returns {object|null}
   */
  function getByOrderId(orderId) {
    if (!orderId) return null;
    return byOrderId.value[orderId] || null;
  }

  /**
   * 工具：把虚拟号中间四位掩码（138****0001）用于展示，避免暴露完整号段。
   *
   * @param {string} phone
   * @returns {string}
   */
  function maskedPhone(phone) {
    if (typeof phone !== 'string' || phone.length < 7) return phone || '';
    return phone.substring(0, 3) + '****' + phone.substring(phone.length - 4);
  }

  return {
    // state
    current,
    byOrderId,
    loading,
    error,
    // actions
    allocate,
    loadDetail,
    getByOrderId,
    maskedPhone,
  };
});

export default {
  useVirtualNumberStore,
  VIRTUAL_NUMBER_STATUS_ACTIVE,
  VIRTUAL_NUMBER_STATUS_RELEASED,
  VIRTUAL_NUMBER_STATUS_EXPIRED,
  VIRTUAL_NUMBER_STATUS_LABEL,
};