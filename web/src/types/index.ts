export interface User {
  id: number
  email: string
  display_name: string
  role: string
  created_at: string
  updated_at: string
}

export interface Node {
  id: number
  name: string
  host: string
  api_port: number
  bind_port: number
  tls_enabled: boolean
  auth_token?: string
  frp_version?: string
  region?: string
  country?: string
  city?: string
  latitude?: number
  longitude?: number
  max_connections?: number
  weight?: number
  is_active?: boolean
  score?: number
  last_heartbeat: string
  created_at: string
  updated_at: string
}

export interface Proxy {
  id: number
  user_id: number
  node_id: number | null
  name: string
  type: string
  local_ip: string
  local_port: number
  remote_port: number | null
  subdomain: string | null
  custom_domains: string | null
  host_header_rewrite: string | null
  http_user: string | null
  http_pass: string | null
  enable_tls: boolean
  status: string
  annotations: string | null
  created_at: string
  updated_at: string
}

export interface ProxyStats {
  id: number
  proxy_id: number
  timestamp: string
  bytes_in: number
  bytes_out: number
  conn_count: number
}

export interface AuditLog {
  id: number
  actor_user_id: number | null
  action_type: string
  resource_type: string
  resource_id: number | null
  detail: string | null
  ip: string
  created_at: string
}

export interface ApiKey {
  id: number
  user_id: number
  name: string | null
  scopes: string | null
  revoked: boolean
  created_at: string
}

export interface FrpsServerInfo {
  version: string
  total_conns: number
  current_conns: number
  total_traffic_in: number
  total_traffic_out: number
  uptime: string
}

export interface FrpsProxyEntry {
  name: string
  type: string
  status: string
  local_addr: string
  remote_addr: string
  bytes_in: number
  bytes_out: number
  conn_count: number
  last_err: string
}

export interface NodeStatusResponse {
  node_id: number
  node_name: string
  server_info: FrpsServerInfo
  proxies: FrpsProxyEntry[]
}
