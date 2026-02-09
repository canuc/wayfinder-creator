import { Navigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center font-mono text-text-tertiary text-xs">
        loading<span className="animate-[blink_1s_step-end_infinite]">_</span>
      </div>
    )
  }

  if (!user) return <Navigate to="/connect" replace />
  if (!user.approved) return <Navigate to="/awaiting" replace />

  return <>{children}</>
}

export function AdminGuard({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center font-mono text-text-tertiary text-xs">
        loading<span className="animate-[blink_1s_step-end_infinite]">_</span>
      </div>
    )
  }

  if (!user) return <Navigate to="/connect" replace />
  if (user.role !== 'admin') return <Navigate to="/" replace />

  return <>{children}</>
}
