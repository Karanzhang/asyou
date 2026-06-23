import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import Layout from './components/Layout'
import LoginPage from './components/LoginPage'
import ForgotPassword from './components/ForgotPassword'
import ResetPassword from './components/ResetPassword'
import ProxyList from './components/ProxyList'
import ProxyDetail from './components/ProxyDetail'
import NodeList from './components/NodeList'
import AuditLogs from './components/AuditLogs'
import ApiKeys from './components/ApiKeys'
import Docs from './components/Docs'

export default function App() {
  const auth = useAuth()

  if (auth.loading) {
    return <div className="login-page"><p>Loading…</p></div>
  }

  if (!auth.isLoggedIn) {
    // Allow access to forgot/reset pages without login
    const isPublicRoute = window.location.pathname.startsWith('/forgot-password') ||
      window.location.pathname.startsWith('/reset-password')
    if (isPublicRoute) {
      return (
        <Routes>
          <Route path="/forgot-password" element={<ForgotPassword />} />
          <Route path="/reset-password" element={<ResetPassword />} />
        </Routes>
      )
    }
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
        <Route path="/docs" element={<Docs />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  )
}
