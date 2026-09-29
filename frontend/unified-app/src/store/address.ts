/**
 * addressStore（Pinia setup syntax）：地址管理。
 *
 * 设计原则：
 *   - state：list（地址数组）+ loading + error
 *   - getters：defaultAddress、count、atLimit（最多 5 条）
 *   - actions：fetchList / create / update / remove / setDefault
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.4（业务 store 模式）
 */
import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import {
  listAddresses,
  createAddress,
  updateAddress,
  deleteAddress,
  setDefaultAddress,
} from '@/api/user';
import type { Address, CreateAddressRequest } from '@/api/user';

const ADDRESS_LIMIT = 5;

export const useAddressStore = defineStore('address', () => {
  // ── state ──────────────────────────────────────────────────────
  const list = ref<Address[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // ── getters ────────────────────────────────────────────────────
  const count = computed(() => list.value.length);
  const atLimit = computed(() => list.value.length >= ADDRESS_LIMIT);

  const defaultAddress = computed<Address | null>(
    () => list.value.find((a) => a.is_default) ?? null,
  );

  function findById(id: number): Address | undefined {
    return list.value.find((a) => a.id === id);
  }

  // ── actions ────────────────────────────────────────────────────
  async function fetchList(): Promise<Address[]> {
    loading.value = true;
    error.value = null;
    try {
      const r = await listAddresses();
      list.value = r.items;
      return list.value;
    } catch (e) {
      error.value = (e as Error).message;
      return [];
    } finally {
      loading.value = false;
    }
  }

  async function create(req: CreateAddressRequest): Promise<Address> {
    const a = await createAddress(req);
    list.value = [a, ...list.value];
    if (a.is_default) {
      // 取消其他默认
      list.value = list.value.map((x) => (x.id === a.id ? x : { ...x, is_default: false }));
    }
    return a;
  }

  async function update(id: number, req: Partial<CreateAddressRequest>): Promise<Address> {
    const a = await updateAddress(id, req);
    list.value = list.value.map((x) => (x.id === id ? a : x));
    if (a.is_default) {
      list.value = list.value.map((x) => (x.id === id ? x : { ...x, is_default: false }));
    }
    return a;
  }

  async function remove(id: number): Promise<void> {
    await deleteAddress(id);
    list.value = list.value.filter((a) => a.id !== id);
  }

  async function setDefault(id: number): Promise<Address> {
    const a = await setDefaultAddress(id);
    list.value = list.value.map((x) => ({ ...x, is_default: x.id === id }));
    return a;
  }

  function clearError() {
    error.value = null;
  }

  return {
    // state
    list,
    loading,
    error,
    // getters
    count,
    atLimit,
    defaultAddress,
    findById,
    // actions
    fetchList,
    create,
    update,
    remove,
    setDefault,
    clearError,
  };
});