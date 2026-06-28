import React, { useEffect, useState, useCallback } from 'react'
import { listNodes, createNode, updateNode, deleteNode, getNodeStatus } from '../api/client'
import type { Node as AsyouNode, NodeStatusResponse, User } from '../types'

export default function NodeList({ user }: { user: User | null }) {
  const isAdmin = user?.role === 'admin'
  const [nodes, setNodes] = useState<AsyouNode[]>([])
  const [statuses, setStatuses] = useState<Record<number, NodeStatusResponse>>({})
  const [showCreate, setShowCreate] = useState(false)
  const [editNodeId, setEditNodeId] = useState<number | null>(null)
  const [detailNodeId, setDetailNodeId] = useState<number | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: string } | null>(null)

  const showToast = (msg: string, type = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const load = useCallback(async () => {
    try {
      const nodeList = await listNodes()
      setNodes(nodeList)
      // Load live status for all nodes in parallel
      nodeList.forEach(n => {
        getNodeStatus(n.id).then(data => {
          setStatuses(prev => ({ ...prev, [n.id]: data }))
        }).catch(() => {})
      })
    } catch {
      showToast('Failed to load nodes', 'error')
    }
  }, [])

  useEffect(() => { load(); const t = setInterval(load, 10000); return () => clearInterval(t) }, [load])

  // Aggregate stats across all nodes
  const agg = { clients: 0, curConns: 0, trafficIn: 0, trafficOut: 0, proxies: 0 }
  Object.values(statuses).forEach(st => {
    if (!st?.server_info) return
    agg.clients += st.server_info.clientCounts || 0
    agg.curConns += st.server_info.curConns || 0
    agg.trafficIn += st.server_info.totalTrafficIn || 0
    agg.trafficOut += st.server_info.totalTrafficOut || 0
    if (st.proxies) agg.proxies += st.proxies.length
  })

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this node?')) return
    try {
      await deleteNode(id)
      showToast('Node deleted')
      load()
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  const isOnline = (n: AsyouNode) => {
    // Prefer live status from frps admin API
    const st = statuses[n.id]
    if (st?.server_info) {
      return true // successfully reached frps admin API
    }
    // Fall back to heartbeat from DB
    if (!n.last_heartbeat) return false
    const diff = Date.now() - new Date(n.last_heartbeat).getTime()
    return diff < 5 * 60 * 1000 // 5 min threshold
  }

  return (
    <div>
      {toast && <div className={`toast ${toast.type}`}>{toast.msg}</div>}
      <div className="page-header">
        <h1>Nodes</h1>
        {isAdmin && <button className="btn btn-primary" onClick={() => setShowCreate(true)}>+ Add Node</button>}
      </div>

      {isAdmin && showCreate && (
        <CreateNodeForm onDone={() => { setShowCreate(false); load() }} onToast={showToast} />
      )}

      {/* Aggregated frps dashboard stats */}
      <div className="stats-grid" style={{ marginBottom: '1rem' }}>
        <div className="stat-card">
          <div className="label">frps Nodes</div>
          <div className="value">{nodes.length}</div>
        </div>
        <div className="stat-card">
          <div className="label">Online</div>
          <div className="value" style={{ color: 'var(--success)' }}>{nodes.filter(isOnline).length}</div>
        </div>
        <div className="stat-card">
          <div className="label">Active Clients</div>
          <div className="value">{agg.clients}</div>
        </div>
        <div className="stat-card">
          <div className="label">Active Proxies</div>
          <div className="value">{agg.proxies}</div>
        </div>
        <div className="stat-card">
          <div className="label">Current Conns</div>
          <div className="value">{agg.curConns}</div>
        </div>
        <div className="stat-card">
          <div className="label">Traffic In</div>
          <div className="value" style={{ fontSize: '0.9rem' }}>{formatBytes(agg.trafficIn)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Traffic Out</div>
          <div className="value" style={{ fontSize: '0.9rem' }}>{formatBytes(agg.trafficOut)}</div>
        </div>
      </div>

      <div className="card">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Status</th>
                <th>Name</th>
                <th>Host:Port</th>
                <th>Clients</th>
                <th>Proxies</th>
                <th>Conns</th>
                <th>Traffic In/Out</th>
                <th>Heartbeat</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {nodes.length === 0 && <tr><td colSpan={9} className="empty">No nodes registered.</td></tr>}
              {nodes.map(n => {
                const st = statuses[n.id]
                const si = st?.server_info
                const isDetail = detailNodeId === n.id
                return (
                  <React.Fragment key={n.id}>
                    <tr onClick={() => setDetailNodeId(isDetail ? null : n.id)} style={{ cursor: 'pointer' }}>
                      <td>
                        <span className={`badge badge-${isOnline(n) ? 'running' : 'stopped'}`}>
                          {isOnline(n) ? 'online' : 'offline'}
                        </span>
                      </td>
                      <td>
                        <strong>{n.name}</strong>
                        {si && <span style={{ marginLeft: '0.4rem', fontSize: '0.7rem', color: 'var(--text-muted)' }}>v{si.version}</span>}
                      </td>
                      <td>{n.host}:{n.bind_port}</td>
                      <td style={{ fontWeight: 600, textAlign: 'center' }}>{si?.clientCounts ?? '…'}</td>
                      <td style={{ textAlign: 'center' }}>{st?.proxies?.length ?? '…'}</td>
                      <td style={{ textAlign: 'center' }}>{si?.curConns ?? '…'}</td>
                      <td style={{ fontSize: '0.8rem', whiteSpace: 'nowrap' }}>
                        {si ? `${formatBytes(si.totalTrafficIn)} ↓ / ${formatBytes(si.totalTrafficOut)} ↑` : '…'}
                      </td>
                      <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
                        {n.last_heartbeat ? new Date(n.last_heartbeat).toLocaleString() : '—'}
                      </td>
                      <td onClick={e => e.stopPropagation()}>
                        {isAdmin && (
                          <>
                            <button className="btn btn-outline btn-sm" onClick={() => setEditNodeId(editNodeId === n.id ? null : n.id)} style={{ marginRight: '0.4rem' }}>Edit</button>
                            <button className="btn btn-danger btn-sm" onClick={() => handleDelete(n.id)}>Delete</button>
                          </>
                        )}
                      </td>
                    </tr>
                    {editNodeId === n.id && (
                      <tr key={`${n.id}-edit`}>
                        <td colSpan={9} style={{ padding: 0, background: 'var(--bg)' }}>
                          <EditNodeForm
                            node={n}
                            onDone={() => { setEditNodeId(null); load() }}
                            onToast={showToast}
                          />
                        </td>
                      </tr>
                    )}
                    {isDetail && st?.proxies && st.proxies.length > 0 && (
                      <tr key={`${n.id}-detail`}>
                        <td colSpan={9} style={{ padding: 0, background: 'var(--bg)' }}>
                          <div style={{ padding: '0.8rem 1.5rem' }}>
                            <p style={{ fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                              Active Proxies on {n.name}:
                            </p>
                            <table style={{ fontSize: '0.8rem', width: '100%' }}>
                              <thead>
                                <tr style={{ color: 'var(--text-muted)' }}>
                                  <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Name</th>
                                  <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Status</th>
                                  <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Client ID</th>
                                  <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Local</th>
                                  <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Remote</th>
                                  <th style={{ textAlign: 'right', padding: '0.2rem 0.5rem' }}>Traffic</th>
                                </tr>
                              </thead>
                              <tbody>
                                {st.proxies.map(p => (
                                  <tr key={p.name}>
                                    <td style={{ padding: '0.2rem 0.5rem' }}>{p.name}</td>
                                    <td style={{ padding: '0.2rem 0.5rem' }}>{p.status}</td>
                                    <td style={{ padding: '0.2rem 0.5rem', fontFamily: 'monospace', fontSize: '0.75rem' }}>{p.clientID || '—'}</td>
                                    <td style={{ padding: '0.2rem 0.5rem' }}>{p.conf ? `${p.conf.localIP || '127.0.0.1'}:${p.conf.localPort || '?'}` : '—'}</td>
                                    <td style={{ padding: '0.2rem 0.5rem' }}>{p.conf?.remotePort ? `0.0.0.0:${p.conf.remotePort}` : '—'}</td>
                                    <td style={{ padding: '0.2rem 0.5rem', textAlign: 'right' }}>{formatBytes(p.todayTrafficIn + p.todayTrafficOut)}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                        </td>
                      </tr>
                    )}
                    {isDetail && (!st?.proxies || st.proxies.length === 0) && (
                      <tr key={`${n.id}-detail`}>
                        <td colSpan={9} style={{ padding: 0, background: 'var(--bg)' }}>
                          <div style={{ padding: '0.8rem 1.5rem', color: 'var(--text-muted)', fontSize: '0.85rem' }}>
                            {st ? 'No active proxies on this node.' : 'Failed to reach frps admin API. Check dashboard credentials.'}
                          </div>
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function CreateNodeForm({ onDone, onToast }: { onDone: () => void; onToast: (m: string, t?: string) => void }) {
  const [name, setName] = useState('')
  const [host, setHost] = useState('')
  const [bindPort, setBindPort] = useState('7000')
  const [apiPort, setApiPort] = useState('')
  const [authToken, setAuthToken] = useState('')
  const [dashboardPort, setDashboardPort] = useState('7500')
  const [dashboardUser, setDashboardUser] = useState('admin')
  const [dashboardPwd, setDashboardPwd] = useState('')
  const [region, setRegion] = useState('')
  const [country, setCountry] = useState('')
  const [city, setCity] = useState('')
  const [latitude, setLatitude] = useState('')
  const [longitude, setLongitude] = useState('')
  const [maxConnections, setMaxConnections] = useState('')
  const [weight, setWeight] = useState('1.0')
  const [subdomainHost, setSubdomainHost] = useState('')
  const [busy, setBusy] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    try {
      const body: Record<string, any> = {
        name, host,
        bind_port: parseInt(bindPort),
        auth_token: authToken || undefined,
      }
      if (apiPort) body.api_port = parseInt(apiPort)
      if (dashboardPort) body.dashboard_port = parseInt(dashboardPort)
      if (dashboardUser) body.dashboard_user = dashboardUser
      if (dashboardPwd) body.dashboard_pwd = dashboardPwd
      if (region) body.region = region
      if (country) body.country = country
      if (city) body.city = city
      if (latitude) body.latitude = parseFloat(latitude)
      if (longitude) body.longitude = parseFloat(longitude)
      if (maxConnections) body.max_connections = parseInt(maxConnections)
      if (weight) body.weight = parseFloat(weight)
      if (subdomainHost) body.subdomain_host = subdomainHost
      await createNode(body)
      onToast('Node created')
      onDone()
    } catch (err: any) {
      onToast(err.message, 'error')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="card">
      <h3>Add Node</h3>
      <form onSubmit={handleSubmit}>
        <fieldset>
          <legend>Basic</legend>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
            <div className="form-group">
              <label>Name *</label>
              <input value={name} onChange={e => setName(e.target.value)} required />
            </div>
            <div className="form-group">
              <label>Host *</label>
              <input value={host} onChange={e => setHost(e.target.value)} required placeholder="IP or domain" />
            </div>
            <div className="form-group">
              <label>Bind Port</label>
              <input type="number" value={bindPort} onChange={e => setBindPort(e.target.value)} />
            </div>
            <div className="form-group">
              <label>API Port (dashboard)</label>
              <input type="number" value={apiPort} onChange={e => setApiPort(e.target.value)} placeholder="7500" />
            </div>
            <div className="form-group">
              <label>Auth Token</label>
              <input value={authToken} onChange={e => setAuthToken(e.target.value)} />
            </div>
            <div className="form-group">
              <label>Dashboard Port</label>
              <input type="number" value={dashboardPort} onChange={e => setDashboardPort(e.target.value)} placeholder="7500" />
            </div>
            <div className="form-group">
              <label>Dashboard User</label>
              <input value={dashboardUser} onChange={e => setDashboardUser(e.target.value)} placeholder="admin" />
            </div>
            <div className="form-group">
              <label>Dashboard Password</label>
              <input type="password" value={dashboardPwd} onChange={e => setDashboardPwd(e.target.value)} placeholder="frps dashboard password" />
            </div>
            <div className="form-group">
              <label>Subdomain Host</label>
              <input value={subdomainHost} onChange={e => setSubdomainHost(e.target.value)} placeholder="e.g. tunnel.example.com" />
              <small style={{ color: 'var(--text-muted)' }}>DNS wildcard *.subdomain_host → this node</small>
            </div>
          </div>
        </fieldset>

        <fieldset style={{ marginTop: '1rem' }}>
          <legend>Scheduling</legend>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '1rem' }}>
            <div className="form-group">
              <label>Weight</label>
              <input type="number" step="0.1" min="0.1" value={weight} onChange={e => setWeight(e.target.value)} />
              <small style={{ color: 'var(--text-muted)' }}>Higher = more tunnels assigned</small>
            </div>
            <div className="form-group">
              <label>Max Connections</label>
              <input type="number" value={maxConnections} onChange={e => setMaxConnections(e.target.value)} placeholder="100" />
            </div>
          </div>
        </fieldset>

        <fieldset style={{ marginTop: '1rem' }}>
          <legend>Geo (for proximity-based scheduling)</legend>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '1rem' }}>
            <div className="form-group">
              <label>Region</label>
              <input value={region} onChange={e => setRegion(e.target.value)} placeholder="ap-east" />
            </div>
            <div className="form-group">
              <label>Country</label>
              <input value={country} onChange={e => setCountry(e.target.value)} placeholder="HK" />
            </div>
            <div className="form-group">
              <label>City</label>
              <input value={city} onChange={e => setCity(e.target.value)} placeholder="Hong Kong" />
            </div>
            <div className="form-group">
              <label>Latitude</label>
              <input type="number" step="any" value={latitude} onChange={e => setLatitude(e.target.value)} placeholder="22.3193" />
            </div>
            <div className="form-group">
              <label>Longitude</label>
              <input type="number" step="any" value={longitude} onChange={e => setLongitude(e.target.value)} placeholder="114.1694" />
            </div>
          </div>
        </fieldset>

        <div style={{ marginTop: '1rem' }}>
          <button className="btn btn-primary" type="submit" disabled={busy}>
            {busy ? 'Creating…' : 'Create'}
          </button>
          <button className="btn btn-outline" type="button" onClick={onDone}>Cancel</button>
        </div>
      </form>
    </div>
  )
}

function EditNodeForm({ node, onDone, onToast }: { node: AsyouNode; onDone: () => void; onToast: (m: string, t?: string) => void }) {
  const [subdomainHost, setSubdomainHost] = useState(node.subdomain_host || '')
  const [dashboardUser, setDashboardUser] = useState(node.dashboard_user || '')
  const [dashboardPwd, setDashboardPwd] = useState('')
  const [authToken, setAuthToken] = useState('')
  const [weight, setWeight] = useState(node.weight?.toString() || '1.0')
  const [maxConnections, setMaxConnections] = useState(node.max_connections?.toString() || '')
  const [busy, setBusy] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    try {
      const body: Record<string, any> = {}
      if (subdomainHost) body.subdomain_host = subdomainHost
      if (dashboardUser) body.dashboard_user = dashboardUser
      if (dashboardPwd) body.dashboard_pwd = dashboardPwd
      if (authToken) body.auth_token = authToken
      if (weight) body.weight = parseFloat(weight)
      if (maxConnections) body.max_connections = parseInt(maxConnections)
      await updateNode(node.id, body)
      onToast('Node updated')
      onDone()
    } catch (err: any) {
      onToast(err.message, 'error')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div style={{ padding: '1rem 1.5rem' }}>
      <h4 style={{ marginBottom: '0.8rem' }}>Edit Node: {node.name}</h4>
      <form onSubmit={handleSubmit}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '1rem' }}>
          <div className="form-group">
            <label>Subdomain Host</label>
            <input value={subdomainHost} onChange={e => setSubdomainHost(e.target.value)} placeholder="tunnel.example.com" />
            <small style={{ color: 'var(--text-muted)' }}>DNS wildcard *.subdomain_host → this node</small>
          </div>
          <div className="form-group">
            <label>Dashboard User</label>
            <input value={dashboardUser} onChange={e => setDashboardUser(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Dashboard Password</label>
            <input type="password" value={dashboardPwd} onChange={e => setDashboardPwd(e.target.value)} placeholder="leave blank to keep" />
          </div>
          <div className="form-group">
            <label>Auth Token</label>
            <input value={authToken} onChange={e => setAuthToken(e.target.value)} placeholder="leave blank to keep" />
          </div>
          <div className="form-group">
            <label>Weight</label>
            <input type="number" step="0.1" min="0.1" value={weight} onChange={e => setWeight(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Max Connections</label>
            <input type="number" value={maxConnections} onChange={e => setMaxConnections(e.target.value)} placeholder="100" />
          </div>
        </div>
        <div style={{ marginTop: '0.8rem' }}>
          <button className="btn btn-primary btn-sm" type="submit" disabled={busy}>
            {busy ? 'Saving…' : 'Save'}
          </button>
          <button className="btn btn-outline btn-sm" type="button" onClick={onDone} style={{ marginLeft: '0.4rem' }}>Cancel</button>
        </div>
      </form>
    </div>
  )
}
