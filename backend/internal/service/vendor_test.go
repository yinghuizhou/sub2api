//go:build unit

package service

import (
	"testing"
	"time"
)

func TestVendor_GetAPIPath(t *testing.T) {
	t.Run("openai default", func(t *testing.T) {
		v := &Vendor{APIFormat: VendorAPIFormatOpenAI}
		if got := v.GetAPIPath(); got != "/v1/chat/completions" {
			t.Errorf("expected /v1/chat/completions, got %q", got)
		}
	})

	t.Run("anthropic default", func(t *testing.T) {
		v := &Vendor{APIFormat: VendorAPIFormatAnthropic}
		if got := v.GetAPIPath(); got != "/v1/messages" {
			t.Errorf("expected /v1/messages, got %q", got)
		}
	})

	t.Run("override takes precedence", func(t *testing.T) {
		override := "/custom/path"
		v := &Vendor{APIFormat: VendorAPIFormatOpenAI, APIPathOverride: &override}
		if got := v.GetAPIPath(); got != "/custom/path" {
			t.Errorf("expected /custom/path, got %q", got)
		}
	})

	t.Run("empty override uses default", func(t *testing.T) {
		empty := ""
		v := &Vendor{APIFormat: VendorAPIFormatOpenAI, APIPathOverride: &empty}
		if got := v.GetAPIPath(); got != "/v1/chat/completions" {
			t.Errorf("expected default path, got %q", got)
		}
	})
}

func TestVendor_IsExpired(t *testing.T) {
	t.Run("no expiry", func(t *testing.T) {
		v := &Vendor{}
		if v.IsExpired() {
			t.Error("expected not expired when ExpiresAt is nil")
		}
	})

	t.Run("future expiry", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		v := &Vendor{ExpiresAt: &future}
		if v.IsExpired() {
			t.Error("expected not expired for future date")
		}
	})

	t.Run("past expiry", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		v := &Vendor{ExpiresAt: &past}
		if !v.IsExpired() {
			t.Error("expected expired for past date")
		}
	})
}

func TestVendor_IsDepleted(t *testing.T) {
	t.Run("non-quota billing", func(t *testing.T) {
		v := &Vendor{BillingType: VendorBillingTypeToken}
		if v.IsDepleted() {
			t.Error("non-quota billing should never be depleted")
		}
	})

	t.Run("quota with headroom", func(t *testing.T) {
		total := 100.0
		v := &Vendor{BillingType: VendorBillingTypeQuota, TotalQuotaUSD: &total, UsedQuotaUSD: 50}
		if v.IsDepleted() {
			t.Error("should not be depleted with remaining quota")
		}
	})

	t.Run("quota fully used", func(t *testing.T) {
		total := 100.0
		v := &Vendor{BillingType: VendorBillingTypeQuota, TotalQuotaUSD: &total, UsedQuotaUSD: 100}
		if !v.IsDepleted() {
			t.Error("should be depleted when used >= total")
		}
	})

	t.Run("quota nil total", func(t *testing.T) {
		v := &Vendor{BillingType: VendorBillingTypeQuota, TotalQuotaUSD: nil}
		if v.IsDepleted() {
			t.Error("should not be depleted when TotalQuotaUSD is nil")
		}
	})
}

func TestVendor_NeedsBalanceAlert(t *testing.T) {
	t.Run("alert disabled", func(t *testing.T) {
		v := &Vendor{BalanceAlertEnabled: false}
		if v.NeedsBalanceAlert() {
			t.Error("should not alert when disabled")
		}
	})

	t.Run("no threshold", func(t *testing.T) {
		v := &Vendor{BalanceAlertEnabled: true, BalanceAlertThreshold: nil}
		if v.NeedsBalanceAlert() {
			t.Error("should not alert without threshold")
		}
	})

	t.Run("balance above threshold", func(t *testing.T) {
		balance := 50.0
		threshold := 10.0
		v := &Vendor{BalanceAlertEnabled: true, BalanceUSD: &balance, BalanceAlertThreshold: &threshold}
		if v.NeedsBalanceAlert() {
			t.Error("should not alert when balance > threshold")
		}
	})

	t.Run("balance at threshold", func(t *testing.T) {
		balance := 10.0
		threshold := 10.0
		v := &Vendor{BalanceAlertEnabled: true, BalanceUSD: &balance, BalanceAlertThreshold: &threshold}
		if !v.NeedsBalanceAlert() {
			t.Error("should alert when balance == threshold")
		}
	})

	t.Run("balance below threshold", func(t *testing.T) {
		balance := 5.0
		threshold := 10.0
		v := &Vendor{BalanceAlertEnabled: true, BalanceUSD: &balance, BalanceAlertThreshold: &threshold}
		if !v.NeedsBalanceAlert() {
			t.Error("should alert when balance < threshold")
		}
	})
}

func TestVendor_IsHealthy(t *testing.T) {
	t.Run("nil status is healthy", func(t *testing.T) {
		v := &Vendor{}
		if !v.IsHealthy() {
			t.Error("nil health status should be considered healthy")
		}
	})

	t.Run("ok is healthy", func(t *testing.T) {
		s := VendorHealthOK
		v := &Vendor{LastHealthStatus: &s}
		if !v.IsHealthy() {
			t.Error("ok status should be healthy")
		}
	})

	t.Run("slow is healthy", func(t *testing.T) {
		s := VendorHealthSlow
		v := &Vendor{LastHealthStatus: &s}
		if !v.IsHealthy() {
			t.Error("slow status should be healthy")
		}
	})

	t.Run("error is unhealthy", func(t *testing.T) {
		s := VendorHealthError
		v := &Vendor{LastHealthStatus: &s}
		if v.IsHealthy() {
			t.Error("error status should be unhealthy")
		}
	})
}

func TestVendor_IsHealthCheckDue(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		v := &Vendor{HealthCheckEnabled: false}
		if v.IsHealthCheckDue() {
			t.Error("should not be due when disabled")
		}
	})

	t.Run("never checked", func(t *testing.T) {
		v := &Vendor{HealthCheckEnabled: true, LastHealthCheckAt: nil}
		if !v.IsHealthCheckDue() {
			t.Error("should be due when never checked")
		}
	})

	t.Run("recently checked", func(t *testing.T) {
		recent := time.Now().Add(-10 * time.Second)
		v := &Vendor{HealthCheckEnabled: true, HealthCheckInterval: 300, LastHealthCheckAt: &recent}
		if v.IsHealthCheckDue() {
			t.Error("should not be due when recently checked")
		}
	})

	t.Run("overdue", func(t *testing.T) {
		old := time.Now().Add(-600 * time.Second)
		v := &Vendor{HealthCheckEnabled: true, HealthCheckInterval: 300, LastHealthCheckAt: &old}
		if !v.IsHealthCheckDue() {
			t.Error("should be due when interval exceeded")
		}
	})
}
