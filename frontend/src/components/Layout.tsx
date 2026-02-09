import type { ReactNode } from 'react'
import { useAuth } from '../contexts/AuthContext'

export function Layout({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth()

  return (
    <div className="max-w-[820px] mx-auto px-6 pt-10 pb-16 max-sm:px-4 max-sm:pt-5">
      {/* Header */}
      <header className="mb-8">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-md bg-accent/10 border border-accent/20 flex items-center justify-center font-mono font-semibold text-xs text-accent-text">
              oc
            </div>
            <div>
              <h1 className="text-base font-semibold tracking-tight font-mono">
                openclaw<span className="text-text-tertiary font-normal">/creator</span>
              </h1>
            </div>
          </div>
          {user && (
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1.5 bg-surface/50 border border-border rounded-md px-2.5 py-1">
                <span className="w-1.5 h-1.5 rounded-full bg-accent animate-[pulse-dot_2s_ease-in-out_infinite]" />
                <span className="text-text-secondary font-mono text-[0.75rem]">{user.address.slice(0, 6)}...{user.address.slice(-4)}</span>
              </div>
              {user.role === 'admin' && (
                <a
                  href="/admin/users"
                  className="font-mono text-[0.75rem] text-text-tertiary hover:text-accent-text px-2 py-1 border border-transparent hover:border-border rounded-md transition-colors"
                >
                  /users
                </a>
              )}
              <button
                onClick={logout}
                className="font-mono text-[0.75rem] text-text-tertiary hover:text-text-secondary px-2 py-1 border border-transparent hover:border-border rounded-md transition-colors"
              >
                /logout
              </button>
            </div>
          )}
        </div>
        <div className="tech-divider" />
      </header>

      {children}

      {/* Footer */}
      <footer className="mt-14 pt-4">
        <div className="tech-divider mb-4" />
        <div className="flex items-center justify-between">
          <span className="font-mono text-[0.7rem] text-text-dim">openclaw/creator v0.1</span>
          <span className="font-mono text-[0.7rem] text-text-dim">system nominal</span>
        </div>
      </footer>
    </div>
  )
}
