import { createContext, useContext, useState, useEffect, useCallback, useRef, type ReactNode } from 'react'
import { useAccount, useSignMessage, useDisconnect } from 'wagmi'
import type { User } from '../types'
import * as api from '../lib/api'

interface AuthContextValue {
  user: User | null
  loading: boolean
  error: string | null
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const authenticatingRef = useRef(false)

  const { address, isConnected } = useAccount()
  const { signMessageAsync } = useSignMessage()
  const { disconnect } = useDisconnect()

  const refresh = useCallback(async () => {
    try {
      const { user } = await api.getMe()
      setUser(user)
    } catch {
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  // Auto-trigger challenge-sign-verify when wallet connects and no session exists
  useEffect(() => {
    if (!isConnected || !address || user || authenticatingRef.current) return

    const authenticate = async () => {
      if (authenticatingRef.current) return
      authenticatingRef.current = true
      setError(null)
      try {
        const addr = address.toLowerCase()
        const { challenge } = await api.requestChallenge(addr)
        const signature = await signMessageAsync({ message: challenge })
        const { user } = await api.verifyChallenge(addr, signature, challenge)
        setUser(user)
      } catch (err) {
        setError((err as Error).message)
        disconnect()
      } finally {
        authenticatingRef.current = false
      }
    }

    // Small delay to avoid StrictMode double-invocation race
    const timer = setTimeout(authenticate, 100)
    return () => clearTimeout(timer)
  }, [isConnected, address, user, signMessageAsync, disconnect])

  const logout = useCallback(async () => {
    await api.logout()
    setUser(null)
    disconnect()
  }, [disconnect])

  return (
    <AuthContext.Provider value={{ user, loading, error, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
