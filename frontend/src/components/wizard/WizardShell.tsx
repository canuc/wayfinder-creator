import { useState, type ReactNode } from 'react'

interface Step {
  label: string
  content: ReactNode
  validate?: () => boolean
  nextDisabled?: boolean
}

interface Props {
  steps: Step[]
  onComplete: () => Promise<void> | void
  onCancel: () => void
  title?: string
}

export function WizardShell({ steps, onComplete, onCancel, title }: Props) {
  const [current, setCurrent] = useState(0)
  const [deploying, setDeploying] = useState(false)
  const step = steps[current]
  const isLast = current === steps.length - 1

  const next = async () => {
    if (step.validate && !step.validate()) return
    if (isLast) {
      setDeploying(true)
      try {
        await onComplete()
      } catch {
        setDeploying(false)
      }
    } else {
      setCurrent(current + 1)
    }
  }

  const back = () => {
    if (deploying) return
    if (current > 0) setCurrent(current - 1)
    else onCancel()
  }

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
            {title || 'deploy'}
          </span>
        </div>
        <span className="font-mono text-[0.65rem] text-text-dim">
          step {current + 1}/{steps.length}
        </span>
      </div>

      <div className="p-5">
        {/* Step indicator */}
        <div className="flex items-center gap-1 mb-6">
          {steps.map((s, i) => (
            <div key={i} className="flex items-center gap-1">
              <div
                className={`w-6 h-6 rounded flex items-center justify-center font-mono text-[0.7rem] font-medium border transition-colors ${
                  i === current
                    ? 'bg-accent/15 border-accent/30 text-accent-text'
                    : i < current
                      ? 'bg-accent/8 border-accent/15 text-accent-text/60'
                      : 'bg-surface border-border text-text-tertiary'
                }`}
              >
                {String(i + 1).padStart(2, '0')}
              </div>
              <span
                className={`font-mono text-[0.7rem] ${
                  i === current ? 'text-text-secondary' : 'text-text-dim'
                } hidden sm:inline`}
              >
                {s.label}
              </span>
              {i < steps.length - 1 && (
                <div className={`w-4 h-px mx-0.5 ${i < current ? 'bg-accent/30' : 'bg-border'}`} />
              )}
            </div>
          ))}
        </div>

        {/* Step content */}
        <div className="min-h-[200px]">{step.content}</div>

        {/* Navigation */}
        <div className="flex items-center justify-between mt-6 pt-4 border-t border-border/50">
          <button
            onClick={back}
            disabled={deploying}
            className="font-mono text-[0.75rem] font-medium border border-border rounded-md px-3 py-1.5 cursor-pointer bg-transparent text-text-secondary hover:text-text hover:border-border-hover disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            {current === 0 ? '/cancel' : '/back'}
          </button>
          <button
            onClick={next}
            disabled={step.nextDisabled || deploying}
            className="font-mono text-xs font-medium border border-accent/30 rounded-md px-5 py-1.5 cursor-pointer bg-accent/10 text-accent-text hover:bg-accent/20 hover:border-accent/50 disabled:opacity-40 disabled:cursor-not-allowed transition-all inline-flex items-center gap-2"
          >
            {deploying ? (
              <span>
                deploying<span className="animate-[blink_1s_step-end_infinite]">_</span>
              </span>
            ) : isLast ? (
              '/deploy'
            ) : (
              '/next'
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
