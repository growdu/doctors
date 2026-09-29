<script setup lang="ts">
/**
 * patient/auth/real-name.vue — 实名认证（v2 unified-app）。
 *
 * 入口：profile「未实名」徽标点击
 *   → 姓名 + 身份证号 + 上传身份证正反面（占位）
 *   → 调 authStore 调用真实认证 API（v2 暂用 mock 占位）
 *
 * 设计要点：
 *   - 姓名 2-20 字
 *   - 身份证 18 位（最后一位 X 允许）
 *   - 提交后置 user.real_name_verified = true
 */
import { ref, computed } from 'vue';
import { useAuthStore } from '@/store/auth';
import { realNameAuth } from '@/api/auth';
import UiCard from '@/components/shared/UiCard.vue';
import UiInput from '@/components/shared/UiInput.vue';
import UiButton from '@/components/shared/UiButton.vue';
import UiEmpty from '@/components/shared/UiEmpty.vue';

const auth = useAuthStore();

const name = ref('');
const idCard = ref('');
const submitting = ref(false);
const success = ref(false);
const error = ref<string | null>(null);

const NAME_RE = /^[\u4e00-\u9fa5·]{2,20}$/;
const ID_CARD_RE = /^\d{17}[\dX]$/;

const canSubmit = computed(() =>
  NAME_RE.test(name.value) &&
  ID_CARD_RE.test(idCard.value) &&
  !submitting.value,
);

async function onSubmit() {
  if (!canSubmit.value) return;
  submitting.value = true;
  error.value = null;
  try {
    await realNameAuth(name.value, idCard.value);
    if (auth.user) {
      auth.user.real_name_verified = true;
    }
    success.value = true;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    submitting.value = false;
  }
}

function onBack() {
  if (typeof uni !== 'undefined') uni.navigateBack({ delta: 1 });
}
</script>

<template>
  <view class="ui-page" data-testid="patient-auth-realname">
    <view v-if="success" data-testid="realname-success">
      <UiEmpty icon="✅" title="实名认证成功" description="您的账号已通过实名认证">
        <template #action>
          <UiButton type="primary" size="sm" data-testid="realname-back-btn" @click="onBack">返回个人中心</UiButton>
        </template>
      </UiEmpty>
    </view>

    <template v-else>
      <view class="realname__tip" data-testid="realname-tip">
        实名认证后可使用全部服务（如下单、SOS一键联系等）。
      </view>

      <UiCard title="实名信息">
        <UiInput
          v-model="name"
          label="真实姓名"
          placeholder="请输入身份证姓名"
          :maxlength="20"
          data-testid="realname-name"
        />
        <UiInput
          v-model="idCard"
          label="身份证号"
          placeholder="18 位身份证号"
          :maxlength="18"
          data-testid="realname-idcard"
        />

        <view v-if="error" class="realname__error" data-testid="realname-error">{{ error }}</view>

        <view class="realname__actions">
          <UiButton
            type="primary"
            block
            :loading="submitting"
            :disabled="!canSubmit"
            data-testid="realname-submit-btn"
            @click="onSubmit"
          >
            提交认证
          </UiButton>
        </view>
      </UiCard>
    </template>
  </view>
</template>

<style scoped>
.realname__tip {
  background: var(--ui-color-bg-hover);
  color: var(--ui-color-text-secondary);
  font-size: var(--ui-font-sm);
  padding: var(--ui-space-md);
  border-radius: var(--ui-radius-md);
  margin-bottom: var(--ui-space-md);
}

.realname__error {
  font-size: var(--ui-font-sm);
  color: var(--ui-color-error);
  margin-top: var(--ui-space-md);
}

.realname__actions {
  margin-top: var(--ui-space-base);
}
</style>