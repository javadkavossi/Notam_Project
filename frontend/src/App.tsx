import { createPortal } from 'react-dom'
import { Routes, Route, Navigate } from 'react-router-dom'
import { ToastContainer } from 'react-toastify'
import 'react-toastify/dist/ReactToastify.css'
import { useAuth } from './contexts/AuthContext'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import './styles/toast-overrides.css'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()
  if (isLoading) return <div className="loading">در حال بارگذاری...</div>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

const toastContainer = (
  <ToastContainer
    position="top-center"
    newestOnTop
    limit={10}
    stacked
    theme="dark"
    closeOnClick={false}
    draggable={false}
    className="notam-toast-container"
  />
)

export default function App() {
  return (
    <>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      {typeof document !== 'undefined' &&
        createPortal(toastContainer, document.body)}
    </>
  )
}
