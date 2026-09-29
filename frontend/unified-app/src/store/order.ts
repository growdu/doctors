/**
 * orderStore（Pinia setup syntax）：订单状态管理。
 *
 * 设计原则：
 *   - state：orders（列表）+ currentOrder（详情）+ loading 标志
 *   - actions：fetchList / fetchDetail / createOrder / cancelOrder / confirmAccept / rejectAccept / finishOrder
 *   - 错误处理：统一捕获 + error state
 *
 * 与 authStore 风格一致（setup syntax）；pinia actions 通过 this 调用。
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.4
 */
import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import {
  createOrder as apiCreateOrder,
  listOrders as apiListOrders,
  getOrder as apiGetOrder,
  cancelOrder as apiCancelOrder,
  confirmAccept as apiConfirmAccept,
  rejectAccept as apiRejectAccept,
  finishOrder as apiFinishOrder,
} from '@/api/orders';
import type { Order, OrderStatus } from '@/api/orders';

export const useOrderStore = defineStore('order', () => {
  // ── state ──────────────────────────────────────────────────────
  const orders = ref<Order[]>([]);
  const currentOrder = ref<Order | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const total = ref(0);

  // ── getters ────────────────────────────────────────────────────
  const hasOrders = computed(() => orders.value.length > 0);

  /**
   * 按状态过滤（前端衍生，节省后端 query）。
   * 注意与后端 listOrders({status}) 不同时使用，避免双层过滤错位。
   */
  function filterByStatusLocal(status: OrderStatus): Order[] {
    return orders.value.filter((o) => o.status === status);
  }

  // ── actions ────────────────────────────────────────────────────
  async function fetchList(query: Parameters<typeof apiListOrders>[0] = {}): Promise<void> {
    loading.value = true;
    error.value = null;
    try {
      const r = await apiListOrders(query);
      orders.value = r.items;
      total.value = r.total;
    } catch (e) {
      error.value = (e as Error).message;
    } finally {
      loading.value = false;
    }
  }

  async function fetchDetail(id: number): Promise<Order> {
    loading.value = true;
    error.value = null;
    try {
      const o = await apiGetOrder(id);
      currentOrder.value = o;
      return o;
    } catch (e) {
      error.value = (e as Error).message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function create(req: Parameters<typeof apiCreateOrder>[0]): Promise<Order> {
    loading.value = true;
    error.value = null;
    try {
      const o = await apiCreateOrder(req);
      orders.value = [o, ...orders.value];
      return o;
    } catch (e) {
      error.value = (e as Error).message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function cancel(id: number, reason?: string): Promise<Order> {
    const o = await apiCancelOrder(id, reason);
    upsert(o);
    return o;
  }

  async function confirmAccept(id: number): Promise<Order> {
    const o = await apiConfirmAccept(id);
    upsert(o);
    return o;
  }

  async function rejectAccept(id: number, reason?: string): Promise<Order> {
    const o = await apiRejectAccept(id, reason);
    upsert(o);
    return o;
  }

  async function finish(id: number): Promise<Order> {
    const o = await apiFinishOrder(id);
    upsert(o);
    return o;
  }

  function clearError() {
    error.value = null;
  }

  // ── helpers ────────────────────────────────────────────────────
  function upsert(o: Order) {
    const idx = orders.value.findIndex((x) => x.id === o.id);
    if (idx >= 0) orders.value[idx] = o;
    else orders.value = [o, ...orders.value];
    if (currentOrder.value?.id === o.id) currentOrder.value = o;
  }

  return {
    // state
    orders,
    currentOrder,
    loading,
    error,
    total,
    // getters
    hasOrders,
    // actions
    fetchList,
    fetchDetail,
    create,
    cancel,
    confirmAccept,
    rejectAccept,
    finish,
    clearError,
    // helpers
    filterByStatusLocal,
  };
});