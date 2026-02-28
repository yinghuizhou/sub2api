import { apiClient } from '../client'

export interface VpnTunnel {
  name: string
  config_name: string
  region: string
  server_ip: string
  socks_port: number
  status: string
  tun_device: string
  local_ip: string
  exit_ip: string
  health: string
  latency_ms: number
  uptime: string
  consecutive_failures: number
  last_check: string
  proxy_id?: number
}

export interface VpnOvpnConfig {
  name: string
  region: string
  server_ip: string
  file_path: string
}

export interface CreateTunnelInput {
  name: string
  config_name: string
  region?: string
  socks_port?: number
  proxy_id?: number
}

export interface AgentHealth {
  status: string
  uptime: string
  tunnels: number
}

export async function getAgentHealth(): Promise<AgentHealth> {
  const { data } = await apiClient.get<AgentHealth>('/admin/vpn/health')
  return data
}

export async function listTunnels(): Promise<VpnTunnel[]> {
  const { data } = await apiClient.get<VpnTunnel[]>('/admin/vpn/tunnels')
  return data
}

export async function createTunnel(input: CreateTunnelInput): Promise<VpnTunnel> {
  const { data } = await apiClient.post<VpnTunnel>('/admin/vpn/tunnels', input)
  return data
}

export async function removeTunnel(name: string): Promise<void> {
  await apiClient.delete(`/admin/vpn/tunnels/${encodeURIComponent(name)}`)
}

export async function restartTunnel(name: string): Promise<void> {
  await apiClient.post(`/admin/vpn/tunnels/${encodeURIComponent(name)}/restart`)
}

export async function startTunnel(name: string): Promise<void> {
  await apiClient.post(`/admin/vpn/tunnels/${encodeURIComponent(name)}/start`)
}

export async function stopTunnel(name: string): Promise<void> {
  await apiClient.post(`/admin/vpn/tunnels/${encodeURIComponent(name)}/stop`)
}

export async function getTunnelStatus(name: string): Promise<VpnTunnel> {
  const { data } = await apiClient.get<VpnTunnel>(`/admin/vpn/tunnels/${encodeURIComponent(name)}/status`)
  return data
}

export async function uploadConfigs(files: File[]): Promise<{ uploaded: string[]; count: number }> {
  const formData = new FormData()
  files.forEach((file) => formData.append('files', file))
  const { data } = await apiClient.post<{ uploaded: string[]; count: number }>(
    '/admin/vpn/configs/upload',
    formData,
    { headers: { 'Content-Type': undefined as unknown as string } }
  )
  return data
}

export async function listConfigs(): Promise<VpnOvpnConfig[]> {
  const { data } = await apiClient.get<VpnOvpnConfig[]>('/admin/vpn/configs')
  return data
}

export async function deleteConfig(name: string): Promise<void> {
  await apiClient.delete(`/admin/vpn/configs/${encodeURIComponent(name)}`)
}

export const vpnAPI = {
  getAgentHealth,
  listTunnels,
  createTunnel,
  removeTunnel,
  restartTunnel,
  startTunnel,
  stopTunnel,
  getTunnelStatus,
  uploadConfigs,
  listConfigs,
  deleteConfig
}

export default vpnAPI
