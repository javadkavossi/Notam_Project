import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import './Login.css'

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    const ok = await login(username, password)
    setLoading(false)
    if (ok) {
      navigate('/', { replace: true })
    } else {
      setError('نام کاربری یا رمز عبور اشتباه است')
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>NOTAM Viewer</h1>
        <p className="subtitle">ورود به پنل مدیریت اعلان‌های هوانوردی</p>
        <form onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="نام کاربری"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoFocus
            disabled={loading}
          />
          <input
            type="password"
            placeholder="رمز عبور"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={loading}
          />
          {error && <p className="error">{error}</p>}
          <button type="submit" disabled={loading}>
            {loading ? 'در حال ورود...' : 'ورود'}
          </button>
        </form>
        <p className="hint">پیش‌فرض: admin / admin</p>
      </div>
    </div>
  )
}
