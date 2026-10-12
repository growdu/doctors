-- 0018_widen_order_events_status.up.sql
-- 修复 §48.T P0：order_events.from_status / to_status 字段类型 = VARCHAR(16)
-- v1.1 新增状态 'escort_pending_acceptance' = 25 chars → 装不下
-- 影响：SelectEscort / ConfirmAccept / RejectAccept 等所有 InsertEvent 调用返回 500000
--        （handler 返回 500，但订单状态机正确转移；事件日志丢失）
--
-- 与 0017 同因：v1.1 增加了状态字符串但忘了同步扩展事件表列宽。
--
-- 修复：把 order_events.from_status / to_status 列宽从 16 改成 32。
-- 兼容性：ALTER TABLE ... TYPE 不影响已有数据；新宽度兼容旧数据。
-- 对应：dev.md §48.T（v2.1 event insert 500 bug）+ §48.S e2e 已知限制

ALTER TABLE order_events ALTER COLUMN from_status TYPE VARCHAR(32);
ALTER TABLE order_events ALTER COLUMN to_status TYPE VARCHAR(32);
