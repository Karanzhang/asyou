import { useEffect, useState } from 'react'
import { listApiKeys, createApiKey, deleteApiKey } from '../api/client'
import type { ApiKey } from '../types'

export default function ApiKeys() {
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [newToken, setNewToken] = useState('')
  const [toast, setToast] = useState<{ msg: string; type: string } | null>(null)

  const showToast = (msg: string, type = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  useEffect(() => {
    listApiKeys().then(setKeys).catch(() => showToast('Failed to load', 'error'))
  }, [])

  const handleCreate = async () => {
    try {
      const res = await createApiKey()
      setNewToken(res.token)
      showToast('API key created — copy it now, it won\'t be shown again')
      setKeys(await listApiKeys())
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  const handleRevoke = async (id: number) => {
    try {
      await deleteApiKey(id)
      showToast('Key revoked')
      setKeys(await listApiKeys())
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  return (
    <div>
      {toast && <div className={`toast ${toast.type}`}>{toast.msg}</div>}
      <div className="page-header">
        <h1>API Keys</h1>
        <button className="btn btn-primary" onClick={handleCreate}>+ New Key</button>
      </div>

      {newToken && (
        <div className="card" style={{ borderColor: 'var(--warning)' }}>
          <h3>New API Key Created</h3>
          <p style={{ wordBreak: 'break-all', fontFamily: 'monospace', background: 'var(--bg)', padding: '0.8rem', borderRadius: 'var(--radius)' }}>
            {newToken}
          </p>
          <p style={{ color: 'var(--text-muted)', fontSize: '0.8rem', marginTop: '0.5rem' }}>
            ⚠️ This token will only be shown once. Store it securely.
          </p>
          <button className="btn btn-sm btn-outline" style={{ marginTop: '0.5rem' }} onClick={() => { setNewToken(''); navigator.clipboard.writeText(newToken) }}>Copy & Dismiss</button>
        </div>
      )}

      <div className="card">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Scopes</th>
                <th>Status</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.length === 0 && <tr><td colSpan={5} className="empty">No API keys.</td></tr>}
              {keys.map(k => (
                <tr key={k.id}>
                  <td>{k.name || '—'}</td>
                  <td>{k.scopes || '—'}</td>
                  <td>{k.revoked ? <span className="badge badge-error">Revoked</span> : <span className="badge badge-running">Active</span>}</td>
                  <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{new Date(k.created_at).toLocaleDateString()}</td>
                  <td>
                    {!k.revoked && <button className="btn btn-danger btn-sm" onClick={() => handleRevoke(k.id)}>Revoke</button>}
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
