import type { ServerInfo, CreateServerRequest, CreateServerResponse, ErrorResponse } from '../types'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    credentials: 'include',
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText })) as ErrorResponse
    throw new Error(body.error || res.statusText)
  }
  return res.json() as Promise<T>
}

export async function listServers(): Promise<ServerInfo[]> {
  return request<ServerInfo[]>('/servers')
}

export async function getServer(id: number): Promise<ServerInfo> {
  return request<ServerInfo>(`/servers/${id}`)
}

export async function createServer(req: CreateServerRequest): Promise<CreateServerResponse> {
  return request<CreateServerResponse>('/servers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export async function deleteServer(id: number): Promise<void> {
  await request(`/servers/${id}`, { method: 'DELETE' })
}

export async function setPublicKey(id: number, publicKeyPEM: string): Promise<void> {
  await request(`/servers/${id}/public-key`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ public_key_pem: publicKeyPEM }),
  })
}

export async function signedRequest(
  creatorPath: string,
  _nodePath: string,
  method: string,
  signHeaders: Record<string, string>,
  body?: string,
): Promise<Response> {
  const opts: RequestInit = {
    method,
    headers: { ...signHeaders },
    credentials: 'include',
  }
  if (body) {
    (opts.headers as Record<string, string>)['Content-Type'] = 'application/json'
    opts.body = body
  }
  return fetch(creatorPath, opts)
}

// Auth API
export async function requestChallenge(address: string): Promise<{ challenge: string }> {
  return request('/auth/challenge', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ address }),
  })
}

export async function verifyChallenge(
  address: string,
  signature: string,
  challenge: string,
): Promise<{ user: import('../types').User }> {
  return request('/auth/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ address, signature, challenge }),
  })
}

export async function getConfig(): Promise<{ walletconnect_project_id: string; providers: string[] }> {
  return request('/config')
}

export async function logout(): Promise<void> {
  await request('/auth/logout', { method: 'POST' })
}

export async function getMe(): Promise<{ user: import('../types').User }> {
  return request('/auth/me')
}

// Admin API
export async function listUsers(): Promise<import('../types').User[]> {
  return request('/admin/users')
}

export async function approveUser(id: number): Promise<void> {
  await request(`/admin/users/${id}/approve`, { method: 'POST' })
}

export async function deleteUser(id: number): Promise<void> {
  await request(`/admin/users/${id}`, { method: 'DELETE' })
}

export async function setSSHKey(sshPublicKey: string): Promise<void> {
  await request('/auth/ssh-key', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ssh_public_key: sshPublicKey }),
  })
}
