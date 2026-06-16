import fetch from 'node-fetch'

/** asyou Node.js SDK */

export interface Proxy {
  id: number
  name: string
  type: string
  local_ip: string
  local_port: number
  status: string
  node_id?: number
}

export interface Node {
  id: number
  name: string
  host: string
  bind_port: number
}

export class AsyouError extends Error {
  constructor(msg: string, public status?: number) {
    super(msg)
    this.name = 'AsyouError'
  }
}

export class Client {
  private token = ''

  constructor(public baseURL: string = 'http://localhost:8080') {}

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const url = `${this.baseURL}${path}`
    const res = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...(this.token ? { Authorization: `Bearer ${this.token}` } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
    })

    if (!res.ok) {
      let msg = `HTTP ${res.status}`
      try {
        const err = (await res.json()) as { error?: string }
        if (err.error) msg = err.error
      } catch {}
      throw new AsyouError(msg, res.status)
    }

    const text = await res.text()
    if (!text) return undefined as T
    return JSON.parse(text) as T
  }

  async login(email: string, password: string): Promise<void> {
    const res = await this.request<{ access_token: string }>(
      'POST',
      '/api/v1/auth/login',
      { email, password }
    )
    this.token = res.access_token
  }

  async register(
    email: string,
    password: string,
    displayName?: string
  ): Promise<void> {
    await this.request('POST', '/api/v1/auth/register', {
      email,
      password,
      display_name: displayName || email.split('@')[0],
    })
    await this.login(email, password)
  }

  async listProxies(): Promise<Proxy[]> {
    return this.request<Proxy[]>('GET', '/api/v1/proxies') ?? []
  }

  async createProxy(
    name: string,
    type = 'tcp',
    localPort = 8080,
    nodeId?: number
  ): Promise<Proxy> {
    return this.request<Proxy>('POST', '/api/v1/proxies', {
      name,
      type,
      local_port: localPort,
      ...(nodeId ? { node_id: nodeId } : {}),
    })
  }

  async deleteProxy(id: number): Promise<void> {
    await this.request('DELETE', `/api/v1/proxies/${id}`)
  }

  async proxyAction(id: number, action: 'start' | 'stop' | 'reload'): Promise<void> {
    await this.request('POST', `/api/v1/proxies/${id}/action`, { action })
  }

  async listNodes(): Promise<Node[]> {
    return this.request<Node[]>('GET', '/api/v1/nodes') ?? []
  }

  /** One-click: create + start a tunnel. */
  async expose(localPort: number, name?: string, nodeId?: number): Promise<Proxy> {
    const tunnelName = name || `node-tunnel-${localPort}`
    const proxy = await this.createProxy(tunnelName, 'tcp', localPort, nodeId)
    await this.proxyAction(proxy.id, 'start')
    return proxy
  }
}
