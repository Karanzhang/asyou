-- 0001_init.sql: initial schema for asyou (SQLite)

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  display_name TEXT,
  role TEXT NOT NULL DEFAULT 'user',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS nodes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  host TEXT NOT NULL,
  api_port INTEGER,
  bind_port INTEGER,
  tls_enabled INTEGER DEFAULT 1,
  auth_token TEXT,
  last_heartbeat DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME
);

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

CREATE TABLE IF NOT EXISTS proxy_stats (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  proxy_id INTEGER NOT NULL,
  timestamp DATETIME NOT NULL,
  bytes_in INTEGER DEFAULT 0,
  bytes_out INTEGER DEFAULT 0,
  conn_count INTEGER DEFAULT 0
);

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

CREATE TABLE IF NOT EXISTS api_keys (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name TEXT,
  token_hash TEXT NOT NULL,
  scopes TEXT,
  revoked INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
