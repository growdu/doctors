/**
 * MSW handlers 集合（v1 + v2 合并）。
 *
 * 组装点（7 模块）：
 *   - orderHandlers  （v2 增量适配）
 *   - reportHandlers （v2 增量适配）
 *   - userHandlers   （v1 增量）
 *   - escortHandlers （v1 增量）
 *   - refundHandlers （v1 增量）
 *   - walletHandlers （v1 增量）
 *
 * 注：本批次未给 patients / work-orders / reviews / messages / sos / settings /
 * finance / reports / coupons / hospitals / packages / audit / dashboard-detail
 * 单独建 handler，handler 文件仅覆盖 admin-web v1 计划的 4 大块（users/escorts/refunds/wallets），
 * 其余页面骨架后续按需扩展。
 */
import { orderHandlers } from './admin/orders';
import { reportHandlers } from './admin/reports';
import { userHandlers } from './admin/users';
import { escortHandlers } from './admin/escorts';
import { refundHandlers } from './admin/refunds';
import { walletHandlers } from './admin/wallets';

export const handlers = [
  ...orderHandlers,
  ...reportHandlers,
  ...userHandlers,
  ...escortHandlers,
  ...refundHandlers,
  ...walletHandlers,
];