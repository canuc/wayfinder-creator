import { useState, useEffect, useCallback, useRef } from 'react'
import type { ServerInfo, PairingRequest } from '../types'
import { signRequest } from '../lib/crypto'
import { signedRequest } from '../lib/api'

export function usePairingRequests(servers: ServerInfo[]) {
  const [requests, setRequests] = useState<PairingRequest[]>([])
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const readyServers = servers.filter((s) => s.status === 'ready' && s.has_node_api)

  const refresh = useCallback(async () => {
    if (readyServers.length === 0) {
      setRequests([])
      return
    }

    const allRequests: PairingRequest[] = []
    for (const srv of readyServers) {
      try {
        const headers = await signRequest('GET', '/pairing/requests')
        const res = await signedRequest(
          `/servers/${srv.id}/pairing/requests`,
          '/pairing/requests',
          'GET',
          headers,
        )
        if (!res.ok) continue
        const reqs = (await res.json()) as PairingRequest[]
        for (const req of reqs) {
          req.server_id = srv.id
          req.server_name = srv.name
          allRequests.push(req)
        }
      } catch {
        // ignore errors for individual servers
      }
    }
    setRequests(allRequests)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(readyServers.map((s) => s.id))])

  useEffect(() => {
    if (readyServers.length === 0) {
      if (intervalRef.current) clearInterval(intervalRef.current)
      return
    }
    refresh()
    intervalRef.current = setInterval(refresh, 30000)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refresh])

  const approve = useCallback(
    async (serverId: number, channel: string, code: string) => {
      const body = JSON.stringify({ channel, code })
      const headers = await signRequest('POST', '/pairing/approve', body)
      const res = await signedRequest(
        `/servers/${serverId}/pairing/approve`,
        '/pairing/approve',
        'POST',
        headers,
        body,
      )
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || res.statusText)
      }
      await refresh()
    },
    [refresh],
  )

  const deny = useCallback(
    async (serverId: number, channel: string, code: string) => {
      const body = JSON.stringify({ channel, code })
      const headers = await signRequest('POST', '/pairing/deny', body)
      const res = await signedRequest(
        `/servers/${serverId}/pairing/deny`,
        '/pairing/deny',
        'POST',
        headers,
        body,
      )
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || res.statusText)
      }
      await refresh()
    },
    [refresh],
  )

  return { requests, approve, deny, refresh }
}
