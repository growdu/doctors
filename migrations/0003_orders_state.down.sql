-- 0003_orders_state.down.sql
-- 撤销 0003：删索引 + 删列 + 收紧 CHECK 约束回退到 11 态。

DROP INDEX IF EXISTS idx_orders_lock;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','accepted','in_service','completed',
  'reviewed','refunding','refunded','closed','canceled'
));
ALTER TABLE orders DROP COLUMN IF EXISTS lock_expire_at;
ALTER TABLE orders DROP COLUMN IF EXISTS lock_owner;