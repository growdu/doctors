/**
 * MSW handlers 集合（v2 增量适配）。
 *
 * 组装点：`orderHandlers`（订单模块）+ `reportHandlers`（看板模块）。
 * 后续 v3+ 可在此追加 escorts / refunds / wallets 等模块 handler。
 */
import { orderHandlers } from './admin/orders';
import { reportHandlers } from './admin/reports';

export const handlers = [...orderHandlers, ...reportHandlers];