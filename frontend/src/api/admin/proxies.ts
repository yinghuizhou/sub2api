/**
 * Admin Proxies API endpoints
 * Handles proxy server management for administrators
 */

import { apiClient } from '../client'
import type {
  Proxy,
  ProxyAccountSummary,
  ProxyQualityCheckResult,
  CreateProxyRequest,
  UpdateProxyRequest,
  PaginatedResponse,
  AdminDataPayload,
  AdminDataImportResult
} from '@/types'

/**
 * List all proxies with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters
 * @returns Paginated list of proxies
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    protocol?: string
    status?: 'active' | 'inactive'
    search?: string
    group_name?: string
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<Proxy>> {
  const { data } = await apiClient.get<PaginatedResponse<Proxy>>('/admin/proxies', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

/**
 * Get all active proxies (without pagination)
 * @returns List of all active proxies
 */
export async function getAll(): Promise<Proxy[]> {
  const { data } = await apiClient.get<Proxy[]>('/admin/proxies/all')
  return data
}

/**
 * Get all active proxies with account count (sorted by creation time desc)
 * @returns List of all active proxies with account count
 */
export async function getAllWithCount(): Promise<Proxy[]> {
  const { data } = await apiClient.get<Proxy[]>('/admin/proxies/all', {
    params: { with_count: 'true' }
  })
  return data
}

/**
 * Get proxy by ID
 * @param id - Proxy ID
 * @returns Proxy details
 */
export async function getById(id: number): Promise<Proxy> {
  const { data } = await apiClient.get<Proxy>(`/admin/proxies/${id}`)
  return data
}

/**
 * Create new proxy
 * @param proxyData - Proxy data
 * @returns Created proxy
 */
export async function create(proxyData: CreateProxyRequest): Promise<Proxy> {
  const { data } = await apiClient.post<Proxy>('/admin/proxies', proxyData)
  return data
}

/**
 * Update proxy
 * @param id - Proxy ID
 * @param updates - Fields to update
 * @returns Updated proxy
 */
export async function update(id: number, updates: UpdateProxyRequest): Promise<Proxy> {
  const { data } = await apiClient.put<Proxy>(`/admin/proxies/${id}`, updates)
  return data
}

/**
 * Delete proxy
 * @param id - Proxy ID
 * @returns Success confirmation
 */
export async function deleteProxy(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/proxies/${id}`)
  return data
}

/**
 * Toggle proxy status
 * @param id - Proxy ID
 * @param status - New status
 * @returns Updated proxy
 */
export async function toggleStatus(id: number, status: 'active' | 'inactive'): Promise<Proxy> {
  return update(id, { status })
}

/**
 * Test proxy connectivity
 * @param id - Proxy ID
 * @returns Test result with IP info
 */
export async function testProxy(id: number): Promise<{
  success: boolean
  message: string
  latency_ms?: number
  ip_address?: string
  city?: string
  region?: string
  country?: string
  country_code?: string
}> {
  const { data } = await apiClient.post<{
    success: boolean
    message: string
    latency_ms?: number
    ip_address?: string
    city?: string
    region?: string
    country?: string
    country_code?: string
  }>(`/admin/proxies/${id}/test`)
  return data
}

/**
 * Check proxy quality across common AI targets
 * @param id - Proxy ID
 * @returns Quality check result
 */
export async function checkProxyQuality(id: number): Promise<ProxyQualityCheckResult> {
  const { data } = await apiClient.post<ProxyQualityCheckResult>(`/admin/proxies/${id}/quality-check`)
  return data
}

/**
 * Get proxy usage statistics
 * @param id - Proxy ID
 * @returns Proxy usage statistics
 */
export async function getStats(id: number): Promise<{
  total_accounts: number
  active_accounts: number
  total_requests: number
  success_rate: number
  average_latency: number
}> {
  const { data } = await apiClient.get<{
    total_accounts: number
    active_accounts: number
    total_requests: number
    success_rate: number
    average_latency: number
  }>(`/admin/proxies/${id}/stats`)
  return data
}

/**
 * Get accounts using a proxy
 * @param id - Proxy ID
 * @returns List of accounts using the proxy
 */
export async function getProxyAccounts(id: number): Promise<ProxyAccountSummary[]> {
  const { data } = await apiClient.get<ProxyAccountSummary[]>(`/admin/proxies/${id}/accounts`)
  return data
}

/**
 * Batch create proxies
 * @param proxies - Array of proxy data to create
 * @returns Creation result with count of created and skipped
 */
export async function batchCreate(
  proxies: Array<{
    protocol: string
    host: string
    port: number
    username?: string
    password?: string
  }>
): Promise<{
  created: number
  skipped: number
}> {
  const { data } = await apiClient.post<{
    created: number
    skipped: number
  }>('/admin/proxies/batch', { proxies })
  return data
}

export async function batchDelete(ids: number[]): Promise<{
  deleted_ids: number[]
  skipped: Array<{ id: number; reason: string }>
}> {
  const { data } = await apiClient.post<{
    deleted_ids: number[]
    skipped: Array<{ id: number; reason: string }>
  }>('/admin/proxies/batch-delete', { ids })
  return data
}

export async function exportData(options?: {
  ids?: number[]
  filters?: {
    protocol?: string
    status?: 'active' | 'inactive'
    search?: string
  }
}): Promise<AdminDataPayload> {
  const params: Record<string, string> = {}
  if (options?.ids && options.ids.length > 0) {
    params.ids = options.ids.join(',')
  } else if (options?.filters) {
    const { protocol, status, search } = options.filters
    if (protocol) params.protocol = protocol
    if (status) params.status = status
    if (search) params.search = search
  }
  const { data } = await apiClient.get<AdminDataPayload>('/admin/proxies/data', { params })
  return data
}

export async function importData(payload: {
  data: AdminDataPayload
}): Promise<AdminDataImportResult> {
  const { data } = await apiClient.post<AdminDataImportResult>('/admin/proxies/data', payload)
  return data
}

/**
 * Get all distinct proxy group names
 * Used to populate autocomplete/datalist in proxy and account forms
 * @returns List of group names
 */
export async function getGroupNames(): Promise<string[]> {
  const { data } = await apiClient.get<string[]>('/admin/proxies/group-names')
  return data
}

/**
 * Trigger health check for all proxies
 * @returns Health check result summary
 */
export async function healthCheckAll(): Promise<{
  total: number
  healthy: number
  unhealthy: number
  errors: number
}> {
  const { data } = await apiClient.post<{
    total: number
    healthy: number
    unhealthy: number
    errors: number
  }>('/admin/proxies/health-check-all')
  return data
}

/**
 * Auto assign a proxy to an account
 * @param accountId - Account ID
 * @returns Assignment result
 */
export async function autoAssignProxy(accountId: number): Promise<{
  proxy_group: string
  proxy_id?: number
  message: string
}> {
  const { data } = await apiClient.post<{
    proxy_group: string
    proxy_id?: number
    message: string
  }>(`/admin/accounts/${accountId}/auto-assign-proxy`)
  return data
}

/**
 * Test proxy connectivity for an account
 * @param accountId - Account ID
 * @returns Test result
 */
export async function testAccountProxy(accountId: number): Promise<{
  success: boolean
  message: string
  latency_ms?: number
  exit_ip?: string
}> {
  const { data } = await apiClient.post<{
    success: boolean
    message: string
    latency_ms?: number
    exit_ip?: string
  }>(`/admin/accounts/${accountId}/test-proxy`)
  return data
}

// --- Proxy Group Assignment API ---

export interface ProxyGroupProxy {
  id: number
  name: string
  region: string
  health_status: string
  assignment_count: number
}

export interface ProxyGroupSummary {
  group_name: string
  total_proxies: number
  healthy_count: number
  total_accounts: number
  proxies: ProxyGroupProxy[]
}

export interface ProxyAssignmentDetail {
  account_id: number
  account_name: string
  platform: string
  proxy_id: number
  proxy_name: string
  assigned_by: string
  assigned_at: string
}

export interface DistributeResult {
  assigned: number
  skipped: number
}

export async function getGroupSummary(name: string, signal?: AbortSignal): Promise<ProxyGroupSummary> {
  const { data } = await apiClient.get<ProxyGroupSummary>(`/admin/proxies/groups/${encodeURIComponent(name)}/summary`, { signal })
  return data
}

export async function getGroupAssignments(name: string, signal?: AbortSignal): Promise<ProxyAssignmentDetail[]> {
  const { data } = await apiClient.get<ProxyAssignmentDetail[]>(`/admin/proxies/groups/${encodeURIComponent(name)}/assignments`, { signal })
  return data
}

export async function distributeAccounts(name: string, strategy: string = 'round-robin', signal?: AbortSignal): Promise<DistributeResult> {
  const { data } = await apiClient.post<DistributeResult>(`/admin/proxies/groups/${encodeURIComponent(name)}/distribute`, { strategy }, { signal })
  return data
}

export async function reassignAccount(accountId: number, proxyId: number): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(`/admin/proxies/assignments/${accountId}`, { proxy_id: proxyId })
  return data
}

export const proxiesAPI = {
  list,
  getAll,
  getAllWithCount,
  getById,
  create,
  update,
  delete: deleteProxy,
  toggleStatus,
  testProxy,
  checkProxyQuality,
  getStats,
  getProxyAccounts,
  batchCreate,
  batchDelete,
  exportData,
  importData,
  getGroupNames,
  healthCheckAll,
  autoAssignProxy,
  testAccountProxy,
  getGroupSummary,
  getGroupAssignments,
  distributeAccounts,
  reassignAccount
}

export default proxiesAPI
