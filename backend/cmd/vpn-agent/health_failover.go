package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// handleUnhealthy attempts to recover an unhealthy tunnel.
// Flow: reconnect x2 → switch to same-region backup node.
func (hc *HealthChecker) handleUnhealthy(t *TunnelInfo) {
	log.Printf("Auto-failover triggered for %q (failures: %d)", t.Name, t.Failures)

	// Step 1 & 2: Try reconnecting (restart) up to maxReconnectAttempts times
	for i := 0; i < maxReconnectAttempts; i++ {
		log.Printf("Reconnect attempt %d/%d for %q", i+1, maxReconnectAttempts, t.Name)
		if err := hc.tunnelMgr.Restart(t.Name); err != nil {
			log.Printf("Reconnect failed for %q: %v", t.Name, err)
			continue
		}
		// Check if reconnect succeeded
		if hc.testConnectivity(t.SocksPort) {
			log.Printf("Reconnect succeeded for %q", t.Name)
			hc.tunnelMgr.mu.Lock()
			if rt, ok := hc.tunnelMgr.tunnels[t.Name]; ok {
				rt.Failures = 0
				rt.Health = "healthy"
			}
			hc.tunnelMgr.mu.Unlock()
			return
		}
	}

	// Step 3: Find same-region backup and switch
	log.Printf("Reconnect failed, trying backup node for %q (region: %s)", t.Name, t.Region)
	configs, err := hc.store.List()
	if err != nil {
		log.Printf("Failed to list configs for failover: %v", err)
		return
	}

	// Filter same-region configs, exclude current
	var candidates []OvpnConfig
	for _, c := range configs {
		if c.Region == t.Region && c.Name != t.ConfigName {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		log.Printf("No backup configs available for region %q, tunnel %q is dead", t.Region, t.Name)
		return
	}

	// Sort by stability score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		si := hc.scores.GetScore(candidates[i].Name)
		sj := hc.scores.GetScore(candidates[j].Name)
		return si > sj
	})

	// Try up to maxSwitchAttempts backup nodes
	for i := 0; i < maxSwitchAttempts && i < len(candidates); i++ {
		newCfg := candidates[i].Name
		log.Printf("Switching %q to backup %q (attempt %d)", t.Name, newCfg, i+1)
		if err := hc.tunnelMgr.SwitchConfig(t.Name, newCfg); err != nil {
			log.Printf("Switch failed: %v", err)
			continue
		}
		if hc.testConnectivity(t.SocksPort) {
			log.Printf("Failover succeeded: %q now using %q", t.Name, newCfg)
			hc.tunnelMgr.mu.Lock()
			if rt, ok := hc.tunnelMgr.tunnels[t.Name]; ok {
				rt.Failures = 0
				rt.Health = "healthy"
			}
			hc.tunnelMgr.mu.Unlock()
			// Notify Sub2API of config change
			if t.ProxyID > 0 {
				go hc.callback.UpdateProxyStatus(t.ProxyID, "", "connected", "healthy", 0)
			}
			return
		}
	}

	log.Printf("All failover attempts exhausted for %q — tunnel is dead", t.Name)
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
