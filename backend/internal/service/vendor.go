package service

import "time"

// Vendor 状态常量
const (
	VendorStatusActive    = "active"
	VendorStatusSuspended = "suspended"
	VendorStatusDepleted  = "depleted"
	VendorStatusError     = "error"
)

// Vendor API 格式常量
const (
	VendorAPIFormatAnthropic = "anthropic"
	VendorAPIFormatOpenAI    = "openai"
)

// Vendor 计费类型常量
const (
	VendorBillingTypeToken        = "token"
	VendorBillingTypeQuota        = "quota"
	VendorBillingTypeSubscription = "subscription"
)

// Vendor 认证类型常量
const (
	VendorAuthTypeAPIKey  = "api_key"
	VendorAuthTypeSession = "session"
	VendorAuthTypeBearer  = "bearer"
)

// Vendor 健康状态常量
const (
	VendorHealthOK      = "ok"
	VendorHealthSlow    = "slow"
	VendorHealthError   = "error"
	VendorHealthTimeout = "timeout"
)

// Vendor 供应商领域模型
type Vendor struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	// 渠道类型
	VendorType         string  `json:"vendor_type"`
	OfficialPlatform   *string `json:"official_platform,omitempty"`
	ResellerPlatform   *string `json:"reseller_platform,omitempty"`
	ResellerAPIKey     *string `json:"reseller_api_key,omitempty"`

	// API 配置
	APIFormat       string            `json:"api_format"`
	BaseURL         string            `json:"base_url"`
	AuthType        string            `json:"auth_type"`
	APIPathOverride *string           `json:"api_path_override,omitempty"`
	ExtraHeaders    map[string]string `json:"extra_headers"`

	// 计费信息
	BillingType     string     `json:"billing_type"`
	CostPer1kInput  *float64   `json:"cost_per_1k_input,omitempty"`
	CostPer1kOutput *float64   `json:"cost_per_1k_output,omitempty"`
	TotalQuotaUSD   *float64   `json:"total_quota_usd,omitempty"`
	UsedQuotaUSD    float64    `json:"used_quota_usd"`
	BalanceUSD      *float64   `json:"balance_usd,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`

	// 健康监控
	Status              string     `json:"status"`
	HealthCheckEnabled  bool       `json:"health_check_enabled"`
	HealthCheckInterval int        `json:"health_check_interval"`
	HealthCheckModel    string     `json:"health_check_model"`
	LastHealthCheckAt   *time.Time `json:"last_health_check_at,omitempty"`
	LastHealthStatus    *string    `json:"last_health_status,omitempty"`
	LastHealthLatency   *int       `json:"last_health_latency,omitempty"`
	ErrorMessage        *string    `json:"error_message,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`

	// 自动采购
	AutoPurchaseEnabled bool           `json:"auto_purchase_enabled"`
	AutoPurchaseConfig  map[string]any `json:"auto_purchase_config,omitempty"`

	// 余额预警
	BalanceAlertEnabled   bool     `json:"balance_alert_enabled"`
	BalanceAlertThreshold *float64 `json:"balance_alert_threshold,omitempty"`

	// 调度
	Priority    int `json:"priority"`
	Concurrency int `json:"concurrency"`

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (v *Vendor) IsActive() bool {
	return v.Status == VendorStatusActive
}

func (v *Vendor) IsHealthy() bool {
	if v.LastHealthStatus == nil {
		return true
	}
	return *v.LastHealthStatus == VendorHealthOK || *v.LastHealthStatus == VendorHealthSlow
}

// GetAPIPath 返回供应商的 API 路径
func (v *Vendor) GetAPIPath() string {
	if v.APIPathOverride != nil && *v.APIPathOverride != "" {
		return *v.APIPathOverride
	}
	switch v.APIFormat {
	case VendorAPIFormatOpenAI:
		return "/v1/chat/completions"
	default:
		return "/v1/messages"
	}
}

// NeedsBalanceAlert 检查是否需要余额预警
func (v *Vendor) NeedsBalanceAlert() bool {
	if !v.BalanceAlertEnabled || v.BalanceAlertThreshold == nil {
		return false
	}
	if v.BalanceUSD == nil {
		return false
	}
	return *v.BalanceUSD <= *v.BalanceAlertThreshold
}

// IsExpired 检查供应商是否已过期
func (v *Vendor) IsExpired() bool {
	if v.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*v.ExpiresAt)
}

// IsDepleted 检查供应商额度是否已耗尽
func (v *Vendor) IsDepleted() bool {
	if v.BillingType != VendorBillingTypeQuota {
		return false
	}
	if v.TotalQuotaUSD == nil {
		return false
	}
	return v.UsedQuotaUSD >= *v.TotalQuotaUSD
}

// IsHealthCheckDue 检查是否需要执行健康检查
func (v *Vendor) IsHealthCheckDue() bool {
	if !v.HealthCheckEnabled {
		return false
	}
	if v.LastHealthCheckAt == nil {
		return true
	}
	return time.Since(*v.LastHealthCheckAt) >= time.Duration(v.HealthCheckInterval)*time.Second
}
