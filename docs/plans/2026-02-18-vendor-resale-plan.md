# Vendor Resale System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a vendor procurement and resale system that allows purchasing low-cost API forwarding services from third-party providers and reselling them through Sub2API platform.

**Architecture:** Vendor entity manages supplier metadata (API format, base_url, billing). Vendor accounts are standard Account entities with vendor_id FK, using existing scheduling/billing/monitoring infrastructure. Protocol conversion layer handles Anthropic↔OpenAI format translation for OpenAI-compatible vendors.

**Tech Stack:** Go 1.25.7 (Gin + Ent ORM), Vue 3 + TypeScript + TailwindCSS, PostgreSQL, Redis

**Key Discovery:** Account.GetBaseURL() already supports custom base_url in credentials, and buildUpstreamRequest() already handles api_key type accounts with custom base_url. This means vendor accounts work with the gateway with minimal changes.

---

## Batch 1: Database & Domain Layer (Backend Foundation)

### Task 1: Create Vendor Ent Schema

**Files:**
- Create: `backend/ent/schema/vendor.go`

**Step 1: Create vendor schema file**

Create `backend/ent/schema/vendor.go` with all fields from the design doc:
- Basic: name, description
- API config: api_format, base_url, auth_type, api_path_override, extra_headers (jsonb)
- Billing: billing_type, cost_per_1k_input, cost_per_1k_output, total_quota_usd, used_quota_usd, balance_usd, expires_at
- Health: status, health_check_enabled, health_check_interval, health_check_model, last_health_check_at, last_health_status, last_health_latency, error_message, consecutive_failures
- Auto-purchase: auto_purchase_enabled, auto_purchase_config (jsonb)
- Balance alert: balance_alert_enabled, balance_alert_threshold
- Edges: accounts (one-to-many)
- Mixins: TimeMixin, SoftDeleteMixin
- Indexes: status, api_format, billing_type, deleted_at

Follow the exact pattern from `backend/ent/schema/proxy.go` or `backend/ent/schema/account.go`.

**Step 2: Add vendor_id and source_type to Account schema**

Modify `backend/ent/schema/account.go`:
- Add field `vendor_id` (int64, optional, nillable)
- Add field `source_type` (string, max 20, default "owned")
- Add edge `vendor` (to Vendor, field vendor_id, unique)
- Add index on `vendor_id` and `source_type`

**Step 3: Run Ent code generation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go generate ./ent`
Expected: Generated files updated successfully

**Step 4: Verify compilation**

Run: `cd /Users/zhouyingh/sub2api/backend && go build ./...`
Expected: No errors

**Step 5: Commit**

```bash
git add backend/ent/
git commit -m "feat(vendor): add Vendor ent schema and Account vendor_id field"
```

---

### Task 2: Create Database Migration

**Files:**
- Create: `backend/migrations/054_create_vendors_table.sql`
- Create: `backend/migrations/055_add_vendor_id_to_accounts.sql`

**Step 1: Create vendors table migration**

Create `backend/migrations/054_create_vendors_table.sql`:
- CREATE TABLE vendors with all columns matching the Ent schema
- Add partial unique index on name WHERE deleted_at IS NULL
- Add indexes on status, api_format, billing_type, deleted_at

**Step 2: Create accounts vendor_id migration**

Create `backend/migrations/055_add_vendor_id_to_accounts.sql`:
- ALTER TABLE accounts ADD COLUMN vendor_id BIGINT REFERENCES vendors(id)
- ALTER TABLE accounts ADD COLUMN source_type VARCHAR(20) DEFAULT 'owned'
- CREATE INDEX on vendor_id and source_type

**Step 3: Commit**

```bash
git add backend/migrations/
git commit -m "feat(vendor): add database migrations for vendors table and account vendor_id"
```

---

### Task 3: Create Vendor Domain Model & Service Interfac:**
- Create: `backend/internal/service/vendor.go` (domain model)
- Create: `backend/internal/service/vendor_service.go` (service interface + repository interface)

**Step 1: Create vendor domain model**

Create `backend/internal/service/vendor.go` with:
- `type Vendor struct` with all fields
- Helper methods: `IsActive()`, `IsHealthy()`, `GetAPIPath()`, `NeedsBalanceAlert()`
- Constants for vendor status, api_format, billing_type, auth_type

**Step 2: Create vendor service and repository interfaces**

Create `backend/internal/service/vendor_service.go` with:
- `type VendorRepository interface` (CRUD + List + ListByStatus + ListWithHealthCheckDue)
- `type VendorService struct` with CRUD methods
- Input types: CreateVendorInput, UpdateVendorInput
- Error variables: ErrVendorNotFound, ErrVendorNilInput

Follow the exact pattern from `backend/internal/service/proxy_service.go`.

**Step 3: Verify compilation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go build ./...`

**Step 4: Commit**

```bash
git add backend/internal/service/vendor.go backend/internal/service/vendor_service.go
git commit -m "feat(vendor): add Vendor domain model and service interface"
```

---

### Task 4: Create Vendor Repository Implementation

**Files:**
- Create: `backend/internal/repository/vendor_repo.go`
- Modify: `backend/internal/repository/wire.go` (add VendorRepo provider)

**Step 1: Implement VendorRepository**

Create `backend/internal/repository/vendor_repo.go`:
- Implement all VendorRepository interface methods
- Use Ent client for CRUD operations
- Include proper entity-to-domain conversion (entVendorToService)
- Follow pattern from `backend/internal/repository/proxy_repo.go`

**Step 2: Register in Wire**

Add VendorRepo to the repository provider set in `backend/internal/repository/wire.go`.

**Step 3: Run Wire generation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go generate ./cmd/serve**Step 4: Verify compilation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go build ./...`

**Step 5: Commit**

```bash
git add backend/internal/repository/vendor_repo.go backend/internal/repository/wire.go backend/cmd/server/wire_gen.go
git commit -m "feat(vendor): implement VendorRepository with Ent ORM"
```

---

## Batch 2: Admin API & Handler Layer

### Task 5: Create Vendor Admin Handler

**Files:**
- Create: `backend/internal/handler/admin/vendor_handler.go`
- Modify: `backend/internal/handler/handler.go` (add Vendor to AdminHandlers)
- Modify: `backend/internal/server/routes/admin.go` (register vendor routes)

**Step 1: Create VendorHandler**

Create `backend/internal/handler/admin/vendor_handler.go`:
- CRUD handlers: List, GetByID, Create, Update, Delete
- Special handlers: Test (connectivity test), RefreshBalance, GetStats, Dashboard
- Health check handlers: TriggerHealthCheck, GetHealthHistory, UpdateHealthConfig
- Follow pattern from `backend/internal/handler/admin/proxy_handler.go`

**Step 2: Add to AdminHandlers struct**

In `backend/internal/handler/handler.go`, add:
```go
Vendor *admin.VendorHandler
```

**Step 3: Register routes**

In `backend/internal/server/routes/admin.go`, add `registerVendorRoutes(admin, h)` and implement:
```go
func registerVendorRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
    vendors := admin.Group("/vendors")
    {
        vendors.GET("", h.Admin.Vendor.List)
        vendors.GET("/dashboard", h.Admin.Vendor.Dashboard)
        vendors.GET("/:id", h.Admin.Vendor.GetByID)
        vendors.POST("", h.Admin.Vendor.Create)
        vendors.PUT("/:id", h.Admin.Vendor.Update)
        vendors.DELETE("/:id", h.Admin.Vendor.Delete)
        vendors.POST("/:id/test", h.Admin.Vendor.Test)
        vendors.POST("/:id/refresh-balance", h.Admin.Vendor.RefreshBalance)
        vendors.GET("/:id/stats", h.Admin.Vendor.GetStats)
        vendors.POST("/:id/health-check", h.Admin.Vendor.TriggerHealthCheck)
        vendors.GET("/:id/accounts", h.Admin.Vendor.GetVendorAccounts)
        vendors.POST("/:id/accounts", h.Admin.Vendor.BatchCreateAccounts)
    }
}
```

**Step 4: Wire DI update**

Update Wire to inject VendorHandler with its dependencies.
Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go generate ./cmd/server`

**Step 5: Verify compilation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go build ./...`

**Step 6: Commit**

```bash
git add backend/internal/handler/admin/vendor_handler.go backend/internal/handler/handler.go backend/internal/server/routes/admin.go backend/cmd/server/wire.go backend/cmd/server/wire_gen.go
git commit -m "feat(vendor): add Vendor admin API handler and routes"
```

---

### Task 6: Extend Account API for Vendor Support

**Files:**
- Modify: `backend/internal/service/account_service.go` (add vendor_id to CreateAccountInput/UpdateAccountInput)
- Modify: `backend/internal/handler/admin/account_handler.go` (accept vendor_id in create/update)
- Modify: `backend/internal/handler/dto/account.go` (add vendor fields to DTO)

**Step 1: Extend Account service types**

Add `VendorID *int64` and `SourceType string` to:
- CreateAccountInput / CreateAccountRequest
- UpdateAccountInput / UpdateAccountRequest
- Account domain struct

**Step 2: Extend Account handler**

In account_handler.go Create/Update methods, pass vendor_id through.
In List method, add `source_type` query parameter for filtering.

**Step 3: Extend Account DTO**

Add vendor_id, source_type, and vendor_name to account response DTO.

**Step 4: Verify compilation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go build ./...`

**Step 5: Commit**

```bash
git add backend/internal/service/account_service.go backend/internal/handler/admin/account_handler.go backend/internal/handler/dto/
git commit -m "feat(vendor): extend Account API with vendor_id support"
```

---

## Batch 3: Health Check & Monitoring Services

### Task 7: Vendor Health Check Service

**Files:**
- Create: `backend/internal/service/vendor_health_service.go`

**Step 1: Implement VendorHealthService**

Create health check service with:
- `RunHealthCheck(ctx, vendorID)` — send test request to vendor API, measure latency
- `RunAllDueHealthChecks(ctx)` — find vendors with health_check_enabled where last check is overdue
- `UpdateHealthStatus(ctx, vendorID, status, latency, error)` — update vendor health fields
- `AutoSuspendUnhealthy(ctx)` — suspend vendors with consecutive_failures > threshold
- `AutoResumeHealthy(ctx)` — resume suspended vendors that pass health check

The health check sends a minimal API request (e.g., count_tokens or a tiny messages request) to the vendor's base_url and measures response time.

**Step 2: Verify compilation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go build ./...`

**Step 3: Commit**

```bash
git add backend/internal/service/vendor_health_service.go
git commit -m "feat(vendor): add VendorHealthService for automated health monitoring"
```

---

### Task 8: Vendor Balance & Alert Service

**Files:**
- Create: `backend/internal/service/vendor_balance_service.go`

**Step 1: Implement VendorBalanceService**

- `TrackUsage(ctx, vendorID, inputTokens, outputTokens)` — update used_quota based on token usage
- `CheckBalanceAlerts(ctx)` — find vendors with balance below threshold, send alerts
- `RefreshBalance(ctx, vendorID)` — attempt to query vendor API for current balance
- `AutoSuspendDepleted(ctx)` — suspend vendors with depleted quota or expired subscription
- `GetCostAnalysis(ctx, vendorID, period)` — calculate cost vs revenue for a vendor

**Step 2: Verify compilation and commit**

```bash
git add backend/internal/service/vendor_balance_service.go
git commit -m "feat(vendor): add VendorBalanceService for cost tracking and alerts"
```

---

### Task 9: Vendor Background Jobs

**Files:**
- Create: `backend/internal/service/vendor_background_service.go`
- Modify: Wire DI to register background service

**Step 1: Implement background service**

Create a service that runs periodic tasks:
- Health checks (every vendor.health_check_interval seconds)
- Balance alerts (every 30 minutes)
- Auto-suspend depleted/expired vendors (every 5 minutes)
- Auto-pricing recalculation (every hour, when enabled)

Use the same pattern as `backend/internal/service/account_expiry_service.go`.

**Step 2: Wire DI and verify**

Register in Wire, run go generate, verify compilation.

**Step 3: Commit**

```bash
git add backend/internal/service/vendor_background_service.go backend/cmd/server/wire.go backend/cmd/server/wire_gen.go
git commit -m "feat(vendor): add vendor background jobs for health check and balance monitoring"
```

---

## Batch 4: Protocol Conversion Layer

### Task 10: Anthropic ↔ OpenAI Protocol Adapter

**Files:**
- Create: `backend/internal/service/vendor_adapter.go`
- Create: `backend/internal/service/vendor_adapter_test.go`

**Step 1: Write failing tests**

Create test cases for:
- ConvertAnthropicRequestToOpenAI: messages format, system prompt, model mapping
- ConvertOpenAIResponseToAnthropic: non-streaming response conversion
- ConvertOpenAISSEToAnthropic: streaming SSE event conversion
- Error response conversion

**Step 2: Implement VendorProtocolAdapter**

```go
type VendorProtocolAdapter struct{}

func (a *VendorProtocolAdapter) ConvertRequest(anthropicBody []byte, vendor *Vendor) ([]byte, error)
func (a *VendorProtocolAdapter) ConvertResponse(openaiBody []byte) ([]byte, error)
func (a *VendorProtocolAdapter) ConvertSSEEvent(openaiEvent []byte) ([]byte, error)
```

Conversion rules:
- Anthropic messages → OpenAI messages (role mapping, content block → string/array)
- system prompt → system message at index 0
- model name mapping (configurable per vendor)
- usage: input_tokens/output_tokens → prompt_tokens/completion_tokens
- stop_reason → finish_reason mapping
- SSE: data format conversion

**Step 3: Run tests**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go test -tags=unit -run TestVendorAdapter ./internal/service/...`

**Step 4: Commit**

```bash
git add backend/internal/service/vendor_adapter.go backend/internal/service/vendor_adapter_test.go
git commit -m "feat(vendor): implement Anthropic↔OpenAI protocol adapter with tests"
```

---

### Task 11: Integrate Vendor Adapter into Gateway

**Files:**
- Modify: `backend/internal/service/gateway_service.go` (buildUpstreamRequest + response handling)

**Step 1: Modify buildUpstreamRequest**

At line ~3510 in gateway_service.go, after the existing base_url logic for api_key accounts, add vendor-aware logic:
- If account has vendor_id, load vendor config
- If vendor.api_format == "openai", convert request body before sending
- Set appropriate auth headers based on vendor.auth_type
- Add vendor.extra_headers

**Step 2: Modify response handling**

In the SSE streaming loop and non-streaming response handling:
- If vendor.api_format == "openai", convert response back to Anthropic format
- Handle SSE event-by-event conversion for streaming

**Step 3: Verify compilation**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go build ./...`

**Step 4: Commit**

```bash
git add backend/internal/service/gateway_service.go
git commit -m "feat(vendor): integrate protocol adapter into gateway forwarding pipeline"
```

---

## Batch 5: Auto-Purchase & Discovery (Phase 3)

### Task 12: Vendor Discovery Service

**Files:**
- Create: `backend/internal/service/vendor_discovery_service.go`

**Step 1: Implement discovery service**

- `ScanMarketplaces(ctx)` — placeholder for AI-driven marketplace scanning
- `EvaluateVendor(ctx, url)` — test a potential vendor URL for API compatibility
- `ScoreVendor(ctx, vendorID)` — calculate vendor score based on price, latency, uptime
- `GetDiscoveryResults(ctx)` — return recent discovery results

Initially this is a framework with manual trigger capability. The actual AI crawling logic can be added later.

**Step 2: Commit**

```bash
git add backend/internal/service/vendor_discovery_service.go
git commit -m "feat(vendor): add vendor discovery service framework"
```

---

### Task 13: Auto-Purchase Service

**Files:**
- Create: `backend/internal/service/vendor_auto_purchase_service.go`

**Step 1: Implement auto-purchase service**

- `ProcessAutoPurchase(ctx, vendorID)` — execute auto-purchase workflow
- `CheckPurchaseNeeded(ctx)` — find vendors needing replenishment
- `CreateAccountFromPurchase(ctx, vendorID, apiKey)` — auto-create Account after purchase

This is a framework. Actual payment integration requires vendor-specific adapters.

**Step 2: Commit**

```bash
git add backend/internal/service/vendor_auto_purchase_service.go
git commit -m "feat(vendor): add auto-purchase service framework"
```

---

### Task 14: Dynamic Pricing Engine

**Files:**
- Create: `backend/internal/service/vendor_pricing_service.go`

**Step 1: Implement pricing service**

- `CalculateSuggestedPrice(ctx, vendorID)` — based on vendor cost + margin
- `RecalculateAllPrices(ctx)` — batch recalculate for all active vendors
- `GetPricingAnalysis(ctx)` — cost vs revenue analysis across all vendors
- `ApplyAutoPrice(ctx, vendorID)` — auto-update group rate_multiplier

**Step 2: Commit**

```bash
git add backend/internal/service/vendor_pricing_service.go
git commit -m "feat(vendor): add dynamic pricing engine"
```

---

## Batch 6: Frontend — Vendor Management UI

### Task 15: Vendor API Client

**Files:**
- Create: `frontend/src/api/vendor.ts`

**Step 1: Create vendor API client**

Define TypeScript types and API functions:
- Vendor interface with all fields
- CRUD functions: listVendors, getVendor, createVendor, updateVendor, deleteVendor
- Action functions: testVendor, refreshBalance, triggerHealthCheck, getVendorStats, getVendorDashboard
- Account functions: getVendorAccounts, batchCreateVendorAccounts

Follow pattern from `frontend/src/api/` existing files.

**Step 2: Commit**

```bash
git add frontend/src/api/vendor.ts
git commit -m "feat(vendor): add vendor API client"
```

---

### Task 16: Vendor Management Page

**Files:**
- Create: `frontend/src/views/admin/VendorsView.vue`
- Modify: `frontend/src/router/index.ts` (add vendor route)

**Step 1: Create VendorsView.vue**

Build vendor management page with:
- Vendor list table (name, api_format, billing_type, balance, status, health, actions)
- Create/Edit vendor modal form
- Delete confirmation
- Test connectivity button with result display
- Refresh balance button
- Health status indicators (green/yellow/red)
- Filter by status, api_format

Follow the design patterns from `frontend/src/views/admin/ProxiesView.vue`.

**Step 2: Add route**

Add `/admin/vendors` route in router with lazy loading.

**Step 3: Add to admin sidebar navigation**

Add "供应商管理" menu item in the admin sidebar.

**Step 4: Commit**

```bash
git add frontend/src/views/admin/VendorsView.vue frontend/src/router/index.ts
git commit -m "feat(vendor): add vendor management page"
```

---

### Task 17: Vendor Dashboard Component

**Files:**
- Create: `frontend/src/views/admin/VendorDashboard.vue`

**Step 1: Create dashboard**

- Total cost vs revenue summary cards
- Vendor health status overview (pie chart or status grid)
- Low balance alerts list
- Expiring vendors list
- Cost trend chart (last 30 days)

**Step 2: Commit**

```bash
git add frontend/src/views/admin/VendorDashboard.vue
git commit -m "feat(vendor): add vendor dashboard with cost analysis"
```

---

### Task 18: Extend Account Management for Vendor

**Files:**
- Modify: `frontend/src/views/admin/AccountsView.vue`

**Step 1: Add vendor filtering**

- Add "来源" (Source) column showing "自有" or vendor name
- Add source_type filter dropdown
- In create/edit account modal, add optional vendor_id selector

**Step 2: Commit**

```bash
git add frontend/src/views/admin/AccountsView.vue
git commit -m "feat(vendor): extend account management with vendor filtering"
```

---

## Batch 7: Settings, Tests & Documentation

### Task 19: Vendor System Settings

**Files:**
- Modify: `backend/internal/service/setting_service.go` (add vendor setting keys)
- Modify: `frontend/src/views/admin/SettingsView.vue` (add vendor settings section)

**Step 1: Add setting keys**

Add constants:
- `vendor_system_enabled` (default: false)
- `vendor_health_check_enabled` (default: false)
- `vendor_balance_alert_enabled` (default: false)
- `vendor_auto_purchase_enabled` (default: false)
- `vendor_auto_pricing_enabled` (default: false)

**Step 2: Add settings UI**

Add "供应商系统" section in admin settings with toggle switches for each feature.

**Step 3: Commit**

```bash
git add backend/internal/service/setting_service.go frontend/src/views/admin/SettingsView.vue
git commit -m "feat(vendor): add vendor system settings with feature toggles"
```

---

### Task 20: Unit Tests

**Files:**
- Create: `backend/internal/service/vendor_service_test.go`
- Create: `backend/internal/service/vendor_health_service_test.go`
- Create: `backend/internal/service/vendor_balance_service_test.go`

**Step 1: Write vendor service tests**

Test CRUD operations, validation, error handling.

**Step 2: Write health check tests**

Test health check logic, auto-suspend, auto-resume.

**Step 3: Write balance service tests**

Test usage tracking, alert thresholds, cost calculation.

**Step 4: Run all tests**

Run: `cd /Users/zhouyinghui/work/ai/sub2api/backend && go test -tags=unit ./internal/service/... -run TestVendor`

**Step 5: Commit**

```bash
git add backend/internal/service/vendor_*_test.go
git commit -m "test(vendor): add unit tests for vendor services"
```

---

### Task 21: Usage Tutorial Documentation

**Files:**
- Create: `docs/vendor-resale-guide.md`

**Step 1: Write comprehensive usage guide**

Include:
1. 系统概述 — 什么是供应商采购系统
2. 快速开始 — 5分钟上手指南
3. 添加供应商 — 详细步骤（含截图位置说明）
4. 创建供应商账户 — 如何将供应商 API Key 录入系统
5. 配置调度策略 — 如何设置供应商池为主力、自有账户为兜底
6. 健康监控 — 如何启用和配置自动健康检查
7. 余额管理 — 余额预警和自动暂停
8. 成本分析 — 如何查看利润率和成本报表
9. 自动定价 — 如何启用动态定价
10. 自动采购（高级）— 自动发现和采购供应商
11. 常见问题 FAQ
12. 故障排除

**Step 2: Commit**

```bash
git add -f docs/vendor-resale-guide.md
git commit -m "docs: add comprehensive vendor resale system usage guide"
```

---

## Batch 8: Final Integration & Worktree Setup

### Task 22: Create Feature Branch Worktree

**Before starting any implementation**, create an isolated worktree:

```bash
cd /Users/zhouyinghui/work/ai/sub2api
git worktree add ../sub2api-vendor-resale feature/vendor-resale 2>/dev/null || {
    git branch feature/vendor-resale
    git worktree add ../sub2api-vendor-resale feature/vendor-resale
}
```

All implementation work happens in `/Users/zhouyinghui/work/ai/sub2api-vendor-resale/`.

---

## Execution Order

1. Task 22 (worktree) — FIRST
2. Tasks 1-4 (Batch 1: Database & Domain) — sequential
3. Tasks 5-6 (Batch 2: Admin API) — sequential
4. Tasks 7-9 (Batch 3: Health & Monitoring) — can parallelize 7+8
5. Tasks 10-11 (Batch 4: Protocol Conversion) — sequential
6. Tasks 12-14 (Batch 5: Auto-Purchase) — can parallelize all 3
7. Tasks 15-18 (Batch 6: Frontend) — sequential (15 first, then 16+17 parallel, then 18)
8. Tasks 19-21 (Batch 7: Settings, Tests, Docs) — can parallelize all 3

Total: 22 tasks, ~8 batches, estimated 3-4 sessions to complete.
