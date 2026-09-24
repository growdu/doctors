-- 0009_select_escort.down.sql
-- 反向迁移：删除 escort_availabilities 表 + 删除 selected_escort_id / escort_pending_expire_at 字段 +
-- 恢复 lock_owner / lock_expire_at + 恢复旧 CHECK 约束。

-- 1. 删除 escort_availabilities 表
DROP TABLE IF EXISTS escort_availabilities CASCADE;

-- 2. 删除新字段
ALTER TABLE orders DROP COLUMN IF EXISTS selected_escort_id;
ALTER TABLE orders DROP COLUMN IF EXISTS escort_pending_expire_at;
DROP INDEX IF EXISTS idx_orders_selecting;

-- 3. 恢复旧 CHECK 约束（移除 selecting_escort + escort_pending_acceptance）
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','pending_acceptance','accepted','in_service',
  'completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
));

-- 4. 恢复 lock_owner / lock_expire_at 字段（注意：原数据已丢，仅恢复 schema）
ALTER TABLE orders ADD COLUMN lock_owner BIGINT REFERENCES users(id);
ALTER TABLE orders ADD COLUMN lock_expire_at TIMESTAMPTZ;

CREATE INDEX idx_orders_lock ON orders(lock_expire_at)
  WHERE lock_owner IS NOT NULL;