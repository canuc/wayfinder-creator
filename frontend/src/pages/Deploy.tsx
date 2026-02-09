import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import type { CreateServerRequest, CreateServerResponse } from '../types'
import { Layout } from '../components/Layout'
import { CreateServerWizard } from '../components/wizard/CreateServerWizard'
import { useAuth } from '../contexts/AuthContext'
import * as api from '../lib/api'

function DeploySuccess({
  server,
  sshPublicKey,
}: {
  server: CreateServerResponse
  sshPublicKey: boolean
}) {
  const navigate = useNavigate()
  const [copied, setCopied] = useState('')

  const sshCommand = sshPublicKey
    ? `ssh -i ~/openclaw-ssh-key root@${server.ipv4}`
    : `ssh root@${server.ipv4}`

  const handleCopy = async (text: string, label: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(label)
    setTimeout(() => setCopied(''), 2000)
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
            {server.name}
          </span>
        </div>
        <span className="font-mono text-[0.65rem] text-accent-text">
          deployed
        </span>
      </div>

      <div className="p-5 grid gap-5">
        <div>
          <div className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest mb-3">
            // deployment successful
          </div>
          <p className="text-sm text-text-secondary">
            <span className="font-mono text-text">{server.name}</span> is provisioning at{' '}
            <span className="font-mono text-text">{server.ipv4}</span>
          </p>
        </div>

        {/* SSH access */}
        <div>
          <div className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest mb-2">
            ssh_access
          </div>
          <div className="bg-[#04040a] border border-border/50 rounded-md px-4 py-3 flex items-center justify-between gap-3">
            <code className="text-[0.8rem] text-text-secondary font-mono break-all">{sshCommand}</code>
            <button
              type="button"
              onClick={() => handleCopy(sshCommand, 'ssh')}
              className="shrink-0 bg-surface border border-border rounded px-2 py-0.5 text-text-tertiary font-mono text-[0.65rem] cursor-pointer hover:text-text-secondary hover:border-border-hover transition-colors"
            >
              {copied === 'ssh' ? 'copied' : 'copy'}
            </button>
          </div>
          {sshPublicKey && (
            <p className="font-mono text-[0.7rem] text-text-dim mt-1.5">
              use the private key downloaded during setup
            </p>
          )}
          <p className="text-[0.75rem] text-text-tertiary mt-1.5">
            SSH available after provisioning completes.{' '}
            <a
              href="https://www.ssh.com/academy/ssh/command"
              target="_blank"
              rel="noopener noreferrer"
              className="text-accent-text hover:underline"
            >
              SSH guide
            </a>
          </p>
        </div>

        {/* Fund wallet */}
        <div>
          <div className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest mb-2">
            fund_wallet
          </div>
          <p className="text-xs text-text-secondary mb-2">
            Once provisioning completes, your server's wallet address will appear on the dashboard.
            Fund it so your agent can transact on-chain.
          </p>
          <div className="flex items-center gap-3">
            <a
              href="https://portal.cdp.coinbase.com/products/faucet"
              target="_blank"
              rel="noopener noreferrer"
              className="font-mono text-[0.75rem] text-accent-text hover:underline"
            >
              base faucet (testnet)
            </a>
            <span className="text-text-dim">|</span>
            <a
              href="https://bridge.base.org"
              target="_blank"
              rel="noopener noreferrer"
              className="font-mono text-[0.75rem] text-accent-text hover:underline"
            >
              bridge to base (mainnet)
            </a>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3 pt-3 border-t border-border/50">
          <button
            onClick={() => navigate('/')}
            className="font-mono text-xs font-medium border border-accent/30 rounded-md px-5 py-1.5 cursor-pointer bg-accent/10 text-accent-text hover:bg-accent/20 hover:border-accent/50 transition-all"
          >
            /dashboard
          </button>
          <button
            onClick={() => navigate('/deploy')}
            className="font-mono text-[0.75rem] font-medium border border-border rounded-md px-3 py-1.5 cursor-pointer bg-transparent text-text-secondary hover:text-text hover:border-border-hover transition-colors"
          >
            /deploy-another
          </button>
        </div>
      </div>
    </div>
  )
}

export function Deploy({ providers }: { providers: string[] }) {
  const navigate = useNavigate()
  const { user } = useAuth()
  const [deployedServer, setDeployedServer] = useState<CreateServerResponse | null>(null)
  const [hadSSHKey, setHadSSHKey] = useState(false)

  const handleSubmit = useCallback(
    async (req: CreateServerRequest) => {
      const res = await api.createServer(req)
      setHadSSHKey(!!req.ssh_public_key)
      setDeployedServer(res)
    },
    [],
  )

  const handleCancel = useCallback(() => {
    navigate('/')
  }, [navigate])

  if (deployedServer) {
    return (
      <Layout>
        <DeploySuccess server={deployedServer} sshPublicKey={hadSSHKey} />
      </Layout>
    )
  }

  return (
    <Layout>
      <CreateServerWizard
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        initialSSHKey={user?.ssh_public_key || undefined}
        providers={providers}
      />
    </Layout>
  )
}
