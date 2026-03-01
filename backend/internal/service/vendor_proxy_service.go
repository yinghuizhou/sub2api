package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// VendorProxyService 管理 Vendor 代理账号的创建、同步和删除
type VendorProxyService struct {
	vendorRepo  VendorRepository
	accountRepo AccountRepository
}

// NewVendorProxyService 创建 VendorProxyService
func NewVendorProxyService(vendorRepo VendorRepository, accountRepo AccountRepository) *VendorProxyService {
	return &VendorProxyService{
		vendorRepo:  vendorRepo,
		accountRepo: accountRepo,
	}
}

// inferPlatformFromVendor 从 Vendor 推断平台类型
func inferPlatformFromVendor(vendor *Vendor) string {
	if vendor.VendorType == domain.VendorTypeOfficial && vendor.OfficialPlatform != nil {
		return *vendor.OfficialPlatform
	}
	// reseller 根据 API 格式推断
	switch vendor.APIFormat {
	case VendorAPIFormatAnthropic:
		return PlatformAnthropic
	case VendorAPIFormatOpenAI:
		return PlatformOpenAI
	default:
		return PlatformAnthropic
	}
}

// CreateProxyAccount 为 Vendor 创建代理账号
// 如果已存在则同步更新
func (s *VendorProxyService) CreateProxyAccount(ctx context.Context, vendorID int64) (*Account, error) {
	vendor, err := s.vendorRepo.GetByID(ctx, vendorID)
	if err != nil {
		return nil, fmt.Errorf("get vendor failed: %w", err)
	}

	// 检查是否已存在代理账号
	existing, err := s.accountRepo.GetByVendorProxyID(ctx, vendorID)
	if err == nil && existing != nil {
		// 已存在，同步更新
		if err := s.syncProxyAccount(ctx, existing, vendor); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// 创建新的代理账号
	platform := inferPlatformFromVendor(vendor)
	proxyAccount := &Account{
		Name:          fmt.Sprintf("[Vendor] %s", vendor.Name),
		Platform:      platform,
		Type:          "vendor_proxy",
		Credentials:   map[string]any{},
		Extra:         map[string]any{},
		Concurrency:   vendor.Concurrency,
		Priority:      vendor.Priority,
		Status:        vendorStatusToAccountStatus(vendor.Status),
		Schedulable:   vendor.Status == VendorStatusActive,
		SourceType:    "vendor",
		IsVendorProxy: true,
		VendorProxyID: &vendor.ID,
		VendorID:      &vendor.ID,
	}

	if err := s.accountRepo.Create(ctx, proxyAccount); err != nil {
		return nil, fmt.Errorf("create vendor proxy account failed: %w", err)
	}

	slog.Info("vendor proxy account created",
		"vendor_id", vendor.ID,
		"vendor_name", vendor.Name,
		"account_id", proxyAccount.ID,
		"platform", platform,
	)
	return proxyAccount, nil
}

// SyncProxyAccount 同步 Vendor 状态到代理账号
func (s *VendorProxyService) SyncProxyAccount(ctx context.Context, vendorID int64) error {
	vendor, err := s.vendorRepo.GetByID(ctx, vendorID)
	if err != nil {
		return fmt.Errorf("get vendor failed: %w", err)
	}

	proxyAccount, err := s.accountRepo.GetByVendorProxyID(ctx, vendorID)
	if err != nil {
		return fmt.Errorf("get proxy account for vendor %d failed: %w", vendorID, err)
	}

	return s.syncProxyAccount(ctx, proxyAccount, vendor)
}

// DeleteProxyAccount 删除 Vendor 的代理账号
func (s *VendorProxyService) DeleteProxyAccount(ctx context.Context, vendorID int64) error {
	if err := s.accountRepo.DeleteByVendorProxyID(ctx, vendorID); err != nil {
		return fmt.Errorf("delete vendor proxy account failed: %w", err)
	}

	slog.Info("vendor proxy account deleted", "vendor_id", vendorID)
	return nil
}

// syncProxyAccount 内部方法：同步 Vendor 字段到代理账号
func (s *VendorProxyService) syncProxyAccount(ctx context.Context, account *Account, vendor *Vendor) error {
	account.Name = fmt.Sprintf("[Vendor] %s", vendor.Name)
	account.Platform = inferPlatformFromVendor(vendor)
	account.Priority = vendor.Priority
	account.Concurrency = vendor.Concurrency
	account.Schedulable = vendor.Status == VendorStatusActive
	account.Status = vendorStatusToAccountStatus(vendor.Status)

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("sync vendor proxy account failed: %w", err)
	}

	slog.Debug("vendor proxy account synced",
		"vendor_id", vendor.ID,
		"account_id", account.ID,
		"status", account.Status,
		"schedulable", account.Schedulable,
	)
	return nil
}

// vendorStatusToAccountStatus 将 Vendor 状态映射为 Account 状态
func vendorStatusToAccountStatus(vendorStatus string) string {
	switch vendorStatus {
	case VendorStatusActive:
		return StatusActive
	case VendorStatusSuspended, VendorStatusDepleted, VendorStatusError:
		return StatusDisabled
	default:
		return StatusDisabled
	}
}
