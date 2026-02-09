import { useState, useEffect, useCallback, useRef } from 'react'
import type { ServerInfo } from '../types'
import { listServers } from '../lib/api'

export function useServers() {
  const [servers, setServers] = useState<ServerInfo[]>([])
  const [loading, setLoading] = useState(true)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const refresh = useCallback(async () => {
    try {
      const data = await listServers()
      setServers(data)
    } catch {
      // ignore errors during polling
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  useEffect(() => {
    const hasProvisioning = servers.some((s) => s.status === 'provisioning')
    const interval = hasProvisioning ? 5000 : 30000

    if (intervalRef.current) clearInterval(intervalRef.current)
    intervalRef.current = setInterval(refresh, interval)

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [servers, refresh])

  return { servers, loading, refresh }
}
