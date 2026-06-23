-- 0008_node_port_range.sql: per-node remote port range for proxy auto-assignment
ALTER TABLE nodes ADD COLUMN port_range_start INTEGER DEFAULT 31000;
ALTER TABLE nodes ADD COLUMN port_range_end INTEGER DEFAULT 31999;
