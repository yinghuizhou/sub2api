package service

// PlatformProxyRule defines proxy routing rules for a specific AI platform.
type PlatformProxyRule struct {
	Platform         string
	PreferredRegions []string
	MaxAccountsPerIP int
	StickyIP         bool
}

// PlatformProxyRules holds per-platform proxy routing configuration.
type PlatformProxyRules struct {
	rules map[string]PlatformProxyRule
}

// NewPlatformProxyRules returns rules pre-populated for known platforms.
func NewPlatformProxyRules() *PlatformProxyRules {
	return &PlatformProxyRules{
		rules: map[string]PlatformProxyRule{
			"anthropic": {
				Platform:         "anthropic",
				PreferredRegions: []string{"us-east", "us-central", "us-west"},
				MaxAccountsPerIP: 5,
				StickyIP:         true,
			},
			"gemini": {
				Platform:         "gemini",
				PreferredRegions: []string{"us-east", "us-west", "eu-west"},
				MaxAccountsPerIP: 10,
				StickyIP:         true,
			},
			"openai": {
				Platform:         "openai",
				PreferredRegions: []string{"us-east", "us-west"},
				MaxAccountsPerIP: 8,
				StickyIP:         true,
			},
		},
	}
}

// GetRule returns the proxy rule for the given platform.
func (r *PlatformProxyRules) GetRule(platform string) (PlatformProxyRule, bool) {
	rule, ok := r.rules[platform]
	return rule, ok
}

// AllRules returns all registered platform rules.
func (r *PlatformProxyRules) AllRules() map[string]PlatformProxyRule {
	return r.rules
}
