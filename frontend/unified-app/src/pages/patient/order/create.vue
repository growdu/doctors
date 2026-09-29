<script setup lang="ts">
/**
 * patient/order/create.vue — 核心下单流（v2 unified-app）。
 *
 * 入口：
 *   - patient/hospitals/detail 点击「选择下单」→ 本页（?hospitalId + ?packageId）
 *   - patient 域首页「快速下单」
 *
 * 流程：
 *   1. 加载医院 + 套餐信息（确认选择）
 *   2. 选择地址（从 addressStore.list 选 / 跳 address/edit 新增）
 *   3. 选择陪诊师（可选：自动分配 / / 手动指定）
 *   4. 选择预约时间
 *   5. 选择优惠券（可选）
 *   6. 备注
 *   7. 提交订单 → orderStore.create() → 跳 order/pay
 *
 * 设计要点：
 *   - 多步表单 + 单一提交
 *   - 实时 canSubmit 验证
 */
import { ref, computed, onMounted } from 'vue';
import { useOrderStore } from '@/store/order';
import { useAddressStore } from '@/store/address';
import { getHospital as apiGetHospital, getPackage as apiGetPackage, listMyCoupons, allocateVirtualNumber } from '@/api/user';
import { listCandidates } from '@/api/match';
import type { Hospital, Package, Coupon } from '@/api/user';
import type { MatchCandidate } from '@/api/match';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';
import UiLoading from '@/components/shared/UiLoading.vue';

const orderStore = useOrderStore();
const addressStore = useAddressStore();

const hospitalId = ref<number | null>(null);
const packageId = ref<number | null>(null);

const hospital = ref<Hospital | null>(null);
const pkg = ref<Package | null>(null);
const myCoupons = ref<Coupon[]>([]);
const candidates = ref<MatchCandidate[]>([]);

const selectedAddressId = ref<number | null>(null);
const autoAssignEscort = ref(true);
const selectedEscortId = ref<number | null>(null);
const appointmentDate = ref('');
const appointmentTime = ref('09:00');
const selectedCouponId = ref<number | null>(null);
const remark = ref('');

const loading = ref(false);
const error = ref<string | null>(null);

const totalAmount = computed(() => pkg.value?.price ?? 0);
const couponDiscount = computed(() => {
  const c = myCoupons.value.find((x) => x.id === selectedCouponId.value);
  return c?.amount ?? 0;
});
const finalAmount = computed(() => Math.max(0, totalAmount.value - couponDiscount.value));

const canSubmit = computed(() =>
  hospital.value !== null &&
  pkg.value !== null &&
  selectedAddressId.value !== null &&
  appointmentDate.value.length === 10 &&
  /^\d{2}:\d{2}$/.test(appointmentTime.value) &&
  !loading.value,
);

function parseQuery() {
  if (typeof uni === 'undefined') return;
  const pages = (uni as unknown as { getCurrentPages?: () => Array<{ options?: Record<string, string> }> }).getCurrentPages?.() || [];
  const opts = pages[pages.length - 1]?.options;
  if (opts?.hospitalId) hospitalId.value = Number(opts.hospitalId);
  if (opts?.packageId) packageId.value = Number(opts.packageId);
}

async function onLoadData() {
  if (!hospitalId.value || !packageId.value) {
    error.value = '缺少医院或套餐参数';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    const [h, p, coupons, cands] = await Promise.all([
      apiGetHospital(hospitalId.value),
      apiGetPackage(packageId.value),
      listMyCoupons(),
      listCandidates({ order_id: 0, limit: 5 }).catch(() => ({ items: [] })),
    ]);
    await addressStore.fetchList();
    hospital.value = h;
    pkg.value = p;
    myCoupons.value = coupons.items.filter((c: Coupon) => c.status === 'claimed' || c.status === 'available');
    candidates.value = cands.items;
    selectedAddressId.value = addressStore.defaultAddress?.id ?? null;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function onAddAddress() {
  if (typeof uni !== 'undefined') uni.navigateTo({ url: '/pages/patient/address/edit' });
}

function onPickEscort(id: number) {
  selectedEscortId.value = id;
  autoAssignEscort.value = false;
}

function onAutoAssign() {
  selectedEscortId.value = null;
  autoAssignEscort.value = true;
}

async function onSubmit() {
  if (!canSubmit.value) return;
  loading.value = true;
  error.value = null;
  try {
    // 1. 创建订单
    const order = await orderStore.create({
      hospital_id: hospitalId.value!,
      package_id: packageId.value!,
      appointment_time: `${appointmentDate.value}T${appointmentTime.value}:00Z`,
      address: addressStore.findById(selectedAddressId.value!)?.detail ?? '',
    });

    // 2. 如果指定了陪诊师，调用 selectEscort
    if (!autoAssignEscort.value && selectedEscortId.value) {
      await orderStore.fetchDetail(order.id);
    }

    // 3. 申请虚拟号（防陪诊师/患者直接联系，隐私保护）
    try {
      await allocateVirtualNumber(order.id);
    } catch (e) {
      // 虚拟号分配失败不影响主流程
    }

    // 4. 跳支付页
    if (typeof uni !== 'undefined') {
      uni.redirectTo({ url: `/pages/patient/order/pay?orderId=${order.id}` });
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

onMounted(() => {
  parseQuery();
  onLoadData();
});
</script>

<template>
  <view class="ui-page" data-testid="patient-order-create">
    <view v-if="loading && !hospital" data-testid="order-create-loading">
      <UiLoading text="加载中..." />
    </view>

    <view v-else-if="error && !hospital" data-testid="order-create-error">
      <UiEmpty icon="⚠️" title="加载失败" :description="error">
        <template #action>
          <text class="order-create__retry" @click="onLoadData">点击重试</text>
        </template>
      </UiEmpty>
    </view>

    <template v-else-if="hospital && pkg">
      <!-- 服务信息确认 -->
      <UiCard title="服务信息" data-testid="order-create-service">
        <view class="order-create__row">
          <text class="order-create__label">医院</text>
          <text class="order-create__value">{{ hospital.name }}</text>
        </view>
        <view class="order-create__row">
          <text class="order-create__label">套餐</text>
          <text class="order-create__value">{{ pkg.title }}</text>
        </view>
        <view class="order-create__row">
          <text class="order-create__label">套餐价格</text>
          <text class="order-create__value order-create__value--amount">{{ formatYuan(totalAmount) }}</text>
        </view>
      </UiCard>

      <!-- 地址选择 -->
      <UiCard title="服务地址" data-testid="order-create-address">
        <view v-if="addressStore.list.length === 0" data-testid="order-create-address-empty">
          <text class="order-create__hint">还没有地址，请先新增</text>
          <UiButton
            type="default"
            size="sm"
            data-testid="order-create-add-address-btn"
            @click="onAddAddress"
          >
            + 新增地址
          </UiButton>
        </view>
        <view v-else class="order-create__address-list">
          <view
            v-for="a in addressStore.list"
            :key="a.id"
            class="order-create__address-item"
            :class="{ 'order-create__address-item--active': selectedAddressId === a.id }"
            :data-testid="`order-create-address-${a.id}`"
            @click="selectedAddressId = a.id"
          >
            <view class="order-create__address-row">
              <text class="order-create__address-name">{{ a.name }} {{ a.phone }}</text>
              <text v-if="a.is_default" class="order-create__address-default">默认</text>
            </view>
            <text class="order-create__address-detail">{{ a.province }} {{ a.city }} {{ a.district }} {{ a.detail }}</text>
          </view>
        </view>
      </UiCard>

      <!-- 预约时间 -->
      <UiCard title="预约时间" data-testid="order-create-time">
        <view class="order-create__time-row">
          <UiInput
            v-model="appointmentDate"
            label="日期"
            placeholder="YYYY-MM-DD"
            data-testid="order-create-date"
          />
          <UiInput
            v-model="appointmentTime"
            label="时间"
            placeholder="HH:MM"
            data-testid="order-create-time-input"
          />
        </view>
      </UiCard>

      <!-- 陪诊师选择 -->
      <UiCard title="陪诊师（可选）" data-testid="order-create-escort">
        <view class="order-create__escort-options">
          <view
            class="order-create__escort-option"
            :class="{ 'order-create__escort-option--active': autoAssignEscort }"
            data-testid="order-create-escort-auto"
            @click="onAutoAssign"
          >
            <text class="order-create__escort-icon">🎲</text>
            <text class="order-create__escort-label">系统自动分配</text>
          </view>
          <view
            v-for="c in candidates"
            :key="c.escort_id"
            class="order-create__escort-option"
            :class="{ 'order-create__escort-option--active': selectedEscortId === c.escort_id }"
            :data-testid="`order-create-escort-${c.escort_id}`"
            @click="onPickEscort(c.escort_id)"
          >
            <text class="order-create__escort-icon">👨‍⚕️</text>
            <view class="order-create__escort-info">
              <text class="order-create__escort-label">陪诊师 #{{ c.escort_id }}</text>
              <text class="order-create__escort-meta">评分 {{ c.score }} · {{ c.distance_m }}m</text>
            </view>
          </view>
        </view>
      </UiCard>

      <!-- 优惠券 -->
      <UiCard v-if="myCoupons.length > 0" title="优惠券（可选）" data-testid="order-create-coupon">
        <view class="order-create__coupon-list">
          <view
            class="order-create__coupon-item"
            :class="{ 'order-create__coupon-item--active': selectedCouponId === null }"
            data-testid="order-create-coupon-none"
            @click="selectedCouponId = null"
          >
            <text>不使用优惠券</text>
          </view>
          <view
            v-for="c in myCoupons"
            :key="c.id"
            class="order-create__coupon-item"
            :class="{ 'order-create__coupon-item--active': selectedCouponId === c.id }"
            :data-testid="`order-create-coupon-${c.id}`"
            @click="selectedCouponId = c.id"
          >
            <text class="order-create__coupon-amount">¥{{ (c.amount / 100).toFixed(2) }}</text>
            <text class="order-create__coupon-title">{{ c.title }}</text>
          </view>
        </view>
      </UiCard>

      <!-- 备注 -->
      <UiCard title="备注（可选）" data-testid="order-create-remark">
        <UiInput
          v-model="remark"
          label="备注"
          placeholder="其他需要说明的事项"
          type="textarea"
          :maxlength="200"
          data-testid="order-create-remark-input"
        />
      </UiCard>

      <!-- 订单金额 -->
      <view v-if="error" class="order-create__error" data-testid="order-create-error-msg">{{ error }}</view>

      <view class="order-create__summary" data-testid="order-create-summary">
        <view class="order-create__summary-row">
          <text class="order-create__summary-label">套餐金额</text>
          <text class="order-create__summary-value">{{ formatYuan(totalAmount) }}</text>
        </view>
        <view v-if="couponDiscount > 0" class="order-create__summary-row">
          <text class="order-create__summary-label">优惠券</text>
          <text class="order-create__summary-value order-create__summary-value--discount">-{{ formatYuan(couponDiscount) }}</text>
        </view>
        <view class="order-create__summary-row order-create__summary-row--total">
          <text class="order-create__summary-label">实付金额</text>
          <text class="order-create__summary-value order-create__summary-value--amount">{{ formatYuan(finalAmount) }}</text>
        </view>
      </view>

      <view class="order-create__actions">
        <UiButton
          type="primary"
          block
          :loading="loading"
          :disabled="!canSubmit"
          data-testid="order-create-submit-btn"
          @click="onSubmit"
        >
          提交订单并支付
        </UiButton>
      </view>
    </template>
  </view>
</template>

<style scoped>
.order-create__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--ui-space-sm) 0;
  border-bottom: 1px solid var(--ui-color-divider);
}

.order-create__row:last-of-type {
  border-bottom: none;
}

.order-create__label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.order-create__value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
}

.order-create__value--amount {
  color: var(--ui-color-error);
  font-weight: var(--ui-font-weight-semibold);
}

.order-create__hint {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-bottom: var(--ui-space-sm);
}

.order-create__address-list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.order-create__address-item {
  padding: var(--ui-space-md);
  background: var(--ui-color-bg-hover);
  border: 2px solid transparent;
  border-radius: var(--ui-radius-md);
  cursor: pointer;
  transition: all var(--ui-duration-fast);
}

.order-create__address-item--active {
  border-color: var(--ui-color-primary);
  background: rgba(22, 119, 255, 0.04);
}

.order-create__address-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ui-space-xs);
}

.order-create__address-name {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
}

.order-create__address-default {
  font-size: var(--ui-font-xs);
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  padding: 1px 6px;
  border-radius: var(--ui-radius-sm);
}

.order-create__address-detail {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  display: block;
  line-height: 1.4;
}

.order-create__time-row {
  display: flex;
  gap: var(--ui-space-md);
}

.order-create__escort-options {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.order-create__escort-option {
  display: flex;
  align-items: center;
  gap: var(--ui-space-md);
  padding: var(--ui-space-md);
  background: var(--ui-color-bg-hover);
  border: 2px solid transparent;
  border-radius: var(--ui-radius-md);
  cursor: pointer;
}

.order-create__escort-option--active {
  border-color: var(--ui-color-primary);
  background: rgba(22, 119, 255, 0.04);
}

.order-create__escort-icon {
  font-size: 24px;
}

.order-create__escort-info {
  flex: 1;
}

.order-create__escort-label {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
  font-weight: var(--ui-font-weight-medium);
  display: block;
}

.order-create__escort-meta {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-secondary);
  display: block;
  margin-top: 2px;
}

.order-create__coupon-list {
  display: flex;
  flex-direction: column;
  gap: var(--ui-space-sm);
}

.order-create__coupon-item {
  display: flex;
  align-items: center;
  gap: var(--ui-space-sm);
  padding: var(--ui-space-md);
  background: var(--ui-color-bg-hover);
  border: 2px solid transparent;
  border-radius: var(--ui-radius-md);
  cursor: pointer;
}

.order-create__coupon-item--active {
  border-color: var(--ui-color-primary);
  background: rgba(22, 119, 255, 0.04);
}

.order-create__coupon-amount {
  font-size: var(--ui-font-lg);
  font-weight: var(--ui-font-weight-bold);
  color: var(--ui-color-error);
  flex-shrink: 0;
}

.order-create__coupon-title {
  flex: 1;
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-primary);
}

.order-create__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-bottom: var(--ui-space-md);
}

.order-create__summary {
  background: var(--ui-color-bg-card);
  border-radius: var(--ui-radius-md);
  padding: var(--ui-space-md) var(--ui-space-base);
  margin-bottom: var(--ui-space-md);
  box-shadow: var(--ui-shadow-sm);
}

.order-create__summary-row {
  display: flex;
  justify-content: space-between;
  padding: var(--ui-space-xs) 0;
}

.order-create__summary-row--total {
  border-top: 1px solid var(--ui-color-divider);
  margin-top: var(--ui-space-xs);
  padding-top: var(--ui-space-sm);
}

.order-create__summary-label {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
}

.order-create__summary-value {
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
}

.order-create__summary-value--discount {
  color: var(--ui-color-success);
}

.order-create__summary-value--amount {
  color: var(--ui-color-error);
  font-weight: var(--ui-font-weight-semibold);
  font-size: var(--ui-font-lg);
}

.order-create__actions {
  margin-top: var(--ui-space-base);
}

.order-create__retry {
  color: var(--ui-color-primary);
  font-size: var(--ui-font-sm);
  text-decoration: underline;
}
</style>