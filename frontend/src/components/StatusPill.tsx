const statusStyles = {
  provisioning: 'text-warning bg-warning-muted border-warning/20',
  ready: 'text-accent-text bg-accent-muted border-accent/20',
  failed: 'text-red-400 bg-danger-muted border-danger/20',
} as const

const dotStyles = {
  provisioning: 'bg-warning animate-[blink_1s_step-end_infinite]',
  ready: 'bg-accent',
  failed: 'bg-danger',
} as const

type Status = 'provisioning' | 'ready' | 'failed'

export function StatusPill({ status }: { status: string }) {
  const s = (status === 'ready' ? 'ready' : status === 'failed' ? 'failed' : 'provisioning') as Status

  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded border font-mono text-[0.7rem] font-medium uppercase tracking-wider ${statusStyles[s]}`}>
      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${dotStyles[s]}`} />
      {status}
    </span>
  )
}
