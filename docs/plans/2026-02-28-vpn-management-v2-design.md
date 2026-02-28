# VPN Management V2 — Enhanced Architecture Design

> Date: 2026-02-28
> Status: Approved
> Approach: Enhance existing VPN Agent + Sub2API backend (no new services)

## Problem Statement

Current VPN management pain points:
1. **Config update friction** — Astrill .ovpn files must be manually downloaded and uploaded via frontend
2. **Failover not smart enough** — Only same-region switching, no cross-region or weighted selection
3. **No unified monitoring** — No dashboard, event history, or alerting

## Solution Overview

Three enhancement modules on existing architecture:

| Module | Purpose |
|--------|---------|
| Config Push API | Browser extension pushes .ovpn directly to server API |
| Smart Failover | Cross-region switching, weighted scoring, cooldown, staleness detection |
| Monitoring Dashboard | Real-time status, event log, alert rules with webhook notifications |

## Module 1: Config Push API + Auto-Deploy

### New Endpoint

```
POST /api/v1/admin/vpn/configs/push
Auth: x-api-key header
```

### Request

```json
{
  "configs": [
    {
      "filename": "us-losangeles-tcp443.ovpn",
      "content": "<base64 encoded>",
      "region": "us-west",
      "auto_deploy": true
    }
  ],
  "replace_existing": true
}
```

### Auto-Deploy Logic

1. Save config to VPN Agent config store
2. If `auto_deploy=true`, find tunnels using same-name config
3. Execute `SwitchConfig` on matching tunnels (hot-swap)
4. Record event in `vpn_events` table
5. Return deployment results

### Config Versioning

- Track `uploaded_at` timestamp per config
- Content hash deduplication (skip identical uploads)
- Retain last N versions (default: 3)
- Mark configs as `stale` if unused > X days

### Browser Extension Integration

- Extension settings: server URL + API key
- On .ovpn download → POST to push endpoint
- Show push result notification in extension popup

## Module 2: Smart Failover

### Current Behavior

```
3 consecutive health failures → restart 2x → switch same-region 3x
```

### Enhanced Behavior

```
3 failures → restart 2x → same-region switch → nearby-region → global
```

### Weighted Selection Algorithm

```go
score = (successRate * 0.4) + (1/latencyMs * 0.3) + (freshness * 0.2) + (regionBonus * 0.1)
```

| Factor | Weight | Source |
|--------|--------|--------|
| successRate | 0.4 | Existing scores.json |
| latency | 0.3 | Average latency (inverse, normalized) |
| freshness | 0.2 | Config upload recency |
| regionMatch | 0.1 | Same region = 1.0, nearby = 0.5, other = 0 |

### Region Proximity Map

```go
var regionProximity = map[string][]string{
    "us-west":  {"us-east", "us-central"},
    "us-east":  {"us-west", "us-central", "eu-west"},
    "eu-west":  {"eu-central", "us-east"},
    "eu-central": {"eu-west"},
    "ap-east":  {"ap-southeast"},
    // ...
}
```

### Cooldown Mechanism

- After failover away from a config: cooldown = min(30min * failures, 6h)
- During cooldown, config is excluded from selection
- Cooldown state stored in memory (reset on agent restart)

### Staleness Detection

- Track `last_success_at` per config in scores.json
- Config unused > 7 days → mark `stale`
- Stale configs deprioritized (score * 0.3)
- Dashboard shows stale config warnings

## Module 3: Monitoring Dashboard

### Database Schema

#### vpn_events table

```sql
CREATE TABLE vpn_events (
    id          BIGSERIAL PRIMARY KEY,
    tunnel_name VARCHAR(100) NOT NULL,
    event_type  VARCHAR(50)  NOT NULL,
    details     JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_vpn_events_tunnel ON vpn_events(tunnel_name);
CREATE INDEX idx_vpn_events_type ON vpn_events(event_type);
CREATE INDEX idx_vpn_events_time ON vpn_events(created_at);
```

Event types: `connected`, `disconnected`, `failover`, `config_push`, `config_switch`, `alert_triggered`

#### vpn_alert_rules table

```sql
CREATE TABLE vpn_alert_rules (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    condition   VARCHAR(50)  NOT NULL,
    threshold   INT          NOT NULL,
    webhook_url TEXT,
    enabled     BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

Conditions: `tunnel_offline` (minutes), `consecutive_failover` (count), `all_offline`, `config_stale` (days)

#### Proxy table additions

```sql
ALTER TABLE proxies ADD COLUMN config_version    INT DEFAULT 1;
ALTER TABLE proxies ADD COLUMN last_connected_at TIMESTAMPTZ;
ALTER TABLE proxies ADD COLUMN config_stale      BOOLEAN DEFAULT FALSE;
```

### Backend APIs

```
GET  /api/v1/admin/vpn/events              # List events (paginated, filterable)
GET  /api/v1/admin/vpn/dashboard            # Aggregated status summary

POST /api/v1/admin/vpn/alert-rules          # Create alert rule
GET  /api/v1/admin/vpn/alert-rules          # List alert rules
PUT  /api/v1/admin/vpn/alert-rules/:id      # Update alert rule
DELETE /api/v1/admin/vpn/alert-rules/:id    # Delete alert rule
```

### Dashboard API Response

```json
{
  "tunnels": {
    "active": 5,
    "offline": 1,
    "degraded": 0,
    "total_configs": 23
  },
  "recent_events": [...],
  "latency_history": {
    "tunnel-name": [{"time": "...", "ms": 45}, ...]
  },
  "stale_configs": ["old-config-1.ovpn", "old-config-2.ovpn"]
}
```

### Alert Execution Flow

1. VPN Agent health check detects issue
2. Agent callback → backend receives status update
3. Backend evaluates alert rules against current state
4. If threshold met → send webhook POST
5. Record `alert_triggered` event

### Webhook Payload

```json
{
  "alert": "tunnel_offline",
  "tunnel": "us-west-1",
  "message": "Tunnel us-west-1 offline for 10 minutes",
  "details": {"last_health": "...", "failures": 6},
  "timestamp": "2026-02-28T14:32:00Z"
}
```

### Frontend Dashboard Page

New route: `/admin/vpn-monitor`

Components:
- StatusCards: active/offline/degraded counts
- LatencyChart: 24h trend (lightweight, no charting library — use CSS bars or sparklines)
- TunnelTable: real-time status with health indicators
- EventLog: scrollable event timeline
- AlertConfig: CRUD for alert rules

## Implementation Priority

| Phase | Module | Effort |
|-------|--------|--------|
| P0 | Config Push API (backend + agent) | Medium |
| P0 | Browser extension push integration | Small |
| P1 | Smart Failover (agent enhancement) | Medium |
| P1 | Event logging (backend + agent) | Medium |
| P2 | Monitoring Dashboard (frontend) | Large |
| P2 | Alert rules + webhook | Medium |

## Files to Modify

### Backend (Go)

| File | Change |
|------|--------|
| `cmd/vpn-agent/server_handlers.go` | Add push config handler |
| `cmd/vpn-agent/config_store.go` | Add versioning, staleness tracking |
| `cmd/vpn-agent/health_failover.go` | Cross-region, weighted scoring, cooldown |
| `cmd/vpn-agent/health.go` | Event emission on state changes |
| `cmd/vpn-agent/callback.go` | Add event callback to backend |
| `internal/handler/admin/vpn_handler.go` | Add push, events, dashboard, alert APIs |
| `internal/service/vpn_agent_service.go` | Add push proxy, event service calls |
| `internal/service/vpn_event_service.go` | **NEW** — event CRUD + alert evaluation |
| `internal/service/vpn_alert_service.go` | **NEW** — alert rule CRUD + webhook dispatch |
| `ent/schema/vpn_event.go` | **NEW** — Ent schema |
| `ent/schema/vpn_alert_rule.go` | **NEW** — Ent schema |
| `ent/schema/proxy.go` | Add config_version, last_connected_at, config_stale |
| `internal/server/routes/admin.go` | Register new routes |

### Frontend (Vue)

| File | Change |
|------|--------|
| `src/views/admin/VpnMonitorView.vue` | **NEW** — monitoring dashboard |
| `src/views/admin/VpnView.vue` | Add push status indicators |
| `src/api/vpn.ts` | Add push, events, dashboard, alert API calls |
| `src/router/index.ts` | Add /admin/vpn-monitor route |

### Database

| Migration | Description |
|-----------|-------------|
| `NNN_add_vpn_events.sql` | Create vpn_events table |
| `NNN_add_vpn_alert_rules.sql` | Create vpn_alert_rules table |
| `NNN_add_proxy_config_fields.sql` | Add proxy table columns |
