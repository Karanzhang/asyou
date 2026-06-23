-- 0001_init.sql: complete asyou schema (consolidated)
-- This single file replaces all incremental migration files.
-- CREATE TABLE IF NOT EXISTS + CREATE UNIQUE INDEX IF NOT EXISTS
-- are idempotent for new databases.
-- ALTER TABLE ADD COLUMN statements are kept for existing databases;
-- "duplicate column name" errors are silently ignored by the runner.

-- ============================================================
-- Users
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  display_name TEXT,
  role TEXT NOT NULL DEFAULT 'user',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME
);

-- ============================================================
-- frps Nodes (all columns from all migrations)
-- ============================================================
CREATE TABLE IF NOT EXISTS nodes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  host TEXT NOT NULL,
  api_port INTEGER,
  bind_port INTEGER,
  tls_enabled INTEGER DEFAULT 1,
  auth_token TEXT,
  -- 0004: geo scheduling
  region TEXT DEFAULT '',
  country TEXT DEFAULT '',
  city TEXT DEFAULT '',
  latitude REAL DEFAULT 0,
  longitude REAL DEFAULT 0,
  max_connections INTEGER DEFAULT 100,
  weight REAL DEFAULT 1.0,
  is_active INTEGER DEFAULT 1,
  -- 0005: frp version
  frp_version TEXT DEFAULT '',
  -- 0006: dashboard credentials
  dashboard_port INTEGER DEFAULT 7500,
  dashboard_user TEXT DEFAULT 'admin',
  dashboard_pwd TEXT DEFAULT '',
  -- 0008: port range
  port_range_start INTEGER DEFAULT 31000,
  port_range_end INTEGER DEFAULT 31999,
  -- 0010: subdomain host
  subdomain_host TEXT DEFAULT '',
  -- original columns
  last_heartbeat DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME
);

-- ALTER TABLE statements for existing databases that were created
-- before these columns were added. The runner ignores "duplicate column" errors.
ALTER TABLE nodes ADD COLUMN region TEXT DEFAULT '';
ALTER TABLE nodes ADD COLUMN country TEXT DEFAULT '';
ALTER TABLE nodes ADD COLUMN city TEXT DEFAULT '';
ALTER TABLE nodes ADD COLUMN latitude REAL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN longitude REAL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN max_connections INTEGER DEFAULT 100;
ALTER TABLE nodes ADD COLUMN weight REAL DEFAULT 1.0;
ALTER TABLE nodes ADD COLUMN is_active INTEGER DEFAULT 1;
ALTER TABLE nodes ADD COLUMN frp_version TEXT DEFAULT '';
ALTER TABLE nodes ADD COLUMN dashboard_port INTEGER DEFAULT 7500;
ALTER TABLE nodes ADD COLUMN dashboard_user TEXT DEFAULT 'admin';
ALTER TABLE nodes ADD COLUMN dashboard_pwd TEXT DEFAULT '';
ALTER TABLE nodes ADD COLUMN port_range_start INTEGER DEFAULT 31000;
ALTER TABLE nodes ADD COLUMN port_range_end INTEGER DEFAULT 31999;
ALTER TABLE nodes ADD COLUMN subdomain_host TEXT DEFAULT '';

-- ============================================================
-- Node health metrics
-- ============================================================
CREATE TABLE IF NOT EXISTS node_health (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id INTEGER NOT NULL,
  latency_ms INTEGER DEFAULT 0,
  current_connections INTEGER DEFAULT 0,
  cpu_load REAL DEFAULT 0,
  memory_usage REAL DEFAULT 0,
  bandwidth_mbps REAL DEFAULT 0,
  recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Proxies (tunnels)
-- ============================================================
CREATE TABLE IF NOT EXISTS proxies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  node_id INTEGER,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  local_ip TEXT DEFAULT '127.0.0.1',
  local_port INTEGER NOT NULL,
  remote_port INTEGER,
  subdomain TEXT,
  custom_domains TEXT,
  host_header_rewrite TEXT,
  http_user TEXT,
  http_pass TEXT,
  enable_tls INTEGER DEFAULT 0,
  status TEXT DEFAULT 'stopped',
  annotations TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME,
  UNIQUE(user_id, name)
);

-- 0007: per-user proxy name uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_proxies_user_name ON proxies(user_id, name);

-- 0009: subdomain uniqueness per node
CREATE UNIQUE INDEX IF NOT EXISTS idx_proxies_node_subdomain
  ON proxies(node_id, subdomain)
  WHERE subdomain IS NOT NULL AND subdomain != '';

-- ============================================================
-- Proxy stats (traffic)
-- ============================================================
CREATE TABLE IF NOT EXISTS proxy_stats (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  proxy_id INTEGER NOT NULL,
  timestamp DATETIME NOT NULL,
  bytes_in INTEGER DEFAULT 0,
  bytes_out INTEGER DEFAULT 0,
  conn_count INTEGER DEFAULT 0
);

-- ============================================================
-- Audit logs
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_user_id INTEGER,
  action_type TEXT NOT NULL,
  resource_type TEXT,
  resource_id INTEGER,
  detail TEXT,
  ip TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- API keys
-- ============================================================
CREATE TABLE IF NOT EXISTS api_keys (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name TEXT,
  token_hash TEXT NOT NULL,
  scopes TEXT,
  revoked INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Certificates (ACME/TLS)
-- ============================================================
CREATE TABLE IF NOT EXISTS certificates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  proxy_id INTEGER NOT NULL,
  domain TEXT NOT NULL,
  cert_pem TEXT NOT NULL,
  key_pem TEXT NOT NULL,
  issuer TEXT NOT NULL DEFAULT 'letsencrypt',
  expires_at DATETIME NOT NULL,
  auto_renew INTEGER DEFAULT 1,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME,
  UNIQUE(domain, user_id),
  FOREIGN KEY (proxy_id) REFERENCES proxies(id) ON DELETE CASCADE
);

-- ============================================================
-- frp versions registry
-- ============================================================
CREATE TABLE IF NOT EXISTS frp_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  version TEXT NOT NULL UNIQUE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Password reset tokens
-- ============================================================
CREATE TABLE IF NOT EXISTS password_resets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  token TEXT NOT NULL,
  expires_at DATETIME NOT NULL,
  used INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
