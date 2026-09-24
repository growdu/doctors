// src/utils/trace.js
//
// Trace ID 生成器 —— 注入 uni.request 拦截器，给每个请求打唯一 id 便于后端日志侧按端拆分。
//
// 格式：`mp-{ms}-{rand6}` —— 与后端约定的 patient miniapp 端前缀一致。
//   - ms: DateTime.now() 毫秒数（人类可读时间序）
//   - rand6: 6 位随机数（0..999999）—— 提升同 ms 内多请求唯一性
//
// 设计：纯函数 + 全局 Math.random；无 platform 依赖；单测可直接调用并注入 `now`。
//
// 历史：v1 骨架阶段 `utils/request.js` 内联了一份 `newTraceId`，现统一收敛到本文件；
// `utils/request.js` 已 import 此处导出，重复定义已删除。

/// 患者端 trace id 前缀；与后端 / 日志约定一致（mp = miniapp patient）。
export const TRACE_PREFIX = 'mp';

/**
 * 生成新 trace id。
 *
 * @param {number} [now] 可选注入的 ms（便于单测断言；默认 `Date.now()`）。
 * @returns {string} `mp-<ms13>-<rand6>` 形式；rand 段左补 0 定长。
 * @example
 *   newTraceId()                 // 'mp-1727187600123-458721'
 *   newTraceId(1727187600123)    // 'mp-1727187600123-458721'
 */
export function newTraceId(now) {
  const ts = typeof now === 'number' ? now : Date.now();
  // Math.random 范围 [0,1) → 6 位 base36 字符串 → padEnd 截到 6 位
  const rand = Math.random().toString(36).slice(2, 8).padEnd(6, '0');
  return `${TRACE_PREFIX}-${ts}-${rand}`;
}

export default {
  TRACE_PREFIX,
  newTraceId,
};