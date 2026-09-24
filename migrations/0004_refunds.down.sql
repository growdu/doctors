-- 0004_refunds.down.sql
-- 撤销 0004：删表 + 索引。

DROP INDEX IF EXISTS idx_refunds_order;
DROP INDEX IF EXISTS idx_refunds_payment;
DROP TABLE IF EXISTS refund_policies;
DROP TABLE IF EXISTS refunds;