import { useState, useEffect, useCallback } from 'react'
import * as bridge from './api/bridge'

export interface UserInfo {
  id: number
  email: string
  display_name: string
}

export default function App() {
  const [user, setUser] = useState<UserInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState<'login' | 'main'>('login')
  const [error, setError] = useState('')

  useEffect(() => {
    bridge.isLoggedIn().then(async (loggedIn) => {
      if (loggedIn) {
        try {
          const u = await bridge.getCurrentUser()
          setUser(u)
          setPage('main')
        } catch {
          await bridge.logout()
        }
      }
      setLoading(false)
    })
  }, [])

  if (loading) {
    return (
      <div className="app" style={{ alignItems: 'center', justifyContent: 'center' }}>
        <div className="spinner" />
      </div>
    )
  }

  if (page === 'login') {
    return <LoginPage
      onLogin={async (email, password) => {
        setError('')
        try {
          await bridge.login(email, password)
          const u = await bridge.getCurrentUser()
          setUser(u)
          setPage('main')
        } catch (err: any) {
          setError(err.message || err)
        }
      }}
      onRegister={async (email, password, name) => {
        setError('')
        try {
          await bridge.register(email, password, name)
          const u = await bridge.getCurrentUser()
          setUser(u)
          setPage('main')
        } catch (err: any) {
          setError(err.message || err)
        }
      }}
      error={error}
    />
  }

  return <MainPage user={user!} onLogout={async () => { await bridge.logout(); setUser(null); setPage('login') }} />
}

// --- Login Page ---

function LoginPage({ onLogin, onRegister, error }: {
  onLogin: (e: string, p: string) => Promise<void>
  onRegister: (e: string, p: string, n: string) => Promise<void>
  error: string
}) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (mode === 'login') {
      await onLogin(email, password)
    } else {
      await onRegister(email, password, name || email.split('@')[0])
    }
  }

  return (
    <div className="app" style={{ alignItems: 'center', justifyContent: 'center' }}>
      <div className="card" style={{ width: 360, padding: '2rem' }}>
        <h1 style={{ textAlign: 'center', color: 'var(--primary)', marginBottom: '0.3rem' }}>asyou</h1>
        <p style={{ textAlign: 'center', color: 'var(--text-muted)', marginBottom: '1.5rem', fontSize: '0.9rem' }}>Desktop Tunnel Client</p>
        {error && <div className="toast error" style={{ position: 'static', transform: 'none', marginBottom: '1rem' }}>{error}</div>}
        <form onSubmit={handleSubmit}>
          {mode === 'register' && (
            <div className="form-group">
              <label>Display Name</label>
              <input value={name} onChange={e => setName(e.target.value)} placeholder="Optional" />
            </div>
          )}
          <div className="form-group">
            <label>Email</label>
            <input type="email" value={email} onChange={e => setEmail(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>Password</label>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)} required />
          </div>
          <button className="btn btn-primary" type="submit" style={{ width: '100%', marginTop: '0.5rem' }}>
            {mode === 'login' ? 'Sign In' : 'Create Account'}
          </button>
        </form>
        <p style={{ textAlign: 'center', marginTop: '1rem', fontSize: '0.85rem', color: 'var(--text-muted)' }}>
          {mode === 'login' ? (
            <>Don't have an account? <a href="#" onClick={e => { e.preventDefault(); setMode('register'); }} style={{ color: 'var(--primary)' }}>Register</a></>
          ) : (
            <>Already have one? <a href="#" onClick={e => { e.preventDefault(); setMode('login'); }} style={{ color: 'var(--primary)' }}>Sign In</a></>
          )}
        </p>
      </div>
    </div>
  )
}

// --- Main Page ---

function MainPage({ user, onLogout }: { user: UserInfo; onLogout: () => void }) {
  const [proxies, setProxies] = useState<any[]>([])
  const [nodes, setNodes] = useState<any[]>([])
  const [ports, setPorts] = useState<any[]>([])
  const [toast, setToast] = useState<{ msg: string; type: string } | null>(null)
  const [selectedPort, setSelectedPort] = useState<number | null>(null)
  const [tunnelName, setTunnelName] = useState('')
  const [busy, setBusy] = useState(false)

  const show = (msg: string, type = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const refresh = useCallback(async () => {
    try {
      const [p, n] = await Promise.all([bridge.listProxies(), bridge.listNodes()])
      setProxies(p)
      setNodes(n)
    } catch { show('Failed to load data', 'error') }
  }, [])

  const scanPorts = useCallback(async () => {
    try {
      setPorts(await bridge.discoverPorts())
    } catch { show('Port scan failed', 'error') }
  }, [])

  useEffect(() => { refresh(); scanPorts() }, [refresh, scanPorts])

  const handleQuickTunnel = async () => {
    if (!selectedPort && !tunnelName) return
    setBusy(true)
    try {
      const name = tunnelName || `tunnel-${selectedPort}`
      const port = selectedPort || 0
      const nodeID = nodes.length > 0 ? nodes[0].id : 0
      const id = await bridge.quickTunnel(name, port, nodeID)
      show(`Tunnel #${id} created & started on port ${port}!`)
      setSelectedPort(null)
      setTunnelName('')
      refresh()
    } catch (err: any) {
      show(err.message || String(err), 'error')
    } finally { setBusy(false) }
  }

  const handleAction = async (id: number, action: 'start' | 'stop') => {
    try {
      if (action === 'start') await bridge.startProxy(id)
      else await bridge.stopProxy(id)
      show(`Proxy ${action}ed`)
      refresh()
    } catch (err: any) { show(err.message, 'error') }
  }

  return (
    <div className="app">
      {toast && <div className={`toast ${toast.type}`}>{toast.msg}</div>}

      <div className="header">
        <h1>asyou</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          <span className="user-info">{user.email}</span>
          <button className="btn btn-outline btn-sm" onClick={onLogout}>Sign Out</button>
        </div>
      </div>

      <div className="main">
        {/* Quick Tunnel Card */}
        <div className="card full quick-tunnel">
          <div className="big-icon">🚀</div>
          <h2>One-Click Tunnel</h2>
          <p>Select a local port or enter a name, then click to create & start</p>
          <div style={{ display: 'flex', gap: '0.5rem', justifyContent: 'center', flexWrap: 'wrap', marginBottom: '1rem' }}>
            <input
              placeholder="Tunnel name (optional)"
              value={tunnelName}
              onChange={e => setTunnelName(e.target.value)}
              style={{ width: 200, padding: '0.5rem 0.7rem', background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', color: 'var(--text)', outline: 'none' }}
            />
            <button className="btn btn-primary" onClick={handleQuickTunnel} disabled={busy || (!selectedPort && !tunnelName)}>
              {busy ? <><span className="spinner" /> Creating…</> : 'Create & Start'}
            </button>
            <button className="btn btn-outline" onClick={scanPorts}>🔄 Scan Ports</button>
          </div>
        </div>

        {/* Discovered Ports */}
        <div className="card">
          <h2>Discovered Ports ({ports.length})</h2>
          <div className="port-grid">
            {ports.length === 0 && <div className="empty">No ports found. Click "Scan Ports".</div>}
            {ports.map((p, i) => (
              <div
                key={i}
                className={`port-item ${selectedPort === p.port ? 'port-item-active' : ''}`}
                style={selectedPort === p.port ? { borderColor: 'var(--primary)' } : {}}
                onClick={() => { setSelectedPort(p.port); setTunnelName('') }}
              >
                <div>
                  <div className="port-num">{p.port}</div>
                  <div className="port-proto">{p.protocol}</div>
                </div>
                <div className="port-proc">{p.process || ''}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Proxies */}
        <div className="card">
          <h2>Your Tunnels ({proxies.length})</h2>
          <div style={{ overflowX: 'auto' }}>
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Type</th>
                  <th>Port</th>
                  <th>Status</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {proxies.length === 0 && <tr><td colSpan={5} className="empty">No tunnels. Use Quick Tunnel above!</td></tr>}
                {proxies.map((p: any) => (
                  <tr key={p.id}>
                    <td><strong>{p.name}</strong></td>
                    <td>{p.type}</td>
                    <td>{p.local_port}</td>
                    <td>
                      <span className={`badge badge-${p.status === 'running' ? 'running' : 'stopped'}`}>{p.status}</span>
                    </td>
                    <td>
                      {p.status === 'stopped' ? (
                        <button className="btn btn-success btn-sm" onClick={() => handleAction(p.id, 'start')}>Start</button>
                      ) : (
                        <button className="btn btn-danger btn-sm" onClick={() => handleAction(p.id, 'stop')}>Stop</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Status Bar */}
      <div className="status-bar">
        <span className={`dot ${nodes.length > 0 ? 'online' : 'offline'}`} />
        <span>{nodes.length > 0 ? `${nodes.length} node(s) configured` : 'No nodes configured'}</span>
        <span style={{ marginLeft: 'auto' }}>{proxies.filter((p: any) => p.status === 'running').length} tunnels active</span>
      </div>
    </div>
  )
}
