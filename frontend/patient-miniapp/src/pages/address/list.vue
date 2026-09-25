<!--
  src/pages/address/list.vue

  地址管理页 —— CRUD + 默认地址设置（plan Task M7）
  （spec §4.4 + plan v1 重构）

  页面流向：
    入口：个人中心 menu「地址管理」/ 订单创建页选择地址
       → 本页
       → onLoad+mounted 调 useAddressStore().loadList() 拉所有地址
       → 渲染地址卡片列表 + 「+ 新增地址」按钮
       → 长按卡片 → 删除 / 编辑菜单
       → 点击「设为默认」 → store.setDefault(id)

  模板要点：
    - 顶部 <u-navbar>「地址管理」+ 自动返回
    - 主体 ListView 循环地址卡片
        卡片：收件人 + 手机号 + 详细地址 + 「默认」徽标 + 操作区（设为默认 / 删除）
    - 空状态：u-empty「暂无地址」+ 引导新增按钮
    - 底部「+ 新增地址」浮动按钮（fixed bottom）
    - loading / error / empty / loaded 四态机

  数据来源：
    - 列表镜像自 useAddressStore().list（store 暴露的 ref）

  测试覆盖：src/pages/address/list.test.js
-->
<template>
  <view class="page-address" data-test="address-page">
    <u-navbar title="地址管理" :auto-back="true" />

    <!-- loading -->
    <view v-if="loading && addresses.length === 0" class="page-address__loading" data-test="loading">
      <u-skeleton :rows="3" :title="true" />
    </view>

    <!-- error -->
    <view v-else-if="loadError" class="page-address__error" data-test="error-state">
      <u-empty text="加载失败" mode="data" />
      <u-button type="primary" plain data-test="retry-btn" @click="onRetry">重试</u-button>
    </view>

    <!-- empty -->
    <view v-else-if="addresses.length === 0" class="page-address__empty" data-test="empty-state">
      <u-empty text="暂无地址" mode="list" />
    </view>

    <!-- list -->
    <view v-else class="page-address__list" data-test="address-list">
      <view
        v-for="a in addresses"
        :key="a.id"
        class="page-address__card"
        :class="{ 'page-address__card--default': a.id === defaultId }"
        :data-test="'address-card-' + a.id"
        :data-address-id="a.id"
      >
        <view class="page-address__card-body">
          <view class="page-address__card-row">
            <text class="page-address__card-name">{{ a.recipient }}</text>
            <text class="page-address__card-phone">{{ a.phone }}</text>
            <text
              v-if="a.id === defaultId"
              class="page-address__card-default"
              data-test="default-badge"
            >默认</text>
          </view>
          <text class="page-address__card-detail">{{ a.detail }}</text>
        </view>
        <view class="page-address__card-actions">
          <u-button
            v-if="a.id !== defaultId"
            size="mini"
            plain
            type="primary"
            :data-test="'set-default-' + a.id"
            @click="onSetDefault(a)"
          >设为默认</u-button>
          <u-button
            size="mini"
            plain
            type="error"
            :data-test="'delete-' + a.id"
            @click="onDelete(a)"
          >删除</u-button>
        </view>
      </view>
    </view>

    <!-- 浮动「+ 新增地址」按钮（弹层表单占位） -->
    <view class="page-address__bottom">
      <u-button
        type="primary"
        data-test="add-address-btn"
        @click="onAdd"
      >+ 新增地址</u-button>
    </view>

    <!-- 新增地址弹层（占位：v1 直接 prompt + store.add；后续接表单页） -->
    <view v-if="showAddDialog" class="page-address__dialog-mask" data-test="add-dialog">
      <view class="page-address__dialog">
        <text class="page-address__dialog-title">新增地址</text>
        <view class="page-address__dialog-field">
          <text class="page-address__dialog-label">收件人</text>
          <input
            v-model="draft.recipient"
            class="page-address__dialog-input"
            data-test="input-recipient"
            placeholder="姓名"
          />
        </view>
        <view class="page-address__dialog-field">
          <text class="page-address__dialog-label">手机号</text>
          <input
            v-model="draft.phone"
            class="page-address__dialog-input"
            data-test="input-phone"
            placeholder="11 位手机号"
          />
        </view>
        <view class="page-address__dialog-field">
          <text class="page-address__dialog-label">详细地址</text>
          <input
            v-model="draft.detail"
            class="page-address__dialog-input"
            data-test="input-detail"
            placeholder="街道 + 门牌号"
          />
        </view>
        <view class="page-address__dialog-buttons">
          <u-button size="small" plain data-test="dialog-cancel" @click="showAddDialog = false">取消</u-button>
          <u-button
            size="small"
            type="primary"
            data-test="dialog-confirm"
            @click="onAddConfirm"
          >保存</u-button>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
// 地址管理页 —— Vue 3 Options API（与项目既有页面风格统一 —— order/index / home）。
//
// 设计要点：
//   - 三态机：loading / error / loaded + empty / loaded + list
//   - 长按 / 点击操作：「设为默认」/「删除」直接调 store（无中间页）
//   - 新增地址：v1 用 inline dialog（recipient / phone / detail）；后续 plan 接 picker 地图
//   - 上限 5：后端 CodeConflict 由 store 抛错 → toast 提示

import { useAddressStore } from '@/stores/address.js';

const EMPTY_DRAFT = () => ({ recipient: '', phone: '', detail: '' });

export default {
  name: 'AddressListPage',
  data() {
    return {
      addresses: [],
      defaultId: null,
      loading: false,
      loadError: false,
      showAddDialog: false,
      draft: EMPTY_DRAFT(),
    };
  },
  computed: {
    addressStore() {
      return useAddressStore();
    },
  },
  methods: {
    async fetchList() {
      this.loading = true;
      this.loadError = false;
      try {
        const r = await this.addressStore.loadList();
        this.addresses = Array.isArray(r) ? r : [];
        this.defaultId = this.addressStore.defaultId;
      } catch (_e) {
        this.loadError = true;
        this.addresses = [];
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '加载失败，请重试', icon: 'none' });
        }
      } finally {
        this.loading = false;
      }
    },

    onRetry() {
      this.fetchList();
    },

    /** 「设为默认」 */
    async onSetDefault(a) {
      if (!a || !a.id) return;
      try {
        await this.addressStore.setDefault(a.id);
        this.defaultId = a.id;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '已设为默认', icon: 'success' });
        }
        await this.fetchList();
      } catch (_e) {
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '设置失败', icon: 'none' });
        }
      }
    },

    /** 「删除」 → confirm → store.remove */
    onDelete(a) {
      if (!a || !a.id) return;
      const doDel = async () => {
        try {
          await this.addressStore.remove(a.id);
          if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
            uni.showToast({ title: '已删除', icon: 'success' });
          }
          await this.fetchList();
        } catch (_e) {
          if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
            uni.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      };
      if (typeof uni === 'undefined' || typeof uni.showModal !== 'function') {
        doDel();
        return;
      }
      uni.showModal({
        title: '删除地址',
        content: '确定要删除该地址吗？',
        success: (res) => {
          if (res.confirm) doDel();
        },
      });
    },

    /** 「+ 新增地址」 → 打开 dialog */
    onAdd() {
      this.draft = EMPTY_DRAFT();
      this.showAddDialog = true;
    },

    /** dialog 「保存」 → store.add → 关闭 dialog */
    async onAddConfirm() {
      const d = this.draft || {};
      if (!d.recipient || !d.phone || !d.detail) {
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '请填写完整', icon: 'none' });
        }
        return;
      }
      try {
        await this.addressStore.add({
          recipient: d.recipient,
          phone: d.phone,
          detail: d.detail,
        });
        this.showAddDialog = false;
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({ title: '已添加', icon: 'success' });
        }
        await this.fetchList();
      } catch (e) {
        if (typeof uni !== 'undefined' && typeof uni.showToast === 'function') {
          uni.showToast({
            title: (e && e.message) || '添加失败（上限 5 条）',
            icon: 'none',
          });
        }
      }
    },
  },
  onLoad() {
    // 占位
  },
  mounted() {
    this.fetchList();
  },
};
</script>

<style lang="scss" scoped>
.page-address {
  min-height: 100vh;
  background: #f5f7fa;
  padding-bottom: 96px;
}

.page-address__loading,
.page-address__empty,
.page-address__error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.page-address__error .u-button { margin-top: 16px; width: 50%; }

.page-address__list {
  padding: 12px 16px;
}

.page-address__card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.page-address__card--default {
  border: 1px solid #1989fa;
}

.page-address__card-body {
  padding-bottom: 10px;
  border-bottom: 1px dashed #f0f0f0;
}

.page-address__card-row {
  display: flex;
  align-items: center;
  margin-bottom: 6px;
}

.page-address__card-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-right: 12px;
}

.page-address__card-phone {
  font-size: 14px;
  color: #606266;
  flex: 1;
}

.page-address__card-default {
  font-size: 11px;
  background: #1989fa;
  color: #fff;
  padding: 2px 8px;
  border-radius: 4px;
  flex-shrink: 0;
}

.page-address__card-detail {
  font-size: 13px;
  color: #606266;
  display: block;
  line-height: 20px;
}

.page-address__card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 10px;
}

.page-address__bottom {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: #fff;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  z-index: 10;
  .u-button { width: 100%; }
}

.page-address__dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-address__dialog {
  width: 80%;
  background: #fff;
  border-radius: 12px;
  padding: 20px 16px;
}

.page-address__dialog-title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  text-align: center;
}

.page-address__dialog-field {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.page-address__dialog-label {
  font-size: 14px;
  color: #909399;
  width: 80px;
  flex-shrink: 0;
}

.page-address__dialog-input {
  flex: 1;
  font-size: 14px;
  color: #303133;
}

.page-address__dialog-buttons {
  display: flex;
  gap: 12px;
  margin-top: 16px;
  .u-button { flex: 1; }
}
</style>