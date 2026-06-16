import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import Layout from './components/Layout'
import LoginPage from './components/LoginPage'
import ProxyList from './components/ProxyList'
import ProxyDetail from './components/ProxyDetail'
import NodeList from './components/NodeList'
import AuditLogs from './components/AuditLogs'
import ApiKeys from './components/ApiKeys'

export default function App() {
  const auth = useAuth()

  if (auth.loading) {
    return <div className="login-page"><p>Loading…</p></div>
  }

  if (!auth.isLoggedIn) {
    return <LoginPage onLogin={auth.login} />
  }

  return (
    <Layout user={auth.user} onLogout={auth.logout}>
      <Routes>
        <Route path="/" element={<ProxyList />} />
        <Route path="/proxies/:id" element={<ProxyDetail />} />
        <Route path="/nodes" element={<NodeList />} />
        <Route path="/audit-logs" element={<AuditLogs />} />
        <Route path="/api-keys" element={<ApiKeys />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  )
}
