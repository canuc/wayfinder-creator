import { useNavigate } from 'react-router-dom'
import type { ServerInfo } from '../types'
import { StatusPill } from './StatusPill'

interface Props {
  server: ServerInfo
  onDelete: (id: number) => void
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return '\u2014'
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function ServerCard({ server, onDelete }: Props) {
  const navigate = useNavigate()
  const wallet = server.wallet_address
    ? `${server.wallet_address.slice(0, 6)}...${server.wallet_address.slice(-4)}`
    : null

  return (
    <div
      className="tech-panel p-4 hover:border-border-hover transition-all cursor-pointer group"
      onClick={() => navigate(`/servers/${server.id}`)}
    >
      {/* Top row: name + status */}
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="font-mono text-sm font-medium text-text group-hover:text-accent-text transition-colors">
            {server.name}
          </div>
          <div className="font-mono text-[0.7rem] text-text-dim mt-0.5">
            id:{server.id}
          </div>
        </div>
        <StatusPill status={server.status} />
      </div>

      {/* Data grid */}
      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 mb-3">
        <div className="flex items-center gap-2">
          <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">ip</span>
          <span className="font-mono text-xs text-text-secondary">
            {server.ipv4 || '\u2014'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">created</span>
          <span className="font-mono text-xs text-text-secondary">
            {formatDate(server.created_at)}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">ch</span>
          <span className="font-mono text-xs text-text-secondary">
            {(server.channel_count ?? 0) > 0 ? server.channel_count : '\u2014'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">wallet</span>
          {wallet ? (
            <span className="font-mono text-xs text-cyan">{wallet}</span>
          ) : (
            <span className="font-mono text-xs text-text-tertiary">{'\u2014'}</span>
          )}
        </div>
      </div>

      {/* Bottom bar */}
      <div className="flex items-center justify-between pt-2.5 border-t border-border/50">
        <span className="font-mono text-[0.7rem] text-text-dim">
          {server.status === 'provisioning' ? 'deploying...' : server.status === 'ready' ? 'operational' : 'error state'}
        </span>
        <button
          onClick={(e) => { e.stopPropagation(); onDelete(server.id) }}
          className="bg-transparent text-text-tertiary border border-border font-mono text-[0.7rem] uppercase tracking-wider px-2 py-0.5 rounded hover:text-danger hover:border-danger/40 hover:bg-danger-muted transition-colors cursor-pointer"
        >
          terminate
        </button>
      </div>
    </div>
  )
}
