import { useAuth } from '../contexts/AuthContext'
import { Navigate } from 'react-router-dom'

export function AwaitingApproval() {
  const { user, logout, refresh } = useAuth()

  if (!user) return <Navigate to="/connect" replace />
  if (user.approved) return <Navigate to="/" replace />

  return (
    <div className="min-h-screen flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        {/* Branding */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-10 h-10 rounded-lg bg-accent/10 border border-accent/20 flex items-center justify-center font-mono font-semibold text-sm text-accent-text mb-4">
            oc
          </div>
          <h1 className="text-lg font-semibold tracking-tight font-mono mb-1">
            openclaw<span className="text-text-tertiary font-normal">/creator</span>
          </h1>
        </div>

        <div className="tech-panel p-6 text-center">
          <div className="flex items-center justify-center gap-2 mb-4">
            <span className="w-2 h-2 rounded-full bg-warning animate-[pulse-dot_2s_ease-in-out_infinite]" />
            <span className="font-mono text-sm font-medium text-warning">
              pending approval
            </span>
          </div>
          <p className="text-xs text-text-secondary mb-2 leading-relaxed">
            Wallet <span className="font-mono text-text">{user.address.slice(0, 6)}...{user.address.slice(-4)}</span> has been registered.
          </p>
          <p className="text-xs text-text-tertiary mb-5 leading-relaxed">
            An administrator must approve your account before you can access the dashboard.
          </p>
          <div className="flex gap-2 justify-center">
            <button
              onClick={refresh}
              className="font-mono text-[0.75rem] font-medium border border-border rounded-md px-3 py-1.5 cursor-pointer bg-transparent text-text-secondary hover:text-text hover:border-border-hover transition-colors"
            >
              /check-status
            </button>
            <button
              onClick={logout}
              className="font-mono text-[0.75rem] font-medium border border-border rounded-md px-3 py-1.5 cursor-pointer bg-transparent text-text-tertiary hover:text-text-secondary hover:border-border-hover transition-colors"
            >
              /disconnect
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
