package service

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/vpnevent"
)

// ListEventsOpts specifies filters and pagination for listing VPN events.
type ListEventsOpts struct {
	TunnelName string
	EventType  string
	Since      *time.Time
	Page       int
	PageSize   int
}

// VpnEventDTO is the API representation of a VPN event.
type VpnEventDTO struct {
	ID         int64                  `json:"id"`
	TunnelName string                 `json:"tunnel_name"`
	EventType  string                 `json:"event_type"`
	Details    map[string]interface{} `json:"details"`
	CreatedAt  time.Time              `json:"created_at"`
}

// DashboardData contains aggregated VPN status for the dashboard.
type DashboardData struct {
	ActiveTunnels   int           `json:"active_tunnels"`
	OfflineTunnels  int           `json:"offline_tunnels"`
	DegradedTunnels int           `json:"degraded_tunnels"`
	TotalConfigs    int           `json:"total_configs"`
	RecentEvents    []VpnEventDTO `json:"recent_events"`
	StaleConfigs    []string      `json:"stale_configs"`
}

// VpnEventService handles VPN event recording, listing, and aggregation.
type VpnEventService struct {
	client *ent.Client
}

// NewVpnEventService creates a new VpnEventService.
func NewVpnEventService(client *ent.Client) *VpnEventService {
	return &VpnEventService{client: client}
}

// RecordEvent creates a new VPN event record.
func (s *VpnEventService) RecordEvent(ctx context.Context, tunnelName, eventType string, details map[string]interface{}) error {
	_, err := s.client.VpnEvent.Create().
		SetTunnelName(tunnelName).
		SetEventType(eventType).
		SetDetails(details).
		Save(ctx)
	return err
}

// ListEvents returns paginated VPN events with optional filters.
func (s *VpnEventService) ListEvents(ctx context.Context, opts ListEventsOpts) ([]VpnEventDTO, int, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 50
	}

	query := s.client.VpnEvent.Query()

	if opts.TunnelName != "" {
		query = query.Where(vpnevent.TunnelNameEQ(opts.TunnelName))
	}
	if opts.EventType != "" {
		query = query.Where(vpnevent.EventTypeEQ(opts.EventType))
	}
	if opts.Since != nil {
		query = query.Where(vpnevent.CreatedAtGTE(*opts.Since))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (opts.Page - 1) * opts.PageSize
	events, err := query.
		Order(vpnevent.ByCreatedAt(sql.OrderDesc())).
		Offset(offset).
		Limit(opts.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]VpnEventDTO, len(events))
	for i, e := range events {
		dtos[i] = eventToDTO(e)
	}
	return dtos, total, nil
}

// GetDashboard returns aggregated VPN status including tunnel counts,
// config count, and recent events.
func (s *VpnEventService) GetDashboard(ctx context.Context, vpnSvc *VpnAgentService) (*DashboardData, error) {
	data := &DashboardData{
		StaleConfigs: []string{},
	}

	// Get tunnel status counts from Agent.
	tunnels, err := vpnSvc.ListTunnels(ctx)
	if err == nil {
		for _, t := range tunnels {
			switch t.Health {
			case "healthy":
				data.ActiveTunnels++
			case "degraded":
				data.DegradedTunnels++
			default:
				data.OfflineTunnels++
			}
		}
	}

	// Get config count from Agent.
	configs, err := vpnSvc.ListConfigs(ctx)
	if err == nil {
		data.TotalConfigs = len(configs)
	}

	// Get recent events from database.
	events, err := s.client.VpnEvent.Query().
		Order(vpnevent.ByCreatedAt(sql.OrderDesc())).
		Limit(20).
		All(ctx)
	if err == nil {
		data.RecentEvents = make([]VpnEventDTO, len(events))
		for i, e := range events {
			data.RecentEvents[i] = eventToDTO(e)
		}
	} else {
		data.RecentEvents = []VpnEventDTO{}
	}

	return data, nil
}

// CleanupOldEvents removes events older than the given retention duration.
// Returns the number of deleted events.
func (s *VpnEventService) CleanupOldEvents(ctx context.Context, retention time.Duration) (int, error) {
	cutoff := time.Now().Add(-retention)
	count, err := s.client.VpnEvent.Delete().
		Where(vpnevent.CreatedAtLT(cutoff)).
		Exec(ctx)
	return count, err
}

// eventToDTO converts an Ent VpnEvent entity to a DTO.
func eventToDTO(e *ent.VpnEvent) VpnEventDTO {
	return VpnEventDTO{
		ID:         e.ID,
		TunnelName: e.TunnelName,
		EventType:  e.EventType,
		Details:    e.Details,
		CreatedAt:  e.CreatedAt,
	}
}
