import { useEffect, useRef } from 'react'
import { StatusPill } from './StatusPill'

interface Props {
  serverName: string
  serverIP: string
  status: string
  logs: string[]
  onBack: () => void
  onDelete?: () => void
}

export function LogViewer({ serverName, serverIP, status, logs, onBack, onDelete }: Props) {
  const terminalRef = useRef<HTMLPreElement>(null)

  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight
    }
  }, [logs])

  return (
    <div className="tech-panel overflow-hidden">
      {/* Terminal header */}
      <div className="flex items-center justify-between px-4 py-2.5 bg-surface/50 border-b border-border">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5">
            <span className="w-2 h-2 rounded-full bg-red-500/60" />
            <span className="w-2 h-2 rounded-full bg-yellow-500/60" />
            <span className="w-2 h-2 rounded-full bg-green-500/60" />
          </div>
          <div className="w-px h-3 bg-border" />
          <span className="font-mono text-[0.7rem] text-text-secondary">
            {serverName}
          </span>
          <span className="font-mono text-[0.65rem] text-text-dim">
            {serverIP}
          </span>
        </div>
        {status && <StatusPill status={status} />}
      </div>

      <div className="p-4">
        {/* Nav */}
        <div className="flex items-center gap-3 mb-3">
          <button
            onClick={onBack}
            className="bg-transparent border border-border rounded-md text-text-secondary cursor-pointer font-mono text-[0.75rem] px-2.5 py-1 hover:text-text hover:border-border-hover transition-colors"
          >
            &larr; back
          </button>
        </div>

        {/* Log output */}
        <pre
          ref={terminalRef}
          className="bg-[#04040a] rounded-md border border-border/50 font-mono text-[0.8rem] leading-relaxed text-text-secondary max-h-[70vh] overflow-y-auto whitespace-pre-wrap break-all p-4"
        >
          {logs.join('\n')}
        </pre>

        {status === 'ready' && (
          <div className="mt-3 flex items-center gap-3 pt-3 border-t border-border/50">
            <span className="font-mono text-[0.75rem] text-accent-text">
              provisioning complete
            </span>
          </div>
        )}
        {status === 'failed' && (
          <div className="mt-3 pt-3 border-t border-border/50">
            <div className="rounded-md border border-red-500/30 bg-red-500/5 p-3 flex items-start gap-3">
              <div className="flex-1">
                <p className="font-mono text-[0.75rem] text-red-400 font-medium">
                  Provisioning failed
                </p>
                <p className="font-mono text-[0.7rem] text-text-dim mt-1">
                  This server cannot be recovered. Delete it and create a new one.
                </p>
              </div>
              {onDelete && (
                <button
                  onClick={onDelete}
                  className="bg-red-500/10 text-red-400 border border-red-500/30 font-mono text-[0.75rem] font-medium px-3 py-1 rounded-md cursor-pointer hover:bg-red-500/20 transition-colors shrink-0"
                >
                  terminate
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
