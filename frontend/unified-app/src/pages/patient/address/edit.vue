<script setup lang="ts">
/**
 * patient/address/edit.vue — 地址编辑/新增页（v2 unified-app）。
 *
 * 入口 A（新增）：address/list「+ 新增地址」→ 本页（无 addressId）
 * 入口 B（编辑）：address/list「编辑」→ 本页（?addressId=xxx）
 *
 * 表单字段：
 *   - 收件人 / 手机号 / 省 / 市 / 区 / 详细地址 / 设为默认（switch）
 *
 * 验证：手机号 11 位 + 详细地址 ≥ 5 字
 *
 * 对应：dev.md §45.5 Phase 3.1a
 */
import { computed, onMounted, ref } from 'vue';
import { useAddressStore } from '@/store/address';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiCard from '@/components/shared/UiCard.vue';

const PHONE_RE = /^1\d{10}$/;

const store = useAddressStore();

const addressId = ref<number | null>(null);
const mode = ref<'create' | 'edit'>('create');

const name = ref('');
const phone = ref('');
const province = ref('');
const city = ref('');
const district = ref('');
const detail = ref('');
const isDefault = ref(false);

const saving = ref(false);
const isCurrentDefault = ref(false);

const phoneValid = computed(() => PHONE_RE.test(phone.value));
const detailValid = computed(() => detail.value.trim().length >= 5);
const canSave = computed(() => !saving.value && name.value.trim() && phoneValid.value && detailValid.value && province.value.trim() && city.value.trim());

// onLoad query 由 uni-app 框架在 Vue 入口传入；这里通过 onLoad hook 读取
// 用 onMounted 简化处理：URL 路径中的 query 在真实环境由 uni 传入；当前 demo 阶段
// 手动通过 router 传参（Phase 3.1 patient 域迁移时接入真路由）
async function onMount() {
  if (mode.value === 'edit' && addressId.value) {
    await store.fetchList();
    const found = store.findById(addressId.value);
    if (found) {
      name.value = found.name;
      phone.value = found.phone;
      province.value = found.province;
      city.value = found.city;
      district.value = found.district;
      detail.value = found.detail;
      isDefault.value = found.is_default;
      isCurrentDefault.value = found.is_default;
    }
  }
}

onMounted(onMount);

async function onSave() {
  if (!canSave.value) return;
  saving.value = true;
  const payload = {
    name: name.value.trim(),
    phone: phone.value.trim(),
    province: province.value.trim(),
    city: city.value.trim(),
    district: district.value.trim(),
    detail: detail.value.trim(),
    is_default: isDefault.value,
  };
  try {
    if (mode.value === 'edit' && addressId.value) {
      await store.update(addressId.value, payload);
    } else {
      await store.create(payload);
    }
    if (typeof uni !== 'undefined') uni.navigateBack({ delta: 1 });
  } catch (e) {
    // store.error 已记录
  } finally {
    saving.value = false;
  }
}

async function onSetDefault() {
  if (mode.value !== 'edit' || !addressId.value) return;
  try {
    await store.setDefault(addressId.value);
    isCurrentDefault.value = true;
    if (typeof uni !== 'undefined') uni.navigateBack({ delta: 1 });
  } catch (e) {
    // store.error 已记录
  }
}
</script>

<template>
  <view class="ui-page" data-testid="patient-address-edit">
    <UiCard title="地址信息">
      <UiInput v-model="name" label="收件人" placeholder="请输入姓名" :maxlength="32" data-testid="address-name" />
      <UiInput v-model="phone" label="手机号" placeholder="11 位手机号" type="tel" :maxlength="11" data-testid="address-phone" />
      <UiInput v-model="province" label="省" placeholder="如：北京市" :maxlength="16" data-testid="address-province" />
      <UiInput v-model="city" label="市" placeholder="如：北京" :maxlength="16" data-testid="address-city" />
      <UiInput v-model="district" label="区" placeholder="如：东城区" :maxlength="32" data-testid="address-district" />
      <UiInput v-model="detail" label="详细地址" placeholder="街道 + 楼栋 + 门牌号" type="textarea" :maxlength="200" data-testid="address-detail" />
      <view class="address-edit__row">
        <text class="address-edit__label">设为默认地址</text>
        <switch :checked="isDefault" @change="(e: any) => (isDefault = e.detail.value)" data-testid="address-default-switch" />
      </view>
    </UiCard>

    <view class="address-edit__footer">
      <UiButton
        v-if="mode === 'edit' && !isCurrentDefault"
        type="default"
        block
        data-testid="set-default-btn"
        @click="onSetDefault"
      >
        设为默认
      </UiButton>
      <UiButton
        type="primary"
        block
        :loading="saving"
        :disabled="!canSave"
        data-testid="save-btn"
        @click="onSave"
      >
        保存
      </UiButton>
    </view>
  </view>
</template>

<style scoped>
.address-edit__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--ui-space-md) 0;
}

.address-edit__label {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
}

.address-edit__footer {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-md);
  margin-top: var(--ui-space-base);
}
</style>