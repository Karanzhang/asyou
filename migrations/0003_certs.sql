-- 0003_certs.sql: certificate storage for automated TLS (ACME)

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
