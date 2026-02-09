import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { WagmiProvider } from 'wagmi'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RainbowKitProvider, darkTheme } from '@rainbow-me/rainbowkit'
import { AuthProvider } from './contexts/AuthContext'
import { AuthGuard, AdminGuard } from './components/AuthGuard'
import { Connect } from './pages/Connect'
import { AwaitingApproval } from './pages/AwaitingApproval'
import { Dashboard } from './pages/Dashboard'
import { ServerDetail } from './pages/ServerDetail'
import { Deploy } from './pages/Deploy'
import { AdminUsers } from './pages/AdminUsers'
import { createWagmiConfig } from './lib/wagmi'
import '@rainbow-me/rainbowkit/styles.css'

const queryClient = new QueryClient()

export function App({ walletConnectProjectId }: { walletConnectProjectId: string }) {
  const wagmiConfig = createWagmiConfig(walletConnectProjectId)

  return (
    <WagmiProvider config={wagmiConfig}>
      <QueryClientProvider client={queryClient}>
        <RainbowKitProvider theme={darkTheme()}>
          <BrowserRouter>
            <AuthProvider>
              <Routes>
                <Route path="/connect" element={<Connect />} />
                <Route path="/awaiting" element={<AwaitingApproval />} />
                <Route
                  path="/"
                  element={
                    <AuthGuard>
                      <Dashboard />
                    </AuthGuard>
                  }
                />
                <Route
                  path="/servers/:id"
                  element={
                    <AuthGuard>
                      <ServerDetail />
                    </AuthGuard>
                  }
                />
                <Route
                  path="/deploy"
                  element={
                    <AuthGuard>
                      <Deploy />
                    </AuthGuard>
                  }
                />
                <Route
                  path="/admin/users"
                  element={
                    <AdminGuard>
                      <AdminUsers />
                    </AdminGuard>
                  }
                />
              </Routes>
            </AuthProvider>
          </BrowserRouter>
        </RainbowKitProvider>
      </QueryClientProvider>
    </WagmiProvider>
  )
}
