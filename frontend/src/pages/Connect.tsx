import { Navigate } from 'react-router-dom'
import { ConnectButton } from '@rainbow-me/rainbowkit'
import { useAuth } from '../contexts/AuthContext'

export function Connect() {
  const { user, error } = useAuth()

  if (user && user.approved) return <Navigate to="/" replace />
  if (user && !user.approved) return <Navigate to="/awaiting" replace />

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
          <div className="font-mono text-[0.7rem] text-text-dim">
            cloud server provisioning
          </div>
        </div>

        {/* Connect card */}
        <div className="tech-panel p-6">
          <div className="flex items-center gap-2 mb-4">
            <span className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest">
              // authenticate
            </span>
          </div>
          <p className="text-xs text-text-secondary mb-5 leading-relaxed">
            Connect your Ethereum wallet to sign in. MetaMask, Rabby, WalletConnect, and more are supported.
          </p>
          {error && (
            <div className="text-xs text-red-400 bg-danger-muted border border-danger/20 font-mono px-3 py-2 rounded-md mb-4">{error}</div>
          )}
          <div className="flex justify-center">
            <ConnectButton />
          </div>
        </div>

        {/* Footer detail */}
        <div className="text-center mt-6">
          <span className="font-mono text-[0.65rem] text-text-dim">
            wallet signature required for session
          </span>
        </div>
      </div>
    </div>
  )
}
