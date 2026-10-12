-- 0017_widen_status_varchar32.down.sql
-- 回滚：把 status 列宽改回 VARCHAR(16)。如果已有超过 16 字符的状态值，ALTER 会失败。

ALTER TABLE orders ALTER COLUMN status TYPE VARCHAR(16);
ALTER TABLE escort_availabilities ALTER COLUMN status TYPE VARCHAR(16);
