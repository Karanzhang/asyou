import React, { useEffect, useState, useCallback } from 'react'
import { listNodes, createNode, deleteNode, getNodeStatus } from '../api/client'
import type { Node as AsyouNode, NodeStatusResponse } from '../types'

export default function NodeList() {
  const [nodes, setNodes] = useState<AsyouNode[]>([])
  const [statuses, setStatuses] = useState<Record<number, NodeStatusResponse>>({})
  const [loadingStatus, setLoadingStatus] = useState<Record<number, boolean>>({})
  const [expanded, setExpanded] = useState<Record<number, boolean>>({})
  const [showCreate, setShowCreate] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: string } | null>(null)

  const showToast = (msg: string, type = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const load = useCallback(async () => {
    try { setNodes(await listNodes()) }
    catch { showToast('Failed to load nodes', 'error') }
  }, [])

  useEffect(() => { load() }, [load])

  const toggleStatus = async (nodeId: number) => {
    if (expanded[nodeId]) {
      setExpanded(prev => ({ ...prev, [nodeId]: false }))
      return
    }
    setExpanded(prev => ({ ...prev, [nodeId]: true }))
    if (!statuses[nodeId]) {
      setLoadingStatus(prev => ({ ...prev, [nodeId]: true }))
      try {
        const data = await getNodeStatus(nodeId)
        setStatuses(prev => ({ ...prev, [nodeId]: data }))
      } catch {
        showToast('Failed to load frps status', 'error')
      } finally {
        setLoadingStatus(prev => ({ ...prev, [nodeId]: false }))
      }
    }
  }

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
    if (!n.last_heartbeat) return false
    const diff = Date.now() - new Date(n.last_heartbeat).getTime()
    return diff < 5 * 60 * 1000 // 5 min threshold
  }

  return (
    <div>
      {toast && <div className={`toast ${toast.type}`}>{toast.msg}</div>}
      <div className="page-header">
        <h1>Nodes</h1>
        <button className="btn btn-primary" onClick={() => setShowCreate(true)}>+ Add Node</button>
      </div>

      {showCreate && (
        <CreateNodeForm onDone={() => { setShowCreate(false); load() }} onToast={showToast} />
      )}

      {/* Stats summary */}
      <div className="stats-grid" style={{ marginBottom: '1rem' }}>
        <div className="stat-card">
          <div className="label">Total Nodes</div>
          <div className="value">{nodes.length}</div>
        </div>
        <div className="stat-card">
          <div className="label">Online</div>
          <div className="value" style={{ color: 'var(--success)' }}>{nodes.filter(isOnline).length}</div>
        </div>
        <div className="stat-card">
          <div className="label">Offline</div>
          <div className="value" style={{ color: 'var(--danger)' }}>{nodes.filter(n => !isOnline(n)).length}</div>
        </div>
      </div>

      <div className="card">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th></th>
                <th>Status</th>
                <th>Name</th>
                <th>Host:Port</th>
                <th>Region</th>
                <th>Weight</th>
                <th>Live Conns</th>
                <th>Traffic</th>
                <th>Heartbeat</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {nodes.length === 0 && <tr><td colSpan={10} className="empty">No nodes registered.</td></tr>}
              {nodes.map(n => {
                const st = statuses[n.id]
                const si = st?.server_info
                const isExpanded = expanded[n.id]
                const isLoading = loadingStatus[n.id]

                return (
                  <React.Fragment key={n.id}>
                    <tr>
                      <td>
                        <button className="btn btn-outline btn-sm" onClick={() => toggleStatus(n.id)} style={{ padding: '0.1rem 0.4rem', fontSize: '0.75rem' }}>
                          {isExpanded ? '▼' : '▶'}
                        </button>
                      </td>
                      <td>
                        <span className={`badge badge-${isOnline(n) ? 'running' : 'stopped'}`}>
                          {isOnline(n) ? 'online' : 'offline'}
                        </span>
                      </td>
                      <td>
                        <strong>{n.name}</strong>
                        {si && <span style={{ marginLeft: '0.4rem', fontSize: '0.7rem', color: 'var(--text-muted)' }}>v{si.version}</span>}
                        {n.score !== undefined && n.score > 0 && (
                          <span style={{ marginLeft: '0.3rem', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                            score:{n.score.toFixed(1)}
                          </span>
                        )}
                      </td>
                      <td>{n.host}:{n.bind_port}</td>
                      <td>
                        {n.region ? (
                          <span title={`${n.country || ''} ${n.city || ''}`}>
                            {n.region}{n.country ? ` (${n.country})` : ''}
                          </span>
                        ) : '—'}
                      </td>
                      <td>{n.weight ?? '1.0'}</td>
                      <td style={{ fontWeight: 600 }}>
                        {isLoading ? '…' : si ? `${si.curConns} / ${si.clientCounts}` : '—'}
                      </td>
                      <td style={{ fontSize: '0.8rem' }}>
                        {isLoading ? '…' : si ? formatBytes(si.totalTrafficIn + si.totalTrafficOut) : '—'}
                      </td>
                      <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
                        {n.last_heartbeat
                          ? new Date(n.last_heartbeat).toLocaleString()
                          : '—'}
                      </td>
                      <td>
                        <button className="btn btn-danger btn-sm" onClick={() => handleDelete(n.id)}>Delete</button>
                      </td>
                    </tr>
                    {isExpanded && (
                      <tr key={`n-${n.id}-detail`}>
                        <td colSpan={10} style={{ padding: 0, background: 'var(--bg)' }}>
                          <div style={{ padding: '0.8rem 1.5rem' }}>
                            {isLoading ? (
                              <p style={{ color: 'var(--text-muted)' }}>Loading frps status…</p>
                            ) : si ? (
                              <div>
                                <div style={{ display: 'flex', gap: '2rem', marginBottom: '0.8rem', fontSize: '0.85rem' }}>
                                  <span>frps v{si.version}</span>
                                  <span>Clients: {si.clientCounts}</span>
                                  <span>Current: {si.curConns}</span>
                                  <span>↓ {formatBytes(si.totalTrafficIn)}</span>
                                  <span>↑ {formatBytes(si.totalTrafficOut)}</span>
                                </div>
                                {st?.proxies && st.proxies.length > 0 && (
                                  <>
                                    <p style={{ fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>Active Proxies on frps:</p>
                                    <table style={{ fontSize: '0.8rem', width: '100%' }}>
                                      <thead>
                                        <tr style={{ color: 'var(--text-muted)' }}>
                                          <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Name</th>
                                          <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Status</th>
                                          <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Local</th>
                                          <th style={{ textAlign: 'left', padding: '0.2rem 0.5rem' }}>Remote</th>
                                          <th style={{ textAlign: 'right', padding: '0.2rem 0.5rem' }}>Conns</th>
                                          <th style={{ textAlign: 'right', padding: '0.2rem 0.5rem' }}>Traffic</th>
                                        </tr>
                                      </thead>
                                      <tbody>
                                        {st.proxies.map(p => (
                                          <tr key={p.name}>
                                            <td style={{ padding: '0.2rem 0.5rem' }}>{p.name}</td>
                                            <td style={{ padding: '0.2rem 0.5rem' }}>{p.status}</td>
                                            <td style={{ padding: '0.2rem 0.5rem' }}>{p.conf ? `${p.conf.localIP || '127.0.0.1'}:${p.conf.localPort || '?'}` : '—'}</td>
                                            <td style={{ padding: '0.2rem 0.5rem' }}>{p.conf?.remotePort ? `0.0.0.0:${p.conf.remotePort}` : '—'}</td>
                                            <td style={{ padding: '0.2rem 0.5rem', textAlign: 'right' }}>{p.curConns}</td>
                                            <td style={{ padding: '0.2rem 0.5rem', textAlign: 'right' }}>{formatBytes(p.todayTrafficIn + p.todayTrafficOut)}</td>
                                          </tr>
                                        ))}
                                      </tbody>
                                    </table>
                                  </>
                                )}
                                {(!st?.proxies || st.proxies.length === 0) && (
                                  <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>No active proxies on this frps node.</p>
                                )}
                              </div>
                            ) : (
                              <p style={{ color: 'var(--text-muted)' }}>Failed to reach frps admin API. Check dashboard credentials.</p>
                            )}
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
