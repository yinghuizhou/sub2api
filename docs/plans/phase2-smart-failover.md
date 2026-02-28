# Phase 2: Smart Failover

> Enhance VPN Agent failover with cross-region switching, weighted scoring, cooldown, and staleness detection.

## Task 4: Agent — Weighted Selection Algorithm

**Files:**
- Modify: `backend/cmd/vpn-agent/health_failover.go` — replace simple scoring with weighted algorithm

**Step 1: Add region proximity map and weighted scoring**

Add to `health_failover.go`:

```go
// regionProximity defines nearby regions for cross-region failover
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

// weightedScore calculates failover priority for a config
func (hc *HealthChecker) weightedScore(configName, targetRegion string) float64 {
	// successRate (0.4): from StabilityScores
	successRate := hc.scores.GetScore(configName)

	// latency factor (0.3): inverse of avg latency, normalized
	// Use score as proxy since we don't track per-config latency yet
	latencyFactor := successRate // simplified: good success ≈ good latency

	// freshness (0.2): based on config metadata upload time
	freshness := hc.configFreshness(configName)

	// regionMatch (0.1): same=1.0, nearby=0.5, other=0.0
	configRegion := parseRegion(configName)
	regionBonus := 0.0
	if configRegion == targetRegion {
		regionBonus = 1.0
	} else if isNearbyRegion(targetRegion, configRegion) {
		regionBonus = 0.5
	}

	return successRate*0.4 + latencyFactor*0.3 + freshness*0.2 + regionBonus*0.1
}

func isNearbyRegion(target, candidate string) bool {
	nearby, ok := regionProximity[target]
	if !ok { return false }
	for _, r := range nearby {
		if r == candidate { return true }
	}
	return false
}

func (hc *HealthChecker) configFreshness(configName string) float64 {
	meta := hc.store.GetMeta(configName)
	if meta == nil { return 0.5 } // unknown = neutral
	age := time.Since(meta.UploadedAt)
	if age < 24*time.Hour { return 1.0 }
	if age < 7*24*time.Hour { return 0.8 }
	if age < 30*24*time.Hour { return 0.5 }
	return 0.2
}
```

**Step 2: Replace findAlternativeConfig with weighted version**

Replace the existing candidate sorting in `handleUnhealthy`:

```go
func (hc *HealthChecker) findBestAlternative(currentConfig, region string, exclude []string) string {
	configs, err := hc.store.List()
	if err != nil { return "" }

	type candidate struct {
		name  string
		score float64
	}
	var candidates []candidate

	excludeSet := make(map[string]bool)
	for _, e := range exclude { excludeSet[e] = true }
	excludeSet[currentConfig] = true

	for _, cfg := range configs {
		if excludeSet[cfg.Name] { continue }
		if hc.isInCooldown(cfg.Name) { continue }

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

	if len(candidates) == 0 { return "" }
	return candidates[0].name
}
```

**Step 3: Build and verify**

```bash
cd backend && go build ./cmd/vpn-agent/
```

**Step 4: Commit**

```bash
git add backend/cmd/vpn-agent/health_failover.go
git commit -m "feat(vpn-agent): weighted failover scoring with region proximity"
```

---

## Task 5: Agent — Cooldown Mechanism

**Files:**
- Modify: `backend/cmd/vpn-agent/health_failover.go` — add cooldown tracking

**Step 1: Add cooldown state**

```go
// cooldownEntry tracks when a config was failed-over away from
type cooldownEntry struct {
	failedAt       time.Time
	consecutiveFails int
}

// Add to HealthChecker struct:
// cooldowns map[string]*cooldownEntry
// cooldownMu sync.Mutex

func (hc *HealthChecker) addCooldown(configName string) {
	hc.cooldownMu.Lock()
	defer hc.cooldownMu.Unlock()
	entry, exists := hc.cooldowns[configName]
	if exists {
		entry.failedAt = time.Now()
		entry.consecutiveFails++
	} else {
		hc.cooldowns[configName] = &cooldownEntry{
			failedAt:       time.Now(),
			consecutiveFails: 1,
		}
	}
}

func (hc *HealthChecker) isInCooldown(configName string) bool {
	hc.cooldownMu.Lock()
	defer hc.cooldownMu.Unlock()
	entry, exists := hc.cooldowns[configName]
	if !exists { return false }

	// Cooldown = min(30min * failures, 6h)
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

func (hc *HealthChecker) clearCooldown(configName string) {
	hc.cooldownMu.Lock()
	defer hc.cooldownMu.Unlock()
	delete(hc.cooldowns, configName)
}
```

**Step 2: Integrate cooldown into handleUnhealthy**

After a failed switchover, call `addCooldown(oldConfigName)`.
After a successful connection, call `clearCooldown(configName)`.

**Step 3: Initialize cooldowns map in NewHealthChecker**

```go
func NewHealthChecker(...) *HealthChecker {
	return &HealthChecker{
		// ... existing fields
		cooldowns: make(map[string]*cooldownEntry),
	}
}
```

**Step 4: Build and verify**

```bash
cd backend && go build ./cmd/vpn-agent/
```

**Step 5: Commit**

```bash
git add backend/cmd/vpn-agent/health_failover.go
git commit -m "feat(vpn-agent): add cooldown mechanism to prevent failover ping-pong"
```

---

## Task 6: Agent — Staleness Detection

**Files:**
- Modify: `backend/cmd/vpn-agent/config_store.go` — add staleness check
- Modify: `backend/cmd/vpn-agent/health.go` — record success on healthy check

**Step 1: Add staleness detection to ConfigStore**

```go
const staleThreshold = 7 * 24 * time.Hour // 7 days

func (s *ConfigStore) CheckAndMarkStale() {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta := s.loadMeta()
	for name, m := range meta.Configs {
		if m.LastSuccessAt == nil {
			// Never succeeded — stale if uploaded > 7 days ago
			m.Stale = time.Since(m.UploadedAt) > staleThreshold
		} else {
			m.Stale = time.Since(*m.LastSuccessAt) > staleThreshold
		}
		meta.Configs[name] = m
	}
	s.saveMeta(meta)
}
```

**Step 2: Call RecordSuccess on healthy check**

In `health.go`, inside `checkTunnel()`, when tunnel is healthy:
```go
if healthy {
	hc.store.RecordSuccess(t.ConfigName)
}
```

**Step 3: Add periodic staleness check**

In `health.go`, inside the health check loop, every 1 hour:
```go
// Add staleness check counter
if checkCount % 120 == 0 { // every ~1 hour (30s * 120)
	hc.store.CheckAndMarkStale()
}
```

**Step 4: Build and verify**

```bash
cd backend && go build ./cmd/vpn-agent/
```

**Step 5: Commit**

```bash
git add backend/cmd/vpn-agent/
git commit -m "feat(vpn-agent): add config staleness detection and success tracking"
```

---

## Task 7: Agent — Cross-Region Failover Integration

**Files:**
- Modify: `backend/cmd/vpn-agent/health_failover.go` — update handleUnhealthy flow

**Step 1: Update handleUnhealthy to use new weighted selection**

Replace the existing failover logic to:
1. Try restart (2x) — unchanged
2. Try same-region alternatives using `findBestAlternative(config, region, tried)`
3. If no same-region available, try cross-region: `findBestAlternative(config, "", tried)`
4. On each failed switch, add to `tried` list and call `addCooldown`
5. On success, call `clearCooldown` and `store.RecordSuccess`

Key change: remove the region filter from `findBestAlternative` when same-region exhausted.

**Step 2: Build and verify**

```bash
cd backend && go build ./cmd/vpn-agent/
```

**Step 3: Commit**

```bash
git add backend/cmd/vpn-agent/health_failover.go
git commit -m "feat(vpn-agent): cross-region failover with cooldown integration"
```
