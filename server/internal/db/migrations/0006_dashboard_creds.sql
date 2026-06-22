-- 0006_dashboard_creds.sql: store frps dashboard credentials per node
ALTER TABLE nodes ADD COLUMN dashboard_port INTEGER DEFAULT 7500;
ALTER TABLE nodes ADD COLUMN dashboard_user TEXT DEFAULT 'admin';
ALTER TABLE nodes ADD COLUMN dashboard_pwd TEXT DEFAULT '';
