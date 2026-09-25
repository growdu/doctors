// src/stores/address.js
//
// 地址簿 store —— 患者端的地址 CRUD + 默认地址设置 + loading/error 状态机
// (spec §4.4 + plan Task M2)
//
// 范围（本文件）：
//   - list：地址列表（响应式）
//   - defaultId：默认地址 id（响应式；UI 高亮 + 选中用）
//   - loading / error：动作执行态
//   - lastAddedId：最近一次 addAddress 成功的 id（UI toast 用）
//
// 不在本文件范围：
//   - 上限校验（5 条 / CodeConflict 由后端兜底；前端按 error.code 提示）
//   - 地图选点（plan 后续）

import { defineStore } from 'pinia';
import { ref } from 'vue';

async function _apiGetAddresses() {
  const mod = await import('@/api/address.js');
  return mod.getAddressList();
}
async function _apiAddAddress(payload) {
  const mod = await import('@/api/address.js');
  return mod.addAddress(payload);
}
async function _apiUpdateAddress(id, payload) {
  const mod = await import('@/api/address.js');
  return mod.updateAddress(id, payload);
}
async function _apiDeleteAddress(id) {
  const mod = await import('@/api/address.js');
  return mod.deleteAddress(id);
}
async function _apiSetDefaultAddress(id) {
  const mod = await import('@/api/address.js');
  return mod.setDefaultAddress(id);
}

export const useAddressStore = defineStore('address', () => {
  // ---- 响应式状态
  const list = ref([]);
  const defaultId = ref(null);
  const loading = ref(false);
  const error = ref(null);
  const lastAddedId = ref(null);

  // ---- actions

  /**
   * 加载地址列表（地址管理页 onLoad）。
   */
  async function loadList() {
    loading.value = true;
    error.value = null;
    try {
      const r = await _apiGetAddresses();
      const items = (r && (r.items || r.data)) || (Array.isArray(r) ? r : []);
      list.value = items;
      // 计算默认 id：is_default === true 的第一条
      const def = items.find((a) => a && a.is_default);
      defaultId.value = def ? def.id : (items[0] && items[0].id) || null;
      return list.value;
    } catch (e) {
      error.value = e;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 新增地址；成功后写入 list + 触发 reload。
   *
   * @param {object} payload
   * @returns {Promise<object>}
   */
  async function add(payload) {
    const created = await _apiAddAddress(payload || {});
    lastAddedId.value = created && created.id;
    // 若新增时设了默认 → 重新拉以同步 defaultId
    if (created && created.is_default) {
      await loadList();
    } else {
      // 列表追加到头部
      list.value = [created].concat(list.value);
    }
    return created;
  }

  /**
   * 修改地址；成功后刷新对应项。
   *
   * @param {number|string} id
   * @param {object} payload
   * @returns {Promise<object>}
   */
  async function update(id, payload) {
    const updated = await _apiUpdateAddress(id, payload || {});
    const idx = list.value.findIndex((a) => a.id === id);
    if (idx >= 0) {
      list.value = list.value.map((a) => (a.id === id ? Object.assign({}, a, updated) : a));
    } else {
      list.value = [updated].concat(list.value);
    }
    return updated;
  }

  /**
   * 删除地址。
   *
   * @param {number|string} id
   */
  async function remove(id) {
    await _apiDeleteAddress(id);
    list.value = list.value.filter((a) => a.id !== id);
    if (defaultId.value === id) {
      defaultId.value = (list.value[0] && list.value[0].id) || null;
    }
  }

  /**
   * 设置默认地址。
   *
   * @param {number|string} id
   */
  async function setDefault(id) {
    await _apiSetDefaultAddress(id);
    // 同步本地 is_default：所有项 is_default=false，仅 id 这一项 true
    list.value = list.value.map((a) =>
      Object.assign({}, a, { is_default: a.id === id }),
    );
    defaultId.value = id;
  }

  /**
   * 工具：取默认地址对象（无 → null）。
   *
   * @returns {object|null}
   */
  function defaultAddress() {
    if (!defaultId.value) return null;
    return list.value.find((a) => a.id === defaultId.value) || null;
  }

  return {
    // state
    list,
    defaultId,
    loading,
    error,
    lastAddedId,
    // actions
    loadList,
    add,
    update,
    remove,
    setDefault,
    defaultAddress,
  };
});

export default { useAddressStore };