import { useState } from 'react'
import type { PairingRequest } from '../types'

interface Props {
  requests: PairingRequest[]
  onApprove: (serverId: number, channel: string, code: string) => void
  onDeny: (serverId: number, channel: string, code: string) => void
}

function requestKey(r: PairingRequest) {
  return `${r.server_id}-${r.channel}-${r.code}`
}

export function PairingRequests({ requests, onApprove, onDeny }: Props) {
  const pending = requests.filter((r) => !r.status || r.status === 'pending')
  const [busy, setBusy] = useState<Record<string, 'approving' | 'denying'>>({})

  if (pending.length === 0) return null

  const handleAction = async (
    r: PairingRequest,
    action: 'approving' | 'denying',
    fn: (serverId: number, channel: string, code: string) => void,
  ) => {
    const key = requestKey(r)
    if (busy[key]) return
    setBusy((prev) => ({ ...prev, [key]: action }))
    try {
      await fn(r.server_id, r.channel, r.code)
    } catch (err) {
      alert(`${action === 'approving' ? 'Approve' : 'Deny'} failed: ${(err as Error).message}`)
    } finally {
      setBusy((prev) => {
        const next = { ...prev }
        delete next[key]
        return next
      })
    }
  }

  return (
    <section className="mb-8">
      <div className="flex items-center gap-3 mb-4">
        <span className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest">
          // pairing requests
        </span>
        <span className="font-mono text-[0.7rem] text-text-dim">
          [{pending.length}]
        </span>
      </div>
      <div className="tech-panel p-5">
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr>
                {['server', 'channel', 'user', 'requested', ''].map((h) => (
                  <th
                    key={h}
                    className="text-left font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-wider px-3 pb-2 border-b border-border whitespace-nowrap"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {pending.map((r) => {
                const userName = r.meta
                  ? [r.meta.firstName, r.meta.lastName].filter(Boolean).join(' ')
                  : ''
                const key = requestKey(r)
                const rowBusy = busy[key]
                return (
                  <tr key={key}>
                    <td className="px-3 py-2.5 border-b border-border/30 font-mono font-medium text-text whitespace-nowrap text-xs">
                      {r.server_name}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 font-mono text-[0.75rem] text-text-secondary whitespace-nowrap">
                      {r.channel}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 font-mono text-[0.75rem] text-text-secondary whitespace-nowrap">
                      {userName || r.id}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 font-mono text-[0.75rem] text-text-tertiary whitespace-nowrap">
                      {r.created_at}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 whitespace-nowrap">
                      {rowBusy ? (
                        <span className="font-mono text-[0.7rem] text-text-tertiary">
                          {rowBusy}<span className="animate-[blink_1s_step-end_infinite]">_</span>
                        </span>
                      ) : (
                        <div className="flex gap-1.5">
                          <button
                            onClick={() => handleAction(r, 'approving', onApprove)}
                            className="bg-accent/10 text-accent-text border border-accent/30 font-mono text-[0.7rem] font-medium rounded px-2 py-0.5 cursor-pointer hover:bg-accent/20 hover:border-accent/50 transition-colors"
                          >
                            approve
                          </button>
                          <button
                            onClick={() => handleAction(r, 'denying', onDeny)}
                            className="bg-transparent text-danger border border-danger/30 font-mono text-[0.7rem] font-medium rounded px-2 py-0.5 cursor-pointer hover:bg-danger-muted transition-colors"
                          >
                            deny
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
