-- 0007_unique_proxy_name.sql: per-user proxy name uniqueness
-- Each user can have unique proxy names; different users can reuse the same name.
CREATE UNIQUE INDEX IF NOT EXISTS idx_proxies_user_name ON proxies(user_id, name);
