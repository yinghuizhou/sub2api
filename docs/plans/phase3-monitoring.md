# Phase 3: Monitoring Dashboard

> Event logging, alert rules, and monitoring dashboard UI.

## Task 8: Database — VPN Events and Alert Rules

**Files:**
- Create: `backend/migrations/068_add_vpn_events.sql`
- Create: `backend/migrations/069_add_vpn_alert_rules.sql`
- Create: `backend/ent/schema/vpnevent.go`
- Create: `backend/ent/schema/vpnalertrule.go`
- Modify: `backend/ent/schema/proxy.go` — add config tracking fields

**Step 1: Create migration for vpn_events**

`068_add_vpn_events.sql`:
```sql
-- VPN event log for monitoring and alerting
CREATE TABLE vpn_events (
    id          BIGSERIAL PRIMARY KEY,
    tunnel_name VARCHAR(100) NOT NULL,
    event_type  VARCHAR(50)  NOT NULL,
    details     JSONB        DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vpn_events_tunnel ON vpn_events(tunnel_name);
CREATE INDEX idx_vpn_events_type ON vpn_events(event_type);
CREATE INDEX idx_vpn_events_time ON vpn_events(created_at DESC);

-- Auto-cleanup events older than 30 days (run via cron or app logic)
COMMENT ON TABLE vpn_events IS 'VPN tunnel event log for monitoring. Auto-cleanup >30d recommended.';
```

**Step 2: Create migration for vpn_alert_rules**

`069_add_vpn_alert_rules.sql`:
```sql
CREATE TABLE vpn_alert_rules (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    condition   VARCHAR(50)  NOT NULL,
    threshold   INT          NOT NULL,
    webhook_url TEXT         NOT NULL DEFAULT '',
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Add config tracking to proxies
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS config_version    INT DEFAULT 1;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS last_connected_at TIMESTAMPTZ;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS config_stale      BOOLEAN DEFAULT FALSE;
```

**Step 3: Create Ent schema for VpnEvent**

`backend/ent/schema/vpnevent.go`:
```go
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type VpnEvent struct {
	ent.Schema
}

func (VpnEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("tunnel_name").MaxLen(100),
		field.String("event_type").MaxLen(50),
		field.JSON("details", map[string]interface{}{}).Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

func (VpnEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tunnel_name"),
		index.Fields("event_type"),
		index.Fields("created_at"),
	}
}
```

**Step 4: Create Ent schema for VpnAlertRule**

`backend/ent/schema/vpnalertrule.go`:
```go
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"time"
)

type VpnAlertRule struct {
	ent.Schema
}

func (VpnAlertRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").MaxLen(100),
		field.String("condition").MaxLen(50),
		field.Int("threshold"),
		field.String("webhook_url").Default(""),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
```

**Step 5: Update proxy schema**

Add to `proxy.go` Fields():
```go
field.Int("config_version").Default(1).Optional(),
field.Time("last_connected_at").Optional().Nillable(),
field.Bool("config_stale").Default(false).Optional(),
```

**Step 6: Regenerate Ent code**

```bash
cd backend && go generate ./ent
```

**Step 7: Commit**

```bash
git add backend/migrations/ backend/ent/
git commit -m "feat(db): add vpn_events, vpn_alert_rules tables and proxy config fields"
```

---

## Task 9: Backend — VPN Event Service

**Files:**
- Create: `backend/internal/service/vpn_event_service.go`
- Modify: `backend/internal/handler/admin/vpn_handler.go` — add event endpoints

**Step 1: Create VpnEventService**

```go
type VpnEventService struct {
	client *ent.Client
}

func NewVpnEventService(client *ent.Client) *VpnEventService

// RecordEvent creates a new VPN event
func (s *VpnEventService) RecordEvent(ctx context.Context, tunnelName, eventType string,
    details map[string]interface{}) error

// ListEvents returns paginated events with optional filters
func (s *VpnEventService) ListEvents(ctx context.Context, opts ListEventsOpts) ([]VpnEventDTO, int, error)

type ListEventsOpts struct {
	TunnelName string
	EventType  string
	Since      *time.Time
	Page       int
	PageSize   int
}

type VpnEventDTO struct {
	ID         int64                  `json:"id"`
	TunnelName string                 `json:"tunnel_name"`
	EventType  string                 `json:"event_type"`
	Details    map[string]interface{} `json:"details"`
	CreatedAt  time.Time              `json:"created_at"`
}

// GetDashboard returns aggregated VPN status
func (s *VpnEventService) GetDashboard(ctx context.Context,
    vpnSvc *VpnAgentService) (*DashboardData, error)

type DashboardData struct {
	ActiveTunnels  int              `json:"active_tunnels"`
	OfflineTunnels int              `json:"offline_tunnels"`
	DegradedTunnels int             `json:"degraded_tunnels"`
	TotalConfigs   int              `json:"total_configs"`
	RecentEvents   []VpnEventDTO    `json:"recent_events"`
	StaleConfigs   []string         `json:"stale_configs"`
}

// CleanupOldEvents removes events older than retention period
func (s *VpnEventService) CleanupOldEvents(ctx context.Context, retention time.Duration) (int, error)
```

**Step 2: Add handler endpoints**

In `vpn_handler.go`:
```go
func (h *VpnHandler) ListEvents(c *gin.Context)    // GET /vpn/events
func (h *VpnHandler) GetDashboard(c *gin.Context)  // GET /vpn/dashboard
```

Query params for ListEvents: `tunnel_name`, `event_type`, `since`, `page`, `page_size`

**Step 3: Register routes**

In `admin.go`:
```go
vpn.GET("/events", h.Admin.Vpn.ListEvents)
vpn.GET("/dashboard", h.Admin.Vpn.GetDashboard)
```

**Step 4: Wire DI — add VpnEventService**

Update provider to inject VpnEventService into VpnHandler.

**Step 5: Build and verify**

```bash
cd backend && go generate ./cmd/server && go build -tags embed ./cmd/server/
```

**Step 6: Commit**

```bash
git add backend/internal/
git commit -m "feat(backend): add VPN event service and dashboard endpoint"
```

---

## Task 10: Backend — Alert Rule Service

**Files:**
- Create: `backend/internal/service/vpn_alert_service.go`
- Modify: `backend/internal/handler/admin/vpn_handler.go` — add alert CRUD
- Modify: `backend/internal/server/routes/admin.go` — register routes

**Step 1: Create VpnAlertService**

```go
type VpnAlertService struct {
	client     *ent.Client
	eventSvc   *VpnEventService
	httpClient *http.Client
}

func NewVpnAlertService(client *ent.Client, eventSvc *VpnEventService) *VpnAlertService

// CRUD
func (s *VpnAlertService) Create(ctx context.Context, input CreateAlertRuleInput) (*AlertRuleDTO, error)
func (s *VpnAlertService) List(ctx context.Context) ([]AlertRuleDTO, error)
func (s *VpnAlertService) Update(ctx context.Context, id int64, input UpdateAlertRuleInput) (*AlertRuleDTO, error)
func (s *VpnAlertService) Delete(ctx context.Context, id int64) error

// Evaluation — called from proxy status callback
func (s *VpnAlertService) EvaluateAndNotify(ctx context.Context, tunnelName, status, health string) error

// SendWebhook sends alert notification
func (s *VpnAlertService) sendWebhook(webhookURL string, payload AlertPayload) error

type CreateAlertRuleInput struct {
	Name       string `json:"name"`
	Condition  string `json:"condition"`   // tunnel_offline, consecutive_failover, all_offline, config_stale
	Threshold  int    `json:"threshold"`
	WebhookURL string `json:"webhook_url"`
}

type AlertRuleDTO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Condition  string    `json:"condition"`
	Threshold  int       `json:"threshold"`
	WebhookURL string    `json:"webhook_url"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

type AlertPayload struct {
	Alert     string                 `json:"alert"`
	Tunnel    string                 `json:"tunnel"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
}
```

**Step 2: Add handler endpoints**

```go
func (h *VpnHandler) CreateAlertRule(c *gin.Context)  // POST /vpn/alert-rules
func (h *VpnHandler) ListAlertRules(c *gin.Context)   // GET /vpn/alert-rules
func (h *VpnHandler) UpdateAlertRule(c *gin.Context)   // PUT /vpn/alert-rules/:id
func (h *VpnHandler) DeleteAlertRule(c *gin.Context)   // DELETE /vpn/alert-rules/:id
```

**Step 3: Register routes**

```go
vpn.POST("/alert-rules", h.Admin.Vpn.CreateAlertRule)
vpn.GET("/alert-rules", h.Admin.Vpn.ListAlertRules)
vpn.PUT("/alert-rules/:id", h.Admin.Vpn.UpdateAlertRule)
vpn.DELETE("/alert-rules/:id", h.Admin.Vpn.DeleteAlertRule)
```

**Step 4: Wire DI update**

**Step 5: Build and verify**

```bash
cd backend && go generate ./cmd/server && go build -tags embed ./cmd/server/
```

**Step 6: Commit**

```bash
git add backend/internal/
git commit -m "feat(backend): add VPN alert rule CRUD and webhook notifications"
```

---

## Task 11: Agent — Event Emission

**Files:**
- Modify: `backend/cmd/vpn-agent/callback.go` — add event reporting
- Modify: `backend/cmd/vpn-agent/health.go` — emit events on state changes
- Modify: `backend/cmd/vpn-agent/health_failover.go` — emit failover events

**Step 1: Add event callback**

In `callback.go`:
```go
func (c *CallbackClient) ReportEvent(tunnelName, eventType string,
    details map[string]interface{}) error {
	// POST /api/v1/admin/vpn/events/report
	// Body: {tunnel_name, event_type, details}
	// Non-blocking, log errors but don't fail
}
```

**Step 2: Add event report endpoint to backend**

In `vpn_handler.go`:
```go
func (h *VpnHandler) ReportEvent(c *gin.Context)  // POST /vpn/events/report
```

In `admin.go`:
```go
vpn.POST("/events/report", h.Admin.Vpn.ReportEvent)
```

**Step 3: Emit events in health check**

In `health.go`, emit events when:
- Tunnel status changes (connected → disconnected, etc.)
- Health status changes (healthy → degraded → unhealthy)

In `health_failover.go`, emit events when:
- Failover initiated
- Failover succeeded (with from/to config)
- Failover failed (all alternatives exhausted)

**Step 4: Build and verify**

```bash
cd backend && go build ./cmd/vpn-agent/ && go build -tags embed ./cmd/server/
```

**Step 5: Commit**

```bash
git add backend/
git commit -m "feat(vpn-agent): emit events on status changes and failovers"
```

---

## Task 12: Frontend — VPN Monitor Dashboard

**Files:**
- Create: `frontend/src/views/admin/VpnMonitorView.vue`
- Modify: `frontend/src/api/vpn.ts` — add dashboard, events, alert APIs
- Modify: `frontend/src/router/index.ts` — add route

**Step 1: Add API functions**

In `vpn.ts`:
```typescript
// Dashboard
export interface DashboardData {
  active_tunnels: number
  offline_tunnels: number
  degraded_tunnels: number
  total_configs: number
  recent_events: VpnEvent[]
  stale_configs: string[]
}

export interface VpnEvent {
  id: number
  tunnel_name: string
  event_type: string
  details: Record<string, unknown>
  created_at: string
}

export function getDashboard() {
  return apiClient.get<DashboardData>('/admin/vpn/dashboard')
}

export function listEvents(params?: {
  tunnel_name?: string; event_type?: string; page?: number; page_size?: number
}) {
  return apiClient.get<{ items: VpnEvent[]; total: number }>('/admin/vpn/events', { params })
}

// Alert Rules
export interface AlertRule { id: number; name: string; condition: string;
  threshold: number; webhook_url: string; enabled: boolean }

export function listAlertRules() {
  return apiClient.get<AlertRule[]>('/admin/vpn/alert-rules')
}
export function createAlertRule(input: Omit<AlertRule, 'id' | 'enabled'>) {
  return apiClient.post<AlertRule>('/admin/vpn/alert-rules', input)
}
export function updateAlertRule(id: number, input: Partial<AlertRule>) {
  return apiClient.put<AlertRule>(`/admin/vpn/alert-rules/${id}`, input)
}
export function deleteAlertRule(id: number) {
  return apiClient.delete(`/admin/vpn/alert-rules/${id}`)
}
```

**Step 2: Create VpnMonitorView**

4 sections:
1. **Status Cards** — active/offline/degraded/total counts
2. **Tunnel Status Table** — real-time health with color indicators
3. **Event Log** — scrollable timeline with filtering
4. **Alert Rules** — CRUD table with enable/disable toggle

Use existing DataTable/Badge components. Auto-refresh every 30s.

**Step 3: Add route**

In `router/index.ts`, add after the vpn route:
```typescript
{
  path: '/admin/vpn-monitor',
  name: 'AdminVpnMonitor',
  component: () => import('@/views/admin/VpnMonitorView.vue'),
  meta: { requiresAuth: true, requiresAdmin: true,
    title: 'VPN Monitor', titleKey: 'admin.vpnMonitor.title' }
}
```

**Step 4: Build and verify**

```bash
cd frontend && pnpm build
```

**Step 5: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add VPN monitoring dashboard with events and alerts"
```

---

## Task 13: Integration — End-to-End Verification

**Step 1: Run full build**

```bash
make build
```

**Step 2: Run tests**

```bash
cd backend && go test -tags=unit ./...
cd frontend && pnpm test:run
```

**Step 3: Run linting**

```bash
make lint
```

**Step 4: Fix any issues found**

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore: fix lint and build issues for VPN management v2"
```
