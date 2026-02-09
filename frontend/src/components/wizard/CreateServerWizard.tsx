import { useState, useMemo, useCallback, useEffect } from 'react'
import type { CreateServerRequest, ChannelConfig } from '../../types'
import { WizardShell } from './WizardShell'
import { ChannelRow } from '../ChannelRow'
import { getOrCreateKeypair } from '../../lib/crypto'
import { generateSSHKeypair, downloadFile, isEd25519Supported } from '../../lib/ssh'
import * as api from '../../lib/api'

const PROVIDER_META: Record<string, { label: string; placeholder: string }> = {
  anthropic: { label: 'Anthropic API key', placeholder: 'sk-ant-api03-...' },
  openai: { label: 'OpenAI API key', placeholder: 'sk-...' },
  gemini: { label: 'Gemini API key', placeholder: 'AI...' },
}

interface Props {
  onSubmit: (req: CreateServerRequest) => Promise<void>
  onCancel: () => void
  initialSSHKey?: string
}

export function CreateServerWizard({ onSubmit, onCancel, initialSSHKey }: Props) {
  const [name, setName] = useState('')

  // SSH key state
  const [sshMode, setSSHMode] = useState<'generate' | 'own'>(initialSSHKey ? 'own' : 'generate')
  const [sshKey, setSSHKey] = useState(initialSSHKey || '')
  const [generatedPublicKey, setGeneratedPublicKey] = useState('')
  const [generatedPrivateKey, setGeneratedPrivateKey] = useState('')
  const [downloadTriggered, setDownloadTriggered] = useState(false)
  const [keySaved, setKeySaved] = useState(false)
  const [ed25519Supported, setEd25519Supported] = useState(true)

  // AI provider state
  const [provider, setProvider] = useState('anthropic')
  const [providerKey, setProviderKey] = useState('')
  const [showProviderKey, setShowProviderKey] = useState(false)
  const [wayfinderKey, setWayfinderKey] = useState('')
  const [showWayfinderKey, setShowWayfinderKey] = useState(false)

  // Channels
  const [channels, setChannels] = useState<ChannelConfig[]>([])

  const [disclaimerAccepted, setDisclaimerAccepted] = useState(false)
  const [error, setError] = useState('')

  const meta = PROVIDER_META[provider] || PROVIDER_META.anthropic

  useEffect(() => {
    isEd25519Supported().then((supported) => {
      setEd25519Supported(supported)
      if (!supported && !initialSSHKey) {
        setSSHMode('own')
      }
    })
  }, [initialSSHKey])

  const handleGenerate = useCallback(async () => {
    try {
      const { privateKeyPEM, publicKeySSH } = await generateSSHKeypair()
      setGeneratedPublicKey(publicKeySSH)
      setGeneratedPrivateKey(privateKeyPEM)
      setSSHKey(publicKeySSH)
      setDownloadTriggered(false)
      setKeySaved(false)
    } catch (err) {
      setError('Failed to generate keypair: ' + (err as Error).message)
    }
  }, [])

  const handleDownload = useCallback(() => {
    if (!generatedPrivateKey) return
    downloadFile('openclaw-ssh-key', generatedPrivateKey)
    setDownloadTriggered(true)
  }, [generatedPrivateKey])

  // Determine the effective SSH key for step 2 validation
  const effectiveSSHKey = sshMode === 'generate' ? generatedPublicKey : sshKey

  // Step 2 next is disabled if:
  // - generate mode: need generated key + download + checkbox
  // - own mode: need a non-empty key
  const sshStepDisabled =
    sshMode === 'generate'
      ? !generatedPublicKey || !downloadTriggered || !keySaved
      : !sshKey.trim()

  const steps = useMemo(
    () => [
      // Step 1: Name
      {
        label: 'Name',
        content: (
          <div className="grid gap-4">
            <div>
              <label className="block font-mono text-[0.75rem] font-medium text-text-tertiary uppercase tracking-wider mb-2">
                server_name{' '}
                <span className="text-text-dim font-normal normal-case tracking-normal">(auto-generated if blank)</span>
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="claw-myserver"
                autoComplete="off"
                spellCheck={false}
                className="w-full bg-bg-input border border-border rounded-md text-text font-mono text-sm px-3 py-2 focus:outline-none focus:border-accent/50 placeholder:text-text-dim transition-colors"
              />
            </div>
          </div>
        ),
      },
      // Step 2: SSH Key
      {
        label: 'SSH Key',
        nextDisabled: sshStepDisabled,
        validate: () => {
          // Save SSH key to account if it changed
          if (effectiveSSHKey && effectiveSSHKey !== initialSSHKey) {
            api.setSSHKey(effectiveSSHKey).catch(() => {})
          }
          return true
        },
        content: (
          <div className="grid gap-4">
            <div className="flex gap-2">
              {ed25519Supported && (
                <button
                  type="button"
                  onClick={() => setSSHMode('generate')}
                  className={`font-mono text-[0.75rem] font-medium px-3 py-1.5 rounded-md border transition-colors cursor-pointer ${
                    sshMode === 'generate'
                      ? 'bg-accent/10 border-accent/30 text-accent-text'
                      : 'bg-transparent border-border text-text-tertiary hover:text-text-secondary hover:border-border-hover'
                  }`}
                >
                  generate
                </button>
              )}
              <button
                type="button"
                onClick={() => setSSHMode('own')}
                className={`font-mono text-[0.75rem] font-medium px-3 py-1.5 rounded-md border transition-colors cursor-pointer ${
                  sshMode === 'own'
                    ? 'bg-accent/10 border-accent/30 text-accent-text'
                    : 'bg-transparent border-border text-text-tertiary hover:text-text-secondary hover:border-border-hover'
                }`}
              >
                use own
              </button>
            </div>

            {!ed25519Supported && sshMode === 'own' && (
              <div className="font-mono text-[0.75rem] text-warning bg-warning-muted border border-warning/20 px-3 py-2 rounded-md">
                Browser doesn't support Ed25519. Paste your own public key.
              </div>
            )}

            {sshMode === 'generate' && (
              <div className="grid gap-3">
                {!generatedPublicKey ? (
                  <button
                    type="button"
                    onClick={handleGenerate}
                    className="font-mono text-xs font-medium border border-accent/30 rounded-md px-5 py-2 cursor-pointer bg-accent/10 text-accent-text hover:bg-accent/20 hover:border-accent/50 transition-all w-fit"
                  >
                    generate ed25519 keypair
                  </button>
                ) : (
                  <>
                    <div>
                      <label className="block font-mono text-[0.75rem] font-medium text-text-tertiary uppercase tracking-wider mb-1.5">
                        public_key
                      </label>
                      <textarea
                        readOnly
                        value={generatedPublicKey}
                        spellCheck={false}
                        className="w-full bg-bg-input border border-border rounded-md text-text-secondary font-mono text-[0.75rem] px-3 py-2 resize-none h-[52px] leading-relaxed"
                      />
                    </div>
                    <div className="flex items-center gap-3">
                      <button
                        type="button"
                        onClick={handleDownload}
                        className="font-mono text-[0.75rem] font-medium border border-border rounded-md px-3 py-1.5 cursor-pointer bg-surface text-text-secondary hover:text-text hover:border-border-hover transition-colors"
                      >
                        download private key
                      </button>
                      {downloadTriggered && (
                        <span className="font-mono text-[0.7rem] text-accent-text">downloaded</span>
                      )}
                    </div>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={keySaved}
                        onChange={(e) => setKeySaved(e.target.checked)}
                        className="accent-accent"
                      />
                      <span className="text-xs text-text-secondary">
                        I have saved my private key
                      </span>
                    </label>
                    <div className="font-mono text-[0.75rem] text-warning bg-warning-muted border border-warning/20 px-3 py-2 rounded-md">
                      This key is required to recover your wallet key - keep it safe.
                    </div>
                  </>
                )}
              </div>
            )}

            {sshMode === 'own' && (
              <div>
                <label className="block font-mono text-[0.75rem] font-medium text-text-tertiary uppercase tracking-wider mb-1.5">
                  ssh_public_key
                </label>
                <textarea
                  value={sshKey}
                  onChange={(e) => setSSHKey(e.target.value)}
                  placeholder="ssh-ed25519 AAAA..."
                  spellCheck={false}
                  className="w-full bg-bg-input border border-border rounded-md text-text font-mono text-sm px-3 py-2 focus:outline-none focus:border-accent/50 resize-y min-h-[72px] leading-relaxed placeholder:text-text-dim transition-colors"
                />
              </div>
            )}
          </div>
        ),
      },
      // Step 3: AI Provider
      {
        label: 'AI Provider',
        validate: () => {
          if (!wayfinderKey.trim()) {
            setError('Wayfinder API key is required')
            return false
          }
          setError('')
          return true
        },
        content: (
          <div className="grid gap-4">
            <div>
              <label className="block font-mono text-[0.75rem] font-medium text-text-tertiary uppercase tracking-wider mb-1.5">
                provider
              </label>
              <select
                value={provider}
                onChange={(e) => {
                  setProvider(e.target.value)
                  setProviderKey('')
                }}
                className="w-full bg-bg-input border border-border rounded-md text-text font-mono text-sm px-3 py-2 focus:outline-none focus:border-accent/50 appearance-none cursor-pointer bg-[url('data:image/svg+xml,%3Csvg%20xmlns=%27http://www.w3.org/2000/svg%27%20width=%2712%27%20height=%2712%27%20fill=%27%234a4a64%27%20viewBox=%270%200%2016%2016%27%3E%3Cpath%20d=%27M8%2011L3%206h10z%27/%3E%3C/svg%3E')] bg-no-repeat bg-[right_10px_center] transition-colors"
              >
                <option value="anthropic">Anthropic (Claude)</option>
                <option value="openai">OpenAI (ChatGPT)</option>
                <option value="gemini">Google Gemini</option>
              </select>
            </div>
            <div>
              <label className="block font-mono text-[0.75rem] font-medium text-text-tertiary uppercase tracking-wider mb-1.5">
                {meta.label.replace(/ /g, '_').toLowerCase()}
              </label>
              <div className="relative">
                <input
                  type={showProviderKey ? 'text' : 'password'}
                  value={providerKey}
                  onChange={(e) => setProviderKey(e.target.value)}
                  placeholder={meta.placeholder}
                  autoComplete="off"
                  spellCheck={false}
                  className="w-full bg-bg-input border border-border rounded-md text-text font-mono text-sm px-3 py-2 focus:outline-none focus:border-accent/50 placeholder:text-text-dim transition-colors"
                />
                <button
                  type="button"
                  onClick={() => setShowProviderKey(!showProviderKey)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 bg-surface border border-border rounded px-1.5 py-0.5 text-text-tertiary font-mono text-[0.65rem] cursor-pointer hover:text-text-secondary hover:border-border-hover transition-colors"
                >
                  {showProviderKey ? 'hide' : 'show'}
                </button>
              </div>
            </div>
            <div>
              <label className="block font-mono text-[0.75rem] font-medium text-text-tertiary uppercase tracking-wider mb-1.5">
                wayfinder_api_key{' '}
                <span className="text-red-400/60 font-normal">*</span>
              </label>
              <div className="relative">
                <input
                  type={showWayfinderKey ? 'text' : 'password'}
                  value={wayfinderKey}
                  onChange={(e) => setWayfinderKey(e.target.value)}
                  placeholder="wf-..."
                  autoComplete="off"
                  spellCheck={false}
                  className="w-full bg-bg-input border border-border rounded-md text-text font-mono text-sm px-3 py-2 focus:outline-none focus:border-accent/50 placeholder:text-text-dim transition-colors"
                />
                <button
                  type="button"
                  onClick={() => setShowWayfinderKey(!showWayfinderKey)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 bg-surface border border-border rounded px-1.5 py-0.5 text-text-tertiary font-mono text-[0.65rem] cursor-pointer hover:text-text-secondary hover:border-border-hover transition-colors"
                >
                  {showWayfinderKey ? 'hide' : 'show'}
                </button>
              </div>
            </div>
            {error && (
              <div className="font-mono text-[0.75rem] text-red-400 bg-danger-muted border border-danger/20 px-3 py-2 rounded-md">
                {error}
              </div>
            )}
          </div>
        ),
      },
      // Step 4: Channels
      {
        label: 'Channels',
        content: (
          <div>
            <label className="block font-mono text-[0.75rem] font-medium text-text-tertiary uppercase tracking-wider mb-3">
              channels <span className="text-text-dim font-normal normal-case tracking-normal">(optional)</span>
            </label>
            {channels.map((ch, i) => (
              <ChannelRow
                key={i}
                channel={ch}
                onChange={(updated) =>
                  setChannels(channels.map((c, j) => (j === i ? updated : c)))
                }
                onRemove={() => setChannels(channels.filter((_, j) => j !== i))}
              />
            ))}
            <button
              type="button"
              onClick={() =>
                setChannels([...channels, { type: 'telegram', token: '' }])
              }
              className="bg-transparent border border-dashed border-border rounded-md text-text-tertiary cursor-pointer font-mono text-[0.75rem] px-3 py-1.5 mt-1 hover:text-accent-text hover:border-accent/30 transition-colors"
            >
              + add channel
            </button>
            {channels.length === 0 && (
              <div className="font-mono text-[0.75rem] text-warning bg-warning-muted border border-warning/20 px-3 py-2 rounded-md mt-3">
                No channels configured. Your agent won't be reachable.
              </div>
            )}
          </div>
        ),
      },
      // Step 5: Review
      {
        label: 'Review',
        nextDisabled: !disclaimerAccepted,
        content: (
          <div className="grid gap-4 text-sm">
            <div className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest">
              // review configuration
            </div>
            <div className="grid grid-cols-[120px_1fr] gap-y-2.5 text-xs border border-border/50 rounded-md p-3 bg-surface/30">
              <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">name</span>
              <span className="text-text font-mono text-[0.8rem]">{name || '(auto)'}</span>
              <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">ssh_key</span>
              <span className="text-text font-mono text-[0.8rem]">
                {effectiveSSHKey ? effectiveSSHKey.slice(0, 40) + '...' : '(none)'}
              </span>
              <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">provider</span>
              <span className="text-text font-mono text-[0.8rem]">{provider}</span>
              <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">api_key</span>
              <span className="text-text font-mono text-[0.8rem]">
                {providerKey ? providerKey.slice(0, 8) + '...' : '(none)'}
              </span>
              <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">wayfinder</span>
              <span className="text-text font-mono text-[0.8rem]">
                {wayfinderKey.slice(0, 8) + '...'}
              </span>
              <span className="font-mono text-[0.7rem] text-text-tertiary uppercase">channels</span>
              <span className="text-text font-mono text-[0.8rem]">
                {channels.length > 0
                  ? channels.map((c) => c.type).join(', ')
                  : '(none)'}
              </span>
            </div>

            {/* Disclaimer */}
            <div className="border border-warning/20 rounded-md bg-warning-muted p-4 grid gap-2 text-xs text-text-secondary font-mono leading-relaxed">
              <div className="text-warning text-[0.7rem] font-medium uppercase tracking-widest font-mono">
                // disclaimer
              </div>
              <p>
                Any funds transferred to this server's wallet may be permanently
                lost due to software bugs, misconfiguration, or security
                vulnerabilities. Positions held by the agent are subject to
                market risk and may lose value.
              </p>
              <p>
                You are solely responsible for the maintenance, security, and
                operation of this server. This includes keeping the system
                updated, managing access credentials, and monitoring agent
                activity.
              </p>
              <p>
                This software is provided as-is with no warranty. By deploying,
                you acknowledge these risks and accept full responsibility.
              </p>
              <label className="flex items-start gap-2 cursor-pointer mt-1 font-sans">
                <input
                  type="checkbox"
                  checked={disclaimerAccepted}
                  onChange={(e) => setDisclaimerAccepted(e.target.checked)}
                  className="accent-accent mt-0.5"
                />
                <span className="text-text-secondary text-xs">
                  I understand and accept the risks
                </span>
              </label>
            </div>

            {error && (
              <div className="font-mono text-[0.75rem] text-red-400 bg-danger-muted border border-danger/20 px-3 py-2 rounded-md">
                {error}
              </div>
            )}
          </div>
        ),
      },
    ],
    [
      name, sshMode, sshKey, generatedPublicKey, generatedPrivateKey, downloadTriggered,
      keySaved, ed25519Supported, provider, providerKey, showProviderKey, wayfinderKey,
      showWayfinderKey, channels, meta, error, effectiveSSHKey, initialSSHKey,
      sshStepDisabled, disclaimerAccepted, handleGenerate, handleDownload,
    ],
  )

  const handleComplete = async () => {
    try {
      const kp = await getOrCreateKeypair()
      const req: CreateServerRequest = {
        wayfinder_api_key: wayfinderKey.trim(),
        public_key_pem: kp.publicKeyPEM,
      }
      if (name.trim()) req.name = name.trim()
      if (effectiveSSHKey.trim()) req.ssh_public_key = effectiveSSHKey.trim()
      if (providerKey.trim()) {
        if (provider === 'anthropic') req.anthropic_api_key = providerKey.trim()
        else if (provider === 'openai') req.openai_api_key = providerKey.trim()
        else if (provider === 'gemini') req.gemini_api_key = providerKey.trim()
      }
      const validChannels = channels.filter((c) => c.type && c.token.trim())
      if (validChannels.length > 0) req.channels = validChannels
      await onSubmit(req)
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <WizardShell steps={steps} onComplete={handleComplete} onCancel={onCancel} />
  )
}
