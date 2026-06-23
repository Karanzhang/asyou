-- 0009_unique_subdomain_per_node.sql
-- Each subdomain must be unique per node, since frps routes by subdomain name.
-- SQLite UNIQUE treats NULLs as distinct, so multiple NULL subdomains are fine.

CREATE UNIQUE INDEX IF NOT EXISTS idx_proxies_node_subdomain
  ON proxies(node_id, subdomain)
  WHERE subdomain IS NOT NULL AND subdomain != '';
