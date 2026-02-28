import { apiClient } from '../client'

export interface VpnTunnel {
  name: string
  tunnel_type: 'openvpn' | 'wireguard'
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
  cert_id?: string
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
  cert_id?: string
  tunnel_type?: 'openvpn' | 'wireguard'
}

export interface AgentHealth {
  status: string
  uptime: string
  tunnels: number
}

export interface VpnCertificate {
  id: string
  label: string
  created_at: string
  in_use: boolean
  active_tunnel?: string
}

export interface ImportCertInput {
  label: string
  ovpn_data: string
}

export async function getAgentHealth(): Promise<AgentHealth> {
  const { data } = await apiClient.get<AgentHealth>('/admin/vpn/health')
  return data
}

export async function listTunnels(): Promise<VpnTunnel[]> {
  const { data } = await apiClient.get<VpnTunnel[]>('/admin/vpn/tunnels')
  return data ?? []
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
  return data ?? []
}

export async function deleteConfig(name: string): Promise<void> {
  await apiClient.delete(`/admin/vpn/configs/${encodeURIComponent(name)}`)
}

export async function listCertificates(): Promise<VpnCertificate[]> {
  const { data } = await apiClient.get<VpnCertificate[]>('/admin/vpn/certificates')
  return data ?? []
}

export async function importCertificate(input: ImportCertInput): Promise<VpnCertificate> {
  const { data } = await apiClient.post<VpnCertificate>('/admin/vpn/certificates/import', input)
  return data
}

export async function deleteCertificate(id: string): Promise<void> {
  await apiClient.delete(`/admin/vpn/certificates/${encodeURIComponent(id)}`)
}

export interface PushConfigItem {
  filename: string
  content: string // base64
  region?: string
  auto_deploy: boolean
}

export interface PushConfigsInput {
  configs: PushConfigItem[]
  replace_existing: boolean
}

export interface PushConfigsResult {
  saved: string[]
  skipped: string[]
  deployed: string[]
  errors: string[]
}

export async function pushConfigs(input: PushConfigsInput): Promise<PushConfigsResult> {
  const { data } = await apiClient.post<PushConfigsResult>('/admin/vpn/configs/push', input)
  return data
}

export interface SyncProxiesResult {
  updated: number
  skipped: number
  failed: number
}

export async function syncTunnelProxies(): Promise<SyncProxiesResult> {
  const { data } = await apiClient.post<SyncProxiesResult>('/admin/vpn/sync-proxies')
  return data
}

// --- Dashboard & Events & Alert Rules ---

export interface DashboardData {
  active_tunnels: number
  offline_tunnels: number
  degraded_tunnels: number
  total_configs: number
  recent_events: VpnEvent[]
  stale_configs: string[]
}

export interface VpnEvent {
  id: number
  tunnel_name: string
  event_type: string
  details: Record<string, unknown>
  created_at: string
}

export interface AlertRule {
  id: number
  name: string
  condition: string
  threshold: number
  webhook_url: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateAlertRuleInput {
  name: string
  condition: string
  threshold: number
  webhook_url: string
}

export interface UpdateAlertRuleInput {
  name?: string
  condition?: string
  threshold?: number
  webhook_url?: string
  enabled?: boolean
}

export async function getDashboard(): Promise<DashboardData> {
  const { data } = await apiClient.get<DashboardData>('/admin/vpn/dashboard')
  return data
}

export async function listEvents(params?: {
  tunnel_name?: string
  event_type?: string
  page?: number
  page_size?: number
}): Promise<{ items: VpnEvent[]; total: number }> {
  const { data } = await apiClient.get<{ items: VpnEvent[]; total: number }>('/admin/vpn/events', { params })
  return data ?? { items: [], total: 0 }
}

export async function listAlertRules(): Promise<AlertRule[]> {
  const { data } = await apiClient.get<AlertRule[]>('/admin/vpn/alert-rules')
  return data ?? []
}

export async function createAlertRule(input: CreateAlertRuleInput): Promise<AlertRule> {
  const { data } = await apiClient.post<AlertRule>('/admin/vpn/alert-rules', input)
  return data
}

export async function updateAlertRule(id: number, input: UpdateAlertRuleInput): Promise<AlertRule> {
  const { data } = await apiClient.put<AlertRule>(`/admin/vpn/alert-rules/${id}`, input)
  return data
}

export async function deleteAlertRule(id: number): Promise<void> {
  await apiClient.delete(`/admin/vpn/alert-rules/${id}`)
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
  deleteConfig,
  listCertificates,
  importCertificate,
  deleteCertificate,
  syncTunnelProxies,
  pushConfigs,
  getDashboard,
  listEvents,
  listAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule
}

export default vpnAPI
