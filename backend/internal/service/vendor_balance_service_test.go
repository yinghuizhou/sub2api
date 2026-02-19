//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestVendorBalanceService_TrackUsage(t *testing.T) {
	ctx := context.Background()

	t.Run("token billing tracks cost", func(t *testing.T) {
		repo := newMockVendorRepo()
		inputCost := 0.01
		outputCost := 0.03
		balance := 10.0
		repo.vendors[1] = &Vendor{
			ID:              1,
			BillingType:     VendorBillingTypeToken,
			CostPer1kInput:  &inputCost,
			CostPer1kOutput: &outputCost,
			BalanceUSD:      &balance,
			UsedQuotaUSD:    0,
			Status:          VendorStatusActive,
		}

		svc := NewVendorBalanceService(repo)
		err := svc.TrackUsage(ctx, 1, 1000, 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		v := repo.vendors[1]
		// expected cost: 1000/1000*0.01 + 1000/1000*0.03 = 0.04
		expectedUsed := 0.04
		if v.UsedQuotaUSD < expectedUsed-0.001 || v.UsedQuotaUSD > expectedUsed+0.001 {
			t.Errorf("expected used ~%.4f, got %.4f", expectedUsed, v.UsedQuotaUSD)
		}
		if v.BalanceUSD == nil || *v.BalanceUSD < 9.959 || *v.BalanceUSD > 9.961 {
			t.Errorf("expected balance ~9.96, got %v", v.BalanceUSD)
		}
	})

	t.Run("non-token billing is no-op", func(t *testing.T) {
		repo := newMockVendorRepo()
		repo.vendors[1] = &Vendor{
			ID:          1,
			BillingType: VendorBillingTypeSubscription,
			Status:      VendorStatusActive,
		}

		svc := NewVendorBalanceService(repo)
		err := svc.TrackUsage(ctx, 1, 5000, 5000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// UsedQuotaUSD should remain 0
		if repo.vendors[1].UsedQuotaUSD != 0 {
			t.Errorf("expected no usage tracked, got %.4f", repo.vendors[1].UsedQuotaUSD)
		}
	})

	t.Run("vendor not found", func(t *testing.T) {
		repo := newMockVendorRepo()
		svc := NewVendorBalanceService(repo)
		err := svc.TrackUsage(ctx, 999, 100, 100)
		if !errors.Is(err, ErrVendorNotFound) {
			t.Errorf("expected ErrVendorNotFound, got %v", err)
		}
	})
}

func TestVendorBalanceService_AutoSuspendDepleted(t *testing.T) {
	ctx := context.Background()

	t.Run("suspends depleted quota vendor", func(t *testing.T) {
		repo := newMockVendorRepo()
		totalQuota := 100.0
		repo.vendors[1] = &Vendor{
			ID:            1,
			Name:          "depleted-vendor",
			BillingType:   VendorBillingTypeQuota,
			TotalQuotaUSD: &totalQuota,
			UsedQuotaUSD:  100.0, // fully used
			Status:        VendorStatusActive,
		}

		svc := NewVendorBalanceService(repo)
		count, err := svc.AutoSuspendDepleted(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 suspended, got %d", count)
		}
		if repo.vendors[1].Status != VendorStatusDepleted {
			t.Errorf("expected status %q, got %q", VendorStatusDepleted, repo.vendors[1].Status)
		}
	})

	t.Run("suspends expired vendor", func(t *testing.T) {
		repo := newMockVendorRepo()
		past := time.Now().Add(-24 * time.Hour)
		repo.vendors[1] = &Vendor{
			ID:          1,
			Name:        "expired-vendor",
			BillingType: VendorBillingTypeSubscription,
			ExpiresAt:   &past,
			Status:      VendorStatusActive,
		}

		svc := NewVendorBalanceService(repo)
		count, err := svc.AutoSuspendDepleted(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 suspended, got %d", count)
		}
		if repo.vendors[1].Status != VendorStatusSuspended {
			t.Errorf("expected status %q, got %q", VendorStatusSuspended, repo.vendors[1].Status)
		}
	})

	t.Run("skips healthy active vendor", func(t *testing.T) {
		repo := newMockVendorRepo()
		totalQuota := 100.0
		repo.vendors[1] = &Vendor{
			ID:            1,
			BillingType:   VendorBillingTypeQuota,
			TotalQuotaUSD: &totalQuota,
			UsedQuotaUSD:  50.0,
			Status:        VendorStatusActive,
		}

		svc := NewVendorBalanceService(repo)
		count, err := svc.AutoSuspendDepleted(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 suspended, got %d", count)
		}
	})
}

func TestVendorBalanceService_GetCostAnalysis(t *testing.T) {
	ctx := context.Background()
	repo := newMockVendorRepo()
	totalQuota := 200.0
	balance := 150.0
	repo.vendors[1] = &Vendor{
		ID:            1,
		Name:          "analysis-vendor",
		TotalQuotaUSD: &totalQuota,
		UsedQuotaUSD:  50.0,
		BalanceUSD:    &balance,
		Status:        VendorStatusActive,
	}

	svc := NewVendorBalanceService(repo)
	analysis, err := svc.GetCostAnalysis(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if analysis.TotalQuotaUSD != 200.0 {
		t.Errorf("expected total 200, got %.2f", analysis.TotalQuotaUSD)
	}
	if analysis.UsedQuotaUSD != 50.0 {
		t.Errorf("expected used 50, got %.2f", analysis.UsedQuotaUSD)
	}
	// When BalanceUSD is set, RemainingUSD should use it
	if analysis.RemainingUSD != 150.0 {
		t.Errorf("expected remaining 150, got %.2f", analysis.RemainingUSD)
	}
	if analysis.UsagePercent < 24.9 || analysis.UsagePercent > 25.1 {
		t.Errorf("expected usage ~25%%, got %.2f%%", analysis.UsagePercent)
	}
}
