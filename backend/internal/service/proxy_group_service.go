package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ProxyGroupService manages proxy group selection for accounts.
// It selects healthy proxy nodes from a named group for shared-proxy scenarios.
type ProxyGroupService struct {
	proxyRepo ProxyRepository

	mu         sync.RWMutex
	groupCache map[string][]Proxy
	cacheTime  map[string]time.Time
	cacheTTL   time.Duration
}

// NewProxyGroupService creates a new ProxyGroupService.
func NewProxyGroupService(proxyRepo ProxyRepository) *ProxyGroupService {
	return &ProxyGroupService{
		proxyRepo:  proxyRepo,
		groupCache: make(map[string][]Proxy),
		cacheTime:  make(map[string]time.Time),
		cacheTTL:   30 * time.Second,
	}
}

// SelectProxyForAccount picks a proxy for the given account.
// Priority: ProxyID (dedicated) > ProxyGroup (shared) > none.
// Returns the proxy URL, or empty string if no proxy is needed.
func (s *ProxyGroupService) SelectProxyForAccount(ctx context.Context, account *Account) string {
	if account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}

	if account.ProxyGroup == "" {
		return ""
	}

	proxy := s.selectFromGroup(ctx, account.ProxyGroup, account.ID)
	if proxy == nil {
		return ""
	}
	return proxy.URL()
}

// selectFromGroup picks a healthy proxy from the named group.
// Algorithm: accountID % len(healthyProxies) for sticky selection;
// automatically skips to the next node on failure.
func (s *ProxyGroupService) selectFromGroup(ctx context.Context, groupName string, accountID int64) *Proxy {
	proxies := s.getGroupProxies(ctx, groupName)
	if len(proxies) == 0 {
		slog.Warn("no proxies in group", "group", groupName)
		return nil
	}

	healthy := make([]Proxy, 0, len(proxies))
	for i := range proxies {
		if proxies[i].IsHealthy() {
			healthy = append(healthy, proxies[i])
		}
	}

	if len(healthy) == 0 {
		slog.Warn("no healthy proxies in group, falling back to all active",
			"group", groupName, "total", len(proxies))
		healthy = proxies
	}

	idx := int(accountID % int64(len(healthy)))
	selected := &healthy[idx]

	slog.Debug("proxy group selection",
		"group", groupName,
		"account_id", accountID,
		"selected_proxy", selected.Name,
		"proxy_id", selected.ID,
		"healthy_count", len(healthy),
		"total_count", len(proxies),
	)

	return selected
}

// getGroupProxies returns proxies for a group, with a short TTL cache.
func (s *ProxyGroupService) getGroupProxies(ctx context.Context, groupName string) []Proxy {
	s.mu.RLock()
	cached, ok := s.groupCache[groupName]
	cacheTime := s.cacheTime[groupName]
	s.mu.RUnlock()

	if ok && time.Since(cacheTime) < s.cacheTTL {
		return cached
	}

	proxies, err := s.proxyRepo.ListByGroupName(ctx, groupName)
	if err != nil {
		slog.Error("failed to list proxies by group", "group", groupName, "error", err)
		if ok {
			return cached
		}
		return nil
	}

	s.mu.Lock()
	s.groupCache[groupName] = proxies
	s.cacheTime[groupName] = time.Now()
	s.mu.Unlock()

	return proxies
}

// InvalidateGroupCache clears the cache for a specific group.
func (s *ProxyGroupService) InvalidateGroupCache(groupName string) {
	s.mu.Lock()
	delete(s.groupCache, groupName)
	delete(s.cacheTime, groupName)
	s.mu.Unlock()
}

// InvalidateAllCache clears all group caches.
func (s *ProxyGroupService) InvalidateAllCache() {
	s.mu.Lock()
	s.groupCache = make(map[string][]Proxy)
	s.cacheTime = make(map[string]time.Time)
	s.mu.Unlock()
}

// ListGroupNames returns all distinct proxy group names.
func (s *ProxyGroupService) ListGroupNames(ctx context.Context) ([]string, error) {
	names, err := s.proxyRepo.ListGroupNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("list group names: %w", err)
	}
	return names, nil
}
