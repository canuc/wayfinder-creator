import { useState, useCallback, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import type { ServerInfo } from '../types'
import { Layout } from '../components/Layout'
import { LogViewer } from '../components/LogViewer'
import { useWebSocket } from '../hooks/useWebSocket'
import * as api from '../lib/api'

export function ServerDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const serverId = id ? parseInt(id, 10) : null
  const [server, setServer] = useState<ServerInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const ws = useWebSocket(serverId)

  useEffect(() => {
    if (serverId === null || isNaN(serverId)) {
      setError('Invalid server ID')
      return
    }
    api.getServer(serverId).then(setServer).catch((err) => {
      setError((err as Error).message)
    })
  }, [serverId])

  const handleDelete = useCallback(async () => {
    if (!serverId) return
    if (!confirm('Are you sure you want to terminate this server? This cannot be undone.')) return
    try {
      await api.deleteServer(serverId)
      navigate('/')
    } catch (err) {
      alert('Delete failed: ' + (err as Error).message)
    }
  }, [serverId, navigate])

  const handleBack = useCallback(() => {
    navigate('/')
  }, [navigate])

  if (error) {
    return (
      <Layout>
        <div className="text-center py-10">
          <div className="text-text-tertiary text-sm mb-4">{error}</div>
          <button
            onClick={() => navigate('/')}
            className="font-sans text-sm font-medium border-none rounded-lg px-5 py-2 cursor-pointer bg-accent text-black hover:bg-green-600 transition-colors"
          >
            Back to dashboard
          </button>
        </div>
      </Layout>
    )
  }

  if (!server) {
    return (
      <Layout>
        <div className="text-center py-10 text-text-tertiary text-sm">
          Loading...
        </div>
      </Layout>
    )
  }

  return (
    <Layout>
      <LogViewer
        serverName={ws.serverName || server.name}
        serverIP={ws.serverIP || server.ipv4}
        status={ws.status || server.status}
        logs={ws.logs}
        onBack={handleBack}
        onDelete={handleDelete}
      />
    </Layout>
  )
}
