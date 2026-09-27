// jest.setup.js
//
// Jest 全局 setup —— 解决 Vue 3 + vue3-jest 下两类测试障碍：
//
// 1. uni-app Page 生命周期钩子（onLoad / onShow / onReady / ...）在
//    Vue 3 Options API SFC 中以顶层方法声明，但 Vue 3 不识别，
//    编译后 wrapper.vm.onLoad === undefined。
//    → 全局 mixin 在 created() 阶段把组件 $options 中这些钩子
//    重新绑回实例，w.vm.onLoad(query) 可调用。
//
// 2. Vue 3 自身的 mounted 生命周期钩子在 mount() 时自动触发，
//    不需要也无法手动 w.vm.mounted() 调用（undefined）。
//    → 测试代码必须移除 `await w.vm.mounted()` 行。
//
// 注：此 setup 不修改任何组件源码，只让单测能在 jsdom 下覆盖完整流程。

import { config } from '@vue/test-utils';

// uni-app Page 钩子清单（按 spec 与 plan 约定）
const UNI_PAGE_HOOKS = [
  'onLoad',
  'onShow',
  'onReady',
  'onPullDownRefresh',
  'onReachBottom',
  'onShareAppMessage',
  'onShareTimeline',
  'onAddToFavorites',
  'onPageScroll',
  'onTabItemTap',
  'onBackPress',
  'onNavigationBarButtonTap',
  'onNavigationBarSearchInputChanged',
  'onNavigationBarSearchInputConfirmed',
  'onNavigationBarSearchInputClicked',
];

// 兜底 mixin：在 created 钩子中遍历组件 $options 中的 uni-app 钩子 + Vue 生命周期，
// 若组件自身声明了对应函数，重新挂到实例上，使 wrapper.vm.xxx() 可调用。
//
// 关键：Vue 3 标准生命周期（mounted / created / beforeUnmount 等）也以顶层
// 方法形式声明 —— Vue 自己会调用，但测试场景中常需在 onLoad(query) 设置初始
// 数据后再次手动触发（如 await w.vm.mounted()），所以一并暴露。
//
// 注意：mixin 必须放在 test 文件 import 之前（@vue/test-utils config
// 在第一次 mount() 时生效），jest setupFiles 在每个 test 文件加载前
// 先行执行，配置已生效。
const VUE_LIFECYCLE_HOOKS = [
  'beforeMount',
  'mounted',
  'beforeUpdate',
  'updated',
  'beforeUnmount',
  'unmounted',
  'errorCaptured',
];

config.global.mixins = [
  {
    created() {
      const opts = this.$options;
      if (!opts) return;
      const ALL_HOOKS = [...UNI_PAGE_HOOKS, ...VUE_LIFECYCLE_HOOKS];
      ALL_HOOKS.forEach((hook) => {
        const fn = opts[hook];
        if (typeof fn === 'function' && typeof this[hook] !== 'function') {
          // 绑定 this 到当前实例；hook 内访问 this.xxx 等于访问 vm.xxx
          this[hook] = fn.bind(this);
        }
      });
    },
  },
];