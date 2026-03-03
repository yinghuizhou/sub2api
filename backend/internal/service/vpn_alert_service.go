package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/vpnalertrule"
)

// Valid alert conditions.
var validConditions = map[string]bool{
	"tunnel_offline":       true,
	"consecutive_failover": true,
	"all_offline":          true,
	"config_stale":         true,
}

// CreateAlertRuleInput is the input for creating a new alert rule.
type CreateAlertRuleInput struct {
	Name       string `json:"name"`
	Condition  string `json:"condition"`
	Threshold  int    `json:"threshold"`
	WebhookURL string `json:"webhook_url"`
}

// UpdateAlertRuleInput is the input for updating an alert rule.
type UpdateAlertRuleInput struct {
	Name       *string `json:"name,omitempty"`
	Condition  *string `json:"condition,omitempty"`
	Threshold  *int    `json:"threshold,omitempty"`
	WebhookURL *string `json:"webhook_url,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
}

// AlertRuleDTO is the API representation of a VPN alert rule.
type AlertRuleDTO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Condition  string    `json:"condition"`
	Threshold  int       `json:"threshold"`
	WebhookURL string    `json:"webhook_url"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AlertPayload is the payload sent to webhook URLs when an alert triggers.
type AlertPayload struct {
	Alert     string                 `json:"alert"`
	Tunnel    string                 `json:"tunnel,omitempty"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// VpnAlertService handles VPN alert rule CRUD, evaluation, and webhook dispatch.
type VpnAlertService struct {
	client     *ent.Client
	eventSvc   *VpnEventService
	httpClient *http.Client
}

// NewVpnAlertService creates a new VpnAlertService.
func NewVpnAlertService(client *ent.Client, eventSvc *VpnEventService) *VpnAlertService {
	return &VpnAlertService{
		client:     client,
		eventSvc:   eventSvc,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Create validates the condition and creates a new alert rule.
func (s *VpnAlertService) Create(ctx context.Context, input CreateAlertRuleInput) (*AlertRuleDTO, error) {
	if !validConditions[input.Condition] {
		return nil, fmt.Errorf("invalid condition %q, must be one of: tunnel_offline, consecutive_failover, all_offline, config_stale", input.Condition)
	}

	rule, err := s.client.VpnAlertRule.Create().
		SetName(input.Name).
		SetCondition(input.Condition).
		SetThreshold(input.Threshold).
		SetWebhookURL(input.WebhookURL).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create alert rule: %w", err)
	}

	dto := alertRuleToDTO(rule)
	return &dto, nil
}

// List returns all alert rules ordered by created_at.
func (s *VpnAlertService) List(ctx context.Context) ([]AlertRuleDTO, error) {
	rules, err := s.client.VpnAlertRule.Query().
		Order(vpnalertrule.ByCreatedAt(sql.OrderAsc())).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}

	dtos := make([]AlertRuleDTO, len(rules))
	for i, r := range rules {
		dtos[i] = alertRuleToDTO(r)
	}
	return dtos, nil
}

// Update updates the specified fields of an alert rule.
func (s *VpnAlertService) Update(ctx context.Context, id int64, input UpdateAlertRuleInput) (*AlertRuleDTO, error) {
	if input.Condition != nil && !validConditions[*input.Condition] {
		return nil, fmt.Errorf("invalid condition %q, must be one of: tunnel_offline, consecutive_failover, all_offline, config_stale", *input.Condition)
	}

	update := s.client.VpnAlertRule.UpdateOneID(id)

	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Condition != nil {
		update.SetCondition(*input.Condition)
	}
	if input.Threshold != nil {
		update.SetThreshold(*input.Threshold)
	}
	if input.WebhookURL != nil {
		update.SetWebhookURL(*input.WebhookURL)
	}
	if input.Enabled != nil {
		update.SetEnabled(*input.Enabled)
	}

	rule, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update alert rule: %w", err)
	}

	dto := alertRuleToDTO(rule)
	return &dto, nil
}

// Delete removes an alert rule by ID.
func (s *VpnAlertService) Delete(ctx context.Context, id int64) error {
	err := s.client.VpnAlertRule.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	return nil
}

// EvaluateAndNotify evaluates all enabled alert rules against current tunnel status.
// Called when proxy status is updated.
func (s *VpnAlertService) EvaluateAndNotify(ctx context.Context, tunnelName, vpnStatus, healthStatus string, failures int) error {
	rules, err := s.client.VpnAlertRule.Query().
		Where(vpnalertrule.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query enabled alert rules: %w", err)
	}

	for _, rule := range rules {
		triggered := false
		var message string

		switch rule.Condition {
		case "tunnel_offline":
			if vpnStatus == "disconnected" && failures >= rule.Threshold {
				triggered = true
				message = fmt.Sprintf("Tunnel %q is offline (failures: %d, threshold: %d)", tunnelName, failures, rule.Threshold)
			}
		case "consecutive_failover":
			if failures >= rule.Threshold {
				triggered = true
				message = fmt.Sprintf("Tunnel %q has %d consecutive failures (threshold: %d)", tunnelName, failures, rule.Threshold)
			}
		case "all_offline":
			// Not applicable in single-tunnel context; skip for now.
			continue
		case "config_stale":
			// Handled by periodic check; skip for now.
			continue
		}

		if triggered {
			payload := AlertPayload{
				Alert:   rule.Name,
				Tunnel:  tunnelName,
				Message: message,
				Details: map[string]interface{}{
					"vpn_status":    vpnStatus,
					"health_status": healthStatus,
					"failures":      failures,
					"threshold":     rule.Threshold,
				},
				Timestamp: time.Now(),
			}

			if rule.WebhookURL != "" {
				if err := s.sendWebhook(rule.WebhookURL, payload); err != nil {
					log.Printf("[VpnAlert] webhook send failed for rule %q: %v", rule.Name, err)
				}
			}

			// Record alert event.
			if s.eventSvc != nil {
				_ = s.eventSvc.RecordEvent(ctx, tunnelName, "alert_triggered", map[string]interface{}{
					"rule_name":   rule.Name,
					"condition":   rule.Condition,
					"message":     message,
					"webhook_url": rule.WebhookURL,
				})
			}
		}
	}

	return nil
}

// sendWebhook sends an HTTP POST with JSON payload to the webhook URL.
func (s *VpnAlertService) sendWebhook(webhookURL string, payload AlertPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// alertRuleToDTO converts an Ent VpnAlertRule entity to a DTO.
func alertRuleToDTO(rule *ent.VpnAlertRule) AlertRuleDTO {
	return AlertRuleDTO{
		ID:         rule.ID,
		Name:       rule.Name,
		Condition:  rule.Condition,
		Threshold:  rule.Threshold,
		WebhookURL: rule.WebhookURL,
		Enabled:    rule.Enabled,
		CreatedAt:  rule.CreatedAt,
		UpdatedAt:  rule.UpdatedAt,
	}
}
