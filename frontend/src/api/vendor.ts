import { apiClient } from './client'

export interface Vendor {
  id: number
  name: string
  description?: string
  api_format: 'anthropic' | 'openai'
  base_url: string
  auth_type: 'api_key' | 'session' | 'bearer'
  api_path_override?: string
  extra_headers: Record<string, string>
  billing_type: 'token' | 'quota' | 'subscription'
  cost_per_1k_input?: number
  cost_per_1k_output?: number
  total_quota_usd?: number
  used_quota_usd: number
  balance_usd?: number
  expires_at?: string
  status: 'active' | 'suspended' | 'depleted' | 'error'
  health_check_enabled: boolean
  health_check_interval: number
  health_check_model: string
  last_health_check_at?: string
  last_health_status?: 'ok' | 'slow' | 'error' | 'timeout'
  last_health_latency?: number
  error_message?: string
  consecutive_failures: number
  auto_purchase_enabled: boolean
  balance_alert_enabled: boolean
  balance_alert_threshold?: number
  created_at: string
  updated_at: string
}

export interface CreateVendorRequest {
  name: string
  description?: string
  api_format: string
  base_url: string
  auth_type: string
  api_path_override?: string
  extra_headers?: Record<string, string>
  billing_type: string
  cost_per_1k_input?: number
  cost_per_1k_output?: number
  total_quota_usd?: number
  balance_usd?: number
  expires_at?: string
  health_check_enabled?: boolean
  health_check_interval?: number
  health_check_model?: string
  balance_alert_enabled?: boolean
  balance_alert_threshold?: number
}

export type UpdateVendorRequest = Partial<CreateVendorRequest> & { status?: string }

export const vendorApi = {
  list: (params?: Record<string, any>) => apiClient.get('/admin/vendors', { params }),
  getById: (id: number) => apiClient.get(`/admin/vendors/${id}`),
  create: (data: CreateVendorRequest) => apiClient.post('/admin/vendors', data),
  update: (id: number, data: UpdateVendorRequest) => apiClient.put(`/admin/vendors/${id}`, data),
  delete: (id: number) => apiClient.delete(`/admin/vendors/${id}`),
  test: (id: number) => apiClient.post(`/admin/vendors/${id}/test`),
  refreshBalance: (id: number) => apiClient.post(`/admin/vendors/${id}/balance`),
  getStats: (id: number) => apiClient.get(`/admin/vendors/${id}/stats`),
  dashboard: () => apiClient.get('/admin/vendors/dashboard'),
  triggerHealthCheck: (id: number) => apiClient.post(`/admin/vendors/${id}/health-check`),
  getAccounts: (id: number) => apiClient.get(`/admin/vendors/${id}/accounts`),
  batchCreateAccounts: (id: number, data: any) =>
    apiClient.post(`/admin/vendors/${id}/accounts`, data),
}
