-- 0002_orders.down.sql
DROP INDEX IF EXISTS idx_orders_patient_created;
DROP INDEX IF EXISTS idx_orders_status_start;
DROP INDEX IF EXISTS idx_order_events_order;
DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS orders;