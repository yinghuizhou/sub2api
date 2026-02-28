package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// CallbackClient sends status updates back to the Sub2API server.
type CallbackClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewCallbackClient creates a new CallbackClient.
func NewCallbackClient(baseURL, apiKey string) *CallbackClient {
	return &CallbackClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// proxyUpdatePayload is the JSON body sent to Sub2API.
type proxyUpdatePayload struct {
	VpnExitIP    string `json:"vpn_exit_ip,omitempty"`
	VpnStatus    string `json:"vpn_status,omitempty"`
	HealthStatus string `json:"health_status,omitempty"`
	LatencyMs    *int   `json:"latency_ms,omitempty"`
}

// UpdateProxyStatus updates a proxy record in Sub2API.
// PUT /api/v1/admin/proxies/{id}
func (c *CallbackClient) UpdateProxyStatus(proxyID int, exitIP, vpnStatus, healthStatus string, latencyMs int) error {
	if c.baseURL == "" || c.apiKey == "" || proxyID == 0 {
		return nil // no callback configured or no proxy ID
	}

	payload := proxyUpdatePayload{
		VpnExitIP:    exitIP,
		VpnStatus:    vpnStatus,
		HealthStatus: healthStatus,
	}
	if latencyMs > 0 {
		payload.LatencyMs = &latencyMs
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/admin/proxies/%d", c.baseURL, proxyID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Sub2API returned %d", resp.StatusCode)
	}
	return nil
}

// ReportEvent sends a VPN event to the Sub2API backend.
// This is fire-and-forget — call it with `go` from health check code.
func (c *CallbackClient) ReportEvent(tunnelName, eventType string, details map[string]interface{}) {
	if c.baseURL == "" || c.apiKey == "" {
		return
	}

	payload := map[string]interface{}{
		"tunnel_name": tunnelName,
		"event_type":  eventType,
		"details":     details,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[callback] failed to marshal event: %v", err)
		return
	}

	url := c.baseURL + "/api/v1/admin/vpn/events/report"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[callback] failed to create event request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[callback] failed to send event: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("[callback] event report returned status %d", resp.StatusCode)
	}
}

// ReportStatus sends tunnel status updates to Sub2API for all tunnels with a ProxyID.
func (c *CallbackClient) ReportStatus(tunnels []TunnelInfo) error {
	for _, t := range tunnels {
		if t.ProxyID == 0 {
			continue
		}
		if err := c.UpdateProxyStatus(t.ProxyID, t.ExitIP, t.Status, t.Health, t.LatencyMs); err != nil {
			log.Printf("Callback error for tunnel %q (proxy %d): %v", t.Name, t.ProxyID, err)
		}
	}
	return nil
}
