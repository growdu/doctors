-- 0003_orders_state.up.sql
-- 状态机统一：加锁单字段 + CHECK 约束包含新状态。
-- 评审 C-01：04 / 07 状态机不一致，新增 pending_acceptance / settling / disputed 三态。
-- 评审 C-04 部分：锁单字段为 §4.2 order-lock plan 铺路。
--
-- 注意：这是兼容迁移；老数据 status 不在新 CHECK 列表里会失败，需先跑数据迁移脚本（生产单独排期）。

ALTER TABLE orders
  ADD COLUMN lock_owner BIGINT REFERENCES users(id),
  ADD COLUMN lock_expire_at TIMESTAMPTZ;

-- 索引服务于"查锁单即将到期"任务（§4.2 order-lock plan）。
CREATE INDEX idx_orders_lock ON orders(lock_expire_at)
  WHERE lock_owner IS NOT NULL;

-- 替换 CHECK 约束（Postgres 改 CHECK 约束要先 DROP 再 ADD）。
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','pending_acceptance','accepted','in_service',
  'completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
));