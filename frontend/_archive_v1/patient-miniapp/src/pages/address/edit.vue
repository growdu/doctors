<!--
  src/pages/address/edit.vue

  地址编辑/新增页 —— 联系人 + 电话 + 详细地址 + 默认切换
  （spec §4.4 + plan Task M3）

  页面流向：
    入口 A（新增）：address/list「+ 新增地址」
       → 本页（无 addressId）
    入口 B（编辑）：address/list「编辑」/「默认」入口
       → 本页（?addressId=xxx）
       → mounted 调 store.loadList() 取列表 → findById 写入表单
    提交：
       - 新增 → store.add(payload) → 成功后 navigateBack
       - 编辑 → store.update(id, payload) → 成功后 navigateBack
    设为默认 → store.setDefault(id) → toast 后 navigateBack

  模板要点：
    - 顶部 <u-navbar> 标题根据 mode 变（新增 / 编辑地址）
    - 4 字段表单：联系人 / 电话 / 详细地址 / 默认切换（u-switch）
    - 「保存」主按钮（disabled 条件：3 必填项有空）
    - 「设为默认」按钮（仅编辑模式且非默认时显示）
    - loading 态：保存按钮 loading

  行为：
    - onLoad(query)：从 query 读 addressId；写入 mode
    - mounted()：地址列表里 findById 写入表单；新增模式跳过
    - onSave()：根据 mode 调 add / update
    - onSetDefault()：调 store.setDefault

  数据来源：
    - address store：useAddressStore().list / .add / .update / .setDefault

  测试覆盖：src/pages/address/edit.test.js
-->
<template>
  <view class="page-address-edit" data-test="address-edit-page">
    <u-navbar :title="mode === 'edit' ? '编辑地址' : '新增地址'" :auto-back="true" />

    <view class="page-address-edit__form" data-test="form">
      <view class="page-address-edit__field">
        <text class="page-address-edit__label">联系人</text>
        <input
          v-model="recipient"
          class="page-address-edit__input"
          type="text"
          maxlength="32"
          placeholder="请输入收件人姓名"
          data-test="recipient-input"
        />
      </view>

      <view class="page-address-edit__field">
        <text class="page-address-edit__label">手机号</text>
        <input
          v-model="phone"
          class="page-address-edit__input"
          type="number"
          maxlength="11"
          placeholder="11 位手机号"
          data-test="phone-input"
        />
      </view>

      <view class="page-address-edit__field">
        <text class="page-address-edit__label">详细地址</text>
        <textarea
          v-model="detail"
          class="page-address-edit__textarea"
          maxlength="200"
          placeholder="街道、楼栋、门牌号"
          data-test="detail-input"
        />
      </view>

      <view class="page-address-edit__field page-address-edit__field--switch">
        <text class="page-address-edit__label">设为默认地址</text>
        <u-switch
          v-model="isDefault"
          :disabled="isCurrentDefault && mode === 'edit'"
          data-test="default-switch"
        />
      </view>
    </view>

    <view class="page-address-edit__footer">
      <u-button
        v-if="mode === 'edit' && !isCurrentDefault"
        type="warning"
        plain
        size="large"
        :loading="settingDefault"
        data-test="set-default-btn"
        @click="onSetDefault"
      >设为默认</u-button>
      <u-button
        type="primary"
        size="large"
        :loading="saving"
        :disabled="!canSave"
        data-test="save-btn"
        @click="onSave"
      >保存</u-button>
    </view>
  </view>
</template>

<script>
// address/edit 页 —— Vue 3 Options API。
//
// 关键设计：
//   - onLoad/mounted 放在 methods 内（uni-app 支持 + 测试可通过 vm 调用）
//   - 编辑模式：loadList 后 findById 写入表单
//   - 设为默认与「保存」是两件事；v1 先让用户明确点选

import { useAddressStore } from '@/stores/address.js';

const PHONE_RE = /^1\d{10}$/;

export default {
  name: 'AddressEditPage',
  data() {
    return {
      addressId: null,
      mode: 'create',         // 'create' | 'edit'
      recipient: '',
      phone: '',
      detail: '',
      isDefault: false,
      isCurrentDefault: false,
      saving: false,
      settingDefault: false,
      _loaded: false,
    };
  },
  computed: {
    addressStore() {
      return useAddressStore();
    },
    phoneValid() {
      return PHONE_RE.test(String(this.phone || ''));
    },
    detailValid() {
      return String(this.detail || '').trim().length >= 5;
    },
    canSave() {
      if (this.saving) return false;
      return String(this.recipient || '').trim().length > 0
        && this.phoneValid
        && this.detailValid;
    },
  },
  methods: {
    /**
     * uni-app Page 钩子：onLoad(query)
     */
    onLoad(query) {
      this.addressId = (query && (query.addressId || query.id)) || null;
      this.mode = this.addressId ? 'edit' : 'create';
    },

    /**
     * Vue mounted 钩子（async + methods 内）。
     */
    async mounted() {
      if (this.mode === 'edit' && this.addressId && !this._loaded) {
        await this.hydrateFromList();
      }
    },

    /**
     * 编辑模式：拉地址列表 + findById 写入表单。
     */
    async hydrateFromList() {
      try {
        const list = await this.addressStore.loadList();
        const found = (list || []).find((a) => String(a.id) === String(this.addressId));
        if (found) {
          this.recipient = found.recipient || '';
          this.phone = found.phone || '';
          this.detail = found.detail || '';
          this.isDefault = !!found.is_default;
          this.isCurrentDefault = !!found.is_default;
          this._loaded = true;
        }
      } catch (e) {
        this._toast(this._errMsg(e, '加载地址失败'));
      }
    },

    /**
     * 「保存」点击：新增 / 编辑分支。
     */
    async onSave() {
      if (!this.canSave) return;
      this.saving = true;
      const payload = {
        recipient: String(this.recipient || '').trim(),
        phone: String(this.phone || '').trim(),
        detail: String(this.detail || '').trim(),
        is_default: !!this.isDefault,
      };
      try {
        if (this.mode === 'edit') {
          await this.addressStore.update(this.addressId, payload);
          this._toast('地址已更新');
        } else {
          await this.addressStore.add(payload);
          this._toast('地址已新增');
        }
        if (typeof uni !== 'undefined' && typeof uni.navigateBack === 'function') {
          uni.navigateBack({ delta: 1 });
        }
      } catch (e) {
        this._toast(this._errMsg(e, '保存失败，请重试'));
      } finally {
        this.saving = false;
      }
    },

    /**
     * 「设为默认」点击（编辑模式专属）。
     */
    async onSetDefault() {
      if (this.mode !== 'edit' || !this.addressId) return;
      this.settingDefault = true;
      try {
        await this.addressStore.setDefault(this.addressId);
        this._toast('已设为默认地址');
        if (typeof uni !== 'undefined' && typeof uni.navigateBack === 'function') {
          uni.navigateBack({ delta: 1 });
        }
      } catch (e) {
        this._toast(this._errMsg(e, '设置失败，请重试'));
      } finally {
        this.settingDefault = false;
      }
    },

    _toast(title) {
      if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
        uni.showToast({ title, icon: 'none' });
      }
    },

    _errMsg(e, fallback) {
      if (e && e.message) return e.message;
      return fallback;
    },
  },
};
</script>

<style lang="scss" scoped>
.page-address-edit {
  min-height: 100vh;
  padding: 16px;
  padding-bottom: 120px;
  background: #f5f5f5;
  box-sizing: border-box;
}

.page-address-edit__form {
  background: #ffffff;
  border-radius: 8px;
  padding: 8px 16px;
}

.page-address-edit__field {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #f0f0f0;
  padding: 12px 0;
  min-height: 56px;
}

.page-address-edit__field:last-child {
  border-bottom: none;
}

.page-address-edit__field--switch {
  justify-content: space-between;
}

.page-address-edit__label {
  width: 80px;
  font-size: 14px;
  color: #595959;
}

.page-address-edit__input,
.page-address-edit__textarea {
  flex: 1;
  font-size: 16px;
  color: #1f1f1f;
}

.page-address-edit__textarea {
  min-height: 48px;
  padding: 4px 0;
  border: none;
  resize: none;
  font-family: inherit;
}

.page-address-edit__footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 12px 16px;
  background: #ffffff;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>