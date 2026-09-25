// src/api/index.js
//
// 统一 API 模块入口 —— re-export 全部 api 子模块 + 命名空间聚合。
// （plan Task M1）
//
// 用法：
//   // 1) named 导入（按业务域精确取）
//   import { getOrder, getHospitals, getMyCoupons } from '@/api';
//
//   // 2) namespace 导入（一次拿全）
//   import api from '@/api';
//   api.hospital.getHospitals({ city_id: 1 });
//   api.coupon.getMyCoupons();
//   api.review.submitReview({ order_id: 7, escort_id: 11, rating: 5 });
//
// 约定：
//   - 每个 api 子模块各自可独立 import（store / 页面 / 组件按需）
//   - default 导出是 namespace 对象，便于 IDE 跳转与 type 自动提示
//   - 不在 default export 里塞 utils/request.js —— request 是跨 api 共用底层，不属于任一业务命名空间

import * as candidates from './candidates.js';
import * as order from './order.js';
import * as hospital from './hospital.js';
import * as address from './address.js';
import * as coupon from './coupon.js';
import * as review from './review.js';
import * as virtualnumber from './virtualnumber.js';

// named re-export：让 `import { xxx } from '@/api'` 也能工作
export * from './candidates.js';
export * from './order.js';
export * from './hospital.js';
export * from './address.js';
export * from './coupon.js';
export * from './review.js';
export * from './virtualnumber.js';

/**
 * 命名空间聚合（默认导出）。
 */
const api = {
  candidates,
  order,
  hospital,
  address,
  coupon,
  review,
  virtualnumber,
};

export default api;