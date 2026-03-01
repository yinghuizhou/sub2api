package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrVendorNotFound    = infraerrors.NotFound("VENDOR_NOT_FOUND", "vendor not found")
	ErrVendorNilInput    = infraerrors.BadRequest("VENDOR_NIL_INPUT", "vendor input cannot be nil")
	ErrVendorHasAccounts = infraerrors.BadRequest("VENDOR_HAS_ACCOUNTS", "cannot delete vendor with associated accounts")
)

// VendorRepository 供应商数据访问接口
type VendorRepository interface {
	Create(ctx context.Context, vendor *Vendor) error
	GetByID(ctx context.Context, id int64) (*Vendor, error)
	Update(ctx context.Context, vendor *Vendor) error
	Delete(ctx context.Context, id int64) error
	CountAccountsByVendorID(ctx context.Context, vendorID int64) (int, error)
	List(ctx context.Context, params pagination.PaginationParams) ([]Vendor, *pagination.PaginationResult, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, status, apiFormat, billingType, search string) ([]Vendor, *pagination.PaginationResult, error)
	ListActive(ctx context.Context) ([]Vendor, error)
	ListByIDs(ctx context.Context, ids []int64) ([]Vendor, error)
	ListByStatus(ctx context.Context, status string) ([]Vendor, error)
	ListHealthCheckDue(ctx context.Context) ([]Vendor, error)
	ListBalanceAlertDue(ctx context.Context) ([]Vendor, error)
	UpdateHealthStatus(ctx context.Context, id int64, status string, latency *int, errMsg *string, consecutiveFailures int) error
	UpdateBalance(ctx context.Context, id int64, balanceUSD *float64, usedQuotaUSD float64) error
	UpdateStatus(ctx context.Context, id int64, status string) error
}

// CreateVendorInput 创建供应商请求
type CreateVendorInput struct {
	Name                  string            `json:"name"`
	Description           *string           `json:"description"`
	APIFormat             string            `json:"api_format"`
	BaseURL               string            `json:"base_url"`
	AuthType              string            `json:"auth_type"`
	APIPathOverride       *string           `json:"api_path_override"`
	ExtraHeaders          map[string]string `json:"extra_headers"`
	BillingType           string            `json:"billing_type"`
	CostPer1kInput        *float64          `json:"cost_per_1k_input"`
	CostPer1kOutput       *float64          `json:"cost_per_1k_output"`
	TotalQuotaUSD         *float64          `json:"total_quota_usd"`
	BalanceUSD            *float64          `json:"balance_usd"`
	ExpiresAt             *time.Time        `json:"expires_at"`
	HealthCheckEnabled    bool              `json:"health_check_enabled"`
	HealthCheckInterval   int               `json:"health_check_interval"`
	HealthCheckModel      string            `json:"health_check_model"`
	BalanceAlertEnabled   bool              `json:"balance_alert_enabled"`
	BalanceAlertThreshold *float64          `json:"balance_alert_threshold"`
}

// UpdateVendorInput 更新供应商请求
type UpdateVendorInput struct {
	Name                  *string           `json:"name"`
	Description           *string           `json:"description"`
	APIFormat             *string           `json:"api_format"`
	BaseURL               *string           `json:"base_url"`
	AuthType              *string           `json:"auth_type"`
	APIPathOverride       *string           `json:"api_path_override"`
	ExtraHeaders          map[string]string `json:"extra_headers"`
	BillingType           *string           `json:"billing_type"`
	CostPer1kInput        *float64          `json:"cost_per_1k_input"`
	CostPer1kOutput       *float64          `json:"cost_per_1k_output"`
	TotalQuotaUSD         *float64          `json:"total_quota_usd"`
	BalanceUSD            *float64          `json:"balance_usd"`
	ExpiresAt             *time.Time        `json:"expires_at"`
	Status                *string           `json:"status"`
	HealthCheckEnabled    *bool             `json:"health_check_enabled"`
	HealthCheckInterval   *int              `json:"health_check_interval"`
	HealthCheckModel      *string           `json:"health_check_model"`
	BalanceAlertEnabled   *bool             `json:"balance_alert_enabled"`
	BalanceAlertThreshold *float64          `json:"balance_alert_threshold"`
	AutoPurchaseEnabled   *bool             `json:"auto_purchase_enabled"`
	AutoPurchaseConfig    map[string]any    `json:"auto_purchase_config"`
}

// VendorService 供应商业务逻辑服务
type VendorService struct {
	vendorRepo VendorRepository
}

// NewVendorService 创建供应商服务
func NewVendorService(vendorRepo VendorRepository) *VendorService {
	return &VendorService{vendorRepo: vendorRepo}
}

// Create 创建供应商
func (s *VendorService) Create(ctx context.Context, input *CreateVendorInput) (*Vendor, error) {
	if input == nil {
		return nil, ErrVendorNilInput
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, infraerrors.BadRequest("VENDOR_INVALID_INPUT", "vendor name is required")
	}
	if strings.TrimSpace(input.BaseURL) == "" {
		return nil, infraerrors.BadRequest("VENDOR_INVALID_INPUT", "vendor base_url is required")
	}
	validAPIFormats := map[string]bool{VendorAPIFormatAnthropic: true, VendorAPIFormatOpenAI: true}
	if !validAPIFormats[input.APIFormat] {
		return nil, infraerrors.BadRequest("VENDOR_INVALID_INPUT", "vendor api_format must be 'anthropic' or 'openai'")
	}
	validAuthTypes := map[string]bool{VendorAuthTypeAPIKey: true, VendorAuthTypeSession: true, VendorAuthTypeBearer: true}
	if !validAuthTypes[input.AuthType] {
		return nil, infraerrors.BadRequest("VENDOR_INVALID_INPUT", "vendor auth_type must be 'api_key', 'session', or 'bearer'")
	}

	vendor := &Vendor{
		Name:                  input.Name,
		Description:           input.Description,
		APIFormat:             input.APIFormat,
		BaseURL:               input.BaseURL,
		AuthType:              input.AuthType,
		APIPathOverride:       input.APIPathOverride,
		ExtraHeaders:          input.ExtraHeaders,
		BillingType:           input.BillingType,
		CostPer1kInput:        input.CostPer1kInput,
		CostPer1kOutput:       input.CostPer1kOutput,
		TotalQuotaUSD:         input.TotalQuotaUSD,
		BalanceUSD:            input.BalanceUSD,
		ExpiresAt:             input.ExpiresAt,
		Status:                VendorStatusActive,
		HealthCheckEnabled:    input.HealthCheckEnabled,
		HealthCheckInterval:   input.HealthCheckInterval,
		HealthCheckModel:      input.HealthCheckModel,
		BalanceAlertEnabled:   input.BalanceAlertEnabled,
		BalanceAlertThreshold: input.BalanceAlertThreshold,
	}

	if vendor.ExtraHeaders == nil {
		vendor.ExtraHeaders = map[string]string{}
	}
	if vendor.HealthCheckInterval <= 0 {
		vendor.HealthCheckInterval = 300
	}
	if vendor.HealthCheckModel == "" {
		vendor.HealthCheckModel = VendorDefaultHealthCheckModel
	}

	if err := s.vendorRepo.Create(ctx, vendor); err != nil {
		return nil, err
	}
	return vendor, nil
}

// GetByID 获取供应商
func (s *VendorService) GetByID(ctx context.Context, id int64) (*Vendor, error) {
	return s.vendorRepo.GetByID(ctx, id)
}

// Update 更新供应商
func (s *VendorService) Update(ctx context.Context, id int64, input *UpdateVendorInput) (*Vendor, error) {
	if input == nil {
		return nil, ErrVendorNilInput
	}

	vendor, err := s.vendorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		vendor.Name = *input.Name
	}
	if input.Description != nil {
		vendor.Description = input.Description
	}
	if input.APIFormat != nil {
		vendor.APIFormat = *input.APIFormat
	}
	if input.BaseURL != nil {
		vendor.BaseURL = *input.BaseURL
	}
	if input.AuthType != nil {
		vendor.AuthType = *input.AuthType
	}
	if input.APIPathOverride != nil {
		vendor.APIPathOverride = input.APIPathOverride
	}
	if input.ExtraHeaders != nil {
		vendor.ExtraHeaders = input.ExtraHeaders
	}
	if input.BillingType != nil {
		vendor.BillingType = *input.BillingType
	}
	if input.CostPer1kInput != nil {
		vendor.CostPer1kInput = input.CostPer1kInput
	}
	if input.CostPer1kOutput != nil {
		vendor.CostPer1kOutput = input.CostPer1kOutput
	}
	if input.TotalQuotaUSD != nil {
		vendor.TotalQuotaUSD = input.TotalQuotaUSD
	}
	if input.BalanceUSD != nil {
		vendor.BalanceUSD = input.BalanceUSD
	}
	if input.ExpiresAt != nil {
		vendor.ExpiresAt = input.ExpiresAt
	}
	if input.Status != nil {
		vendor.Status = *input.Status
	}
	if input.HealthCheckEnabled != nil {
		vendor.HealthCheckEnabled = *input.HealthCheckEnabled
	}
	if input.HealthCheckInterval != nil {
		vendor.HealthCheckInterval = *input.HealthCheckInterval
	}
	if input.HealthCheckModel != nil {
		vendor.HealthCheckModel = *input.HealthCheckModel
	}
	if input.BalanceAlertEnabled != nil {
		vendor.BalanceAlertEnabled = *input.BalanceAlertEnabled
	}
	if input.BalanceAlertThreshold != nil {
		vendor.BalanceAlertThreshold = input.BalanceAlertThreshold
	}
	if input.AutoPurchaseEnabled != nil {
		vendor.AutoPurchaseEnabled = *input.AutoPurchaseEnabled
	}
	if input.AutoPurchaseConfig != nil {
		vendor.AutoPurchaseConfig = input.AutoPurchaseConfig
	}

	if err := s.vendorRepo.Update(ctx, vendor); err != nil {
		return nil, err
	}
	return vendor, nil
}

// Delete 删除供应商（软删除）
func (s *VendorService) Delete(ctx context.Context, id int64) error {
	// 检查是否有关联的账户
	count, err := s.vendorRepo.CountAccountsByVendorID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrVendorHasAccounts
	}
	return s.vendorRepo.Delete(ctx, id)
}

// List 供应商列表
func (s *VendorService) List(ctx context.Context, params pagination.PaginationParams, status, apiFormat, billingType, search string) ([]Vendor, *pagination.PaginationResult, error) {
	return s.vendorRepo.ListWithFilters(ctx, params, status, apiFormat, billingType, search)
}

// ListActive 获取所有活跃供应商
func (s *VendorService) ListActive(ctx context.Context) ([]Vendor, error) {
	return s.vendorRepo.ListActive(ctx)
}
