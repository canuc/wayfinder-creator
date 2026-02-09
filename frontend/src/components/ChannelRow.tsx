import { useState } from 'react'
import type { ChannelConfig } from '../types'

const CHANNEL_TYPES = ['telegram', 'discord', 'slack', 'whatsapp', 'signal', 'googlechat', 'mattermost']

interface Props {
  channel: ChannelConfig
  onChange: (channel: ChannelConfig) => void
  onRemove: () => void
}

const inputClass =
  'w-full bg-bg-input border border-border rounded-md text-text font-mono text-[0.75rem] px-2 py-1.5 focus:outline-none focus:border-accent/50 placeholder:text-text-dim transition-colors'

const labelClass = 'block font-mono text-[0.65rem] font-medium text-text-tertiary uppercase tracking-wider mb-1'

function TokenInput({
  value,
  onChange,
  placeholder,
  label,
}: {
  value: string
  onChange: (v: string) => void
  placeholder: string
  label?: string
}) {
  const [show, setShow] = useState(false)
  return (
    <div>
      {label && <label className={labelClass}>{label}</label>}
      <div className="relative">
        <input
          type={show ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          autoComplete="off"
          spellCheck={false}
          className={inputClass}
        />
        <button
          type="button"
          onClick={() => setShow(!show)}
          className="absolute right-2 top-1/2 -translate-y-1/2 bg-surface border border-border rounded px-1.5 py-0.5 text-text-tertiary font-mono text-[0.6rem] cursor-pointer hover:text-text-secondary hover:border-border-hover transition-colors"
        >
          {show ? 'hide' : 'show'}
        </button>
      </div>
    </div>
  )
}

function ChannelGuide({ type }: { type: string }) {
  if (type === 'telegram') {
    return (
      <div className="text-[0.7rem] text-text-tertiary leading-relaxed bg-surface/40 border border-border/50 rounded-md px-3 py-2 mt-1.5">
        <div className="font-mono text-[0.65rem] text-text-dim uppercase tracking-wider mb-1">// setup</div>
        <ol className="list-decimal list-inside space-y-0.5">
          <li>Open Telegram and message <span className="font-mono text-text-secondary">@BotFather</span></li>
          <li>Send <span className="font-mono text-text-secondary">/newbot</span> and follow the prompts</li>
          <li>Choose a name and a username ending in <span className="font-mono text-text-secondary">bot</span></li>
          <li>Copy the bot token BotFather gives you and paste it above</li>
        </ol>
      </div>
    )
  }

  if (type === 'discord') {
    return (
      <div className="text-[0.7rem] text-text-tertiary leading-relaxed bg-surface/40 border border-border/50 rounded-md px-3 py-2 mt-1.5">
        <div className="font-mono text-[0.65rem] text-text-dim uppercase tracking-wider mb-1">// setup</div>
        <ol className="list-decimal list-inside space-y-0.5">
          <li>Go to <a href="https://discord.com/developers/applications" target="_blank" rel="noopener noreferrer" className="text-accent-text hover:underline">discord.com/developers/applications</a> and create a New Application</li>
          <li>Go to <span className="font-mono text-text-secondary">Bot</span> section and copy the <span className="font-mono text-text-secondary">Token</span></li>
          <li>Enable <span className="font-mono text-text-secondary">Message Content Intent</span> and <span className="font-mono text-text-secondary">Server Members Intent</span> under Privileged Gateway Intents</li>
          <li>Go to <span className="font-mono text-text-secondary">OAuth2 &gt; URL Generator</span>, select scopes <span className="font-mono text-text-secondary">bot</span> + <span className="font-mono text-text-secondary">applications.commands</span></li>
          <li>Select permissions: View Channels, Send Messages, Read Message History, Embed Links, Attach Files</li>
          <li>Open the generated URL to invite the bot to your server</li>
        </ol>
      </div>
    )
  }

  if (type === 'slack') {
    return (
      <div className="text-[0.7rem] text-text-tertiary leading-relaxed bg-surface/40 border border-border/50 rounded-md px-3 py-2 mt-1.5">
        <div className="font-mono text-[0.65rem] text-text-dim uppercase tracking-wider mb-1">// setup</div>
        <ol className="list-decimal list-inside space-y-0.5">
          <li>Create a Slack app at <a href="https://api.slack.com/apps" target="_blank" rel="noopener noreferrer" className="text-accent-text hover:underline">api.slack.com/apps</a> (select "From scratch")</li>
          <li>Enable <span className="font-mono text-text-secondary">Socket Mode</span> in app settings and generate an <span className="font-mono text-text-secondary">App Token</span> (<span className="font-mono text-cyan">xapp-...</span>) with <span className="font-mono text-text-secondary">connections:write</span> scope</li>
          <li>Under <span className="font-mono text-text-secondary">OAuth &amp; Permissions</span>, add bot scopes: <span className="font-mono text-text-secondary">chat:write</span>, <span className="font-mono text-text-secondary">channels:history</span>, <span className="font-mono text-text-secondary">channels:read</span>, <span className="font-mono text-text-secondary">app_mentions:read</span>, <span className="font-mono text-text-secondary">im:history</span>, <span className="font-mono text-text-secondary">im:read</span>, <span className="font-mono text-text-secondary">im:write</span></li>
          <li>Install the app to your workspace and copy the <span className="font-mono text-text-secondary">Bot Token</span> (<span className="font-mono text-cyan">xoxb-...</span>)</li>
          <li>Under <span className="font-mono text-text-secondary">Event Subscriptions</span>, subscribe to: <span className="font-mono text-text-secondary">message.channels</span>, <span className="font-mono text-text-secondary">message.im</span>, <span className="font-mono text-text-secondary">app_mention</span></li>
          <li>Invite the bot to desired channels with <span className="font-mono text-text-secondary">/invite @yourbot</span></li>
        </ol>
      </div>
    )
  }

  return null
}

export function ChannelRow({ channel, onChange, onRemove }: Props) {
  const [showGuide, setShowGuide] = useState(false)
  const hasGuide = channel.type === 'telegram' || channel.type === 'discord' || channel.type === 'slack'

  return (
    <div className="border border-border/50 rounded-md p-3 mb-2 bg-surface/20">
      {/* Header row: type selector + remove */}
      <div className="flex items-center gap-2 mb-2">
        <select
          value={channel.type}
          onChange={(e) => onChange({ ...channel, type: e.target.value, token: '', account: '' })}
          className="bg-bg-input border border-border rounded-md text-text font-mono text-[0.75rem] px-2 py-1.5 focus:outline-none focus:border-accent/50 appearance-none cursor-pointer bg-[url('data:image/svg+xml,%3Csvg%20xmlns=%27http://www.w3.org/2000/svg%27%20width=%2712%27%20height=%2712%27%20fill=%27%234a4a64%27%20viewBox=%270%200%2016%2016%27%3E%3Cpath%20d=%27M8%2011L3%206h10z%27/%3E%3C/svg%3E')] bg-no-repeat bg-[right_10px_center] pr-7 transition-colors"
        >
          {CHANNEL_TYPES.map((t) => (
            <option key={t} value={t} className="bg-surface text-text">
              {t}
            </option>
          ))}
        </select>
        {hasGuide && (
          <button
            type="button"
            onClick={() => setShowGuide(!showGuide)}
            className="font-mono text-[0.65rem] text-text-dim hover:text-text-tertiary transition-colors cursor-pointer"
          >
            {showGuide ? 'hide guide' : 'setup guide'}
          </button>
        )}
        <div className="flex-1" />
        <button
          type="button"
          onClick={onRemove}
          className="bg-transparent border border-border rounded text-text-tertiary cursor-pointer font-mono text-[0.7rem] px-1.5 py-0.5 hover:text-danger hover:border-danger/40 transition-colors"
        >
          &times;
        </button>
      </div>

      {/* Inline guide */}
      {showGuide && <ChannelGuide type={channel.type} />}

      {/* Token fields */}
      <div className="grid gap-2 mt-2">
        {channel.type === 'slack' ? (
          <>
            <TokenInput
              label="bot_token (xoxb-...)"
              value={channel.token}
              onChange={(v) => onChange({ ...channel, token: v })}
              placeholder="xoxb-..."
            />
            <TokenInput
              label="app_token (xapp-...)"
              value={channel.account || ''}
              onChange={(v) => onChange({ ...channel, account: v })}
              placeholder="xapp-..."
            />
          </>
        ) : channel.type === 'telegram' ? (
          <TokenInput
            label="bot_token"
            value={channel.token}
            onChange={(v) => onChange({ ...channel, token: v })}
            placeholder="123456:ABC-DEF1234ghIkl-zyx57W2v..."
          />
        ) : channel.type === 'discord' ? (
          <TokenInput
            label="bot_token"
            value={channel.token}
            onChange={(v) => onChange({ ...channel, token: v })}
            placeholder="MTIzNDU2Nzg5MDEyMzQ1Njc4OQ..."
          />
        ) : (
          <TokenInput
            value={channel.token}
            onChange={(v) => onChange({ ...channel, token: v })}
            placeholder="bot token..."
          />
        )}
      </div>
    </div>
  )
}
