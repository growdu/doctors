<!--
  src/components/CountdownBadge.vue

  CountdownBadge —— 倒计时徽章（uView Plus plain-tag 风格）

  用途：
    在订单详情 / 服务卡片 / 邀请列表中显示订单/邀请的剩余确认时间。
    单一职责：仅渲染剩余秒数 + 颜色态（红/蓝），不参与业务状态。

  Props：
    - expireAt  (String, 必填) ISO 8601 datetime（任意时区，内部以 Date.now() 对齐）
    - label     (String, 可选) 徽章前缀文案，默认 "待确认剩余"

  行为：
    - mounted：启动 setInterval(1000) 每秒计算一次剩余秒数
    - watch(expireAt)：prop 变化立刻重算（不等下一个 tick）
    - 减到 0 时：clearInterval 停止 tick，颜色保持红色（不归零反弹）
    - beforeUnmount：clearInterval 防内存泄漏

  颜色态：
    - 剩余 < 60s → #ff4d4f（红，uView Plus error 主色）
    - 剩余 ≥ 60s → #1677ff（蓝，uView Plus primary 主色）

  样式：uView Plus plain tag 风格
    - display: inline-flex; align-items: center
    - border: 1px solid <color>; background: #fff
    - border-radius: 4px
    - padding: 4px 8px
    - font-size: 12px

  测试覆盖：src/components/CountdownBadge.test.js
-->
<template>
  <view class="countdown-badge" :style="badgeStyle">
    <text class="countdown-badge__label">{{ label }}</text>
    <text class="countdown-badge__time">剩余 {{ remainingSeconds }}s</text>
  </view>
</template>

<script>
// CountdownBadge —— 倒计时徽章 SFC（Vue 3 Composition API）。
//
// 为何选 Composition API：
//   - 与本项目 src/stores/*.js 风格统一（Vue 3 + ref/computed）
//   - props 在 setup 里直接以 Proxy 形式被 watch 访问，无需 this.xxxx 中转
//   - intervalId 用闭包变量持有（不暴露到 template，不需要响应式）
//
// 为何 watch 选 immediate: true：
//   - mount 之前就希望首屏渲染出真实的剩余秒数（避免一闪 0 再变成真实值）
//   - 同时承担 "prop 变化立刻重算" 的职责，无需再写额外 onChange 钩子

import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue';

export default {
  name: 'CountdownBadge',
  props: {
    expireAt: {
      type: String,
      required: true,
    },
    label: {
      type: String,
      default: '待确认剩余',
    },
  },
  setup(props) {
    const remainingSeconds = ref(0);
    // intervalId 不进 ref：定时器 ID 无需触发渲染更新
    let intervalId = null;

    const tick = () => {
      const target = new Date(props.expireAt).getTime();
      // 防御：expireAt 不是合法 ISO 串时回退到 0（界面显示 0s 红色，外部排查）
      if (Number.isNaN(target)) {
        remainingSeconds.value = 0;
        return;
      }
      const diff = target - Date.now();
      const next = Math.max(0, Math.floor(diff / 1000));
      remainingSeconds.value = next;
      // 减到 0 → 停止 tick（颜色保持红色由 computed 决定，不再变化）
      if (next <= 0 && intervalId !== null) {
        clearInterval(intervalId);
        intervalId = null;
      }
    };

    // initial render & prop 变化立刻重算
    watch(() => props.expireAt, () => tick(), { immediate: true });

    onMounted(() => {
      // mount 时再起一次（即便 setup 阶段 immediate 已跑过，覆盖极短的页面挂载延迟）
      intervalId = setInterval(tick, 1000);
    });

    onBeforeUnmount(() => {
      if (intervalId !== null) {
        clearInterval(intervalId);
        intervalId = null;
      }
    });

    const color = computed(() =>
      remainingSeconds.value < 60 ? '#ff4d4f' : '#1677ff'
    );

    const badgeStyle = computed(() => ({
      color: color.value,
      borderColor: color.value,
    }));

    return {
      remainingSeconds,
      badgeStyle,
    };
  },
};
</script>

<style scoped>
.countdown-badge {
  display: inline-flex;
  align-items: center;
  border-style: solid;
  border-width: 1px;
  border-radius: 4px;
  padding: 4px 8px;
  font-size: 12px;
  line-height: 1.2;
  background-color: #ffffff;
  font-variant-numeric: tabular-nums;
}
.countdown-badge__label {
  margin-right: 4px;
}
.countdown-badge__time {
  /* 数字等宽，避免每秒宽度抖动 */
  font-variant-numeric: tabular-nums;
}
</style>