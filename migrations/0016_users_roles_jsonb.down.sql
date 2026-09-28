-- 0016_users_roles_jsonb.down.sql
-- 配套 0016_users_roles_jsonb.up.sql 回滚。

DROP INDEX IF EXISTS idx_users_roles_gin;
ALTER TABLE users DROP COLUMN IF EXISTS roles;