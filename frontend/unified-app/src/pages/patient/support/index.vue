<script setup lang="ts">
/**
 * patient/support/index.vue — 客服与帮助（v2 unified-app）。
 *
 * 入口：profile「客服中心」」/ patient 域 Tab「我的」附近
 *   → FAQ 列表（mock）
 *   → 联系客服（电话 + 在线咨询）
 *   → 关于版本
 */
import { ref } from 'vue';
import UiCard from '@/components/shared/UiCard.vue';
import UiButton from '@/components/shared/UiButton.vue';

const faqOpen = ref<string | null>(null);

interface FAQ {
  q: string;
  a: string;
}

const FAQS: FAQ[] = [
  { q: '如何预约陪诊师？', a: '在「首页」选择医院 → 选择套餐 → 选择陪诊师或由系统分配 → 提交订单 → 支付 → 陪诊师确认后服务生效。' },
  { q: '订单可以取消吗？', a: '订单状态为「待陪诊师接单」或「已确认」时可取消；服务中或已完成订单不可取消，可申请退款。' },
  { q: '退款多久到多久？', a: '退款申请提交后，1-3 个工作日内审核。审核通过后原路退回，微信 / 支付宝即时到账。' },
  { q: '陪诊师如何收费？', a: '陪诊师收费 = 套餐价格；您无需额外支付小费。套餐价格已包含服务费。' },
  { q: '如何切换患者 / 陪诊师 / 管理员？', a: '在「我的」→「切换激活角色」中切换；切换后界面和功能会随 active_role 切换。' },
];

function toggleFaq(q: string) {
  faqOpen.value = faqOpen.value === q ? null : q;
}

function onCallPhone() {
  if (typeof uni !== 'undefined') {
    uni.showModal({
      title: '客服热线',
      content: '400-123-4567（工作时间 9:00-21:00）',
      showCancel: false,
    });
  }
}

function onOnlineChat() {
  if (typeof uni !== 'undefined') {
    uni.showToast({ title: '在线咨询接入中', icon: 'none' });
  }
}
</script>

<template>
  <view class="ui-page" data-testid="patient-support">
    <UiCard title="联系方式" data-testid="support-contact">
      <view class="support__contact-row">
        <UiButton type="primary" size="sm" data-testid="support-call-btn" @click="onCallPhone">
          📞 电话客服
        </UiButton>
        <UiButton type="default" size="sm" data-testid="support-chat-btn" @click="onOnlineChat">
          💬 在线咨询
        </UiButton>
      </view>
    </UiCard>

    <view class="support__section-title" data-testid="support-faq-title">常见问题</view>
    <view data-testid="support-faq-list">
      <UiCard
        v-for="(f, i) in FAQS"
        :key="i"
        :data-testid="`support-faq-${i}`"
        shadow="none"
      >
        <view class="support__faq-q" @click="toggleFaq(f.q)">
          <text class="support__faq-icon">Q</text>
          <text class="support__faq-q-text">{{ f.q }}</text>
          <text class="support__faq-arrow">{{ faqOpen === f.q ? '−' : '+' }}</text>
        </view>
        <view v-if="faqOpen === f.q" class="support__faq-a">
          <text class="support__faq-icon support__faq-icon--a">A</text>
          <text class="support__faq-a-text">{{ f.a }}</text>
        </view>
      </UiCard>
    </view>

    <view class="support__about" data-testid="support-about">
      <text class="support__about-version">Doctors 统一 App v2.0.0-sp2</text>
      <text class="support__about-copyright">© 2026 Doctors Team</text>
    </view>
  </view>
</template>

<style scoped>
.support__contact-row {
  display: flex;
  gap: var(--ui-space-md);
}

.support__section-title {
  font-size: var(--ui-font-md);
  font-weight: var(--ui-font-weight-semibold);
  color: var(--ui-color-text-primary);
  padding: var(--ui-space-base) 0 var(--ui-space-sm);
}

.support__faq-q {
  display: flex;
  align-items: center;
  gap: var(--ui-space-sm);
  cursor: pointer;
}

.support__faq-icon {
  width: 24px;
  height: 24px;
  background: var(--ui-color-primary);
  color: var(--ui-color-text-inverse);
  font-size: var(--ui-font-xs);
  font-weight: var(--ui-font-weight-bold);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.support__faq-icon--a {
  background: var(--ui-color-success);
}

.support__faq-q-text {
  flex: 1;
  font-size: var(--ui-font-base);
  color: var(--ui-color-text-primary);
}

.support__faq-arrow {
  font-size: var(--ui-font-lg);
  color: var(--ui-color-text-secondary);
  flex-shrink: 0;
}

.support__faq-a {
  display: flex;
  gap: var(--ui-space-sm);
  margin-top: var(--ui-space-md);
  padding-top: var(--ui-space-md);
  border-top: 1px solid var(--ui-color-divider);
}

.support__faq-a-text {
  flex: 1;
  font-size: var(--ui-font-sm);
  color: var(--ui-color-text-secondary);
  line-height: 1.6;
}

.support__about {
  text-align: center;
  padding: var(--ui-space-lg) 0;
}

.support__about-version,
.support__about-copyright {
  font-size: var(--ui-font-xs);
  color: var(--ui-color-text-disabled);
  display: block;
}

.support__about-version {
  margin-bottom: 2px;
}
</style>