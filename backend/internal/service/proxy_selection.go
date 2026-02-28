package service

import "sort"

// selectBestProxy picks the best proxy from a list based on:
// 1. Priority (lower number = higher priority)
// 2. Assignment count (fewer = preferred)
// Assumes input is pre-filtered to healthy proxies (or all if no healthy ones).
func selectBestProxy(proxies []Proxy, counts map[int64]int64) *Proxy {
	if len(proxies) == 0 {
		return nil
	}

	// Sort by priority ASC, then assignment count ASC
	sorted := make([]Proxy, len(proxies))
	copy(sorted, proxies)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return counts[sorted[i].ID] < counts[sorted[j].ID]
	})

	return &sorted[0]
}

// selectBestProxyForMigration picks a replacement proxy when an existing one is removed.
// Prefers same proxy_type, then falls back to any healthy proxy.
func selectBestProxyForMigration(proxies []Proxy, preferType string, counts map[int64]int64) *Proxy {
	// Try same type first
	sameType := make([]Proxy, 0)
	for i := range proxies {
		if proxies[i].IsHealthy() && proxies[i].ProxyType == preferType {
			sameType = append(sameType, proxies[i])
		}
	}
	if len(sameType) > 0 {
		return selectBestProxy(sameType, counts)
	}

	// Fall back to any healthy proxy
	healthy := make([]Proxy, 0)
	for i := range proxies {
		if proxies[i].IsHealthy() {
			healthy = append(healthy, proxies[i])
		}
	}
	if len(healthy) > 0 {
		return selectBestProxy(healthy, counts)
	}

	// Last resort: any proxy
	return selectBestProxy(proxies, counts)
}
