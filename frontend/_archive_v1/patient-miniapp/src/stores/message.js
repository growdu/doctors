// src/stores/message.js
//
// 站内信 store —— 列表 + 当前详情 + 未读数 + loading/error 状态机
// (spec §4.8 + plan Task M2)
//
// 范围（本文件）：
//   - list：站内信列表（响应式）
//   - current：当前详情（响应式）
//   - unread：未读总数（响应式；首屏 badge 用）
//   - pagination：分页元数据
//   - loading / loadingDetail / error：动作执行态
//
// 不在本文件范围：
//   - WebSocket 实时推送（plan 后续）

import { defineStore } from 'pinia';
import { ref } from 'vue';

async function _apiListMessages(query) {
  const mod = await import('@/api/message.js');
  return mod.listMessages(query || {});
}
async function _apiGetMessageDetail(id) {
  const mod = await import('@/api/message.js');
  return mod.getMessageDetail(id);
}
async function _apiMarkRead(id) {
  const mod = await import('@/api/message.js');
  return mod.markRead(id);
}
async function _apiMarkAllRead() {
  const mod = await import('@/api/message.js');
  return mod.markAllRead();
}

export const useMessageStore = defineStore('message', () => {
  // ---- 响应式状态
  const list = ref([]);
  const current = ref(null);
  const unread = ref(0);
  const pagination = ref({ total: 0, page: 1, page_size: 20 });
  const loading = ref(false);
  const loadingDetail = ref(false);
  const error = ref(null);

  // ---- actions

  /**
   * 加载站内信列表。
   *
   * @param {object} [query]
   * @param {number} [query.page]
   * @param {number} [query.page_size]
   * @param {string} [query.category]
   * @param {boolean} [query.unread]
   */
  async function loadList(query) {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiListMessages(query || {});
      const items = (r && (r.items || r.data)) || [];
      list.value = Array.isArray(items) ? items : [];
      pagination.value = {
        total: r && typeof r.total === 'number' ? r.total : list.value.length,
        page: r && typeof r.page === 'number' ? r.page : (query && query.page) || 1,
        page_size: r && typeof r.page_size === 'number' ? r.page_size : (query && query.page_size) || 20,
      };
      if (r && typeof r.unread === 'number') {
        unread.value = r.unread;
      } else {
        // 兜底：本地计算未读
        unread.value = list.value.filter((m) => !m.read_at).length;
      }
      return { items: list.value, unread: unread.value, ...pagination.value };
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 加载单条详情（含标记已读副作用）。
   *
   * @param {number|string} id
   */
  async function loadDetail(id) {
    loadingDetail.value = true;
    error.value = null;
    try {
      const r = await _apiGetMessageDetail(id);
      current.value = r || null;
      // 副作用：标记已读（若当前消息原本未读）
      if (current.value && !current.value.read_at) {
        try {
          await _apiMarkRead(id);
          // 本地同步
          current.value.read_at = new Date().toISOString();
          const idx = list.value.findIndex((m) => String(m.id) === String(id));
          if (idx >= 0) list.value[idx] = Object.assign({}, list.value[idx], { read_at: current.value.read_at });
          if (unread.value > 0) unread.value -= 1;
        } catch (_e) {
          // 标记已读失败不阻塞详情展示
        }
      }
      return current.value;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loadingDetail.value = false;
    }
  }

  /**
   * 全部已读。
   */
  async function markAllRead() {
    try {
      await _apiMarkAllRead();
      const now = new Date().toISOString();
      list.value = list.value.map((m) => Object.assign({}, m, { read_at: m.read_at || now }));
      unread.value = 0;
    } catch (e) {
      error.value = e;
      throw e;
    }
  }

  /**
   * 清空状态（页面卸载）。
   */
  function clearAll() {
    list.value = [];
    current.value = null;
    unread.value = 0;
    pagination.value = { total: 0, page: 1, page_size: 20 };
  }

  return {
    // state
    list,
    current,
    unread,
    pagination,
    loading,
    loadingDetail,
    error,
    // actions
    loadList,
    loadDetail,
    markAllRead,
    clearAll,
  };
});

export default { useMessageStore };