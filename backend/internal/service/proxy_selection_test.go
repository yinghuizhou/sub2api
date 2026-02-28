//go:build unit

package service

import (
	"testing"
)

func TestSelectBestProxy_PreferHigherPriority(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "astrill-1", ProxyType: "astrill", Priority: 90, Status: StatusActive},
		{ID: 2, Name: "wg-1", ProxyType: "wireguard", Priority: 50, Status: StatusActive},
		{ID: 3, Name: "isp-1", ProxyType: "static", Priority: 10, Status: StatusActive},
	}
	counts := map[int64]int64{}

	best := selectBestProxy(proxies, counts)
	if best == nil || best.ID != 3 {
		t.Errorf("expected proxy ID 3 (isp-1, priority=10), got %v", best)
	}
}

func TestSelectBestProxy_TiebreakByCount(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "isp-1", ProxyType: "static", Priority: 10, Status: StatusActive},
		{ID: 2, Name: "isp-2", ProxyType: "static", Priority: 10, Status: StatusActive},
	}
	counts := map[int64]int64{1: 5, 2: 2}

	best := selectBestProxy(proxies, counts)
	if best == nil || best.ID != 2 {
		t.Errorf("expected proxy ID 2 (fewer assignments), got %v", best)
	}
}

func TestSelectBestProxy_EmptyList(t *testing.T) {
	best := selectBestProxy([]Proxy{}, map[int64]int64{})
	if best != nil {
		t.Errorf("expected nil for empty list, got %v", best)
	}
}

func TestSelectBestProxy_SingleProxy(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "only-one", ProxyType: "static", Priority: 50, Status: StatusActive},
	}
	counts := map[int64]int64{1: 10}

	best := selectBestProxy(proxies, counts)
	if best == nil || best.ID != 1 {
		t.Errorf("expected proxy ID 1 (only option), got %v", best)
	}
}

func TestSelectBestProxyForMigration_PreferSameType(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "wg-1", ProxyType: "wireguard", Priority: 50, Status: StatusActive, HealthStatus: "healthy"},
		{ID: 2, Name: "isp-2", ProxyType: "static", Priority: 10, Status: StatusActive, HealthStatus: "healthy"},
	}
	counts := map[int64]int64{}

	// Migrating from a static proxy should prefer another static
	best := selectBestProxyForMigration(proxies, "static", counts)
	if best == nil || best.ID != 2 {
		t.Errorf("expected proxy ID 2 (same type static), got %v", best)
	}
}

func TestSelectBestProxyForMigration_FallbackToOtherType(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "wg-1", ProxyType: "wireguard", Priority: 50, Status: StatusActive, HealthStatus: "healthy"},
	}
	counts := map[int64]int64{}

	// No static proxies available, should fall back to wireguard
	best := selectBestProxyForMigration(proxies, "static", counts)
	if best == nil || best.ID != 1 {
		t.Errorf("expected proxy ID 1 (fallback to wireguard), got %v", best)
	}
}

func TestSelectBestProxyForMigration_SameTypePriorityMatters(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "isp-low", ProxyType: "static", Priority: 90, Status: StatusActive, HealthStatus: "healthy"},
		{ID: 2, Name: "isp-high", ProxyType: "static", Priority: 10, Status: StatusActive, HealthStatus: "healthy"},
		{ID: 3, Name: "wg-1", ProxyType: "wireguard", Priority: 5, Status: StatusActive, HealthStatus: "healthy"},
	}
	counts := map[int64]int64{}

	// Same-type filter should narrow to static, then pick by priority
	best := selectBestProxyForMigration(proxies, "static", counts)
	if best == nil || best.ID != 2 {
		t.Errorf("expected proxy ID 2 (static, priority=10), got %v", best)
	}
}

func TestSelectBestProxyForMigration_SkipsUnhealthy(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "isp-unhealthy", ProxyType: "static", Priority: 10, Status: StatusActive, HealthStatus: "unhealthy"},
		{ID: 2, Name: "wg-healthy", ProxyType: "wireguard", Priority: 50, Status: StatusActive, HealthStatus: "healthy"},
	}
	counts := map[int64]int64{}

	// The static one is unhealthy, should fall back to healthy wireguard
	best := selectBestProxyForMigration(proxies, "static", counts)
	if best == nil || best.ID != 2 {
		t.Errorf("expected proxy ID 2 (healthy wireguard fallback), got %v", best)
	}
}
