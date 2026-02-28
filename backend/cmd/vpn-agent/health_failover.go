package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// regionProximity defines nearby regions for cross-region failover.
var regionProximity = map[string][]string{
	"us": {"ca", "uk", "de"},
	"uk": {"de", "fr", "nl", "us"},
	"de": {"nl", "fr", "uk"},
	"fr": {"de", "nl", "uk"},
	"nl": {"de", "fr", "uk"},
	"ca": {"us", "uk"},
	"au": {"sg", "jp"},
	"jp": {"sg", "kr", "hk", "tw"},
	"sg": {"hk", "jp", "au", "tw"},
	"hk": {"sg", "tw", "jp", "kr"},
	"kr": {"jp", "hk", "sg"},
	"tw": {"hk", "sg", "jp"},
	"in": {"sg", "hk"},
}

// isNearbyRegion checks if candidate is a nearby region to target.
func isNearbyRegion(target, candidate string) bool {
	nearby, ok := regionProximity[target]
	if !ok {
		return false
	}
	for _, r := range nearby {
		if r == candidate {
			return true
		}
	}
	return false
}

// handleUnhealthy attempts to recover an unhealthy tunnel.
// Flow: reconnect x2 → same-region alternatives → cross-region alternatives.
func (hc *HealthChecker) handleUnhealthy(t *TunnelInfo) {
	log.Printf("[failover] Tunnel %q unhealthy (%d failures), starting failover", t.Name, t.Failures)
	originalConfig := t.ConfigName

	// Emit failover start event
	go hc.callback.ReportEvent(t.Name, "failover", map[string]interface{}{
		"reason":      "unhealthy",
		"failures":    t.Failures,
		"config_name": t.ConfigName,
	})

	// Phase 1: Try restart up to maxReconnectAttempts times
	for i := 0; i < maxReconnectAttempts; i++ {
		log.Printf("[failover] %s: restart attempt %d/%d", t.Name, i+1, maxReconnectAttempts)
		if err := hc.tunnelMgr.Restart(t.Name); err != nil {
			log.Printf("[failover] %s: restart failed: %v", t.Name, err)
			continue
		}
		if hc.testConnectivity(t.SocksPort) {
			log.Printf("[failover] %s: restart succeeded", t.Name)
			hc.tunnelMgr.mu.Lock()
			if rt, ok := hc.tunnelMgr.tunnels[t.Name]; ok {
				rt.Failures = 0
				rt.Health = "healthy"
			}
			hc.tunnelMgr.mu.Unlock()
			hc.clearCooldown(t.ConfigName)
			hc.scores.RecordSuccess(t.ConfigName)
			hc.store.RecordSuccess(t.ConfigName)
			return
		}
	}

	// Phase 2: Try same-region alternatives using weighted scoring
	tried := []string{originalConfig}
	log.Printf("[failover] %s: restart failed, trying same-region alternatives (region=%s)", t.Name, t.Region)
	for i := 0; i < maxSwitchAttempts; i++ {
		alt := hc.findBestAlternative(originalConfig, t.Region, tried)
		if alt == "" {
			log.Printf("[failover] %s: no more same-region alternatives", t.Name)
			break
		}
		log.Printf("[failover] %s: switching to %s (same-region, attempt %d)", t.Name, alt, i+1)
		tried = append(tried, alt)

		if err := hc.tunnelMgr.SwitchConfig(t.Name, alt); err != nil {
			log.Printf("[failover] %s: switch to %s failed: %v", t.Name, alt, err)
			hc.addCooldown(alt)
			hc.scores.RecordFailure(alt)
			continue
		}
		if hc.testConnectivity(t.SocksPort) {
			log.Printf("[failover] %s: switched to %s successfully", t.Name, alt)
			hc.updateTunnelAfterSwitch(t, alt)
			hc.addCooldown(originalConfig)
			go hc.callback.ReportEvent(t.Name, "config_switch", map[string]interface{}{
				"from_config": originalConfig,
				"to_config":   alt,
				"phase":       "same_region",
			})
			return
		}
		hc.addCooldown(alt)
		hc.scores.RecordFailure(alt)
	}

	// Phase 3: Try cross-region alternatives (any region)
	log.Printf("[failover] %s: same-region exhausted, trying cross-region", t.Name)
	for i := 0; i < maxSwitchAttempts; i++ {
		alt := hc.findBestAlternative(originalConfig, "", tried) // empty region = any
		if alt == "" {
			log.Printf("[failover] %s: no more alternatives available", t.Name)
			break
		}
		log.Printf("[failover] %s: switching to %s (cross-region, attempt %d)", t.Name, alt, i+1)
		tried = append(tried, alt)

		if err := hc.tunnelMgr.SwitchConfig(t.Name, alt); err != nil {
			log.Printf("[failover] %s: switch to %s failed: %v", t.Name, alt, err)
			hc.addCooldown(alt)
			hc.scores.RecordFailure(alt)
			continue
		}
		if hc.testConnectivity(t.SocksPort) {
			log.Printf("[failover] %s: switched to %s successfully (cross-region)", t.Name, alt)
			hc.updateTunnelAfterSwitch(t, alt)
			hc.addCooldown(originalConfig)
			go hc.callback.ReportEvent(t.Name, "config_switch", map[string]interface{}{
				"from_config": originalConfig,
				"to_config":   alt,
				"phase":       "cross_region",
			})
			return
		}
		hc.addCooldown(alt)
		hc.scores.RecordFailure(alt)
	}

	log.Printf("[failover] %s: all failover attempts exhausted", t.Name)
	hc.addCooldown(originalConfig)
	go hc.callback.ReportEvent(t.Name, "failover_exhausted", map[string]interface{}{
		"config_name": originalConfig,
		"tried":       tried,
	})
	hc.tunnelMgr.mu.Lock()
	if rt, ok := hc.tunnelMgr.tunnels[t.Name]; ok {
		rt.Health = "unhealthy"
	}
	hc.tunnelMgr.mu.Unlock()
}

// updateTunnelAfterSwitch updates tunnel state and notifies Sub2API after a successful config switch.
func (hc *HealthChecker) updateTunnelAfterSwitch(t *TunnelInfo, newConfig string) {
	hc.tunnelMgr.mu.Lock()
	if rt, ok := hc.tunnelMgr.tunnels[t.Name]; ok {
		rt.ConfigName = newConfig
		rt.Region = parseRegion(newConfig)
		rt.Failures = 0
		rt.Health = "healthy"
	}
	hc.tunnelMgr.mu.Unlock()

	hc.clearCooldown(newConfig)
	hc.scores.RecordSuccess(newConfig)
	hc.store.RecordSuccess(newConfig)

	// Notify Sub2API of config change
	if t.ProxyID > 0 {
		go hc.callback.UpdateProxyStatus(t.ProxyID, "", "connected", "healthy", 0)
	}
}

// configFreshness returns a 0.0-1.0 score based on how recently the config was uploaded.
func (hc *HealthChecker) configFreshness(configName string) float64 {
	meta := hc.store.GetMeta(configName)
	if meta == nil {
		return 0.5 // unknown = neutral
	}
	age := time.Since(meta.UploadedAt)
	switch {
	case age < 24*time.Hour:
		return 1.0
	case age < 7*24*time.Hour:
		return 0.8
	case age < 30*24*time.Hour:
		return 0.5
	default:
		return 0.2
	}
}

// weightedScore calculates failover priority for a config.
// score = successRate*0.4 + latencyFactor*0.3 + freshness*0.2 + regionBonus*0.1
func (hc *HealthChecker) weightedScore(configName, targetRegion string) float64 {
	successRate := hc.scores.GetScore(configName)
	latencyFactor := successRate // simplified: good success ≈ good latency
	freshness := hc.configFreshness(configName)

	configRegion := parseRegion(configName)
	regionBonus := 0.0
	if configRegion == targetRegion {
		regionBonus = 1.0
	} else if isNearbyRegion(targetRegion, configRegion) {
		regionBonus = 0.5
	}

	return successRate*0.4 + latencyFactor*0.3 + freshness*0.2 + regionBonus*0.1
}

// findBestAlternative finds the best failover config using weighted scoring.
// It considers success rate, latency, config freshness, and region proximity.
func (hc *HealthChecker) findBestAlternative(currentConfig, region string, exclude []string) string {
	configs, err := hc.store.List()
	if err != nil {
		return ""
	}

	type candidate struct {
		name  string
		score float64
	}
	var candidates []candidate

	excludeSet := make(map[string]bool)
	for _, e := range exclude {
		excludeSet[e] = true
	}
	excludeSet[currentConfig] = true

	for _, cfg := range configs {
		if excludeSet[cfg.Name] {
			continue
		}
		// Skip configs currently in cooldown
		if hc.isInCooldown(cfg.Name) {
			continue
		}
		score := hc.weightedScore(cfg.Name, region)

		// Deprioritize stale configs
		meta := hc.store.GetMeta(cfg.Name)
		if meta != nil && meta.Stale {
			score *= 0.3
		}

		candidates = append(candidates, candidate{cfg.Name, score})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].name
}

// addCooldown records a failover event for a config, increasing its cooldown period.
func (hc *HealthChecker) addCooldown(configName string) {
	hc.cooldownMu.Lock()
	defer hc.cooldownMu.Unlock()
	entry, exists := hc.cooldowns[configName]
	if exists {
		entry.failedAt = time.Now()
		entry.consecutiveFails++
	} else {
		hc.cooldowns[configName] = &cooldownEntry{
			failedAt:         time.Now(),
			consecutiveFails: 1,
		}
	}
}

// isInCooldown checks if a config is still in its cooldown period after a failover.
// Cooldown duration = min(30min * consecutiveFails, 6h).
func (hc *HealthChecker) isInCooldown(configName string) bool {
	hc.cooldownMu.Lock()
	defer hc.cooldownMu.Unlock()
	entry, exists := hc.cooldowns[configName]
	if !exists {
		return false
	}
	cooldownDuration := time.Duration(entry.consecutiveFails) * 30 * time.Minute
	maxCooldown := 6 * time.Hour
	if cooldownDuration > maxCooldown {
		cooldownDuration = maxCooldown
	}
	if time.Since(entry.failedAt) > cooldownDuration {
		delete(hc.cooldowns, configName)
		return false
	}
	return true
}

// clearCooldown removes the cooldown entry for a config (e.g., after successful recovery).
func (hc *HealthChecker) clearCooldown(configName string) {
	hc.cooldownMu.Lock()
	defer hc.cooldownMu.Unlock()
	delete(hc.cooldowns, configName)
}

// StabilityScores tracks historical performance of .ovpn configs.
type StabilityScores struct {
	Configs map[string]*ConfigScore `json:"configs"`
	mu      sync.Mutex
}

// ConfigScore tracks success/failure for a single config.
type ConfigScore struct {
	TotalUses int     `json:"total_uses"`
	Successes int     `json:"successes"`
	Score     float64 `json:"score"`
}

func loadScores(stateDir string) *StabilityScores {
	s := &StabilityScores{Configs: make(map[string]*ConfigScore)}
	path := filepath.Join(stateDir, "scores.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, s); err != nil {
		log.Printf("Warning: failed to parse scores.json: %v", err)
	}
	if s.Configs == nil {
		s.Configs = make(map[string]*ConfigScore)
	}
	return s
}

// Save persists scores to disk.
func (s *StabilityScores) Save(stateDir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	path := filepath.Join(stateDir, "scores.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("Warning: failed to save scores.json: %v", err)
	}
}

// RecordSuccess records a successful health check for a config.
func (s *StabilityScores) RecordSuccess(configName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cs := s.getOrCreate(configName)
	cs.TotalUses++
	cs.Successes++
	cs.Score = float64(cs.Successes) / float64(cs.TotalUses)
}

// RecordFailure records a failed health check for a config.
func (s *StabilityScores) RecordFailure(configName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cs := s.getOrCreate(configName)
	cs.TotalUses++
	cs.Score = float64(cs.Successes) / float64(cs.TotalUses)
}

// GetScore returns the stability score for a config (0.0 - 1.0).
func (s *StabilityScores) GetScore(configName string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cs, ok := s.Configs[configName]; ok {
		return cs.Score
	}
	return 0.5 // default score for unknown configs
}

func (s *StabilityScores) getOrCreate(name string) *ConfigScore {
	if cs, ok := s.Configs[name]; ok {
		return cs
	}
	cs := &ConfigScore{}
	s.Configs[name] = cs
	return cs
}
