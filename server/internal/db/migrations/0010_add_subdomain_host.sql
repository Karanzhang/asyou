-- 0010_add_subdomain_host.sql: store subdomain_host per node
-- Used for scoping subdomain uniqueness across nodes that share the same domain.
ALTER TABLE nodes ADD COLUMN subdomain_host TEXT DEFAULT '';
