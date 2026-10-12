-- 0017_widen_status_varchar32.up.sql
-- 修复 v1.1 schema 缺陷：
--   orders.status 字段类型 = VARCHAR(16)
--   v1.1 新增状态 'escort_pending_acceptance' = 25 chars → 装不下
--   CHECK constraint 已经接受该值（0009），但列宽拒绝写入
--
-- 修复：把 orders.status 列宽从 16 改成 32，容纳 v1.1 全部状态字符串。
-- 同时把 orders.escort_availabilities.status 也从 16 改成 32（虽然当前只有 3 个状态最长 9 chars，
-- 防御性扩展，避免后续新增状态字串超 16）。
--
-- 兼容性：ALTER TABLE ... TYPE 不影响已有数据；新宽度兼容旧数据。
-- 对应：dev.md §48.P P0（30s 倒计时调查）+ §48.Q P0 修复

ALTER TABLE orders ALTER COLUMN status TYPE VARCHAR(32);

ALTER TABLE escort_availabilities ALTER COLUMN status TYPE VARCHAR(32);
