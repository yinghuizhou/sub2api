package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

const maxAssignCacheSize = 10000 // evict oldest entries when exceeded

// ProxyGroupService manages proxy group selection for accounts.
// It uses persistent assignments (account_proxy_assignments table) so that
// once an account is bound to a proxy/region, it never switches.
type ProxyGroupService struct {
	proxyRepo ProxyRepository

	mu         sync.RWMutex
	groupCache map[string][]Proxy
	cacheTime  map[string]time.Time
	cacheTTL   time.Duration

	// assignmentCache: accountID -> ProxyAssignment (short-lived, bounded)
	assignMu    sync.RWMutex
	assignCache map[int64]*ProxyAssignment
	assignTime  map[int64]time.Time
}

// NewProxyGroupService creates a new ProxyGroupService.
func NewProxyGroupService(proxyRepo ProxyRepository) *ProxyGroupService {
	return &ProxyGroupService{
		proxyRepo:   proxyRepo,
		groupCache:  make(map[string][]Proxy),
		cacheTime:   make(map[string]time.Time),
		cacheTTL:    30 * time.Second,
		assignCache: make(map[int64]*ProxyAssignment),
		assignTime:  make(map[int64]time.Time),
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

// selectFromGroup picks a proxy for an account using persistent assignment.
//
// Algorithm:
//  1. Check assignment cache / DB for existing binding
//  2. If assigned and proxy healthy → use it
//  3. If assigned but proxy unhealthy → still use it (never switch region)
//  4. If not assigned → pick the least-loaded healthy proxy, persist binding
func (s *ProxyGroupService) selectFromGroup(ctx context.Context, groupName string, accountID int64) *Proxy {
	proxies := s.getGroupProxies(ctx, groupName)
	if len(proxies) == 0 {
		slog.Warn("no proxies in group", "group", groupName)
		return nil
	}

	// Build a quick lookup map
	proxyByID := make(map[int64]*Proxy, len(proxies))
	for i := range proxies {
		proxyByID[proxies[i].ID] = &proxies[i]
	}

	// 1. Check persistent assignment (with in-memory cache)
	assignment := s.getCachedAssignment(ctx, accountID)
	if assignment != nil {
		if p, ok := proxyByID[assignment.ProxyID]; ok {
			if !p.IsHealthy() {
				slog.Warn("assigned proxy unhealthy, using anyway to preserve region",
					"account_id", accountID, "proxy", p.Name, "proxy_id", p.ID,
					"health_status", p.HealthStatus)
			}
			return p
		}
		// Assigned proxy no longer in group (deleted?) — log and reassign
		slog.Warn("assigned proxy not found in group, will reassign",
			"account_id", accountID, "proxy_id", assignment.ProxyID, "group", groupName)
	}

	// 2. No assignment → pick least-loaded healthy proxy
	healthy := make([]Proxy, 0, len(proxies))
	for i := range proxies {
		if proxies[i].IsHealthy() {
			healthy = append(healthy, proxies[i])
		}
	}
	if len(healthy) == 0 {
		slog.Warn("no healthy proxies, assigning to least-loaded overall",
			"group", groupName, "total", len(proxies))
		healthy = proxies
	}

	// Get current assignment counts per proxy (scoped to this group)
	counts, err := s.proxyRepo.CountAssignmentsByProxyInGroup(ctx, groupName)
	if err != nil {
		slog.Error("failed to count assignments", "error", err, "group", groupName)
		counts = make(map[int64]int64)
	}

	// Pick the proxy with fewest assignments
	var best *Proxy
	var bestCount int64 = -1
	for i := range healthy {
		c := counts[healthy[i].ID]
		if bestCount < 0 || c < bestCount {
			bestCount = c
			best = &healthy[i]
		}
	}

	if best == nil {
		return nil
	}

	// 3. Persist the assignment (uses ON CONFLICT DO NOTHING for "auto")
	if err := s.proxyRepo.SetAssignment(ctx, accountID, best.ID, groupName, "auto"); err != nil {
		slog.Error("failed to persist proxy assignment",
			"account_id", accountID, "proxy_id", best.ID, "error", err)
	}

	// Re-read from DB to handle race: another goroutine may have won the insert
	actual, err := s.proxyRepo.GetAssignment(ctx, accountID)
	if err != nil {
		slog.Error("failed to re-read assignment after insert", "account_id", accountID, "error", err)
		return best // fallback to our pick
	}
	if actual != nil {
		s.assignMu.Lock()
		s.assignCache[accountID] = actual
		s.assignTime[accountID] = time.Now()
		s.assignMu.Unlock()

		// If another goroutine won with a different proxy, use that one
		if actual.ProxyID != best.ID {
			if p, ok := proxyByID[actual.ProxyID]; ok {
				slog.Info("using existing assignment from concurrent insert",
					"account_id", accountID, "proxy_id", actual.ProxyID)
				return p
			}
		}
	}

	slog.Info("new proxy assignment created",
		"group", groupName,
		"account_id", accountID,
		"proxy", best.Name,
		"proxy_id", best.ID,
	)

	return best
}

// getCachedAssignment returns the assignment from cache or DB.
func (s *ProxyGroupService) getCachedAssignment(ctx context.Context, accountID int64) *ProxyAssignment {
	s.assignMu.RLock()
	cached, ok := s.assignCache[accountID]
	cacheTime := s.assignTime[accountID]
	s.assignMu.RUnlock()

	if ok && time.Since(cacheTime) < s.cacheTTL {
		return cached
	}

	assignment, err := s.proxyRepo.GetAssignment(ctx, accountID)
	if err != nil {
		slog.Error("failed to get proxy assignment", "account_id", accountID, "error", err)
		if ok {
			return cached
		}
		return nil
	}

	// Only cache non-nil results — caching nil fills the cache with unknowns
	// that can never be invalidated by SetAssignment (which only fires on real assignments).
	if assignment != nil {
		s.assignMu.Lock()
		if len(s.assignCache) >= maxAssignCacheSize {
			s.evictExpiredAssignments()
		}
		s.assignCache[accountID] = assignment
		s.assignTime[accountID] = time.Now()
		s.assignMu.Unlock()
	}

	return assignment
}

// evictExpiredAssignments removes expired entries from the assignment cache.
// Must be called with assignMu held (write lock).
func (s *ProxyGroupService) evictExpiredAssignments() {
	now := time.Now()
	for id, t := range s.assignTime {
		if now.Sub(t) > s.cacheTTL {
			delete(s.assignCache, id)
			delete(s.assignTime, id)
		}
	}
	// If still over limit after evicting expired entries, evict oldest 25%
	if len(s.assignCache) >= maxAssignCacheSize {
		type entry struct {
			id int64
			t  time.Time
		}
		entries := make([]entry, 0, len(s.assignTime))
		for id, t := range s.assignTime {
			entries = append(entries, entry{id, t})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].t.Before(entries[j].t) })
		evictCount := len(entries)/4 + 1
		for i := 0; i < evictCount && i < len(entries); i++ {
			delete(s.assignCache, entries[i].id)
			delete(s.assignTime, entries[i].id)
		}
	}
}

// InvalidateAssignmentCache clears the assignment cache for a specific account.
func (s *ProxyGroupService) InvalidateAssignmentCache(accountID int64) {
	s.assignMu.Lock()
	delete(s.assignCache, accountID)
	delete(s.assignTime, accountID)
	s.assignMu.Unlock()
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

// InvalidateAllCache clears all group caches and assignment caches.
func (s *ProxyGroupService) InvalidateAllCache() {
	s.mu.Lock()
	s.groupCache = make(map[string][]Proxy)
	s.cacheTime = make(map[string]time.Time)
	s.mu.Unlock()

	s.assignMu.Lock()
	s.assignCache = make(map[int64]*ProxyAssignment)
	s.assignTime = make(map[int64]time.Time)
	s.assignMu.Unlock()
}

// ListGroupNames returns all distinct proxy group names.
func (s *ProxyGroupService) ListGroupNames(ctx context.Context) ([]string, error) {
	names, err := s.proxyRepo.ListGroupNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("list group names: %w", err)
	}
	return names, nil
}
