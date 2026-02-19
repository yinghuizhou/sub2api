//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// --- mock VendorRepository ---

type mockVendorRepo struct {
	vendors map[int64]*Vendor
	nextID  int64
	// hooks for error injection
	createErr            error
	updateErr            error
	deleteErr            error
	getErr               error
	listActiveErr        error
	updateHealthErr      error
	updateBalanceErr     error
	updateStatusErr      error
	listHealthCheckDue   []Vendor
	listBalanceAlertDue  []Vendor
	accountCountByVendor map[int64]int
}

func newMockVendorRepo() *mockVendorRepo {
	return &mockVendorRepo{
		vendors:              make(map[int64]*Vendor),
		nextID:               1,
		accountCountByVendor: make(map[int64]int),
	}
}

func (m *mockVendorRepo) Create(_ context.Context, v *Vendor) error {
	if m.createErr != nil {
		return m.createErr
	}
	v.ID = m.nextID
	m.nextID++
	cp := *v
	m.vendors[v.ID] = &cp
	return nil
}

func (m *mockVendorRepo) GetByID(_ context.Context, id int64) (*Vendor, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	v, ok := m.vendors[id]
	if !ok {
		return nil, ErrVendorNotFound
	}
	cp := *v
	return &cp, nil
}

func (m *mockVendorRepo) Update(_ context.Context, v *Vendor) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.vendors[v.ID]; !ok {
		return ErrVendorNotFound
	}
	cp := *v
	m.vendors[v.ID] = &cp
	return nil
}

func (m *mockVendorRepo) Delete(_ context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.vendors, id)
	return nil
}

func (m *mockVendorRepo) CountAccountsByVendorID(_ context.Context, vendorID int64) (int, error) {
	return m.accountCountByVendor[vendorID], nil
}

func (m *mockVendorRepo) List(_ context.Context, params pagination.PaginationParams) ([]Vendor, *pagination.PaginationResult, error) {
	return m.ListWithFilters(context.Background(), params, "", "", "", "")
}

func (m *mockVendorRepo) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string) ([]Vendor, *pagination.PaginationResult, error) {
	var out []Vendor
	for _, v := range m.vendors {
		out = append(out, *v)
	}
	return out, &pagination.PaginationResult{Total: int64(len(out))}, nil
}

func (m *mockVendorRepo) ListActive(_ context.Context) ([]Vendor, error) {
	if m.listActiveErr != nil {
		return nil, m.listActiveErr
	}
	var out []Vendor
	for _, v := range m.vendors {
		if v.Status == VendorStatusActive {
			out = append(out, *v)
		}
	}
	return out, nil
}

func (m *mockVendorRepo) ListByStatus(_ context.Context, status string) ([]Vendor, error) {
	var out []Vendor
	for _, v := range m.vendors {
		if v.Status == status {
			out = append(out, *v)
		}
	}
	return out, nil
}

func (m *mockVendorRepo) ListHealthCheckDue(_ context.Context) ([]Vendor, error) {
	return m.listHealthCheckDue, nil
}

func (m *mockVendorRepo) ListBalanceAlertDue(_ context.Context) ([]Vendor, error) {
	return m.listBalanceAlertDue, nil
}

func (m *mockVendorRepo) UpdateHealthStatus(_ context.Context, id int64, status string, latency *int, errMsg *string, consecutiveFailures int) error {
	if m.updateHealthErr != nil {
		return m.updateHealthErr
	}
	v, ok := m.vendors[id]
	if !ok {
		return ErrVendorNotFound
	}
	v.LastHealthStatus = &status
	v.LastHealthLatency = latency
	v.ErrorMessage = errMsg
	v.ConsecutiveFailures = consecutiveFailures
	return nil
}

func (m *mockVendorRepo) UpdateBalance(_ context.Context, id int64, balanceUSD *float64, usedQuotaUSD float64) error {
	if m.updateBalanceErr != nil {
		return m.updateBalanceErr
	}
	v, ok := m.vendors[id]
	if !ok {
		return ErrVendorNotFound
	}
	v.BalanceUSD = balanceUSD
	v.UsedQuotaUSD = usedQuotaUSD
	return nil
}

func (m *mockVendorRepo) UpdateStatus(_ context.Context, id int64, status string) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	v, ok := m.vendors[id]
	if !ok {
		return ErrVendorNotFound
	}
	v.Status = status
	return nil
}

// --- Tests ---

func TestVendorService_Create(t *testing.T) {
	repo := newMockVendorRepo()
	svc := NewVendorService(repo)
	ctx := context.Background()

	t.Run("success with defaults", func(t *testing.T) {
		v, err := svc.Create(ctx, &CreateVendorInput{
			Name:      "test-vendor",
			APIFormat: VendorAPIFormatOpenAI,
			BaseURL:   "https://api.example.com",
			AuthType:  VendorAuthTypeAPIKey,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.ID == 0 {
			t.Fatal("expected non-zero ID")
		}
		if v.Status != VendorStatusActive {
			t.Errorf("expected status %q, got %q", VendorStatusActive, v.Status)
		}
		if v.HealthCheckInterval != 300 {
			t.Errorf("expected default interval 300, got %d", v.HealthCheckInterval)
		}
		if v.HealthCheckModel != VendorDefaultHealthCheckModel {
			t.Errorf("expected default model, got %q", v.HealthCheckModel)
		}
		if v.ExtraHeaders == nil {
			t.Error("expected ExtraHeaders to be initialized")
		}
	})

	t.Run("nil input returns error", func(t *testing.T) {
		_, err := svc.Create(ctx, nil)
		if !errors.Is(err, ErrVendorNilInput) {
			t.Errorf("expected ErrVendorNilInput, got %v", err)
		}
	})

	t.Run("repo error propagates", func(t *testing.T) {
		repoErr := newMockVendorRepo()
		repoErr.createErr = errors.New("db down")
		svcErr := NewVendorService(repoErr)
		_, err := svcErr.Create(ctx, &CreateVendorInput{
			Name:      "fail",
			BaseURL:   "https://example.com",
			APIFormat: VendorAPIFormatAnthropic,
			AuthType:  VendorAuthTypeAPIKey,
		})
		if err == nil || err.Error() != "db down" {
			t.Errorf("expected db error, got %v", err)
		}
	})
}

func TestVendorService_Update(t *testing.T) {
	repo := newMockVendorRepo()
	svc := NewVendorService(repo)
	ctx := context.Background()

	// seed a vendor
	v, _ := svc.Create(ctx, &CreateVendorInput{
		Name:      "original",
		APIFormat: VendorAPIFormatOpenAI,
		BaseURL:   "https://old.example.com",
		AuthType:  VendorAuthTypeAPIKey,
	})

	t.Run("partial update", func(t *testing.T) {
		newName := "updated"
		newURL := "https://new.example.com"
		updated, err := svc.Update(ctx, v.ID, &UpdateVendorInput{
			Name:    &newName,
			BaseURL: &newURL,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "updated" {
			t.Errorf("expected name 'updated', got %q", updated.Name)
		}
		if updated.BaseURL != "https://new.example.com" {
			t.Errorf("expected new URL, got %q", updated.BaseURL)
		}
		// unchanged field
		if updated.APIFormat != VendorAPIFormatOpenAI {
			t.Errorf("expected APIFormat unchanged, got %q", updated.APIFormat)
		}
	})

	t.Run("nil input returns error", func(t *testing.T) {
		_, err := svc.Update(ctx, v.ID, nil)
		if !errors.Is(err, ErrVendorNilInput) {
			t.Errorf("expected ErrVendorNilInput, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		name := "x"
		_, err := svc.Update(ctx, 9999, &UpdateVendorInput{Name: &name})
		if !errors.Is(err, ErrVendorNotFound) {
			t.Errorf("expected ErrVendorNotFound, got %v", err)
		}
	})
}

func TestVendorService_Delete(t *testing.T) {
	repo := newMockVendorRepo()
	svc := NewVendorService(repo)
	ctx := context.Background()

	t.Run("success when no accounts", func(t *testing.T) {
		v, _ := svc.Create(ctx, &CreateVendorInput{Name: "to-delete", APIFormat: VendorAPIFormatOpenAI, BaseURL: "https://x.com", AuthType: VendorAuthTypeAPIKey})

		if err := svc.Delete(ctx, v.ID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err := svc.GetByID(ctx, v.ID)
		if !errors.Is(err, ErrVendorNotFound) {
			t.Errorf("expected not found after delete, got %v", err)
		}
	})

	t.Run("fails when vendor has accounts", func(t *testing.T) {
		v, _ := svc.Create(ctx, &CreateVendorInput{Name: "with-accounts", APIFormat: VendorAPIFormatOpenAI, BaseURL: "https://y.com", AuthType: VendorAuthTypeAPIKey})

		// Simulate vendor having 2 asciated accounts
		repo.accountCountByVendor[v.ID] = 2

		err := svc.Delete(ctx, v.ID)
		if !errors.Is(err, ErrVendorHasAccounts) {
			t.Errorf("expected ErrVendorHasAccounts, got %v", err)
		}

		// Vendor should still exist
		_, err = svc.GetByID(ctx, v.ID)
		if err != nil {
			t.Errorf("vendor should still exist after failed delete, got error: %v", err)
		}
	})
}

func TestVendorService_ListActive(t *testing.T) {
	repo := newMockVendorRepo()
	svc := NewVendorService(repo)
	ctx := context.Background()

	svc.Create(ctx, &CreateVendorInput{Name: "active1", APIFormat: VendorAPIFormatOpenAI, BaseURL: "https://a.com", AuthType: VendorAuthTypeAPIKey})
	svc.Create(ctx, &CreateVendorInput{Name: "active2", APIFormat: VendorAPIFormatOpenAI, BaseURL: "https://b.com", AuthType: VendorAuthTypeAPIKey})

	// suspend one
	repo.vendors[2].Status = VendorStatusSuspended

	active, err := svc.ListActive(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 active vendor, got %d", len(active))
	}
}
