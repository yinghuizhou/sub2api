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
	ID          int64
	Name        string
	Description *string

	// API 配置
	APIFormat       string
	BaseURL         string
	AuthType        string
	APIPathOverride *string
	ExtraHeaders    map[string]string

	// 计费信息
	BillingType     string
	CostPer1kInput  *float64
	CostPer1kOutput *float64
	TotalQuotaUSD   *float64
	UsedQuotaUSD    float64
	BalanceUSD      *float64
	ExpiresAt       *time.Time

	// 健康监控
	Status              string
	HealthCheckEnabled  bool
	HealthCheckInterval int
	HealthCheckModel    string
	LastHealthCheckAt   *time.Time
	LastHealthStatus    *string
	LastHealthLatency   *int
	ErrorMessage        *string
	ConsecutiveFailures int

	// 自动采购
	AutoPurchaseEnabled bool
	AutoPurchaseConfig  map[string]any

	// 余额预警
	BalanceAlertEnabled   bool
	BalanceAlertThreshold *float64

	// 时间戳
	CreatedAt time.Time
	UpdatedAt time.Time
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
