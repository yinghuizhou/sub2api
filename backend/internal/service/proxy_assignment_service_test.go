//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// proxyAssignRepoMock implements ProxyRepository for assignment tests.
type proxyAssignRepoMock struct {
	activeProxies []ProxyWithAccountCount
	groupCounts   map[string]int64
}

func (m *proxyAssignRepoMock) ListActiveWithAccountCount(_ context.Context) ([]ProxyWithAccountCount, error) {
	return m.activeProxies, nil
}
func (m *proxyAssignRepoMock) CountAccountsByGroupName(_ context.Context) (map[string]int64, error) {
	return m.groupCounts, nil
}

// Unused interface methods — stubs to satisfy ProxyRepository.
func (m *proxyAssignRepoMock) Create(_ context.Context, _ *Proxy) error { return nil }
func (m *proxyAssignRepoMock) GetByID(_ context.Context, _ int64) (*Proxy, error) {
	return nil, nil
}
func (m *proxyAssignRepoMock) ListByIDs(_ context.Context, _ []int64) ([]Proxy, error) {
	return nil, nil
}
func (m *proxyAssignRepoMock) Update(_ context.Context, _ *Proxy) error { return nil }
func (m *proxyAssignRepoMock) Delete(_ context.Context, _ int64) error  { return nil }
func (m *proxyAssignRepoMock) List(_ context.Context, _ pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *proxyAssignRepoMock) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _ string) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *proxyAssignRepoMock) ListWithFiltersAndAccountCount(_ context.Context, _ pagination.PaginationParams, _, _, _ string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *proxyAssignRepoMock) ListActive(_ context.Context) ([]Proxy, error) { return nil, nil }
func (m *proxyAssignRepoMock) ListByGroupName(_ context.Context, _ string) ([]Proxy, error) {
	return nil, nil
}
func (m *proxyAssignRepoMock) ListGroupNames(_ context.Context) ([]string, error) { return nil, nil }
func (m *proxyAssignRepoMock) ExistsByHostPortAuth(_ context.Context, _ string, _ int, _, _ string) (bool, error) {
	return false, nil
}
func (m *proxyAssignRepoMock) CountAccountsByProxyID(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}
func (m *proxyAssignRepoMock) ListAccountSummariesByProxyID(_ context.Context, _ int64) ([]ProxyAccountSummary, error) {
	return nil, nil
}
func (m *proxyAssignRepoMock) GetAssignment(_ context.Context, _ int64) (*ProxyAssignment, error) {
	return nil, nil
}
func (m *proxyAssignRepoMock) SetAssignment(_ context.Context, _, _ int64, _, _ string) error {
	return nil
}
func (m *proxyAssignRepoMock) DeleteAssignment(_ context.Context, _ int64) error { return nil }
func (m *proxyAssignRepoMock) ListAssignmentsByGroup(_ context.Context, _ string) ([]ProxyAssignment, error) {
	return nil, nil
}
func (m *proxyAssignRepoMock) CountAssignmentsByProxy(_ context.Context) (map[int64]int64, error) {
	return nil, nil
}
func (m *proxyAssignRepoMock) BulkSetAssignments(_ context.Context, _ []ProxyAssignment) (int, error) {
	return 0, nil
}
func (m *proxyAssignRepoMock) ListAccountsByProxyGroup(_ context.Context, _ string) ([]ProxyAccountSummary, error) {
	return nil, nil
}

func TestProxyAssignment_PreferredRegionLowestLoad(t *testing.T) {
	mock := &proxyAssignRepoMock{
		activeProxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 1, GroupName: "us-east-1", Region: "us-east"}, AccountCount: 2},
			{Proxy: Proxy{ID: 2, GroupName: "us-east-1", Region: "us-east"}, AccountCount: 2},
			{Proxy: Proxy{ID: 3, GroupName: "us-west-1", Region: "us-west"}, AccountCount: 0},
		},
		groupCounts: map[string]int64{
			"us-east-1": 4,
			"us-west-1": 1,
		},
	}

	svc := NewProxyAssignmentService(mock, NewPlatformProxyRules())
	group, err := svc.AutoAssign(context.Background(), "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// us-west-1 has lower load (1 account / 1 proxy = 1.0) vs us-east-1 (4 / 2 = 2.0)
	if group != "us-west-1" {
		t.Errorf("expected us-west-1 (lowest load), got %q", group)
	}
}

func TestProxyAssignment_SkipsFullGroups(t *testing.T) {
	// anthropic rule: MaxAccountsPerIP = 5
	mock := &proxyAssignRepoMock{
		activeProxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 1, GroupName: "full-group", Region: "us-east"}, AccountCount: 5},
			{Proxy: Proxy{ID: 2, GroupName: "available-group", Region: "us-east"}, AccountCount: 2},
		},
		groupCounts: map[string]int64{
			"full-group":      5, // 5/1 = 5 >= MaxAccountsPerIP(5), should skip
			"available-group": 2, // 2/1 = 2 < 5, ok
		},
	}

	svc := NewProxyAssignmentService(mock, NewPlatformProxyRules())
	group, err := svc.AutoAssign(context.Background(), "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group != "available-group" {
		t.Errorf("expected available-group, got %q", group)
	}
}

func TestProxyAssignment_UnknownPlatform(t *testing.T) {
	mock := &proxyAssignRepoMock{}
	svc := NewProxyAssignmentService(mock, NewPlatformProxyRules())
	group, err := svc.AutoAssign(context.Background(), "unknown-platform")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group != "" {
		t.Errorf("expected empty string for unknown platform, got %q", group)
	}
}

func TestProxyAssignment_FallbackToNonPreferred(t *testing.T) {
	// anthropic preferred: us-east, us-central, us-west
	// Only eu-west available (non-preferred), preferred group is full
	mock := &proxyAssignRepoMock{
		activeProxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 1, GroupName: "us-east-full", Region: "us-east"}, AccountCount: 5},
			{Proxy: Proxy{ID: 2, GroupName: "eu-west-1", Region: "eu-west"}, AccountCount: 1},
		},
		groupCounts: map[string]int64{
			"us-east-full": 5, // 5/1 >= 5, full
			"eu-west-1":    1, // 1/1 < 5, available but non-preferred
		},
	}

	svc := NewProxyAssignmentService(mock, NewPlatformProxyRules())
	group, err := svc.AutoAssign(context.Background(), "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group != "eu-west-1" {
		t.Errorf("expected eu-west-1 as fallback, got %q", group)
	}
}
