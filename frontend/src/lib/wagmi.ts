import { getDefaultConfig } from '@rainbow-me/rainbowkit'
import { mainnet } from 'wagmi/chains'

export function createWagmiConfig(projectId: string) {
  return getDefaultConfig({
    appName: 'openclaw creator',
    projectId,
    chains: [mainnet],
  })
}
