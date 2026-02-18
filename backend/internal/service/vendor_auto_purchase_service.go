package service

import (
	"context"
	"log/slog"
)

// VendorAutoPurchaseService 自动采购服务（Phase 3 框架）
type VendorAutoPurchaseService struct {
	vendorRepo     VendorRepository
	accountService *AccountService
}

// NewVendorAutoPurchaseService 创建自动采购服务
func NewVendorAutoPurchaseService(vendorRepo VendorRepository, accountService *AccountService) *VendorAutoPurchaseService {
	return &VendorAutoPurchaseService{
		vendorRepo:     vendorRepo,
		accountService: accountService,
	}
}

// CheckPurchaseNeeded 检查哪些供应商需要补货
func (s *VendorAutoPurchaseService) CheckPurchaseNeeded(ctx context.Context) ([]Vendor, error) {
	// TODO: Phase 3 - 检查余额低于阈值的供应商
	return []Vendor{}, nil
}

// ProcessAutoPurchase 执行自动采购流程
func (s *VendorAutoPurchaseService) ProcessAutoPurchase(ctx context.Context, vendorID int64) error {
	// TODO: Phase 3 - 对接支付接口自动充值
	slog.Info("[VendorAutoPurchase] auto-purchase not implemented", "vendor_id", vendorID)
	return nil
}

// CreateAccountFromPurchase 采购完成后自动创建 Account
func (s *VendorAutoPurchaseService) CreateAccountFromPurchase(ctx context.Context, vendorID int64, apiKey string, groupIDs []int64) (*Account, error) {
	vendor, err := s.vendorRepo.GetByID(ctx, vendorID)
	if err != nil {
		return nil, err
	}

	sourceType := "vendor"
	req := CreateAccountRequest{
		Name:        vendor.Name + " - Auto",
		Platform:    "claude",
		Type:        "api_key",
		Credentials: map[string]any{"api_key": apiKey, "base_url": vendor.BaseURL},
		Concurrency: 3,
		Priority:    10,
		GroupIDs:    groupIDs,
		VendorID:    &vendorID,
		SourceType:  sourceType,
	}

	return s.accountService.Create(ctx, req)
}
