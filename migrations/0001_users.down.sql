-- 0001_users.down.sql
DROP INDEX IF EXISTS idx_users_phone;
DROP INDEX IF EXISTS idx_users_wx_unionid;
DROP INDEX IF EXISTS idx_users_role_status;
DROP TABLE IF EXISTS users;