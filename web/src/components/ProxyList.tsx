import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSSE } from '../hooks/useSSE'
import { listProxies, createProxy, proxyAction, listNodes, getProxyStats } from '../api/client'
import type { Proxy as AsyouProxy, Node as AsyouNode, ProxyStats } from '../types'
import TrafficChart from './TrafficChart'

export default function ProxyList() {
  const [proxies, setProxies] = useState<AsyouProxy[]>([])
  const [nodes, setNodes] = useState<AsyouNode[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: string } | null>(null)
  const navigate = useNavigate()

  const showToast = (msg: string, type = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const load = useCallback(async () => {
    try {
      const [p, n] = await Promise.all([listProxies(), listNodes()])
      setProxies(p)
      setNodes(n)
    } catch {
      showToast('Failed to load data', 'error')
    }
  }, [])

  useEffect(() => { load() }, [load])

  // Auto-refresh on SSE proxy updates
  useSSE('proxy_update', () => { load() })

  const handleAction = async (id: number, action: 'start' | 'stop' | 'reload') => {
    try {
      await proxyAction(id, action)
      showToast(`Proxy ${action}ed`)
      load()
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  return (
    <div>
      {toast && <div className={`toast ${toast.type}`}>{toast.msg}</div>}

      <div className="stats-grid">
        <div className="stat-card">
          <div className="label">Total Proxies</div>
          <div className="value">{proxies.length}</div>
        </div>
        <div className="stat-card">
          <div className="label">Running</div>
          <div className="value">{proxies.filter(p => p.status === 'running').length}</div>
        </div>
        <div className="stat-card">
          <div className="label">Stopped</div>
          <div className="value">{proxies.filter(p => p.status === 'stopped').length}</div>
        </div>
        <div className="stat-card">
          <div className="label">Nodes</div>
          <div className="value">{nodes.length}</div>
        </div>
      </div>

      <div className="page-header">
        <h1>Proxies</h1>
        <button className="btn btn-primary" onClick={() => setShowCreate(true)}>+ New Tunnel</button>
      </div>

      {showCreate && (
        <CreateProxyForm
          nodes={nodes}
          onDone={() => { setShowCreate(false); load() }}
          onToast={showToast}
        />
      )}

      <div className="card">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Local</th>
                <th>Remote</th>
                <th>Node</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {proxies.length === 0 && (
                <tr><td colSpan={7} className="empty">No tunnels yet. Create your first one!</td></tr>
              )}
              {proxies.map(p => (
                <tr key={p.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/proxies/${p.id}`)}>
                  <td><strong>{p.name}</strong></td>
                  <td>{p.type}</td>
                  <td>{p.local_ip}:{p.local_port}</td>
                  <td>{p.remote_port ?? '—'}</td>
                  <td>{nodes.find(n => n.id === p.node_id)?.name || '—'}</td>
                  <td>
                    <span className={`badge badge-${p.status === 'running' ? 'running' : 'stopped'}`}>
                      {p.status}
                    </span>
                  </td>
                  <td className="actions" onClick={e => e.stopPropagation()}>
                    {p.status === 'stopped' ? (
                      <button className="btn btn-success btn-sm" onClick={() => handleAction(p.id, 'start')}>Start</button>
                    ) : (
                      <button className="btn btn-warning btn-sm" onClick={() => handleAction(p.id, 'stop')}>Stop</button>
                    )}
                    <button className="btn btn-outline btn-sm" onClick={() => handleAction(p.id, 'reload')}>Reload</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {proxies.length > 0 && <TrafficChart proxyId={proxies[0].id} />}
    </div>
  )
}

function CreateProxyForm({ nodes, onDone, onToast }: { nodes: AsyouNode[]; onDone: () => void; onToast: (m: string, t?: string) => void }) {
  const [name, setName] = useState('')
  const [type, setType] = useState('tcp')
  const [localPort, setLocalPort] = useState('')
  const [remotePort, setRemotePort] = useState('')
  const [subdomain, setSubdomain] = useState('')
  const [nodeId, setNodeId] = useState(nodes[0]?.id?.toString() || '')
  const [busy, setBusy] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    try {
      await createProxy({
        name,
        type,
        local_port: parseInt(localPort),
        remote_port: remotePort ? parseInt(remotePort) : undefined,
        subdomain: subdomain || undefined,
        node_id: nodeId ? parseInt(nodeId) : undefined,
      })
      onToast('Tunnel created')
      onDone()
    } catch (err: any) {
      onToast(err.message, 'error')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="card">
      <h3>New Tunnel</h3>
      <form onSubmit={handleSubmit}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
          <div className="form-group">
            <label>Name</label>
            <input value={name} onChange={e => setName(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>Type</label>
            <select value={type} onChange={e => setType(e.target.value)}>
              <option value="tcp">TCP</option>
              <option value="http">HTTP</option>
              <option value="https">HTTPS</option>
              <option value="udp">UDP</option>
            </select>
          </div>
          <div className="form-group">
            <label>Local Port</label>
            <input type="number" value={localPort} onChange={e => setLocalPort(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>Remote Port (optional)</label>
            <input type="number" value={remotePort} onChange={e => setRemotePort(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Subdomain (optional)</label>
            <input value={subdomain} onChange={e => setSubdomain(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Node</label>
            <select value={nodeId} onChange={e => setNodeId(e.target.value)}>
              <option value="">— Select —</option>
              {nodes.map(n => <option key={n.id} value={n.id}>{n.name}</option>)}
            </select>
          </div>
        </div>
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
