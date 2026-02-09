import { useState, useEffect, useCallback, useRef } from 'react'
import type { WSMessage } from '../types'
import { connectWebSocket } from '../lib/websocket'

interface WSState {
  serverName: string
  serverIP: string
  status: string
  defaultKeyRemoved: boolean
  logs: string[]
  connected: boolean
}

export function useWebSocket(serverId: number | null) {
  const [state, setState] = useState<WSState>({
    serverName: '',
    serverIP: '',
    status: '',
    defaultKeyRemoved: false,
    logs: [],
    connected: false,
  })
  const wsRef = useRef<WebSocket | null>(null)

  const reconnect = useCallback(() => {
    if (serverId === null) return

    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    setState({
      serverName: '',
      serverIP: '',
      status: '',
      defaultKeyRemoved: false,
      logs: [],
      connected: false,
    })

    const ws = connectWebSocket(
      serverId,
      (msg: WSMessage) => {
        if (msg.type === 'init') {
          setState((prev) => ({
            ...prev,
            serverName: msg.server.name,
            serverIP: msg.server.ipv4,
            status: msg.server.status,
            defaultKeyRemoved: msg.server.default_key_removed,
            connected: true,
          }))
        } else if (msg.type === 'log') {
          setState((prev) => ({
            ...prev,
            logs: [...prev.logs, msg.line],
          }))
        } else if (msg.type === 'status') {
          setState((prev) => ({
            ...prev,
            status: msg.status,
            defaultKeyRemoved: msg.default_key_removed,
          }))
        }
      },
      () => {
        setState((prev) => ({
          ...prev,
          connected: false,
          logs: [...prev.logs, '[disconnected]'],
        }))
      },
    )

    wsRef.current = ws
  }, [serverId])

  useEffect(() => {
    reconnect()
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [reconnect])

  return { ...state, reconnect }
}
