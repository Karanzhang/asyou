-- 0005_frp_version.sql: track frp version per node for compatibility

ALTER TABLE nodes ADD COLUMN frp_version TEXT DEFAULT '';

CREATE TABLE IF NOT EXISTS frp_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  version TEXT NOT NULL UNIQUE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
