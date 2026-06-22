import type { User, Node, Proxy, ProxyStats, AuditLog, ApiKey, NodeStatusResponse } from '../types'

const BASE = '/api/v1'

export function getToken(): string | null {
  return localStorage.getItem('asyou_token')
}

export function setToken(token: string) {
  localStorage.setItem('asyou_token', token)
}

export function clearToken() {
  localStorage.removeItem('asyou_token')
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(`${BASE}${path}`, { ...options, headers })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || body.code || 'request failed')
  }
  const ct = res.headers.get('content-type') || ''
  if (ct.includes('application/json')) {
    return res.json()
  }
  return res.text() as unknown as T
}

// Auth
export function login(email: string, password: string) {
  return request<{ access_token: string; expires_in: number }>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export function register(email: string, password: string, display_name: string) {
  return request<User>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, display_name }),
  })
}

export function getMe() {
  return request<User>('/users/me')
}

// Nodes
export function listNodes() {
  return request<Node[]>('/nodes')
}

export function getNode(id: number) {
  return request<Node>(`/nodes/${id}`)
}

export function createNode(data: Partial<Node>) {
  return request<Node>('/nodes', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteNode(id: number) {
  return request<void>(`/nodes/${id}`, { method: 'DELETE' })
}

export function getNodeStatus(id: number) {
  return request<NodeStatusResponse>(`/nodes/${id}/status`)
}

// Proxies
export function listProxies() {
  return request<Proxy[]>('/proxies')
}

export function getProxy(id: number) {
  return request<{ proxy: Proxy; frps_client_id?: string }>(`/proxies/${id}`)
}

export function createProxy(data: {
  name: string
  type: string
  local_ip?: string
  local_port: number
  remote_port?: number
  subdomain?: string
  custom_domains?: string[]
  node_id?: number
}) {
  return request<Proxy>('/proxies', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteProxy(id: number) {
  return request<void>(`/proxies/${id}`, { method: 'DELETE' })
}

export function proxyAction(id: number, action: 'start' | 'stop' | 'reload') {
  return request<void>(`/proxies/${id}/action`, {
    method: 'POST',
    body: JSON.stringify({ action }),
  })
}

export function getProxyStats(id: number, limit = 60) {
  return request<ProxyStats[]>(`/proxies/${id}/stats?limit=${limit}`)
}

// Audit
export function listAuditLogs(limit = 50) {
  return request<AuditLog[]>(`/audit-logs?limit=${limit}`)
}

// API Keys
export function listApiKeys() {
  return request<ApiKey[]>('/api-keys')
}

export function createApiKey(name?: string) {
  return request<{ token: string }>('/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export function deleteApiKey(id: number) {
  return request<void>(`/api-keys/${id}`, { method: 'DELETE' })
}
