import { NavLink } from 'react-router-dom'
import type { User } from '../types'

interface Props {
  user: User | null
  onLogout: () => void
  children: React.ReactNode
}

export default function Layout({ user, onLogout, children }: Props) {
  return (
    <div className="layout">
      <aside className="sidebar">
        <h2>asyou</h2>
        <nav>
          <NavLink to="/" className={({ isActive }) => isActive ? 'active' : ''} end>
            📡 Proxies
          </NavLink>
          <NavLink to="/nodes" className={({ isActive }) => isActive ? 'active' : ''}>
            🖥 Nodes
          </NavLink>
          <NavLink to="/audit-logs" className={({ isActive }) => isActive ? 'active' : ''}>
            📋 Audit Logs
          </NavLink>
          <NavLink to="/api-keys" className={({ isActive }) => isActive ? 'active' : ''}>
            🔑 API Keys
          </NavLink>
        </nav>
        <div style={{ padding: '0 1.5rem', marginTop: 'auto', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
          {user?.email}
        </div>
        <button className="logout-btn" onClick={onLogout}>Sign Out</button>
      </aside>
      <main className="main">{children}</main>
    </div>
  )
}
