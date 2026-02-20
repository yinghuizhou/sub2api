import { apiClient } from './client'

export interface RechargePackage {
  id: number
  amount_cny: number
  bonus_rate: number
  bonus_fixed: number
  label: string | null
  is_active: boolean
  sort_order: number
}

export interface CreateOrderRequest {
  amount_cny: number
  package_id?: number
  channel: 'wechat' | 'alipay'
}

export interface PaymentOrder {
  id: number
  order_no: string
  amount_cny: number
  total_credit: number
  channel: string
  status: 'pending' | 'paid' | 'failed'
}

export const paymentApi = {
  listPackages: () =>
    apiClient.get<RechargePackage[]>('/payment/packages').then(r => r.data),

  createOrder: (data: CreateOrderRequest) =>
    apiClient.post<{ order: PaymentOrder; qr_url: string }>('/payment/create', data).then(r => r.data),

  getOrder: (id: number) =>
    apiClient.get<PaymentOrder>(`/payment/orders/${id}`).then(r => r.data),
}
