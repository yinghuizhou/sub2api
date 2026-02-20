import { apiClient } from './client'

export interface ReferralInfo {
  invite_code: string
  total_invitees: number
  total_commission: number
}

export interface ReferralCommission {
  id: number
  invitee_id: number
  order_amount_usd: number
  commission_rate: number
  commission_amount: number
  status: string
  created_at: string
}

export const referralApi = {
  getInfo: () =>
    apiClient.get<ReferralInfo>('/referral/info').then(r => r.data),

  listCommissions: (page = 1, limit = 20) =>
    apiClient.get<{ items: ReferralCommission[]; total: number }>('/referral/commissions', {
      params: { page, limit },
    }).then(r => r.data),
}
