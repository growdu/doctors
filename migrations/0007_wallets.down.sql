-- 0007_wallets.down.sql
-- 撤销 0007：删索引 + 删表 + 删 orders.completed_at 列。

DROP INDEX IF EXISTS idx_billings_user_created;
DROP INDEX IF EXISTS idx_withdrawals_status;
DROP INDEX IF EXISTS idx_withdrawals_user_created;
DROP INDEX IF EXISTS idx_wallets_user;
DROP TABLE IF EXISTS billings;
DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS wallets;

DROP INDEX IF EXISTS idx_orders_completed;
ALTER TABLE orders DROP COLUMN IF EXISTS completed_at;