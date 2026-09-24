// src/utils/format.js
//
// 通用格式化工具 —— 金额 / 日期时间 / 手机号脱敏。
// 单一职责：纯函数，无副作用；可在 uni runtime 与 jest 单测中直接调用。
//
// 设计依据：
//   - formatMoney：spec §10 钱包展示统一 `¥xxx.xx`
//   - formatDateTime：订单详情 / 服务卡片统一 `yyyy-MM-dd HH:mm`
//   - maskPhone：敏感信息脱敏（仅展示前 3 + 后 4）
//
// 错误策略：
//   - formatMoney 负数抛 RangeError（资金不能为负；调用方应先校验）
//   - formatDateTime 接受 ISO 字符串或 Date；无效输入原样返回占位 `-`
//   - maskPhone 非 11 位原样返回（兜底，避免异常打断列表渲染）
//
// 后续 plan 扩展：formatDistance（候选陪诊师卡片显示 km）、formatRating 等。

/**
 * 格式化金额：`¥xxx.xx`；负数抛 RangeError。
 *
 * @param {number} amount 金额（单位：元）
 * @returns {string} `¥<amount.toFixed(2)>`
 * @example
 *   formatMoney(100)       // '¥100.00'
 *   formatMoney(99.999)    // '¥100.00'
 *   formatMoney(0)         // '¥0.00'
 */
export function formatMoney(amount) {
  if (typeof amount !== 'number' || Number.isNaN(amount)) {
    throw new RangeError(`formatMoney: amount must be a number, got ${amount}`);
  }
  if (amount < 0) {
    throw new RangeError(`formatMoney: amount must be >= 0, got ${amount}`);
  }
  return `¥${amount.toFixed(2)}`;
}

/**
 * 格式化日期时间：`yyyy-MM-dd HH:mm`（本地时区）。
 *
 * 接受 ISO 字符串或 Date 对象；无效输入返回 `'-'` 占位（兜底渲染）。
 *
 * @param {string|Date|number|null|undefined} input
 * @returns {string}
 */
export function formatDateTime(input) {
  if (input === null || input === undefined || input === '') return '-';
  const d = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(d.getTime())) return '-';
  const pad = (n) => n.toString().padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/**
 * 手机号脱敏：11 位 → `前3****后4`；非 11 位原样返回（兜底）。
 *
 * @param {string} phone
 * @returns {string}
 */
export function maskPhone(phone) {
  if (typeof phone !== 'string') return '';
  if (phone.length !== 11) return phone;
  return `${phone.substring(0, 3)}****${phone.substring(7)}`;
}

export default {
  formatMoney,
  formatDateTime,
  maskPhone,
};