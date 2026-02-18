package service

import (
	"context"
	"fmt"
)

// VendorPricingSuggestion 定价建议
type VendorPricingSuggestion struct {
	VendorID          int64   `json:"vendor_id"`
	VendorName        string  `json:"vendor_name"`
	CostPer1kInput    float64 `json:"cost_per_1k_input"`
	CostPer1kOutput   float64 `json:"cost_per_1k_output"`
	SuggestedMultiplier float64 `json:"suggested_multiplier"`
	MarginPercent     float64 `json:"margin_percent"`
}

// VendorPricingService 动态定价引擎（Phase 3 框架）
type VendorPricingService struct {
	vendorRepo VendorRepository
}

// NewVendorPricingService 创建定价服务
func NewVendorPricingService(vendorRepo VendorRepository) *VendorPricingService {
	return &VendorPricingService{vendorRepo: vendorRepo}
}

// CalculateSuggestedPrice 根据供应商成本计算建议售价
func (s *VendorPricingService) CalculateSuggestedPrice(ctx context.Context, vendorID int64, targetMargin float64) (*VendorPricingSuggestion, error) {
	vendor, err := s.vendorRepo.GetByID(ctx, vendorID)
	if err != nil {
		return nil, err
	}

	if targetMargin <= 0 {
		targetMargin = 0.3 // 默认 30% 利润率
	}

	suggestion := &VendorPricingSuggestion{
		VendorID:      vendor.ID,
		VendorName:    vendor.Name,
		MarginPercent: targetMargin * 100,
	}

	if vendor.CostPer1kInput != nil {
		suggestion.CostPer1kInput = *vendor.CostPer1kInput
		suggestion.SuggestedMultiplier = 1.0 / (1.0 - targetMargin)
	}
	if vendor.CostPer1kOutput != nil {
		suggestion.CostPer1kOutput = *vendor.CostPer1kOutput
	}

	return suggestion, nil
}

// RecalculateAllPrices 批量重新计算所有活跃供应商的建议售价
func (s *VendorPricingService) RecalculateAllPrices(ctx context.Context, targetMargin float64) ([]VendorPricingSuggestion, error) {
	vendors, err := s.vendorRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active vendors: %w", err)
	}

	var suggestions []VendorPricingSuggestion
	for _, v := range vendors {
		suggestion, err := s.CalculateSuggestedPrice(ctx, v.ID, targetMargin)
		if err != nil {
			continue
		}
		suggestions = append(suggestions, *suggestion)
	}
	return suggestions, nil
}
