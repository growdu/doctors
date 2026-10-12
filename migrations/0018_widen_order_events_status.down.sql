-- 0018_widen_order_events_status.down.sql
-- 回滚：把 order_events status 列宽改回 VARCHAR(16)。
-- 如果已有超过 16 字符的状态值，ALTER 会失败。

ALTER TABLE order_events ALTER COLUMN from_status TYPE VARCHAR(16);
ALTER TABLE order_events ALTER COLUMN to_status TYPE VARCHAR(16);
