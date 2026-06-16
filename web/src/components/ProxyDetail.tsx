import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useSSE } from '../hooks/useSSE'
import { getProxy, proxyAction, getProxyStats, deleteProxy } from '../api/client'
import type { Proxy as AsyouProxy, ProxyStats } from '../types'
import TrafficChart from './TrafficChart'

export default function ProxyDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [proxy, setProxy] = useState<AsyouProxy | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: string } | null>(null)

  const showToast = (msg: string, type = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  useEffect(() => {
    if (!id) return
    getProxy(parseInt(id)).then(setProxy).catch(() => showToast('Failed to load', 'error'))
  }, [id])

  // Auto-refresh on SSE proxy update for this proxy
  useSSE('proxy_update', (data: any) => {
    if (!id || !data) return
    if (data.id === parseInt(id)) {
      getProxy(parseInt(id)).then(setProxy).catch(() => {})
    }
  })

  if (!proxy) return <p>Loading…</p>

  const handleAction = async (action: 'start' | 'stop' | 'reload') => {
    try {
      await proxyAction(proxy.id, action)
      showToast(`Proxy ${action}ed`)
      const updated = await getProxy(proxy.id)
      setProxy(updated)
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  const handleDelete = async () => {
    if (!confirm('Delete this tunnel?')) return
    try {
      await deleteProxy(proxy.id)
      navigate('/')
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  let annotation: { error?: string; when?: string } | null = null
  if (proxy.annotations) {
    try { annotation = JSON.parse(proxy.annotations) } catch {}
  }

  return (
    <div>
      {toast && <div className={`toast ${toast.type}`}>{toast.msg}</div>}
      <div className="page-header">
        <h1>{proxy.name}</h1>
        <div>
          <button className="btn btn-outline" onClick={() => navigate('/')}>← Back</button>
        </div>
      </div>

      <div className="card">
        <div className="detail-grid">
          <div className="detail-item"><div className="label">Status</div><div className="val"><span className={`badge badge-${proxy.status === 'running' ? 'running' : 'stopped'}`}>{proxy.status}</span></div></div>
          <div className="detail-item"><div className="label">Type</div><div className="val">{proxy.type}</div></div>
          <div className="detail-item"><div className="label">Local</div><div className="val">{proxy.local_ip}:{proxy.local_port}</div></div>
          <div className="detail-item"><div className="label">Remote Port</div><div className="val">{proxy.remote_port ?? '—'}</div></div>
          <div className="detail-item"><div className="label">Node ID</div><div className="val">{proxy.node_id ?? '—'}</div></div>
          <div className="detail-item"><div className="label">Subdomain</div><div className="val">{proxy.subdomain ?? '—'}</div></div>
        </div>
      </div>

      {annotation && annotation.error && (
        <div className="card" style={{ borderColor: 'var(--danger)' }}>
          <h3>Last Error</h3>
          <p style={{ color: 'var(--danger)', fontSize: '0.9rem' }}>{annotation.error}</p>
          {annotation.when && <p style={{ color: 'var(--text-muted)', fontSize: '0.8rem', marginTop: '0.3rem' }}>when: {annotation.when}</p>}
        </div>
      )}

      <div className="card">
        <h3>Actions</h3>
        {proxy.status === 'stopped' ? (
          <button className="btn btn-success" onClick={() => handleAction('start')}>Start</button>
        ) : (
          <button className="btn btn-warning" onClick={() => handleAction('stop')}>Stop</button>
        )}
        <button className="btn btn-outline" onClick={() => handleAction('reload')}>Reload</button>
        <button className="btn btn-danger" onClick={handleDelete}>Delete</button>
      </div>

      <TrafficChart proxyId={proxy.id} />
    </div>
  )
}
