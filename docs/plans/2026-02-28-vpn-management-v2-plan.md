# VPN Management V2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enhance existing VPN management with config push API, smart failover, and monitoring dashboard.

**Architecture:** Extend VPN Agent + Sub2API backend. No new services. Three modules: (1) Config Push API for browser extension, (2) Smart failover with cross-region switching and weighted scoring, (3) Monitoring dashboard with event logging and alerts.

**Tech Stack:** Go 1.25+ (Gin, Ent ORM), Vue 3 + TypeScript + Vite, PostgreSQL 15+

**Design doc:** `docs/plans/2026-02-28-vpn-management-v2-design.md`

---

## Plan Index

| Phase | File | Description |
|-------|------|-------------|
| Phase 1 | [phase1-config-push.md](phase1-config-push.md) | Config Push API (Agent + Backend + Frontend) |
| Phase 2 | [phase2-smart-failover.md](phase2-smart-failover.md) | Smart Failover (Agent enhancements) |
| Phase 3 | [phase3-monitoring.md](phase3-monitoring.md) | Monitoring Dashboard (DB + Backend + Frontend) |

## Dependency Graph

```
Phase 1 (Config Push) ──┐
                        ├──→ Phase 3 (Monitoring)
Phase 2 (Failover) ─────┘
```

Phase 1 and Phase 2 are independent, can be done in parallel.
Phase 3 depends on event logging infrastructure from both.

## Key File Paths

### VPN Agent (`backend/cmd/vpn-agent/`)
- `server.go` — Route registration
- `server_handlers.go` — HTTP handlers
- `config_store.go` — Config file management
- `health_failover.go` — Failover logic + stability scores
- `health.go` — Health check loop
- `callback.go` — Callback to Sub2API
- `tunnel.go` — TunnelManager

### Backend (`backend/internal/`)
- `handler/admin/vpn_handler.go` — VPN admin handlers
- `service/vpn_agent_service.go` — VPN agent proxy service
- `server/routes/admin.go` — Route registration
- `ent/schema/proxy.go` — Proxy entity schema

### Frontend (`frontend/src/`)
- `views/admin/VpnView.vue` — VPN management page
- `api/vpn.ts` — VPN API client
- `router/index.ts` — Route definitions

### Database
- `backend/migrations/` — SQL migrations (next: `068_`)

## Conventions

- Migration files: `NNN_description.sql`
- Ent schema changes require: `cd backend && go generate ./ent`
- Wire DI changes require: `cd backend && go generate ./cmd/server`
- Frontend uses `pnpm` (never npm)
- API response format: `{"code": 0, "message": "success", "data": {...}}`
- Admin auth: `x-api-key` header
- All tests: `go test -tags=unit ./...`
