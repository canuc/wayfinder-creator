import type { ServerInfo } from '../types'
import { ServerCard } from './ServerCard'

interface Props {
  servers: ServerInfo[]
  onDelete: (id: number) => void
}

export function ServerList({ servers, onDelete }: Props) {
  if (servers.length === 0) {
    return (
      <div className="text-text-tertiary text-sm py-10 text-center font-mono">
        No servers running. Deploy one to get started.
      </div>
    )
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {servers.map((s) => (
        <ServerCard key={s.id} server={s} onDelete={onDelete} />
      ))}
    </div>
  )
}
