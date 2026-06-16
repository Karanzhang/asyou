import { useEffect, useState } from 'react'
import { listAuditLogs } from '../api/client'
import type { AuditLog } from '../types'

export default function AuditLogs() {
  const [logs, setLogs] = useState<AuditLog[]>([])

  useEffect(() => {
    listAuditLogs().then(setLogs).catch(() => {})
  }, [])

  return (
    <div>
      <div className="page-header"><h1>Audit Logs</h1></div>
      <div className="card">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Action</th>
                <th>Resource</th>
                <th>Actor</th>
                <th>IP</th>
                <th>Detail</th>
              </tr>
            </thead>
            <tbody>
              {logs.length === 0 && <tr><td colSpan={6} className="empty">No audit logs yet.</td></tr>}
              {logs.map(l => (
                <tr key={l.id}>
                  <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{new Date(l.created_at).toLocaleString()}</td>
                  <td><code>{l.action_type}</code></td>
                  <td>{l.resource_type}{l.resource_id ? ` #${l.resource_id}` : ''}</td>
                  <td>{l.actor_user_id ?? '—'}</td>
                  <td>{l.ip || '—'}</td>
                  <td style={{ fontSize: '0.8rem', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>{l.detail || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
