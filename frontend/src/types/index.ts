export interface ServerInfo {
  id: number
  name: string
  status: 'provisioning' | 'ready' | 'failed'
  ipv4: string
  provisioned: boolean
  wallet_address?: string
  default_key_removed: boolean
  has_node_api: boolean
  created_at?: string
  channel_count?: number
}

export interface ChannelConfig {
  type: string
  token: string
  name?: string
  account?: string
}

export interface CreateServerRequest {
  name?: string
  ssh_public_key?: string
  anthropic_api_key?: string
  openai_api_key?: string
  gemini_api_key?: string
  wayfinder_api_key: string
  channels?: ChannelConfig[]
  public_key_pem?: string
}

export interface CreateServerResponse {
  id: number
  name: string
  status: string
  ipv4: string
}

export interface LogEntry {
  id: number
  line: string
}

export interface PairingRequest {
  id: string
  channel: string
  user: string
  code: string
  status: string
  created_at: string
  server_id: number
  server_name: string
  meta?: {
    firstName?: string
    lastName?: string
  }
}

export interface User {
  id: number
  address: string
  role: 'admin' | 'user'
  approved: boolean
  ssh_public_key?: string
  created_at: string
}

export interface AuthState {
  user: User | null
  loading: boolean
}

export interface ErrorResponse {
  error: string
}

export type WSMessage =
  | { type: 'init'; server: { id: number; name: string; ipv4: string; status: string; default_key_removed: boolean } }
  | { type: 'log'; line: string }
  | { type: 'status'; status: string; default_key_removed: boolean }
