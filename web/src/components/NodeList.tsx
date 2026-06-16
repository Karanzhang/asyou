import { useEffect, useState, useCallback } from 'react'
import { listNodes, createNode, deleteNode } from '../api/client'
import type { Node as AsyouNode } from '../types'

export default function NodeList() {
  const [nodes, setNodes] = useState<AsyouNode[]>([])
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

      <div className="card">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Host</th>
                <th>Bind Port</th>
                <th>API Port</th>
                <th>TLS</th>
                <th>Heartbeat</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {nodes.length === 0 && <tr><td colSpan={7} className="empty">No nodes registered.</td></tr>}
              {nodes.map(n => (
                <tr key={n.id}>
                  <td><strong>{n.name}</strong></td>
                  <td>{n.host}</td>
                  <td>{n.bind_port}</td>
                  <td>{n.api_port || '—'}</td>
                  <td>{n.tls_enabled ? '✓' : '—'}</td>
                  <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{n.last_heartbeat || '—'}</td>
                  <td>
                    <button className="btn btn-danger btn-sm" onClick={() => handleDelete(n.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function CreateNodeForm({ onDone, onToast }: { onDone: () => void; onToast: (m: string, t?: string) => void }) {
  const [name, setName] = useState('')
  const [host, setHost] = useState('')
  const [bindPort, setBindPort] = useState('7000')
  const [apiPort, setApiPort] = useState('')
  const [authToken, setAuthToken] = useState('')
  const [busy, setBusy] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    try {
      await createNode({
        name, host,
        bind_port: parseInt(bindPort),
        api_port: apiPort ? parseInt(apiPort) : undefined as any,
        auth_token: authToken || undefined as any,
      } as any)
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
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
          <div className="form-group">
            <label>Name</label>
            <input value={name} onChange={e => setName(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>Host</label>
            <input value={host} onChange={e => setHost(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>Bind Port</label>
            <input type="number" value={bindPort} onChange={e => setBindPort(e.target.value)} />
          </div>
          <div className="form-group">
            <label>API Port</label>
            <input type="number" value={apiPort} onChange={e => setApiPort(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Auth Token</label>
            <input value={authToken} onChange={e => setAuthToken(e.target.value)} />
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
