/**
 * MSW handlers 集合（v1 + v2 合并）。
 *
 * 组装点（12 模块）：
 *   - orderHandlers  （v2 增量适配）
 *   - reportHandlers （v2 增量适配）
 *   - userHandlers   （v1 增量）
 *   - escortHandlers （v1 增量）
 *   - refundHandlers （v1 增量）
 *   - walletHandlers （v1 增量）
 *   - workOrderHandlers （v1 增量补齐 — H1）
 *   - reviewHandlers    （v1 增量补齐 — H2）
 *   - messageHandlers   （v1 增量补齐 — H3）
 *   - sosHandlers       （v1 增量补齐 — H4）
 *   - patientHandlers   （v1 增量补齐 — H5）
 *   - hospitalHandlers  （v1 增量补齐 — H6）
 *   - packageHandlers   （v1 增量补齐 — H7）
 *   - couponHandlers    （v1 增量补齐 — H8）
 *
 * 对应 spec：l2-api-gap §2.3 admin 12 API 清单。
 */
import { orderHandlers } from './admin/orders';
import { reportHandlers } from './admin/reports';
import { userHandlers } from './admin/users';
import { escortHandlers } from './admin/escorts';
import { refundHandlers } from './admin/refunds';
import { walletHandlers } from './admin/wallets';
import { workOrderHandlers } from './admin/work_orders';
import { reviewHandlers } from './admin/reviews';
import { messageHandlers } from './admin/messages';
import { sosHandlers } from './admin/sos';
import { patientHandlers } from './admin/patients';
import { hospitalHandlers } from './admin/hospitals';
import { packageHandlers } from './admin/packages';
import { couponHandlers } from './admin/coupons';

export const handlers = [
  ...orderHandlers,
  ...reportHandlers,
  ...userHandlers,
  ...escortHandlers,
  ...refundHandlers,
  ...walletHandlers,
  ...workOrderHandlers,
  ...reviewHandlers,
  ...messageHandlers,
  ...sosHandlers,
  ...patientHandlers,
  ...hospitalHandlers,
  ...packageHandlers,
  ...couponHandlers,
];