package service

import (
	"context"
	"fmt"
	"log"
)

// ProxyAssignmentService auto-assigns proxy groups to accounts based on platform rules.
type ProxyAssignmentService struct {
	proxyRepo ProxyRepository
	rules     *PlatformProxyRules
}

func NewProxyAssignmentService(proxyRepo ProxyRepository, rules *PlatformProxyRules) *ProxyAssignmentService {
	return &ProxyAssignmentService{proxyRepo: proxyRepo, rules: rules}
}

// AutoAssign finds the best proxy group for an account based on platform rules.
// Returns empty string if no suitable group found or no rule exists for the platform.
func (s *ProxyAssignmentService) AutoAssign(ctx context.Context, platform string) (string, error) {
	rule, ok := s.rules.GetRule(platform)
	if !ok {
		return "", nil
	}

	proxies, err := s.proxyRepo.ListActiveWithAccountCount(ctx)
	if err != nil {
		return "", fmt.Errorf("list proxies: %w", err)
	}

	groupCounts, err := s.proxyRepo.CountAccountsByGroupName(ctx)
	if err != nil {
		return "", fmt.Errorf("count accounts by group: %w", err)
	}

	type groupInfo struct {
		proxyCount   int
		accountCount int64
		isPreferred  bool
	}
	groups := make(map[string]*groupInfo)

	for _, p := range proxies {
		if p.GroupName == "" {
			continue
		}
		g, exists := groups[p.GroupName]
		if !exists {
			g = &groupInfo{}
			groups[p.GroupName] = g
		}
		g.proxyCount++
		g.accountCount = groupCounts[p.GroupName]

		for _, pr := range rule.PreferredRegions {
			if p.Proxy.Region == pr {
				g.isPreferred = true
				break
			}
		}
	}

	var bestGroup string
	var bestLoad float64 = 999999

	for name, g := range groups {
		if g.proxyCount == 0 {
			continue
		}
		accountsPerProxy := float64(g.accountCount) / float64(g.proxyCount)
		if int(accountsPerProxy) >= rule.MaxAccountsPerIP {
			continue
		}

		load := accountsPerProxy
		if !g.isPreferred {
			load += 1000
		}

		if load < bestLoad {
			bestLoad = load
			bestGroup = name
		}
	}

	if bestGroup != "" {
		log.Printf("[ProxyAssignment] Auto-assigned platform=%s to group=%s (load=%.1f)", platform, bestGroup, bestLoad)
	}
	return bestGroup, nil
}
