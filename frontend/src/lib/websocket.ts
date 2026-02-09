import type { WSMessage } from '../types'

export type WSMessageHandler = (msg: WSMessage) => void
export type WSCloseHandler = () => void

export function connectWebSocket(
  serverId: number,
  onMessage: WSMessageHandler,
  onClose?: WSCloseHandler,
): WebSocket {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/servers/${serverId}/ws`
  const ws = new WebSocket(url)

  ws.onmessage = (evt) => {
    try {
      const msg = JSON.parse(evt.data) as WSMessage
      onMessage(msg)
    } catch {
      // ignore parse errors
    }
  }

  ws.onclose = () => onClose?.()
  ws.onerror = () => onClose?.()

  return ws
}
