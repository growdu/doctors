// src/stores/wallet.js
//
// 钱包 store —— 患者端余额 / 冻结 / 流水 + loading/error 状态机
// (spec §4.6 + plan Task M2)
//
// 范围（本文件）：
//   - wallet：当前钱包（balance + frozen + currency）
//   - transactions：流水列表（响应式）
//   - pagination：流水分页元数据
//   - loading / loadingTx / error：动作执行态
//
// 不在本文件范围：
//   - 充值 / 提现（v1 仅展示；后续 plan 接入）
//   - 流水导出 / 对账（plan 后续）

import { defineStore } from 'pinia';
import { ref } from 'vue';

async function _apiGetWallet() {
  const mod = await import('@/api/wallet.js');
  return mod.getWallet();
}
async function _apiListTransactions(query) {
  const mod = await import('@/api/wallet.js');
  return mod.listTransactions(query || {});
}

export const useWalletStore = defineStore('wallet', () => {
  // ---- 响应式状态
  const wallet = ref(null);
  const transactions = ref([]);
  const pagination = ref({ total: 0, page: 1, page_size: 20 });
  const loading = ref(false);
  const loadingTx = ref(false);
  const error = ref(null);

  // ---- actions

  /**
   * 拉钱包余额 + 冻结。
   */
  async function loadWallet() {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiGetWallet();
      wallet.value = r || null;
      return wallet.value;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 拉流水（默认 20 条 / page=1）。
   *
   * @param {object} [query]
   * @param {number} [query.page]
   * @param {number} [query.page_size]
   * @param {string} [query.type]
   */
  async function loadTransactions(query) {
    loadingTx.value = true;
    error.value = null;
    try {
      const r = await _apiListTransactions(query || {});
      const items = (r && (r.items || r.data)) || [];
      transactions.value = Array.isArray(items) ? items : [];
      pagination.value = {
        total: r && typeof r.total === 'number' ? r.total : transactions.value.length,
        page: r && typeof r.page === 'number' ? r.page : (query && query.page) || 1,
        page_size: r && typeof r.page_size === 'number' ? r.page_size : (query && query.page_size) || 20,
      };
      return { items: transactions.value, ...pagination.value };
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loadingTx.value = false;
    }
  }

  /**
   * 工具：清空所有状态（页面卸载时调）。
   */
  function clearAll() {
    wallet.value = null;
    transactions.value = [];
    pagination.value = { total: 0, page: 1, page_size: 20 };
  }

  return {
    // state
    wallet,
    transactions,
    pagination,
    loading,
    loadingTx,
    error,
    // actions
    loadWallet,
    loadTransactions,
    clearAll,
  };
});

export default { useWalletStore };