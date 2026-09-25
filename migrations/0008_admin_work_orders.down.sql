-- 0008_admin_work_orders.down.sql
DROP INDEX IF EXISTS idx_work_orders_subject;
DROP INDEX IF EXISTS idx_work_orders_assignee;
DROP INDEX IF EXISTS idx_work_orders_status_priority;
DROP TABLE IF EXISTS work_orders;