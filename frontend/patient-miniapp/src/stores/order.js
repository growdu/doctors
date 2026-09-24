// src/stores/order.js
//
// 订单 store —— 患者端的订单状态机 + 列表/详情 + 轮询
// (spec §4.2 + §3.3 + plan Task 7 + Task 13 v1.1 增量)
//
// v1.1 增量（依据 spec `2026-09-24-order-matching-redesign.md` §1.2）：
//   - 新增状态 `selectingEscort`：订单已支付，等待患者从候选列表选陪诊师
//   - 新增状态 `escortPendingAcceptance`：患者已选定某 escort，等待 escort 在 30s 倒计时内确认
//   - 替换原抢单 `acceptOrder` 为 `selectEscortBy(orderId, escortId)`（由患者侧发起）
//   - 30s 倒计时由 UI 组件（`components/Countdown/Countdown.vue`）驱动；
//     本 store 只暴露 `escortPendingExpireAt` 字段供 UI 读
//
// 范围（本文件）：
//   - current：详情页当前订单（响应式）
//   - list：订单列表（响应式）
//   - candidates：候选陪诊师列表（响应式；selectingEscort 期间有值）
//   - candidatesGeneratedAt：候选快照生成时间（前端展示"X 秒前刷新"用）
//   - selection：最近一次 selectEscort 响应（含 escort_pending_expire_at 等）
//   - pollTimer：当前轮询句柄（防止多实例）
//
// 不在本文件范围（plan Task 5/6/13）：
//   - candidates / selectEscort 的真实 API（由 src/api/order.js 提供；当前 jest.mock 注入）
//   - 30s 倒计时到期 → 自动跳详情 / 重新拉 candidates 的 UI 行为
//   - 评价 / 退款 / 申诉（plan Task 6+）

import { defineStore } from 'pinia';
import { ref } from 'vue';

// api 模块由 plan Task 5/6 提供；本阶段 jest 单测用 jest.doMock 注入。
// 用动态 import 延迟解析（避免顶层 import 在 api 文件缺失时抛 module not found）。
async function _apiGetOrder(id) {
  const mod = await import('@/api/order.js');
  return mod.getOrder(id);
}
async function _apiListOrders(query) {
  const mod = await import('@/api/order.js');
  return mod.listOrders(query || {});
}
async function _apiGetCandidates(orderId) {
  const mod = await import('@/api/order.js');
  return mod.getCandidates(orderId);
}
async function _apiSelectEscort(orderId, payload) {
  const mod = await import('@/api/order.js');
  return mod.selectEscort(orderId, payload);
}
async function _apiCancelOrder(id, reason) {
  const mod = await import('@/api/order.js');
  return mod.cancelOrder(id, reason);
}

// ---------------------------------------------------------------------------
// 订单状态常量（与后端状态机字段名 1:1 对齐）
// ---------------------------------------------------------------------------

/** 已下单未支付 */
export const ORDER_STATUS_PENDING_PAYMENT = 'pending_payment';
/** 已支付，等待患者从候选列表选陪诊师（v1.1 新增） */
export const ORDER_STATUS_SELECTING_ESCORT = 'selectingEscort';
/** 已选定某 escort，等待 escort 在 30s 内确认（v1.1 新增） */
export const ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE = 'escortPendingAcceptance';
/** escort 已确认，等待出发 */
export const ORDER_STATUS_ESCORT_CONFIRMED = 'escortConfirmed';
/** escort 已到达 / 服务中 */
export const ORDER_STATUS_IN_SERVICE = 'inService';
/** 服务完成 */
export const ORDER_STATUS_COMPLETED = 'completed';
/** 已取消 */
export const ORDER_STATUS_CANCELLED = 'cancelled';
/** 已退款（含部分退款） */
export const ORDER_STATUS_REFUNDED = 'refunded';

/** 全部状态码 → 友好文案（v1.1 仅覆盖新增/常用分支；其余由 UI 兜底） */
export const ORDER_STATUS_LABEL = {
  [ORDER_STATUS_PENDING_PAYMENT]: '待支付',
  [ORDER_STATUS_SELECTING_ESCORT]: '选择陪诊师',
  [ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE]: '陪诊师确认中',
  [ORDER_STATUS_ESCORT_CONFIRMED]: '已确认',
  [ORDER_STATUS_IN_SERVICE]: '服务中',
  [ORDER_STATUS_COMPLETED]: '已完成',
  [ORDER_STATUS_CANCELLED]: '已取消',
  [ORDER_STATUS_REFUNDED]: '已退款',
};

/** 进度条节点顺序（详情页头部进度条 UI 用） */
export const ORDER_PROGRESS_STEPS = [
  ORDER_STATUS_PENDING_PAYMENT,
  ORDER_STATUS_SELECTING_ESCORT,
  ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE,
  ORDER_STATUS_ESCORT_CONFIRMED,
  ORDER_STATUS_IN_SERVICE,
  ORDER_STATUS_COMPLETED,
];

export const useOrderStore = defineStore('order', () => {
  // ---- 响应式状态
  const current = ref(null);
  const list = ref([]);
  const candidates = ref([]);
  const candidatesGeneratedAt = ref('');
  const selection = ref(null); // 最近一次 selectEscort 响应
  let pollTimer = null;
  let pollingOrderId = null;

  // ---- actions

  /**
   * 加载订单详情（详情页打开时调）。
   * @param {number|string} id
   */
  async function loadOrder(id) {
    const detail = await _apiGetOrder(id);
    current.value = detail;
    return detail;
  }

  /**
   * 加载订单列表（订单列表页打开时调）。
   * @param {object} [query]
   * @param {string} [query.status]
   * @param {number} [query.page]
   * @param {number} [query.page_size]
   */
  async function loadList(query) {
    const r = await _apiListOrders(query || {});
    // 兼容两种返回形态：{ items: [...] } 或 [...]（直返数组）
    const items = Array.isArray(r) ? r : (r && r.items) || [];
    list.value = items;
    return items;
  }

  /**
   * 加载候选陪诊师列表（订单处于 selectingEscort 时调）。
   *
   * @param {number|string} orderId
   * @returns {Promise<{ items: object[], generated_at: string }>}
   */
  async function loadCandidates(orderId) {
    const r = await _apiGetCandidates(orderId);
    // 兼容 { items, generated_at } 或 { data } 形态
    const items = (r && (r.items || r.data || r.candidates)) || [];
    const generatedAt = (r && r.generated_at) || '';
    candidates.value = Array.isArray(items) ? items : [];
    candidatesGeneratedAt.value = generatedAt;
    return { items: candidates.value, generated_at: generatedAt };
  }

  /**
   * 患者选定某 escort —— 触发后端把订单状态切到 escortPendingAcceptance。
   *
   * @param {number|string} orderId
   * @param {number|string} escortId
   * @returns {Promise<object>} selectEscort 响应（含 escort_pending_expire_at）
   */
  async function selectEscortBy(orderId, escortId) {
    if (!orderId || !escortId) {
      throw new Error('selectEscortBy: orderId & escortId required');
    }
    const r = await _apiSelectEscort(orderId, { escort_id: escortId });
    selection.value = r || null;
    // 立刻刷新订单详情，状态机应切到 escortPendingAcceptance
    await loadOrder(orderId);
    return r;
  }

  /**
   * 启动订单状态轮询（详情页打开时调；intervalMs 默认 3000ms）。
   * 进行中状态（selectingEscort / escortPendingAcceptance / inService 等）才轮询；
   * 终态（completed / cancelled / refunded）由后端 status 反馈自动停。
   *
   * @param {number|string} orderId
   * @param {number} [intervalMs=3000]
   */
  function startPolling(orderId, intervalMs = 3000) {
    stopPolling();
    if (!orderId) return;
    pollingOrderId = orderId;
    const tick = async () => {
      if (pollingOrderId !== orderId) return; // 已被新轮询或 stopPolling 抢占
      try {
        await loadOrder(orderId);
      } catch (_e) {
        // 网络抖动：下一次 tick 继续
      }
    };
    // 立即拉一次；之后定时
    void tick();
    pollTimer = setInterval(tick, intervalMs);
  }

  /**
   * 停止轮询（页面 onUnload 时调）。
   */
  function stopPolling() {
    if (pollTimer !== null) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
    pollingOrderId = null;
  }

  /**
   * 取消订单（用户主动）。
   * @param {number|string} id
   * @param {string} reason
   */
  async function cancel(id, reason) {
    await _apiCancelOrder(id, reason);
    await loadOrder(id);
  }

  /**
   * 工具：判断当前订单是否处于某状态。
   * @param {string} status
   * @returns {boolean}
   */
  function isStatus(status) {
    return Boolean(current.value && current.value.status === status);
  }

  return {
    // state
    current,
    list,
    candidates,
    candidatesGeneratedAt,
    selection,
    // actions
    loadOrder,
    loadList,
    loadCandidates,
    selectEscortBy,
    startPolling,
    stopPolling,
    cancel,
    isStatus,
  };
});

export default {
  ORDER_STATUS_PENDING_PAYMENT,
  ORDER_STATUS_SELECTING_ESCORT,
  ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE,
  ORDER_STATUS_ESCORT_CONFIRMED,
  ORDER_STATUS_IN_SERVICE,
  ORDER_STATUS_COMPLETED,
  ORDER_STATUS_CANCELLED,
  ORDER_STATUS_REFUNDED,
  ORDER_STATUS_LABEL,
  ORDER_PROGRESS_STEPS,
};