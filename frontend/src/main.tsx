import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App'
import { getConfig } from './lib/api'
import './index.css'

async function init() {
  const config = await getConfig()
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App walletConnectProjectId={config.walletconnect_project_id} providers={config.providers || []} />
    </StrictMode>,
  )
}

init()
