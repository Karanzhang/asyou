import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

export default function ResetPassword() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [done, setDone] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (password !== confirm) {
      setError('Passwords do not match')
      return
    }
    if (password.length < 6) {
      setError('Password must be at least 6 characters')
      return
    }
    setBusy(true)
    try {
      const res = await fetch('/api/v1/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, password }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: 'request failed' }))
        throw new Error(data.error)
      }
      setDone(true)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (!token) {
    return (
      <div className="login-page">
        <div className="login-card" style={{ textAlign: 'center' }}>
          <h1>asyou</h1>
          <p style={{ marginTop: '1rem', color: 'var(--danger)' }}>Invalid reset link. No token provided.</p>
          <a href="/" style={{ display: 'inline-block', marginTop: '1.5rem' }}>Back to Sign In</a>
        </div>
      </div>
    )
  }

  if (done) {
    return (
      <div className="login-page">
        <div className="login-card" style={{ textAlign: 'center' }}>
          <h1>asyou</h1>
          <p style={{ marginTop: '1rem', color: 'var(--success)' }}>Password has been reset successfully!</p>
          <a href="/" style={{ display: 'inline-block', marginTop: '1.5rem' }}>Sign In with New Password</a>
        </div>
      </div>
    )
  }

  return (
    <div className="login-page">
      <form className="login-card" onSubmit={handleSubmit}>
        <h1>asyou</h1>
        <p>Enter your new password</p>
        {error && <div className="error">{error}</div>}
        <div className="form-group">
          <label>New Password</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} required minLength={6} />
        </div>
        <div className="form-group">
          <label>Confirm Password</label>
          <input type="password" value={confirm} onChange={e => setConfirm(e.target.value)} required />
        </div>
        <button className="btn btn-primary" type="submit" disabled={busy}>
          {busy ? 'Resetting…' : 'Reset Password'}
        </button>
      </form>
    </div>
  )
}
