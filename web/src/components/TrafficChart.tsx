import { useEffect, useState } from 'react'
import { useSSE } from '../hooks/useSSE'
import { getProxyStats } from '../api/client'
import type { ProxyStats } from '../types'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

interface Props {
  proxyId: number
}

export default function TrafficChart({ proxyId }: Props) {
  const [stats, setStats] = useState<ProxyStats[]>([])

  // Initial load from REST API
  useEffect(() => {
    getProxyStats(proxyId, 60).then(data => setStats(data.reverse())).catch(() => {})
  }, [proxyId])

  // Real-time updates via SSE
  useSSE('stats_update', (data: any[]) => {
    if (!Array.isArray(data)) return
    const match = data.find((s: any) => s.proxy_id === proxyId)
    if (!match) return
    setStats(prev => {
      const next = [...prev, {
        id: 0,
        proxy_id: match.proxy_id,
        timestamp: new Date().toISOString(),
        bytes_in: match.bytes_in,
        bytes_out: match.bytes_out,
        conn_count: match.conn_count,
      }]
      // Keep last 60 entries
      return next.slice(-60)
    })
  })

  if (stats.length === 0) return null

  const data = stats.map(s => ({
    time: new Date(s.timestamp).toLocaleTimeString(),
    in: Math.round(s.bytes_in / 1024),
    out: Math.round(s.bytes_out / 1024),
    conn: s.conn_count,
  }))

  return (
    <div className="card">
      <h3>Traffic (real-time via SSE)</h3>
      <ResponsiveContainer width="100%" height={250}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
          <XAxis dataKey="time" tick={{ fontSize: 11, fill: 'var(--text-muted)' }} />
          <YAxis tick={{ fontSize: 11, fill: 'var(--text-muted)' }} />
          <Tooltip
            contentStyle={{ background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)' }}
          />
          <Line type="monotone" dataKey="in" stroke="var(--primary)" name="KB/s In" strokeWidth={2} dot={false} />
          <Line type="monotone" dataKey="out" stroke="var(--success)" name="KB/s Out" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
