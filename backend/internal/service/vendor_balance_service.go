package service

import (
	"context"
	"fmt"
	"log/slog"
)

// VendorBalanceService 供应商余额与成本管理服务
type VendorBalanceService struct {
	vendorRepo VendorRepository
}

// NewVendorBalanceService 创建余额服务
func NewVendorBalanceService(vendorRepo VendorRepository) *VendorBalanceService {
	return &VendorBalanceService{vendorRepo: vendorRepo}
}

// TrackUsage 记录供应商用量（按 token 计费模式）
func (s *VendorBalanceService) TrackUsage(ctx context.Context, vendorID int64, inputTokens, outputTokens int) error {
	vendor, err := s.vendorRepo.GetByID(ctx, vendorID)
	if err != nil {
		return err
	}

	if vendor.BillingType != VendorBillingTypeToken {
		return nil // 非 token 计费模式不跟踪
	}

	var cost float64
	if vendor.CostPer1kInput != nil {
		cost += float64(inputTokens) / 1000.0 * *vendor.CostPer1kInput
	}
	if vendor.CostPer1kOutput != nil {
		cost += float64(outputTokens) / 1000.0 * *vendor.CostPer1kOutput
	}

	newUsed := vendor.UsedQuotaUSD + cost
	var newBalance *float64
	if vendor.BalanceUSD != nil {
		b := *vendor.BalanceUSD - cost
		newBalance = &b
	}

	return s.vendorRepo.UpdateBalance(ctx, vendorID, newBalance, newUsed)
}

// CheckBalanceAlerts 检查所有需要余额预警的供应商
func (s *VendorBalanceService) CheckBalanceAlerts(ctx context.Context) ([]Vendor, error) {
	vendors, err := s.vendorRepo.ListBalanceAlertDue(ctx)
	if err != nil {
		return nil, fmt.Errorf("list balance alert due: %w", err)
	}

	for _, v := range vendors {
		slog.Warn("vendor balance alert",
			"vendor_id", v.ID,
			"name", v.Name,
			"balance_usd", v.BalanceUSD,
			"threshold", v.BalanceAlertThreshold,
		)
		// TODO: 发送邮件/通知
	}
	return vendors, nil
}

// AutoSuspendDepleted 自动暂停额度耗尽或过期的供应商
func (s *VendorBalanceService) AutoSuspendDepleted(ctx context.Context) (int, error) {
	vendors, err := s.vendorRepo.ListActive(ctx)
	if err != nil {
		return 0, err
	}

	suspended := 0
	for _, v := range vendors {
		shouldSuspend := false
		reason := ""

		if v.IsDepleted() {
			shouldSuspend = true
			reason = "quota depleted"
		} else if v.IsExpired() {
			shouldSuspend = true
			reason = "subscription expired"
		}

		if shouldSuspend {
			status := VendorStatusDepleted
			if v.IsExpired() {
				status = VendorStatusSuspended
			}
			if err := s.vendorRepo.UpdateStatus(ctx, v.ID, status); err != nil {
				slog.Error("failed to suspend depleted vendor", "vendor_id", v.ID, "error", err)
				continue
			}
			suspended++
			slog.Warn("auto-suspended vendor", "vendor_id", v.ID, "name", v.Name, "reason", reason)
		}
	}
	return suspended, nil
}

// VendorCostAnalysis 供应商成本分析
type VendorCostAnalysis struct {
	VendorID      int64   `json:"vendor_id"`
	VendorName    string  `json:"vendor_name"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	TotalQuotaUSD float64 `json:"total_quota_usd"`
	UsedQuotaUSD  float64 `json:"used_quota_usd"`
	RemainingUSD  float64 `json:"remaining_usd"`
	UsagePercent  float64 `json:"usage_percent"`
}

// GetCostAnalysis 获取供应商成本分析
func (s *VendorBalanceService) GetCostAnalysis(ctx context.Context, vendorID int64) (*VendorCostAnalysis, error) {
	vendor, err := s.vendorRepo.GetByID(ctx, vendorID)
	if err != nil {
		return nil, err
	}

	analysis := &VendorCostAnalysis{
		VendorID:     vendor.ID,
		VendorName:   vendor.Name,
		UsedQuotaUSD: vendor.UsedQuotaUSD,
	}

	if vendor.TotalQuotaUSD != nil {
		analysis.TotalQuotaUSD = *vendor.TotalQuotaUSD
		analysis.RemainingUSD = *vendor.TotalQuotaUSD - vendor.UsedQuotaUSD
		if *vendor.TotalQuotaUSD > 0 {
			analysis.UsagePercent = vendor.UsedQuotaUSD / *vendor.TotalQuotaUSD * 100
		}
	}
	if vendor.BalanceUSD != nil {
		analysis.RemainingUSD = *vendor.BalanceUSD
	}

	return analysis, nil
}
