import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Layout } from '../components/Layout'
import { ServerList } from '../components/ServerList'
import { PairingRequests } from '../components/PairingRequests'
import { useServers } from '../hooks/useServers'
import { usePairingRequests } from '../hooks/usePairingRequests'
import * as api from '../lib/api'

export function Dashboard() {
  const navigate = useNavigate()
  const { servers, refresh } = useServers()
  const { requests, approve, deny } = usePairingRequests(servers)

  const handleDelete = useCallback(
    async (id: number) => {
      if (!confirm(`Terminate server ${id}?`)) return
      try {
        await api.deleteServer(id)
        await refresh()
      } catch (err) {
        alert('Delete failed: ' + (err as Error).message)
      }
    },
    [refresh],
  )

  return (
    <Layout>
      <section className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <span className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest">
              // servers
            </span>
            {servers.length > 0 && (
              <span className="font-mono text-[0.7rem] text-text-dim">
                [{servers.length}]
              </span>
            )}
          </div>
          <button
            onClick={() => navigate('/deploy')}
            className="font-mono text-xs font-medium border border-accent/30 rounded-md px-4 py-1.5 cursor-pointer bg-accent/10 text-accent-text hover:bg-accent/20 hover:border-accent/50 transition-all glow-accent"
          >
            + deploy
          </button>
        </div>

        {servers.length === 0 ? (
          <div className="tech-panel p-10 text-center">
            <div className="font-mono text-sm text-text-tertiary mb-1">
              no active instances
            </div>
            <div className="font-mono text-[0.7rem] text-text-dim mb-5">
              deploy your first server to get started
            </div>
            <button
              onClick={() => navigate('/deploy')}
              className="font-mono text-xs font-medium border border-accent/30 rounded-md px-5 py-2 cursor-pointer bg-accent/10 text-accent-text hover:bg-accent/20 hover:border-accent/50 transition-all"
            >
              + deploy server
            </button>
          </div>
        ) : (
          <ServerList servers={servers} onDelete={handleDelete} />
        )}
      </section>
      <PairingRequests requests={requests} onApprove={approve} onDeny={deny} />
    </Layout>
  )
}
