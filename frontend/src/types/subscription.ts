/**
 * 订阅配置
 */
export interface SubscriptionConfig {
  enabled: boolean
  daily_limit_usd: number
  subscription_period: 'daily' | 'weekly' | 'monthly'
  subscription_start: string  // ISO 8601 格式
  subscription_end: string    // ISO 8601 格式
}

/**
 * 日用量
 */
export interface DailyUsage {
  account_id: number
  date: string  // YYYY-MM-DD
  usage_usd: number
}

/**
 * 订阅状态
 */
export interface SubscriptionStatus {
  config: SubscriptionConfig | null
  usage: DailyUsage | null
  status: 'normal' | 'warning' | 'exceeded' | 'expired' | 'disabled'
  percentage: number  // 用量百分比 (0-100)
}

/**
 * 用量趋势数据
 */
export interface UsageTrend {
  date: string
  usage_usd: number
  limit_usd: number
}

/**
 * 告警规则
 */
export interface AlertRule {
  threshold: number  // 告警阈值 (0-100)
  enabled: boolean
}

/**
 * 批量配置请求
 */
export interface BatchSubscriptionConfigRequest {
  account_ids: number[]
  config: SubscriptionConfig
}

/**
 * 导出报表请求
 */
export interface ExportUsageReportRequest {
  account_ids?: number[]
  start_date: string
  end_date: string
  format: 'csv' | 'excel'
}
